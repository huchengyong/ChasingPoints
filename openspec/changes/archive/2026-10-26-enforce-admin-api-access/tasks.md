## 1. 访问控制中间件

- [x] 1.1 在 `backend/internal/middleware/` 新增后台访问控制中间件：`path.Clean` 后判断 `/api/admin` 前缀；三个公开入口走白名单；其余路径无身份返回 401、非负 ID 返回 403（带 `code`/`success`/`message` 的 JSON）、负数 ID 放行
- [x] 1.2 单元测试：负数 ID 放行；正数 ID 与 0 返回 403 且不调用下一层；无身份返回 401；公开入口在带普通用户令牌时仍放行；非后台路径不受影响；`//`、`/./`、`/x/../` 变体被拦；`/api/admin` 本身与 `/api/administrator` 的边界

## 2. 接入服务

- [x] 2.1 `backend/chasing_points.go`：把全局中间件注册抽成 `registerGlobalMiddlewares`，在 `ActiveUserSessionMiddleware` 之前注册后台访问控制，保持其他中间件顺序不变
- [x] 2.2 `backend/` 新增真实服务测试：启动 go-zero 服务并遍历 `Routes()` 中的 `/api/admin` 路由，普通用户令牌 → 403、匿名 → 401（公开入口例外）；覆盖路径变体；确认非后台接口（如 `/api/user/info`）对同一个用户令牌不返回 403

## 3. 验证

- [x] 3.1 变异检查：临时去掉中间件注册，确认真实服务测试失败；临时去掉 `path.Clean`，确认变体用例失败；检查后恢复
- [x] 3.2 在 `backend/` 运行 `go vet ./...`、`go test ./...`，并对新增 Go 文件执行 `gofmt -l`
- [x] 3.3 `openspec validate enforce-admin-api-access --strict` 通过

## 4. 发布与验收记录

- [x] 4.1 发布开发环境（`backend/deploy-dev.sh`）；匿名请求逐个访问各后台接口，确认 401，公开入口 200
- [x] 4.2 用户在开发环境用管理员账号走一遍后台页面（用户、球馆、配置、装备目录），确认没有 403（需要用户人工确认）
- [x] 4.3 在 `docs/testing/` 新增验收记录：自动化结果、变异检查、开发环境验证、人工确认情况、未执行项
- [x] 4.4 经用户确认后发布生产，并只读核对
