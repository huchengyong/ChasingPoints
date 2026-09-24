# Linux AMD64 构建目标设计

## 目标

在根目录 Makefile 增加 `make build`，将后端 `chasing_points` 交叉编译为可在 Linux AMD64 运行的单个可执行文件。

## 已确认行为

- 命令固定为 `make build`。
- 在 `backend/` 模块内执行 Go 构建。
- 构建环境固定为 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，避免依赖目标系统的 C 运行时。
- 输出文件固定为根目录 `bin/chasing_points`；`bin/` 已被 Git 忽略。
- 现有 `server` 目标保持不变。

## 实现边界

Makefile 增加 `BUILD_DIR`、`BUILD_BINARY` 与 `.PHONY: build`，目标先创建输出目录，再从 `backend/` 执行 `go build -o ../bin/chasing_points .`。

不增加多平台参数、版本注入、压缩、发布或部署逻辑。

## 验证

1. 运行 `make build`。
2. 确认 `bin/chasing_points` 存在。
3. 使用 `file bin/chasing_points` 确认产物为 Linux x86-64 可执行文件。
