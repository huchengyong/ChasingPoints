## Why

原生 iOS 已具备 SwiftUI 导航、主题、请求层和 Keychain 登录态基础，但“收短信 → 登录 → 重启恢复 → 令牌刷新 → 退出”的真实后端链路尚未完成验收。继续迁移业务页面前，需要验证并补齐这条链路，避免将编译通过或模拟响应测试当作登录能力已经可用。

## What Changes

- 对照现有后端契约，联调短信发送、验证码登录、用户信息与令牌刷新四个接口，补齐已发现的原生端兼容问题。
- 完善登录失败反馈，覆盖 HTTP 200 业务失败、HTTP 400 纯文本验证码错误、普通 HTTP 401 与 `SESSION_INVALID`，以及断网、超时、5xx。
- 验证 Keychain 保存、冷启动恢复、刷新后持久化、本机退出清理，以及开发/生产环境凭据隔离。
- 验证并发刷新只发起一次、原请求最多重放一次，以及退出或账号切换后旧响应不能污染新会话。
- 细化共享会话规范的客户端适用范围：原生 iOS 刷新遇暂时性故障保留凭据，只有明确认证失效才清理；UniApp 保留既有刷新失败处理。
- 补充必要的原生自动化测试和可复现的人工联调记录；分别记录构建、可控测试、真实后端验收结果。

本变更不迁移四个 Tab 的业务页面，不扩展第三方登录、账号绑定/合并、注销、推送或 WebSocket，不新增服务端全设备退出能力。最低支持版本保持 iOS 17.0。

## Capabilities

### New Capabilities

- `native-ios-auth-integration`：定义原生 iOS 短信登录、会话恢复与刷新、本机退出的行为及联调验收要求。

### Modified Capabilities

- `stale-auth-session-recovery`：修改“客户端幂等处理失效会话”，区分 UniApp 与原生 iOS 的刷新失败处理，明确原生的暂时性刷新失败不触发清理、重新登录提示或导航；保留其余会话隔离与幂等要求。

沿用 `auth-safe-client-read-coordination` 中认证代次隔离原则，不修改该能力，也不引入业务缓存、Bootstrap 或实时通信范围。

## Impact

- 原生端：预计涉及 `iOS/iOS/APIClient.swift`、`SessionStore.swift`、`LoginView.swift`，必要时调整 `ContentView.swift`、`iOSApp.swift` 的会话提示/登录入口，以及原生测试目标与 Xcode 工程配置。
- API：复用 `POST /api/auth/send-sms`、`POST /api/auth/login`、`POST /api/auth/refresh-token`、`GET /api/user/info`；以 `backend/chasing_points.api` 和实际错误处理为准，当前不计划改变接口契约。
- 验证依赖：可用的开发后端及其数据库、Redis、短信服务；实施真实短信测试前确认专用测试手机号。短信服务和 JWT 签名密钥继续由后端管理，无需提供给原生客户端。
- 交付物：原生修复、必要测试，以及 `docs/testing/native-ios-auth-integration.md` 联调步骤和脱敏结果。已有基础代码应先验证，仅修复有证据的缺口。
