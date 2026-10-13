#!/usr/bin/env bash
# 重启生产后端 api.zhuifen.cn。
# 进程由手动拉起、不由宝塔托管，所以不要用面板的重启按钮（面板不认识这个进程，可能起第二个实例）。
#
# 用法：
#   backend/restart-prod.sh           停止旧进程、启动新进程，并验证健康状态（会有几秒停机）
#   backend/restart-prod.sh --status  只读查看进程和健康状态，不做任何改动
#
# 依赖：~/.ssh/config 中的 chasingpoints-prod 别名（可用 CHASING_POINTS_PROD_HOST 覆盖）。
set -Eeuo pipefail

HOST="${CHASING_POINTS_PROD_HOST:-chasingpoints-prod}"
MODE=restart
case "${1:-}" in
    "") ;;
    --status) MODE=status ;;
    -h|--help) sed -n '2,8p' "$0"; exit 0 ;;
    *) printf 'Unknown argument: %s\n' "$1" >&2; exit 2 ;;
esac

SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=15 -o LogLevel=ERROR)

if ! ssh "${SSH_OPTS[@]}" "$HOST" true; then
    printf 'Cannot reach %s over SSH. Check ~/.ssh/config and your key.\n' "$HOST" >&2
    exit 1
fi

if [ "$MODE" = restart ]; then
    printf 'Restarting production API on %s (expect a few seconds of downtime)...\n' "$HOST" >&2
fi

ssh "${SSH_OPTS[@]}" "$HOST" "bash -s -- $MODE" <<'REMOTE'
MODE="$1"
PROJ=/www/wwwroot/api.zhuifen.cn
BIN="$PROJ/chasing_points"
LOG=/www/wwwlogs/go/chasing_points.log
LOCAL_HEALTH=http://127.0.0.1:8808/api/admin/exists
# 应用日志目录：必须与 yaml 的 Log.Path 一致（当前为绝对路径）。改了 Log.Path 要同步这里。
APP_LOG_DIR=/www/wwwlogs/go/chasing_points

running_pids() { pgrep -x -u www chasing_points || true; }
local_http() { curl -s -o /dev/null -w '%{http_code}' --max-time 3 "$LOCAL_HEALTH" || true; }
public_http() {
    curl -s -o /dev/null -w 'http=%{http_code} tls_verify=%{ssl_verify_result}' --max-time 10 \
        --resolve api.zhuifen.cn:443:127.0.0.1 https://api.zhuifen.cn/api/admin/exists || true
}

show_status() {
    local p
    p=$(running_pids | head -1)
    if [ -n "$p" ]; then
        echo "process: pid=$p running for $(ps -o etime= -p "$p" | tr -d ' ')"
    else
        echo "process: not running"
    fi
    echo "local health (127.0.0.1:8808): http=$(local_http)"
    echo "public health (api.zhuifen.cn, via local nginx): $(public_http)"
}

case "$MODE" in
    status)
        show_status
        exit 0
        ;;
    restart)
        ;;
    *)
        echo "unknown mode: $MODE" >&2
        exit 2
        ;;
esac

# 启动前检查，避免停掉旧进程后才发现新进程起不来
if [ ! -x "$BIN" ]; then echo "binary missing or not executable: $BIN" >&2; exit 1; fi
if [ ! -f "$PROJ/.env" ]; then echo ".env missing: $PROJ/.env" >&2; exit 1; fi

# 1) 停止旧进程：先 SIGTERM 让服务优雅退出，30 秒后仍在则 SIGKILL
if [ -n "$(running_pids)" ]; then
    echo "stopping old process(es): $(running_pids | tr '\n' ' ')"
    for p in $(running_pids); do kill -TERM "$p"; done
    for _ in $(seq 1 30); do
        [ -z "$(running_pids)" ] && break
        sleep 1
    done
    if [ -n "$(running_pids)" ]; then
        echo "still running after 30s, sending SIGKILL"
        for p in $(running_pids); do kill -KILL "$p"; done
        sleep 2
    fi
    if [ -n "$(running_pids)" ]; then
        echo "old process did not exit; aborting to avoid a second instance" >&2
        exit 1
    fi
else
    echo "no running process found; starting fresh"
fi

# 2) 启动新进程：与之前手动启动的方式一致（用户 www、工作目录、日志文件）
LOG_OFFSET=$(stat -c %s "$LOG" 2>/dev/null || echo 0)
APP_ERR_OFFSET=$(stat -c %s "$APP_LOG_DIR/error.log" 2>/dev/null || echo 0)
APP_SEV_OFFSET=$(stat -c %s "$APP_LOG_DIR/severe.log" 2>/dev/null || echo 0)
# su 本身的 stdio 也要断开：否则会继承远端 bash 的 stdin（脚本内容）和 SSH 通道，可能导致会话不退出
su -s /bin/bash www -c "cd $PROJ && setsid nohup $BIN >> $LOG 2>&1 < /dev/null &" >/dev/null 2>&1 </dev/null

# 3) 等待本机健康检查通过
healthy=0
for _ in $(seq 1 60); do
    if [ "$(local_http)" = 200 ]; then healthy=1; break; fi
    sleep 1
done
if [ "$healthy" != 1 ]; then
    echo "service did not become healthy within 60s; last log lines:" >&2
    tail -n 40 "$LOG" >&2
    exit 1
fi

# 4) 启动后检查：
#    - stdout 日志里的 panic / 配置校验失败：直接判定失败
#    - 应用日志 logs/severe.log 有新内容：判定失败；logs/error.log 有新内容：只提示
errors=$(tail -c +$((LOG_OFFSET + 1)) "$LOG" | grep -c -i -E 'panic|fatal|生产安全配置校验失败' || true)
app_errors=$(tail -c +$((APP_ERR_OFFSET + 1)) "$APP_LOG_DIR/error.log" 2>/dev/null | wc -l | tr -d ' ')
app_severe=$(tail -c +$((APP_SEV_OFFSET + 1)) "$APP_LOG_DIR/severe.log" 2>/dev/null | wc -l | tr -d ' ')
echo "started: pid=$(running_pids | head -1)"
echo "local health: http=200"
echo "public health: $(public_http)"
echo "stdout error-like lines since start: $errors"
echo "app log new lines since start: error=$app_errors severe=$app_severe"
if [ "$app_errors" -gt 0 ]; then
    echo "new app errors (last 5 lines, may include request context):" >&2
    tail -c +$((APP_ERR_OFFSET + 1)) "$APP_LOG_DIR/error.log" | tail -n 5 >&2
fi
if [ "$errors" -gt 0 ] || [ "$app_severe" -gt 0 ]; then
    echo "fatal errors found; last lines of stdout log:" >&2
    tail -n 40 "$LOG" >&2
    exit 1
fi
REMOTE
