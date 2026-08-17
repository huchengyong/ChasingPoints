## ADDED Requirements

### Requirement: 已认证请求必须对应启用用户
服务端 SHALL 在 REST JWT 认证成功后、业务 handler 执行前，验证正数 `user_id` 仍对应存在且 `status = 1` 的用户。公开请求和负数管理员身份 MUST 保持既有鉴权行为。

#### Scenario: 有效用户继续访问
- **WHEN** 签名与有效期均合法的 access token 包含一个存在且启用的正数 `user_id`
- **THEN** 服务端 SHALL 继续执行目标业务 handler
- **AND** 既有成功响应的字段与语义 SHALL 保持不变

#### Scenario: Refresh token 不得访问业务路由
- **WHEN** 调用方把带有 `token_type = refresh` 的有效 JWT 用作 REST 业务路由 Bearer token
- **THEN** 服务端 MUST 在业务 handler 前返回 HTTP 401
- **AND** MUST NOT 把 refresh token 当作 access token 放行业务请求

#### Scenario: Token 指向已删除用户
- **WHEN** 签名与有效期均合法的 access token 所含正数 `user_id` 在 `users` 表中不存在
- **THEN** 服务端 MUST 在业务 handler 前终止请求
- **AND** 返回 HTTP 401
- **AND** 不得返回 HTTP 200 的空 `success: false` 业务响应

#### Scenario: Token 指向停用用户
- **WHEN** access token 所含正数 `user_id` 对应用户的 `status` 不为 1
- **THEN** 服务端 MUST 将该请求视为失效会话并返回 HTTP 401

#### Scenario: 用户查询发生服务故障
- **WHEN** 服务端无法完成用户有效性查询且无法确认用户不存在或停用
- **THEN** 服务端 MUST 返回 5xx 服务错误
- **AND** 不得把数据库或依赖故障伪装成 HTTP 401

#### Scenario: 公开与管理员请求不受影响
- **WHEN** 请求 context 中没有 `user_id` 或 `user_id` 为负数管理员身份
- **THEN** 用户有效性校验 SHALL 跳过普通用户查询
- **AND** 请求 SHALL 按既有公开或管理员链路继续处理

### Requirement: 登录与 WebSocket 遵守同一普通用户认证边界
服务端 SHALL 在为已有普通用户签发新会话前确认用户启用，并在 WebSocket 升级前只接受 access token 且确认正数用户仍存在且 `status = 1`。

#### Scenario: 停用用户尝试重新登录
- **WHEN** 手机号、微信小程序或通用 OAuth 登录定位到 `status != 1` 的已有用户
- **THEN** 服务端 MUST 拒绝登录
- **AND** MUST NOT 签发新的 access token 或 refresh token

#### Scenario: 有效用户连接 WebSocket
- **WHEN** 存在且启用的普通用户使用有效 access token 连接用户或对局 WebSocket
- **THEN** 服务端 SHALL 按既有权限规则继续升级连接

#### Scenario: Refresh token 连接 WebSocket
- **WHEN** 调用方使用带有 `token_type = refresh` 的 JWT 连接用户或对局 WebSocket
- **THEN** 服务端 MUST 拒绝升级并返回 HTTP 401

#### Scenario: 缺失或停用用户连接 WebSocket
- **WHEN** WebSocket access token 指向已删除或 `status != 1` 的正数普通用户
- **THEN** 服务端 MUST 拒绝升级并返回 HTTP 401

### Requirement: 失效会话响应具有稳定机器标识
失效主体的 HTTP 401 响应 SHALL 包含稳定原因 `SESSION_INVALID` 和可展示的重新登录提示，使客户端能够区别于可通过 refresh 恢复的普通 JWT 401。

#### Scenario: 返回标准失效原因
- **WHEN** 服务端确认 token 对应普通用户不存在或已停用
- **THEN** 响应 payload SHALL 包含 `success = false`
- **AND** `reason` SHALL 为 `SESSION_INVALID`
- **AND** `message` SHALL 明确要求用户重新登录

#### Scenario: 普通 access token 过期
- **WHEN** JWT 中间件仅因 access token 过期或签名认证失败返回 HTTP 401
- **THEN** 服务端 MUST NOT 虚构 `SESSION_INVALID`
- **AND** 客户端 SHALL 保留既有 refresh 判断能力

### Requirement: 客户端幂等处理失效会话
移动端 SHALL 对 `SESSION_INVALID` 直接执行全局会话失效流程，不得使用同属失效用户的 refresh token重试；并发失败 MUST 只触发一次清理、提示和登录导航。

#### Scenario: 失效主体跳过刷新
- **WHEN** 任一请求收到 HTTP 401 且响应原因为 `SESSION_INVALID`
- **THEN** 客户端 MUST NOT 调用 refresh-token 接口
- **AND** SHALL 立即进入全局会话失效流程

#### Scenario: 普通 401 刷新成功
- **WHEN** 请求收到不含 `SESSION_INVALID` 的 HTTP 401 且 refresh token 有效
- **THEN** 客户端 SHALL 共享一次 refresh 请求
- **AND** 刷新成功后 SHALL 重放原请求
- **AND** 用户登录状态 SHALL 保持有效

#### Scenario: 普通 401 刷新失败
- **WHEN** access token 认证失败且 refresh 请求无法恢复会话
- **THEN** 客户端 SHALL 进入与 `SESSION_INVALID` 相同的全局清理流程

#### Scenario: 多个请求同时发现失效会话
- **WHEN** 多个并发请求同时收到 `SESSION_INVALID` 或共享 refresh 失败
- **THEN** 客户端 MUST 只执行一次状态清理
- **AND** MUST 只显示一次重新登录提示
- **AND** MUST 只安排一次登录页导航
- **AND** 页面级调用方不得重复展示认证错误

#### Scenario: 静默请求与普通请求同时发现失效
- **WHEN** 静默请求与普通请求同时确认同一个会话已失效
- **THEN** 静默请求不得抑制全局重新登录提示与导航
- **AND** 全局失效副作用仍 MUST 只执行一次

#### Scenario: 旧请求在新登录后才返回
- **WHEN** 旧 access token 发出的请求在用户已登录新会话后才返回 `SESSION_INVALID`
- **THEN** 客户端 MUST 仅拒绝旧请求
- **AND** MUST NOT 清除、覆盖或导航离开新会话

#### Scenario: 被新会话淘汰的旧请求不展示重新登录
- **WHEN** 旧请求返回会话错误时 access token 或 auth generation 已不再属于当前会话
- **THEN** 客户端 MUST 将该错误标记为 superseded
- **AND** 页面 MUST NOT 显示当前会话正在退出或即将跳转登录页

#### Scenario: 旧 refresh 响应在会话切换后返回
- **WHEN** refresh 请求发出后用户退出或切换了账号
- **THEN** 旧 refresh 响应 MUST NOT 写回 token 或恢复旧登录状态

### Requirement: 会话清理覆盖全部持久化身份状态
全局退出 SHALL 同步清除 access token、refresh token、持久化用户身份、登录标记和依赖当前用户的运行时缓存，并断开用户 WebSocket，避免旧身份在下次启动时恢复。

#### Scenario: 清理直接存储与 Pinia 持久化状态
- **WHEN** 客户端判定会话失效
- **THEN** `token` 与 `refreshToken` 存储 SHALL 被清除
- **AND** 持久化 `user-store` 中的 token、用户资料和 `isLoggedIn` SHALL 被归零
- **AND** 用户相关段位或未读状态缓存 SHALL 不得继续以旧用户身份展示

#### Scenario: 清理后重新启动
- **WHEN** 会话失效清理完成后用户重新启动或重新编译客户端
- **THEN** 客户端 MUST NOT 从任一持久化键恢复旧用户的已登录状态

### Requirement: App 恢复时验证持久化会话
客户端 SHALL 在使用持久化身份连接用户 WebSocket或执行其他 app 级认证行为前，向服务端验证当前用户仍然有效；验证过程 SHALL 避免重复并发执行。

#### Scenario: 持久化会话仍有效
- **WHEN** App 恢复时本地存在已登录状态且服务端返回当前有效用户资料
- **THEN** 客户端 SHALL 使用服务端资料校正本地用户状态
- **AND** 验证成功后方可连接用户 WebSocket

#### Scenario: 持久化会话已失效
- **WHEN** App 恢复验证收到 `SESSION_INVALID`
- **THEN** 客户端 SHALL 清理全部会话状态
- **AND** 引导用户重新登录

#### Scenario: 恢复验证遇到网络或服务故障
- **WHEN** App 无法验证会话是因为网络失败或 5xx 服务错误
- **THEN** 客户端 MUST NOT 删除仍可能有效的凭证
- **AND** MUST NOT 将该故障宣称为账号失效
- **AND** SHALL 在后续前台恢复或用户重试时再次验证

#### Scenario: 多处同时请求恢复验证
- **WHEN** App 生命周期与页面加载同时触发同一 auth generation 的会话验证
- **THEN** 客户端 SHALL 共享同一个进行中的验证结果

#### Scenario: 验证期间切换账号
- **WHEN** 用户 A 的恢复验证尚未完成时客户端退出或登录用户 B
- **THEN** 用户 A 的验证结果 MUST NOT 更新用户 B 的 store
- **AND** 用户 B MUST NOT 复用用户 A 的进行中验证

#### Scenario: 验证完成前 App 进入后台
- **WHEN** App 发起恢复验证后进入后台且验证随后成功
- **THEN** 客户端 MUST NOT 在后台重新连接用户 WebSocket
- **AND** MUST NOT 执行依赖前台有效身份的 app 级行为

#### Scenario: 前一前台周期的进行中对局响应延迟返回
- **WHEN** 进行中对局检查发出后 App 进入后台并再次进入前台
- **THEN** 前一 lifecycle generation 的响应 MUST NOT 弹出对局提醒

#### Scenario: 验证前不执行 app 级认证行为
- **WHEN** App 恢复持久化登录态
- **THEN** push token 上传、进行中对局检查和用户 WebSocket 连接 SHALL 等待验证成功

### Requirement: 账号合并后切换到目标用户会话
任何手机号绑定路径在成功合并两个账号后 SHALL 返回并应用属于合并目标用户的新 access token、refresh token 和用户资料；客户端不得继续使用已删除源用户的令牌。

#### Scenario: 短信验证码绑定触发账号合并
- **WHEN** 当前 OAuth 用户通过短信验证码绑定一个属于其他有效用户的手机号
- **THEN** 服务端 SHALL 完成 OAuth 归属迁移
- **AND** 成功响应 SHALL 返回目标用户的新 access token、refresh token、有效期和 `user_info`
- **AND** 返回 token 中的 `user_id` SHALL 等于 `user_info.id`

#### Scenario: 微信手机号授权触发账号合并
- **WHEN** 微信小程序手机号授权绑定触发账号合并
- **THEN** 成功响应 SHALL 返回目标用户的新令牌和用户资料
- **AND** 客户端 SHALL 原子替换当前用户会话

#### Scenario: 手机号属于停用目标用户
- **WHEN** 短信或微信手机号绑定发现目标手机号属于 `status != 1` 的用户
- **THEN** 服务端 MUST 拒绝账号合并
- **AND** MUST NOT 迁移 OAuth 归属
- **AND** MUST NOT 删除当前源用户
- **AND** MUST NOT 签发停用目标用户的会话

#### Scenario: 绑定未触发账号合并
- **WHEN** 当前用户绑定的是尚未被其他用户使用的手机号
- **THEN** 当前用户 ID 和现有会话 SHALL 保持不变
- **AND** 客户端 SHALL 更新手机号与 `need_bind_phone` 状态

#### Scenario: 合并响应被客户端应用
- **WHEN** 绑定响应包含 `merged_account = true` 及完整替换令牌
- **THEN** 客户端 SHALL 使用统一登录动作写入新 token、refresh token 和目标用户资料
- **AND** 后续请求 MUST 使用目标用户 token
- **AND** 成功提示 MUST NOT 要求用户重新登录

### Requirement: 页面错误反馈不得把非网络失败归因于网络
共享请求层 SHALL 保留足以区分网络、会话、权限、服务和普通业务失败的信息；荣誉墙初次加载 SHALL 根据错误类别展示反馈。

#### Scenario: 真实网络失败
- **WHEN** 荣誉墙请求因网络连接失败或请求层 fail 回调而失败
- **THEN** 页面 SHALL 提示检查网络并提供重新加载操作

#### Scenario: 服务或业务失败
- **WHEN** 荣誉墙请求已到达服务端但因 5xx 或普通业务错误失败
- **THEN** 页面 SHALL 使用暂时无法加载的服务反馈
- **AND** 不得提示用户检查网络

#### Scenario: 权限拒绝
- **WHEN** 非好友请求查看他人荣誉墙
- **THEN** 服务端 SHALL 返回 HTTP 403 或稳定的等价权限类别
- **AND** 页面 SHALL 告知用户无权查看对应内容
- **AND** 不得把权限拒绝显示为网络异常

#### Scenario: 会话失效已由全局处理
- **WHEN** 荣誉墙请求因 `SESSION_INVALID` 被全局会话流程接管
- **THEN** 页面 MUST NOT 再显示网络失败页或重复 toast
- **AND** SHALL 允许全局重新登录导航完成
