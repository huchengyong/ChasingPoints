## 1. 后端失效会话判定

- [x] 1.1 为 context 身份解析补充测试，覆盖无 `user_id`、正数普通用户、负数管理员、`json.Number`、`float64` 与非法类型。
- [x] 1.2 增加可选且保留正负号的身份解析 helper，供全局中间件复用，不重复解析 Authorization header。
- [x] 1.3 先编写 active-user session middleware 测试，覆盖有效用户放行、缺失用户 401、停用用户 401、公开请求放行、管理员放行及数据库错误 5xx。
- [x] 1.4 实现 active-user session middleware，失效主体返回 `success: false`、`reason: SESSION_INVALID` 和重新登录提示。
- [x] 1.5 在 `backend/chasing_points.go` 中通过 `server.Use` 注册中间件，并增加集成测试确认 JWT 已解析后才执行用户校验且业务 handler 不会在失效会话下运行。
- [x] 1.6 增加荣誉墙或用户信息路由回归测试，复现“有效 JWT 指向已删除用户”时由 HTTP 200 空失败改为标准 HTTP 401，同时确认有效用户响应不变。

## 2. 账号合并后的目标用户会话

- [x] 2.1 先扩展绑定手机号 API 契约测试，断言 `BindPhoneResp` 可加法返回 access token、refresh token、有效期、`need_bind_phone` 和 `user_info`。
- [x] 2.2 修改 `backend/chasing_points.api` 的 `BindPhoneResp` 后立即运行 goctl，检查生成 diff 且确认没有新增 todo logic 或错误 handler 引用。
- [x] 2.3 将短信验证码账号合并的 OAuth 迁移与源用户删除放入事务，成功后为合并目标用户签发新 token pair 并返回目标 `user_info`。
- [x] 2.4 增加短信账号合并测试，验证响应 token 的 `user_id`、`user_info.id` 和 OAuth 归属一致，旧用户已删除且新用户状态有效。
- [x] 2.5 增加合并失败事务回滚测试，以及未发生合并时用户 ID、现有会话和普通绑定响应保持不变的测试。
- [x] 2.6 回归微信手机号授权合并，确认其继续返回目标用户令牌并与扩展后的通用绑定响应语义一致。

## 3. 移动端请求层与全局会话清理

- [x] 3.1 先扩展 `app/tests/request.test.mjs`，覆盖错误类别、HTTP 状态和响应 payload 保留、`SESSION_INVALID` 识别、普通 401 可刷新及网络失败分类。
- [x] 3.2 调整统一请求层：`SESSION_INVALID` 401 跳过 refresh，普通 JWT 401 继续共享 refresh，refresh 失败进入同一会话失效流程。
- [x] 3.3 为会话失效处理增加 single-flight 控制和测试，确保并发 401 只清理一次、提示一次、导航一次，调用方错误带 `_isHandled`。
- [x] 3.4 扩展用户 store 的退出动作，统一清除 access token、refresh token、持久化用户身份、登录标记、段位及现有用户相关未读缓存，并断开 user WebSocket。
- [x] 3.5 增加用户 store 回归测试，确认退出后重新初始化不会从 `token`、`refreshToken` 或 `user-store` 恢复旧身份。

## 4. App 持久化会话恢复验证

- [x] 4.1 为会话恢复逻辑补充纯函数或可注入依赖测试，覆盖有效用户、`SESSION_INVALID`、网络/5xx 保留凭证和并发验证共享结果。
- [x] 4.2 让 `app/api/user.js` 的用户信息门面支持会话恢复所需的静默请求选项，同时保持页面只通过 API 门面访问后端。
- [x] 4.3 在 `app/App.vue` 恢复已登录状态时先验证当前用户，用服务端资料校正 store，验证成功后再连接 user WebSocket。
- [x] 4.4 确保恢复验证遇到网络或 5xx 时不退出用户、不宣称会话失效，并在后续前台恢复时允许重试。
- [x] 4.5 增加 App 生命周期源码或逻辑测试，确认未验证的持久化身份不会直接触发用户级 WebSocket 连接，且失效会话不产生重复登录导航。

## 5. 移动端账号合并会话替换

- [x] 5.1 扩展 `app/utils/bind-phone-flow.js` 测试，覆盖短信绑定返回完整目标用户令牌时 `sessionReplaced = true`，以及旧服务无令牌时仍回退到重新登录。
- [x] 5.2 核对并调整 `bindPhone.vue`，使短信与微信合并响应都通过 `userStore.login(response)` 原子替换 token、refresh token 和用户资料。
- [x] 5.3 更新登录页、欢迎页、设置页和编辑资料页回归断言，确认新响应不再保留源用户会话，兼容回退响应仍执行 logout/relogin。

## 6. 荣誉墙错误反馈

- [x] 6.1 先为荣誉墙加载错误视图模型增加测试，覆盖 network、forbidden、server/business 和已由全局处理的 session 错误。
- [x] 6.2 调整 `app/subPages/achievement/index.vue` 的失败状态，仅真实网络错误提示检查网络；权限和服务失败显示对应文案。
- [x] 6.3 确保 `_isHandled` 的会话错误不设置网络失败态、不弹页面级 toast，并允许全局重新登录流程完成。
- [x] 6.4 更新荣誉墙源码断言，防止后续再次把所有初次加载失败统一归因于网络。

## 7. 回归验证与交付检查

- [x] 7.1 运行后端身份解析、中间件、认证、绑定手机号、用户信息和荣誉墙定向测试。
- [x] 7.2 运行 `cd backend && go test ./...`，确认普通用户、管理员、公开路由和账号合并无回归。
- [x] 7.3 运行 `cd app && node --test tests/*.test.mjs`，确认请求层、持久化状态、绑定流程、App 生命周期和荣誉墙测试通过。
- [x] 7.4 使用有效用户 token、已删除用户 token 和停用用户 token 做本地只读请求验证，确认分别得到成功、`SESSION_INVALID` 401 和 `SESSION_INVALID` 401。
- [x] 7.5 运行 `git diff --check` 与 `openspec validate fix-stale-auth-session-handling --type change --strict`，核对实现、API 生成代码和规格一致。

## 8. Review 并发与端到端加固

- [x] 8.1 补充请求层竞态测试，覆盖静默/非静默混合并发、延迟旧 401 不得清除新会话、原始 401 payload 与 `_isSilent` 元数据保留。
- [x] 8.2 将会话失效处理绑定到发起请求的 access token；旧 token 响应不得影响已切换的新会话，且失效会话始终只执行一次全局提示与登录导航。
- [x] 8.3 为 refresh 写回增加 refresh token 世代校验，并确保刷新成功后的重放网络/业务失败不会被误判为会话失效。
- [x] 8.4 补充会话恢复竞态测试，覆盖不同 auth generation 不共享请求、旧用户结果不应用到新用户、App 隐藏后不连接 WebSocket。
- [x] 8.5 重构 App 会话恢复：按 auth generation 校验结果，在应用资料和连接 WS 前复核前台与当前会话，并把推送 token 上传、进行中对局提醒统一放到验证成功后。
- [x] 8.6 让恢复用户信息请求静默处理网络错误，同时保持 `SESSION_INVALID` 的全局提示和登录导航。
- [x] 8.7 完整清理通知与好友申请未读缓存，并覆盖 logout 与账号会话替换回归测试。
- [x] 8.8 在短信与微信手机号合并事务中拒绝停用目标用户，补充源用户/OAuth 不变的回滚测试。
- [x] 8.9 让荣誉墙非好友访问返回真实 HTTP 403 并补充 handler、请求分类和页面反馈回归测试。
- [x] 8.10 运行前后端全量测试、竞态定向测试、`git diff --check` 与 OpenSpec strict validation。

## 9. 第二轮 Review 认证边界与生命周期收口

- [x] 9.1 更新 proposal、design 与 capability spec，补充 access/refresh 用途隔离、停用用户登录拒绝、WebSocket 活跃用户校验、superseded 页面反馈及 lifecycle generation 要求。
- [x] 9.2 先补后端回归测试，覆盖 refresh token 访问 REST、refresh/deleted/disabled 用户连接 WebSocket，以及三个登录入口拒绝停用用户。
- [x] 9.3 在 REST、用户/对局 WebSocket 和登录入口统一落实 access token 与 active-user 边界，并保持公开请求、负数管理员和兼容期无类型 access token 行为。
- [x] 9.4 先补移动端回归测试，覆盖 superseded 会话错误不显示重新登录、进行中对局响应受 lifecycle generation 约束，以及账号合并成功文案不再要求登录。
- [x] 9.5 将会话失效处理同时绑定 access token 与 auth generation，向旧调用方标记 superseded，并为进行中对局检查增加 lifecycle generation 复核。
- [x] 9.6 修正账号合并成功文案，并让全局 HTTP error handler 对普通错误及 gRPC status error 保持 go-zero 既有兜底状态语义。
- [x] 9.7 运行后端认证/WebSocket/错误映射定向测试、移动端请求/生命周期/绑定定向测试及 `go test -race`。
- [x] 9.8 运行前后端全量测试、真实 refresh-token-as-access/WS 探针、`git diff --check` 与 OpenSpec strict validation。
