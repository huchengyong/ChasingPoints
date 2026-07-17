# 管理后台启动命令设计

## 目标

在根目录 Makefile 增加 `make admin`，一次启动管理后台所需的 Go API 与 Vite 开发服务，并在按下 `Ctrl+C` 或终端退出时停止后端进程。

## 已确认行为

- Go 服务必须从 `backend/` 目录运行，命令复用现有的 `go run . -f etc/chasing_points-api.yaml`。
- 后端环境变量复用现有 `server` 目标的 `ENV_FILE` 选择规则，并在启动子进程前加载。
- 管理端从 `admin/` 目录运行 `npm run dev`；Vite 自行读取该目录的开发环境配置。
- 两个服务的日志都输出到当前终端。
- 不启动 Cloudflare Tunnel，不增加 npm 依赖，不新增端口清理、健康检查或后台守护逻辑。

## 实现边界

Makefile 增加 `.PHONY` 声明和 `admin` 目标。目标先校验后端环境文件存在，再以后台子进程启动 `backend/` 下的 Go 服务并保存其 PID，随后在前台执行 `admin/` 下的 `npm run dev`。

配方通过 `trap` 在接收到 `INT`、`TERM` 或前台 Vite 进程结束时终止后端 PID，确保 `Ctrl+C` 一次停止两个服务。前端保持前台进程，因此无需额外结束它自身。

现有 `server` 与 `build` 目标保持不变。

## 验证

1. 运行 `make -n admin`，确认后端命令在 `backend/` 目录执行、前端命令在 `admin/` 目录执行。
2. 在已配置后端环境变量且已安装管理端依赖的本地环境运行 `make admin`。
3. 确认 Go API 与 Vite 服务均启动，按 `Ctrl+C` 后后端子进程一并退出。
