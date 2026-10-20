# 失败关闭验证记录（P0 任务 8.4）

> 对应 OpenSpec 变更 `fix-client-p0-release-blockers` 的任务 8.4。本记录只含测试名称、命令、结果与状态，不含任何 token、密码或完整凭据。

## 1. 状态

部分完成。本地自动化测试与线上只读检查已完成；带有效票据的 WebSocket 检查、以及 Redis 故障的线上验证仍待完成（见第 4 节）。

## 2. 本地自动化测试（2026-10-20）

命令（在 `backend/` 执行）：

```bash
go test ./internal/pkg/ws/ ./internal/config/ -count=1 -v
go test ./... -count=1
```

| 验证点 | 测试 | 结果 |
| --- | --- | --- |
| 配置的 WebSocket Origin 放行 | `TestWebSocketOriginAllowsConfiguredOrigin` | 通过 |
| 未配置的 Origin 拒绝 | `TestWebSocketOriginRejectsUnconfiguredOrigin` | 通过 |
| 缺失 Origin：默认拒绝，仅在安全传输且开启时放行 | `TestWebSocketOriginAllowsNativeMissingOriginOnlyWhenSecureAndEnabled` | 通过 |
| 用户 WebSocket 在注册前拒绝不允许的 Origin | `TestUserWSHandlerRejectsDisallowedOriginBeforeRegistration` | 通过 |
| 票据只能使用一次，重放被拒绝 | `TestUserWSHandlerAllowsActiveTicketOnce` | 通过 |
| 票据服务故障（store 缺失或 Redis 连接失败）返回 503，不升级、不注册 | `TestUserWSHandlerFailsClosedWhenTicketServiceUnavailable`（本次新增） | 通过 |
| 华为登录开启但缺少 `CLIENT_SECRET` 时拒绝启动 | `TestValidateProductionSecurityRejectsMissingProviderConfig` | 通过 |
| 华为登录开启但缺少 `CLIENT_ID` 时拒绝启动 | `TestValidateProductionSecurityRejectsHuaweiEnabledWithoutClientID`（本次新增） | 通过 |
| 缺失匹配邀请签名密钥时拒绝启动 | `TestValidateProductionSecurityRejectsMissingMatchInviteSecret` | 通过 |
| 后端全量测试 | `go test ./...` | 退出码 0，无 FAIL |

## 3. 线上只读检查（2026-10-20，生产 `api.zhuifen.cn`，未改动数据）

| 检查 | 方法 | 预期 | 结果 |
| --- | --- | --- | --- |
| 允许的 HTTP Origin 被回显 | `Origin: https://www.zhuifen.cn` | `access-control-allow-origin` 回显该 Origin | 通过 |
| 未知 HTTP Origin 不被回显 | `Origin: https://evil.example` | 响应无 ACAO 头 | 通过 |
| 受保护接口未登录 | `GET /api/member/status` | 401 | 通过 |
| WebSocket 无票据 | `GET /api/user/ws`，无 `ticket` | 400 `missing ticket`，不升级 | 通过 |
| 华为登录验证器已启用且拒绝假凭据 | `POST /api/auth/login-by-oauth`，假凭据 | 400 `Huawei rejected OAuth credential`，不建账号（验证在建账号之前返回） | 通过 |

测试环境（`api-zhuifen.kekemate.cn`）的华为假凭据检查结果相同。

## 4. 待完成项

1. **WebSocket Origin 拒绝（带有效票据）**：必须用测试用户的 token 申领票据，票据一次性，每个用例重新申领。预期：Origin 为 `https://evil.example` → 403；`https://www.zhuifen.cn` → 101；无 Origin → 101（线上 `AllowNativeMissingOrigin: true`）。
2. **Redis 故障的线上验证**：不在线上或测试服务器上停 Redis。代码层面的 503 已由第 2 节的测试覆盖；线上只能在隔离环境中做。
3. **完整二进制的启动拒绝**：本地用临时副本启动时，yaml 占位符展开与线上行为不一致，未能复现完整启动。配置校验逻辑已由单元测试覆盖，启动入口为 `chasing_points.go` 中 `ValidateProductionSecurity` 的调用。

## 5. 签署

| 角色 | 姓名 | 日期 | 备注 |
| --- | --- | --- | --- |
| 发布负责人 |  |  |  |
