#!/usr/bin/env bash
# 发布生产后端 api.zhuifen.cn：本地测试并编译 Linux amd64 二进制，上传到服务器暂存目录；
# 人工确认后备份数据库、执行 goose 迁移、替换二进制，再调用 restart-prod.sh 重启并验证，失败时恢复旧二进制。
# 只发布二进制和迁移；线上 etc/chasing_points-api.yaml 与 .env 和仓库版本不同，本脚本不改动，配置变更需人工处理。
# 发布成功后只保留最近 3 个暂存目录（含其中的旧二进制和数据库备份），发布失败时不清理。
#
# 用法：
#   backend/deploy-prod.sh
#
# 依赖：~/.ssh/config 中的 chasingpoints-prod 别名（可用 CHASING_POINTS_PROD_HOST 覆盖），HEAD 已推送到远端。
set -Eeuo pipefail

HOST="${CHASING_POINTS_PROD_HOST:-chasingpoints-prod}"
GOOSE_VERSION=v3.28.0
KEEP_STAGES=3
SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=15 -o LogLevel=ERROR)

case "${1:-}" in
    "") ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
    *) printf 'Unknown argument: %s\n' "$1" >&2; exit 2 ;;
esac

backend_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1) 只发布已提交、已推送的代码，线上二进制里的 vcs.revision 才能对应到仓库
if [ -n "$(git -C "$backend_dir" status --porcelain --untracked-files=no -- .)" ] \
    || [ -n "$(git -C "$backend_dir" ls-files --others --exclude-standard -- '*.go' 'migrations/')" ]; then
    printf 'backend 有未提交的代码或迁移，生产只发布已提交的版本：\n' >&2
    git -C "$backend_dir" status --short -- . >&2
    exit 1
fi
if [ -z "$(git -C "$backend_dir" branch -r --contains HEAD)" ]; then
    printf 'HEAD 尚未推送到任何远端分支，请先推送再发布生产。\n' >&2
    exit 1
fi
revision="$(git -C "$backend_dir" rev-parse HEAD)"
release_id="$(date +%Y%m%d%H%M%S)-${revision:0:12}"

# 2) 预检：读取线上版本，确认服务器上有没有对应版本的 goose
preflight="$(ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- '$GOOSE_VERSION'" <<'REMOTE'
set -Eeuo pipefail
PROJ=/www/wwwroot/api.zhuifen.cn
[ -x "$PROJ/chasing_points" ] || { echo "binary missing: $PROJ/chasing_points" >&2; exit 1; }
[ -f "$PROJ/.env" ] || { echo ".env missing: $PROJ/.env" >&2; exit 1; }
grep -q '^MYSQL_DSN=' "$PROJ/.env" || { echo "MYSQL_DSN missing from $PROJ/.env" >&2; exit 1; }
rev=$(grep -a -o -m1 'vcs.revision=[0-9a-f]*' "$PROJ/chasing_points" | head -1 | cut -d= -f2 || true)
echo "deployed=${rev:-unknown}"
if [ -x "$PROJ/deploy-tools/goose-$1" ]; then echo "goose=present"; else echo "goose=missing"; fi
REMOTE
)" || { printf 'Preflight on %s failed. Check ~/.ssh/config and your key.\n' "$HOST" >&2; exit 1; }
deployed="$(printf '%s\n' "$preflight" | sed -n 's/^deployed=//p')"
goose_state="$(printf '%s\n' "$preflight" | sed -n 's/^goose=//p')"

printf '线上版本：%s\n待发布版本：%s\n' "$deployed" "$revision"
if [ "$deployed" = "$revision" ]; then
    printf '生产已经是该版本，无需发布。只需重启请用 restart-prod.sh。\n'
    exit 0
fi
if git -C "$backend_dir" merge-base --is-ancestor "$deployed" HEAD 2>/dev/null; then
    printf '\n本次包含的 backend 提交：\n'
    git -C "$backend_dir" log --oneline "$deployed..HEAD" -- . | sed 's/^/  /'
    if ! git -C "$backend_dir" diff --quiet "$deployed" HEAD -- etc/chasing_points-api.yaml .env.example; then
        printf '\n注意：仓库配置模板有变化，脚本不会改线上 yaml/.env，请先人工核对：\n' >&2
        git -C "$backend_dir" diff --stat "$deployed" HEAD -- etc/chasing_points-api.yaml .env.example >&2
    fi
else
    printf '\n注意：线上版本不是 HEAD 的祖先（未知版本或回退发布），请确认这是预期的。\n' >&2
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/chasing-points-deploy-prod.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
archive_name="chasing-points-${release_id}.tar.gz"
remote_archive="/tmp/$archive_name"

# 3) 本地测试和编译
printf '\nTesting backend...\n'
(cd "$backend_dir" && go test ./...)

printf 'Building Linux amd64 release %s...\n' "$release_id"
mkdir "$work_dir/release"
(cd "$backend_dir" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$work_dir/release/chasing_points" .)
cp -R "$backend_dir/migrations" "$work_dir/release/migrations"
if [ "$goose_state" = missing ]; then
    printf 'Building goose %s for the server...\n' "$GOOSE_VERSION"
    mkdir "$work_dir/goose-build"
    (
        cd "$work_dir/goose-build"
        go mod init goose-build >/dev/null 2>&1
        go get "github.com/pressly/goose/v3/cmd/goose@$GOOSE_VERSION"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$work_dir/release/goose" github.com/pressly/goose/v3/cmd/goose
    )
fi
COPYFILE_DISABLE=1 tar -C "$work_dir/release" -czf "$work_dir/$archive_name" .

# 4) 上传到暂存目录并列出待执行迁移，此时线上服务和数据库都还没有变化
cleanup_stage() {
    ssh "${SSH_OPTS[@]}" "$HOST" "rm -rf '/www/wwwroot/api.zhuifen.cn/deploy-stage-$release_id'" || true
}

printf 'Uploading release...\n'
scp "${SSH_OPTS[@]}" "$work_dir/$archive_name" "${HOST}:${remote_archive}"

stage_output="$(ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- '$release_id' '$remote_archive' '$GOOSE_VERSION'" <<'REMOTE'
set -Eeuo pipefail
PROJ=/www/wwwroot/api.zhuifen.cn
STAGE="$PROJ/deploy-stage-$1"
GOOSE="$PROJ/deploy-tools/goose-$3"
trap 'rm -f "$2"' EXIT
umask 077
[ ! -e "$STAGE" ] || { echo "stage already exists: $STAGE" >&2; exit 1; }
mkdir -p "$STAGE"
tar -xzf "$2" -C "$STAGE"
chmod 755 "$STAGE/chasing_points"
if [ -f "$STAGE/goose" ]; then
    mkdir -p "$PROJ/deploy-tools"
    install -m 700 "$STAGE/goose" "$GOOSE"
    rm -f "$STAGE/goose"
fi
DSN=$(sed -n 's/^MYSQL_DSN=//p' "$PROJ/.env" | head -1 | sed -e 's/^["'\'']//' -e 's/["'\'']$//')
status=$("$GOOSE" -dir "$STAGE/migrations" mysql "$DSN" status 2>&1)
echo "pending=$(printf '%s\n' "$status" | grep -c 'Pending' || true)"
printf '%s\n' "$status" | grep 'Pending' | sed 's/^/  /' || true
REMOTE
)" || { printf 'Staging on %s failed.\n' "$HOST" >&2; cleanup_stage; exit 1; }
pending="$(printf '%s\n' "$stage_output" | sed -n 's/^pending=//p')"
printf '\n待执行迁移：%s 个\n' "$pending"
printf '%s\n' "$stage_output" | grep -v '^pending=' || true

# 5) 人工确认
printf '\n即将发布到生产 %s：' "$HOST"
if [ "$pending" != 0 ]; then printf '先备份数据库并执行 %s 个迁移，' "$pending"; fi
printf '再替换二进制并重启（会有几秒停机）。\n输入 yes 继续：'
answer=""
read -r answer || true
if [ "$answer" != yes ]; then
    printf '已取消，清理服务器暂存目录。\n'
    cleanup_stage
    exit 1
fi

# 6) 备份数据库、执行迁移、替换二进制；线上进程仍是旧版本，直到下一步重启
ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- '$release_id' '$GOOSE_VERSION' '$pending'" <<'REMOTE'
set -Eeuo pipefail
PROJ=/www/wwwroot/api.zhuifen.cn
STAGE="$PROJ/deploy-stage-$1"
GOOSE="$PROJ/deploy-tools/goose-$2"
umask 077
DSN=$(sed -n 's/^MYSQL_DSN=//p' "$PROJ/.env" | head -1 | sed -e 's/^["'\'']//' -e 's/["'\'']$//')

if [ "$3" != 0 ]; then
    # DSN 格式：user:password@tcp(host:port)/db?params
    user=${DSN%%:*}; rest=${DSN#*:}
    password=${rest%@tcp(*}; hostport=${rest#*@tcp(}; hostport=${hostport%%)*}
    db=${rest#*)/}; db=${db%%\?*}
    dump="$STAGE/db-before-migrations-$(date +%Y%m%dT%H%M%S).sql"
    echo "backing up database $db to $dump"
    MYSQL_PWD="$password" mysqldump -h "${hostport%:*}" -P "${hostport##*:}" -u "$user" \
        --single-transaction --routines --triggers "$db" > "$dump"
    echo "running migrations"
    if ! "$GOOSE" -dir "$STAGE/migrations" mysql "$DSN" up; then
        echo "migration failed; binary not switched. database backup: $dump" >&2
        exit 1
    fi
fi

mkdir -p "$STAGE/backup"
cp -p "$PROJ/chasing_points" "$STAGE/backup/chasing_points.live"
cp -f "$STAGE/chasing_points" "$PROJ/chasing_points.tmp"
chown --reference="$STAGE/backup/chasing_points.live" "$PROJ/chasing_points.tmp"
chmod 755 "$PROJ/chasing_points.tmp"
mv -f "$PROJ/chasing_points.tmp" "$PROJ/chasing_points"
echo "new binary installed; previous binary saved to $STAGE/backup/chasing_points.live"
REMOTE

# 7) 重启并验证；失败则恢复旧二进制再重启（数据库迁移不会自动回滚）
if "$backend_dir/restart-prod.sh"; then
    printf '\nDeployed %s to %s\n' "$release_id" "$HOST"
    # 暂存目录名以时间戳开头，按名称倒序即从新到旧
    ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- '$KEEP_STAGES'" <<'REMOTE' || printf '清理旧暂存目录失败，可稍后手动处理。\n' >&2
set -Eeuo pipefail
PROJ=/www/wwwroot/api.zhuifen.cn
find "$PROJ" -mindepth 1 -maxdepth 1 -type d -name 'deploy-stage-*' | sort -r | tail -n +$(( $1 + 1 )) |
    while read -r dir; do
        echo "removing old stage: $dir"
        rm -rf -- "$dir"
    done
REMOTE
    exit 0
fi

printf '\n新版本重启验证失败，恢复旧二进制...\n' >&2
ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- '$release_id'" <<'REMOTE'
set -Eeuo pipefail
PROJ=/www/wwwroot/api.zhuifen.cn
BACKUP="$PROJ/deploy-stage-$1/backup/chasing_points.live"
cp -p "$BACKUP" "$PROJ/chasing_points.tmp"
mv -f "$PROJ/chasing_points.tmp" "$PROJ/chasing_points"
REMOTE
if "$backend_dir/restart-prod.sh"; then
    printf '已回滚到 %s 并恢复服务。数据库迁移未回滚，如需回滚请使用暂存目录中的备份。\n' "$deployed" >&2
else
    printf '回滚后的重启也失败了，需要立即人工处理！\n' >&2
fi
exit 1
