## Context

当前 HTTP 鉴权只验证 JWT 的签名和有效期，不验证 token 中的正数 `user_id` 是否仍对应 `users` 表中的启用用户。账号合并、用户停用或开发数据重置后，旧 token 因签名仍有效而继续通过 go-zero JWT 中间件；随后各 logic 各自查询用户，并常以 HTTP 200 返回无消息的 `success: false`。

本次故障已复现：微信开发者工具持久化 token 与 Pinia 用户均指向已不存在的用户 2，OAuth 已指向有效用户 1；同一荣誉墙接口使用用户 2 token 返回 HTTP 200 + `success: false`，使用用户 1 token 则正常返回 25 项成就。移动端 `request.js` 将前者转换成普通“请求失败”，荣誉墙再统一展示“请检查网络”。

后端已经通过 `server.Use` 注册全局请求中间件。go-zero 的执行链会先完成路由 JWT 认证，再执行 `server.Use` 中间件，因此可以直接读取 JWT 写入 context 的 `user_id`，无需重复解析 token。管理员 JWT 使用负数 ID，公开路由没有 `user_id`，两者必须保持既有行为。普通用户 access token 与 refresh token 使用同一签名 secret，仅通过 `token_type` 区分；go-zero 默认 JWT 中间件不会验证该字段，手工注册的 WebSocket 路由也只解析 `user_id`，因此 refresh token 当前可被误用为业务访问凭证。

## Goals / Non-Goals

**Goals:**

- 在所有 REST JWT 用户路由进入业务 handler 前统一拒绝 refresh token，以及已删除或停用的用户身份。
- 在用户和对局 WebSocket 升级前只接受 access token，并验证正数普通用户仍存在且启用。
- 在手机号、微信小程序和通用 OAuth 登录入口签发 token 前拒绝停用用户。
- 用可机器识别的 HTTP 401 响应区分“token 本身过期”与“token 主体已失效”。
- 让客户端对失效主体跳过无意义的 refresh，幂等清理完整会话并只触发一次重新登录提示与导航。
- 在 App 恢复持久化登录态时先向服务端确认用户仍有效，再连接用户 WebSocket 或执行依赖身份的 app 级行为。
- 账号合并成功后始终把会话切换到目标用户，避免返回后继续使用已删除用户的 token。
- 让荣誉墙等页面只在真实网络失败时提示检查网络，认证失效由全局流程处理，其他错误使用对应反馈。

**Non-Goals:**

- 不引入 JWT 黑名单、session 表、用户会话版本号或 Redis 会话缓存。
- 不修改 token 有效期、签名算法或 refresh token 的总体续期策略；只补齐 access/refresh 的用途隔离。
- 不重做账号资产合并规则，也不迁移成就、比赛等业务资产。
- 不改变匿名浏览和负数管理员身份的鉴权方式。
- 不在本次建立全站统一错误码目录；只增加本能力需要的稳定失效原因。

## Decisions

### 1. 使用 JWT 后置的全局有效用户中间件

新增依赖 `UserModel` 的 active-user session middleware，并通过 `server.Use` 注册。中间件按 context 身份分流：

```text
无 user_id       -> 公开路由，继续
user_id < 0      -> 管理员身份，继续
user_id > 0      -> token_type = refresh：401
                     └─ access/兼容期空类型：查询 users 主键
                          ├─ status = 1：继续
                          ├─ 不存在/停用：401 SESSION_INVALID
                          └─ 数据库错误：500，不清客户端会话
```

中间件使用可保留正负号的可选身份解析 helper，不重复解析 Authorization header。有效用户检查是主键查询，首期不增加 Redis 缓存；先保证语义正确，再根据真实指标决定是否优化。

**替代方案：** 继续在每个 logic 中检查用户。现状已经证明容易遗漏且响应语义不一致，因此不采用。

**替代方案：** 只在 App 启动时调用 `/api/user/info`。这无法覆盖用户在 App 前台期间被停用、账号刚完成合并但请求并发到达等场景，因此只作为客户端恢复保护，不作为唯一校验。

### 2. 失效主体返回稳定的 401 原因

中间件对不存在或停用的正数用户返回：

```json
{
  "success": false,
  "reason": "SESSION_INVALID",
  "message": "登录状态已失效，请重新登录"
}
```

HTTP 状态码为 401。数据库查询失败返回 5xx 服务错误，防止短暂数据库故障把所有客户端误登出。有效用户的所有成功 payload 保持不变。

**替代方案：** 继续返回 HTTP 200 并增加 message。统一请求层只对 401 承担会话恢复，继续使用 200 会让每个页面重复识别业务失败，因此不采用。

### 3. 客户端区分可刷新 401 与不可恢复会话

`request.js` 保留现有 token 过期续期闭环，但按响应原因处理：

- 普通 JWT 401（没有 `SESSION_INVALID`）：共享一次 refresh 请求，成功后重放原请求。
- `SESSION_INVALID`：不调用 refresh，因为 access/refresh token 指向同一个失效主体；直接进入全局会话失效流程。
- refresh 失败：进入同一会话失效流程。

全局失效流程必须 single-flight。并发请求即使同时收到 401，也只清理一次状态、显示一次提示并安排一次登录导航；每个调用方仍收到带 `_isHandled`、HTTP 状态和错误类别的 rejection，页面不得重复提示。

Review 加固后，失效处理必须同时绑定到发起请求时使用的 access token 与 auth generation：只有二者仍对应当前会话时才允许执行 logout。已经切换到新会话后才到达的旧 401 只能拒绝旧调用方，不得清除新会话，也必须标记为 superseded，避免页面误显示“正在重新登录”。`silent` 仅抑制网络等瞬时失败的页面反馈，不得抑制已确认失效会话的全局登录导航。refresh 响应写回同样必须校验发起刷新时的 refresh token 仍属于当前会话，避免旧刷新响应复活已退出或已切换的账号。

### 4. 会话清理以 Pinia 用户状态为单一入口

扩展现有 `userStore.logout()` 作为完整清理入口：清除 access token、refresh token、持久化 `user-store` 中的身份状态、段位等用户相关缓存，并断开 user WebSocket。直接存储键与 Pinia 持久化状态必须在同一次动作中归零，避免只移除 `token` 后又从 `user-store` 恢复伪登录。

App 恢复时若本地标记为已登录，则先以静默方式获取当前用户信息：

- 成功：用服务端用户资料校正 store，再连接 user WebSocket。
- `SESSION_INVALID`：复用全局清理与重新登录流程。
- 网络或 5xx：不删除仍可能有效的凭证，不把瞬时故障当成登出；跳过本次受身份影响的 app 级连接，并在下次前台恢复或用户重试时再次验证。

验证过程按用户 store 的运行时 auth generation 共享 in-flight Promise，避免同一会话重复请求，同时禁止不同账号世代复用旧 Promise。恢复 helper 只返回服务端资料；App 在应用资料、上传 push token、检查进行中对局和连接 WebSocket 前，必须再次确认 App 仍在前台、用户仍已登录且 auth generation 未变化。进行中对局提醒还必须绑定发起时的 App lifecycle generation，防止前一前台周期的响应在再次进入前台后弹窗。logout、重新登录和账号合并替换会话都会推进 generation，使旧异步结果自动失效。

完整会话清理除段位外还必须归零通知未读数和好友申请未读数；新账号登录或账号合并替换会话时也先清理这些旧用户运行时缓存。

### 5. 所有账号合并路径返回目标用户会话

微信手机号授权合并已经返回目标用户的新 access token、refresh token 和 `user_info`；短信验证码绑定的合并路径改为相同语义。`BindPhoneResp` 以加法方式补充可选 token、有效期、`need_bind_phone` 和 `user_info` 字段，修改 `.api` 后立即运行 goctl。

短信合并的数据迁移在事务中完成，成功响应中的 token 和用户资料必须属于合并目标用户。短信和微信两条路径都必须在锁定目标用户后确认其 `status = 1`；停用目标不得接收 OAuth，也不得导致源用户删除。前端继续复用 `resolveBindPhoneSuccess` 与 `userStore.login(response)` 原子替换会话。已返回并应用完整替换会话时，成功文案不得再要求用户重新登录；旧客户端缺少替换字段时仍按兼容流程重新登录。

**替代方案：** 合并后只要求所有页面主动 logout。该方案依赖每个绑定入口都正确处理事件，任何遗漏都会再次留下已删除用户 token，因此不作为新客户端主路径。

### 6. 请求错误携带类别，页面按原因展示

统一请求层为 rejection 保留 `statusCode`、响应 payload 和最小错误类别：`network`、`session`、`forbidden`、`server`、`business`。荣誉墙初次加载状态基于类别生成文案：

- `network`：提示检查网络并允许重试。
- `forbidden`：提示无权查看，不归因于网络。
- `server` / `business`：提示暂时无法加载并允许重试。
- `session` 且 `_isHandled`：由全局重新登录流程接管，不显示网络失败页或重复 toast。

荣誉墙非好友访问必须在真实 HTTP 链路返回 403，使 `forbidden` 分类不是仅存在于纯函数测试中的不可达分支。被新会话淘汰的旧请求使用独立 superseded 标记，页面不得将其显示为当前会话正在失效。首期只调整本次暴露问题的荣誉墙和共享错误构造，不顺带重写所有页面文案。

### 7. 登录与 WebSocket 复用同一普通用户认证边界

所有普通用户登录入口在为已有用户签发 token 前检查 `status = 1`，停用用户统一返回登录失败且不暴露具体封禁原因。WebSocket 使用类型化 claims，只接受 access token；对正数普通用户在升级连接前查询用户状态，缺失或停用返回 401。该规则同时覆盖 `/api/user/ws` 和需要认证身份的 `/api/match/ws`，避免 REST 与实时链路产生不同会话语义。

历史无 `token_type` 的 access token 在兼容期内继续作为 access 使用；明确标记为 `refresh` 的 token 必须在 REST 与 WebSocket 入口拒绝。历史无类型 refresh token 无法与旧 access token可靠区分，只能随既有有效期自然退出，后续不再签发无类型普通用户 token。

### 8. 自定义 HTTP 错误保留框架兜底语义

结构化领域错误继续通过全局 `httpx` error handler 输出指定状态和 JSON payload；非结构化错误保持既有 HTTP 400 文本响应，gRPC status error 保留 go-zero 默认的 HTTP 状态映射，避免荣誉墙 403 支持意外改变其他 handler 的错误语义。

## Risks / Trade-offs

- **[每个 JWT 用户请求增加一次用户主键查询]** → 首期接受简单一致的数据库校验；使用现有连接池与主键索引，观察请求延迟后再决定是否增加短 TTL 缓存。
- **[数据库故障可能被误判为用户无效]** → 只有明确的不存在或 `status != 1` 返回 401；查询错误必须返回 5xx，客户端保留会话。
- **[并发 401 造成多次 toast 和导航]** → refresh 与 session invalidation 分别采用 single-flight 控制，并用 `_isHandled` 阻止页面重复反馈。
- **[启动验证与页面请求并发]** → 共用一次验证/失效状态；所有路径调用同一个幂等 logout，不依赖执行顺序。
- **[旧客户端收到新的 SESSION_INVALID 401 后仍先尝试 refresh]** → refresh 会因用户不存在而失败并触发现有退出逻辑，功能可回退；新版客户端会直接跳过 refresh。
- **[账号合并 token 已签发但响应丢失]** → 旧 token 对应用户已删除，下一请求会得到 401；用户重新微信或手机号登录后 OAuth 已指向目标用户，可恢复会话。

## Migration Plan

1. 先部署后端 active-user middleware、稳定 401 payload、短信账号合并的新 token 响应和回归测试。
2. 使用有效用户、已删除用户、停用用户及负数管理员 token 验证路由行为；确认数据库错误不会返回 401。
3. 发布移动端请求层 single-flight 失效处理、完整 store 清理、App 恢复验证和荣誉墙分类反馈。
4. 验证历史持久化脏会话首次请求后会自动退出，重新微信登录后 OAuth 指向的有效用户可正常进入荣誉墙。
5. 观察 401 数量、用户查询耗时和重复登录反馈；不运行数据迁移。

回滚时可先回滚客户端分类展示，再移除 active-user middleware；`BindPhoneResp` 的加法字段和客户端兼容读取可以保留。回滚不需要恢复数据库结构或用户数据。

## Open Questions

无阻塞问题。停用用户首期统一使用“登录状态已失效”以避免暴露账号状态；若后续产品需要展示封禁原因，应单独设计可公开的状态码与申诉入口。
