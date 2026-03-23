#!/bin/bash

# Goose 数据库迁移脚本
# 自动从 .env 文件加载数据库连接信息

set -e

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
MIGRATIONS_DIR="$SCRIPT_DIR/migrations"

# 检查 .env 文件是否存在
if [ ! -f "$ENV_FILE" ]; then
    echo "错误: 未找到 .env 文件: $ENV_FILE"
    exit 1
fi

# 加载 .env 文件中的环境变量
export $(grep -v '^#' "$ENV_FILE" | grep -v '^$' | xargs)

# 构建数据库连接字符串
DB_DSN="${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?parseTime=true"

# 显示帮助信息
show_help() {
    echo "用法: $0 <命令> [选项]"
    echo ""
    echo "命令:"
    echo "  up              执行所有待执行的迁移"
    echo "  up-one          执行下一个待执行的迁移"
    echo "  down            回滚最后一次迁移"
    echo "  down-to <版本>  回滚到指定版本"
    echo "  redo            回滚并重新执行最后一次迁移"
    echo "  status          显示迁移状态"
    echo "  version         显示当前数据库版本"
    echo "  create <名称>   创建新的迁移文件"
    echo "  fix             修复迁移版本顺序"
    echo "  validate        验证迁移文件"
    echo ""
    echo "示例:"
    echo "  $0 up            # 执行所有迁移"
    echo "  $0 status        # 查看迁移状态"
    echo "  $0 create users  # 创建名为 users 的迁移文件"
    echo ""
    echo "数据库连接信息 (从 .env 加载):"
    echo "  Host: ${MYSQL_HOST}:${MYSQL_PORT}"
    echo "  Database: ${MYSQL_DATABASE}"
    echo "  User: ${MYSQL_USER}"
}

# 检查 goose 是否已安装
check_goose() {
    if ! command -v goose &> /dev/null; then
        echo "错误: goose 未安装"
        echo "请运行以下命令安装: go install github.com/pressly/goose/v3/cmd/goose@latest"
        exit 1
    fi
}

# 主逻辑
main() {
    check_goose

    case "${1:-help}" in
        up)
            echo "正在执行所有迁移..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" up
            ;;
        up-one)
            echo "正在执行下一个迁移..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" up-by-one
            ;;
        down)
            echo "正在回滚最后一次迁移..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" down
            ;;
        down-to)
            if [ -z "$2" ]; then
                echo "错误: 请指定目标版本"
                exit 1
            fi
            echo "正在回滚到版本 $2..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" down-to "$2"
            ;;
        redo)
            echo "正在重做最后一次迁移..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" redo
            ;;
        status)
            echo "迁移状态:"
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" status
            ;;
        version)
            echo "当前数据库版本:"
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" version
            ;;
        create)
            if [ -z "$2" ]; then
                echo "错误: 请指定迁移名称"
                exit 1
            fi
            echo "正在创建迁移文件: $2"
            goose -dir "$MIGRATIONS_DIR" create "$2" sql
            ;;
        fix)
            echo "正在修复迁移版本..."
            goose -dir "$MIGRATIONS_DIR" fix
            ;;
        validate)
            echo "正在验证迁移文件..."
            goose -dir "$MIGRATIONS_DIR" mysql "$DB_DSN" validate
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
