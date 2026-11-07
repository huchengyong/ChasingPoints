#!/usr/bin/env bash
# 发布管理后台（静态站）到测试站或生产站：类型检查与测试 → 按环境构建 → 核对产物的 API 地址 → 上传到服务器暂存目录 →
# 备份站点目录 → 叠加 assets 并原子替换 index.html → 校验文件与线上访问。
# 旧的哈希 assets 保留在站点里，已打开的页面不会因为找不到旧 chunk 而报错；每次发布前的完整站点都在 .admin-backups 下。
# 后端发布见 ../backend/deploy-dev.sh 与 ../backend/deploy-prod.sh。
#
# 用法（在 admin/ 下执行）：
#   ./deploy.sh dev [--dry-run]     测试站 admin-zhuifen.kekemate.cn；允许未提交的改动（版本号加 -dirty）
#   ./deploy.sh prod [--dry-run]    生产站 admin.zhuifen.cn；要求代码已提交并推送，并需输入 yes 确认
#
# --dry-run：只做测试、构建和产物检查，不上传，不改服务器。
# 依赖：~/.ssh/config 中的 chasingpoints-dev / chasingpoints-prod 别名；已在 admin/ 下执行过 npm install。
set -Eeuo pipefail

usage() { sed -n '2,12p' "$0"; }
die() { printf '%s\n' "$*" >&2; exit 1; }

ENV_NAME="${1:-}"
case "$ENV_NAME" in
    dev|prod) ;;
    -h|--help) usage; exit 0 ;;
    "") usage >&2; exit 2 ;;
    *) usage >&2; die "未知环境：$ENV_NAME" ;;
esac

DRY_RUN=0
shift
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=1 ;;
        *) die "未知参数：$arg" ;;
    esac
done

case "$ENV_NAME" in
    dev)
        SSH_HOST=chasingpoints-dev
        SITE=admin-zhuifen.kekemate.cn
        API_HOST=api-zhuifen.kekemate.cn
        OTHER_API_HOST=api.zhuifen.cn
        BUILD_MODE=development
        # 测试站 443 由 xray 占用，nginx 在 80 端口，按 Host 头访问
        VERIFY=host
        ;;
    prod)
        SSH_HOST=chasingpoints-prod
        SITE=admin.zhuifen.cn
        API_HOST=api.zhuifen.cn
        OTHER_API_HOST=api-zhuifen.kekemate.cn
        BUILD_MODE=production
        VERIFY=resolve
        ;;
esac
SSH=(ssh -o BatchMode=yes -o ConnectTimeout=15 -o LogLevel=ERROR "$SSH_HOST")
# 每次发布都检查的页面：前端路由依赖 SPA 回退，应全部返回 200
ROUTES="/ /gear/pending /gear/products /gear/brands"

ADMIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -d "$ADMIN_DIR/node_modules" ] || die "缺少依赖：请先在 admin/ 下执行 npm install"

REVISION="$(git -C "$ADMIN_DIR" rev-parse --short=12 HEAD)"
if [ -n "$(git -C "$ADMIN_DIR" status --porcelain -- .)" ]; then
    if [ "$ENV_NAME" = prod ]; then
        git -C "$ADMIN_DIR" status --short -- . >&2
        die "生产只发布已提交的代码，admin/ 有未提交的改动（见上方）。"
    fi
    REVISION="${REVISION}-dirty"
fi
if [ "$ENV_NAME" = prod ] && [ -z "$(git -C "$ADMIN_DIR" branch -r --contains HEAD)" ]; then
    die "HEAD 尚未推送到远端，请先推送再发布生产。"
fi
RELEASE_ID="$(date +%Y%m%d%H%M%S)-${REVISION}"

printf '运行测试...\n'
(cd "$ADMIN_DIR" && node --experimental-strip-types --test tests/*.test.mjs)

WORK="$(mktemp -d "${TMPDIR:-/tmp}/admin-deploy.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT
BUILD="$WORK/dist"

printf '类型检查与构建（%s）...\n' "$BUILD_MODE"
(cd "$ADMIN_DIR" && npx vue-tsc && npx vite build --mode "$BUILD_MODE" --outDir "$BUILD" --emptyOutDir)

[ -f "$BUILD/index.html" ] && [ -d "$BUILD/assets" ] || die "构建产物不完整：缺少 index.html 或 assets/"
extra="$(find "$BUILD" -mindepth 1 -maxdepth 1 ! -name index.html ! -name assets)"
[ -z "$extra" ] || die "构建产物有不会上传的顶层文件：$extra"
grep -rqF "https://$API_HOST" "$BUILD/assets" || die "构建产物里找不到 https://$API_HOST，API 地址不对"
if grep -rqF "https://$OTHER_API_HOST" "$BUILD/assets"; then
    die "构建产物里出现了另一个环境的 API 地址：https://$OTHER_API_HOST"
fi

# 上传时一起带上校验清单，服务器端据此确认每个文件都完整写入
(cd "$BUILD" && find . -type f | sort | xargs shasum -a 256) > "$WORK/SHA256SUMS"
ASSET_REFS="$(grep -o '/assets/[^"]*' "$BUILD/index.html" | sort -u | tr '\n' ' ')"

if [ "$DRY_RUN" = 1 ]; then
    printf '预演结束（--dry-run），未上传。版本 %s，产物 %s 个文件。\n' "$RELEASE_ID" "$(find "$BUILD" -type f | wc -l | tr -d ' ')"
    exit 0
fi

if [ "$ENV_NAME" = prod ]; then
    printf '即将发布到生产 %s（版本 %s）：覆盖 assets 并替换 index.html，旧版备份在服务器 .admin-backups。\n输入 yes 继续：' "$SITE" "$RELEASE_ID"
    read -r answer || answer=""
    if [ "$answer" != yes ]; then
        printf '已取消。\n'
        exit 0
    fi
fi

REMOTE_STAGE="/tmp/admin-release-$RELEASE_ID"
printf '上传到 %s...\n' "$SSH_HOST"
tar --no-xattrs -czf - -C "$BUILD" . -C "$WORK" SHA256SUMS |
    "${SSH[@]}" "rm -rf '$REMOTE_STAGE' && mkdir -p '$REMOTE_STAGE' && tar -C '$REMOTE_STAGE' -xzf -"

printf '替换站点文件...\n'
"${SSH[@]}" bash -s -- "$SITE" "$RELEASE_ID" "$REMOTE_STAGE" <<'REMOTE'
set -Eeuo pipefail
SITE=$1; RELEASE=$2; STAGE=$3
ROOT=/www/wwwroot
SITE_DIR="$ROOT/$SITE"
BACKUP="$ROOT/.admin-backups/$SITE/$RELEASE"
SUDO=""
[ "$(id -u)" = 0 ] || SUDO="sudo -n"

[ -d "$SITE_DIR" ] || { echo "站点目录不存在：$SITE_DIR" >&2; exit 1; }
[ -f "$STAGE/index.html" ] && [ -d "$STAGE/assets" ] && [ -f "$STAGE/SHA256SUMS" ] || { echo "暂存目录不完整：$STAGE" >&2; exit 1; }
[ ! -e "$BACKUP" ] || { echo "备份目录已存在：$BACKUP" >&2; exit 1; }

$SUDO mkdir -p "$ROOT/.admin-backups/$SITE"
$SUDO cp -a "$SITE_DIR" "$BACKUP"
echo "备份：$BACKUP"

$SUDO cp -a "$STAGE/assets/." "$SITE_DIR/assets/"
$SUDO chown -R www:www "$SITE_DIR/assets"
$SUDO install -o www -g www -m 644 "$STAGE/index.html" "$SITE_DIR/.index.html.new"
$SUDO mv -f "$SITE_DIR/.index.html.new" "$SITE_DIR/index.html"

if ! (cd "$SITE_DIR" && sha256sum -c --quiet "$STAGE/SHA256SUMS"); then
    echo "校验失败：站点文件与本次构建不一致。旧版在 $BACKUP，可用它恢复。" >&2
    exit 1
fi
rm -rf "$STAGE"
echo "已替换：$SITE_DIR"
REMOTE

printf '线上检查...\n'
"${SSH[@]}" bash -s -- "$SITE" "$VERIFY" $ASSET_REFS $ROUTES <<'REMOTE'
set -u
SITE=$1; MODE=$2; shift 2
failed=0
for path in "$@"; do
    if [ "$MODE" = resolve ]; then
        code=$(curl -s -o /dev/null -w '%{http_code}' --resolve "$SITE:443:127.0.0.1" "https://$SITE$path")
    else
        code=$(curl -s -o /dev/null -w '%{http_code}' -H "Host: $SITE" "http://127.0.0.1$path")
    fi
    echo "$code $path"
    [ "$code" = 200 ] || failed=1
done
exit $failed
REMOTE

printf '发布完成：%s，版本 %s。\n' "$SITE" "$RELEASE_ID"
