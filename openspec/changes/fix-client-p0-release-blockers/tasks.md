## 1. 发布秘密与 Manifest 先行止血

- [x] 1.1 从 `app/manifest.json` 删除 Harmony `default/release` 的非空签名密码、证书/profile/store 路径及本机绝对资源路径，并确认仓库不再包含可用签名秘密。
- [x] 1.2 在仓库忽略规则中加入 Harmony / Android / iOS 私钥、证书、keystore、profile 和本地签名配置模式，并提供 HBuilderX 本地 `build-profile.json5` 生成说明与安全模板。
- [x] 1.3 逐项映射当前 App 功能到 Manifest 模块与权限，移除无已批准用例的 `READ_LOGS`、`GET_ACCOUNTS`、`READ_PHONE_STATE`、`WRITE_SETTINGS`、存储挂载和网络状态修改权限。
- [x] 1.4 调整相机与位置权限触发点，验证只有用户进入扫码或附近球房时才请求对应权限，并补拒绝权限的明确反馈。
- [x] 1.5 增加 App 静态安全契约测试，拒绝非空签名字段、用户目录绝对路径、未批准高敏权限和 Manifest 中无依据的发布模块。
- [x] 1.6 将 Manifest 所引用的 Android/iOS 图标及 Harmony 图标、启动页资源纳入版本控制，并用静态契约验证干净工作区资源完整性；签名材料继续保持本地忽略。

## 2. 后端安全配置、契约与数据约束

- [x] 2.1 在后端配置中增加生产 HTTP Origin、WebSocket Origin、原生缺失 Origin 开关、Huawei 服务端验证配置和独立匹配邀请签名配置，并全部支持环境变量注入。
- [x] 2.2 增加生产配置校验与测试：Origin 为空或包含 `*`、已启用 Provider 缺少验证配置、匹配邀请密钥缺失时必须失败关闭。
- [x] 2.3 将 `rest.WithCors("*", ...)` 替换为配置驱动的精确 HTTP CORS 策略，保留正式管理端来源但不修改 admin 页面，并补允许/拒绝/本地环境测试。
- [x] 2.4 为 WebSocket 实现统一 Origin 判定 helper 与单元测试，覆盖允许来源、拒绝来源、原生缺失 Origin 和生产错误配置。
- [x] 2.5 新增 MySQL migration，为 `user_oauth(provider, open_id)` 建立兼容现有 MySQL 的唯一约束，并为重复数据提供可审计的迁移前检查。
- [x] 2.6 一次性更新 `backend/chasing_points.api`：替换通用 OAuth 凭据字段，增加用户/对局 WS ticket、匹配邀请预览与 `invite_token`、个人数据游标导出和账号注销契约；修改后立即运行 goctl 生成代码。
- [x] 2.7 检查 goctl 生成结果，将新 logic 保持在对应 `internal/logic/<group>/`，确认未手改生成文件且后续任务覆盖全部新 handler 空壳。

## 3. 服务端验证的第三方登录

- [x] 3.1 在 `backend/internal/pkg` 增加最小 Provider verifier 接口、规范化身份和稳定错误类别，并用 fake verifier 覆盖成功、拒绝、超时和异常响应。
- [x] 3.2 实现 Huawei verifier：使用受限超时与官方验证链路校验 credential 的应用归属和有效性，确保日志不含 credential、access token、完整 OpenID 或 UnionID。
- [x] 3.3 将 Huawei verifier 从 `ServiceContext` 注入认证 logic，配置缺失或上游故障时返回稳定 4xx/5xx 类别且不得回退客户端 OpenID。
- [x] 3.4 重写 `LoginByOauthLogic`，只使用 verifier 返回的 Provider 身份，在事务中幂等创建用户与 OAuth 关联，并拒绝停用、删除或注销用户。
- [x] 3.5 增加 OAuth 并发首登、伪造 OpenID、Provider 不支持、凭据过期、Provider 故障和停用用户的后端测试，证明不会创建重复或替代账号。
- [x] 3.6 更新 `app/api/auth.js` 与 Harmony 登录页，只上传新的 Provider credential和最小展示字段，移除 `open_id/union_id` 身份参数及敏感日志。
- [x] 3.7 增加 App 登录契约与组件测试，覆盖成功、授权失效、上游暂不可用、协议未同意和旧 OpenID 请求不再出现。

## 4. 安全且可恢复的 WebSocket 通道

- [x] 4.1 实现可注入的 WS ticket store，生成高熵 ticket、只存哈希与 scope、TTL 不超过 30 秒，并通过 Redis 原子消费阻止重放。
- [x] 4.2 实现用户 WS ticket logic，验证 access token对应启用用户后签发 `user` scope ticket，并覆盖 refresh token、注销用户和 Redis 故障测试。
- [x] 4.3 实现对局 WS ticket logic，验证目标比赛、可见性和参赛者/裁判权限后签发绑定 `match_id` 的 ticket，并保留公开比赛匿名只读规则。
- [x] 4.4 改造 `/api/user/ws` 与 `/api/match/ws` handler：应用 Origin 策略、消费 ticket、拒绝 `?token=` access token，并确保 URL、日志和指标不记录完整凭据。
- [x] 4.5 增加 WebSocket 集成测试，覆盖允许/拒绝 Origin、有效 ticket、过期 ticket、重放、refresh token、私密比赛越权、公开匿名和注销用户。
- [x] 4.6 在 `app/api/` 增加用户与对局 ticket申领门面，并确保错误沿用请求层的 network/session/permission/server 分类。
- [x] 4.7 在 `app/utils` 增加指数退避、抖动、60 秒上限、前后台/网络/人工断开与 connection generation 的纯逻辑策略及单元测试。
- [x] 4.8 改造 `MatchWebSocket`，每次认证连接申领新 ticket、移除 query access token与固定 5 次上限，并在重连成功后请求最新 snapshot。
- [x] 4.9 改造 `UserWebSocket`，复用同一重连策略和 ticket流程，明确会话失效停止、权限失败停止、网络/5xx 继续退避。
- [x] 4.10 在 `App.vue` 与对局页面接入前后台和网络状态，确保后台/离线/logout/切号/卸载取消定时器与心跳，恢复时不接受旧 generation 回调。
- [x] 4.11 增加 App WebSocket 测试，覆盖长时间断网后恢复、固定 5 次后仍可恢复、会话失效、切号、后台暂停、人工断开和两条通道行为一致性。

## 5. 可信二维码、微信小程序扫码与 PK 闭环

- [x] 5.1 评审并锁定 UniApp 兼容的离线二维码依赖（首选 `uqrcodejs`），记录许可证与包体影响，并添加到 App 依赖锁定文件。
- [x] 5.2 在 `app/utils` 实现统一二维码渲染 wrapper与纯逻辑测试，支持匹配码和裁判码且不访问任何第三方二维码域名。
- [x] 5.3 实现服务端短期匹配邀请 token helper与测试，使用独立密钥签名 purpose、邀请用户、签发/过期时间和随机标识，拒绝篡改与过期 token。
- [x] 5.4 实现匹配邀请预览 logic，从 token和数据库返回可信对手信息，且不接受客户端昵称、头像或用户 ID 作为授权依据。
- [x] 5.5 改造开始对局 logic，生产请求必须携带有效 `invite_token`，由服务端解析对手；同时保持 `challenge_id` 双方、球种和原子关联校验。
- [x] 5.6 增加匹配邀请与挑战闭环后端测试，覆盖篡改、过期、错误对手/球种、双方并发开局、重复请求收敛同一 `match_id` 和进行中对局保护。
- [x] 5.7 更新 `app/api/match.js` 和开局 payload helper，加入邀请预览与 `invite_token`，停止解析可修改的 `user_id/nickname/avatar/ts` 为可信身份。
- [x] 5.8 将个人页、对局页和进行中对局裁判弹窗的 `api.qrserver.com` URL 全部替换为本地二维码渲染，并补加载、失败和刷新状态。
- [x] 5.9 在 `app/utils` 实现统一扫码 helper，区分匹配邀请、裁判邀请、取消、权限拒绝、无效内容和过期内容，并补纯逻辑测试。
- [x] 5.10 将首页、对局页、排行榜和“我的”的扫码入口改为统一 helper，并把有效平台条件扩展到 `MP-WEIXIN`；不支持平台必须显式隐藏或说明。
- [x] 5.11 接通微信小程序的匹配邀请预览、正式开局、裁判预览/加入及接受挑战后的 `pending_match_challenge` 消费与清理。
- [x] 5.12 增加 App 组件/契约测试，证明各入口生成相同请求，取消或权限拒绝不创建对局，logout/切号不会复用旧挑战上下文。

## 6. 个人数据导出与账号注销

- [x] 6.1 用 CodeGraph 盘点所有用户相关表和 denormalized 身份字段，建立逐表 `export/delete/anonymize/retain-fact` 分类清单并限定在用户端数据范围。
- [x] 6.2 为个人数据导出实现各领域有界 model 查询，使用 keyset 与硬性 batch 上限，禁止 `ListAll`、无界历史扫描和跨用户私密字段。
- [x] 6.3 实现绑定用户、快照时间、分类和最后主键的不透明签名游标，以及格式版本、条目计数和完成标识测试。
- [x] 6.4 实现个人数据导出 logic，覆盖资料、认证关联摘要、设置、关系和业务记录，排除 token、验证码、完整 OAuth 标识、推送 token和其他用户私密数据。
- [x] 6.5 增加导出后端测试，覆盖多批次、篡改游标、切换用户、空数据、依赖故障、无秘密字段和查询有界性。
- [x] 6.6 扩展短信验证码 scene 支持 `delete_account`，并为手机号复核增加有效、错误、过期和重放测试。
- [x] 6.7 实现 OAuth 注销复核，复用 Provider verifier 验证新的凭据，拒绝旧客户端 OpenID 或仅凭 access token注销。
- [x] 6.8 在 model 层实现注销事务所需的用户行锁、OAuth 删除、关系/偏好/社交清理和冗余身份匿名化方法，不在 logic 直接操作数据库。
- [x] 6.9 实现账号注销 service：处理进行中对局 `account_deleted` 收口、逐表清理、比赛事实去标识化、清空直接身份并软删除用户。
- [x] 6.10 为 WebSocket Hub 增加按用户断开能力并在注销提交后调用，验证旧 access/refresh token、HTTP 和两条 WS 通道立即进入 `SESSION_INVALID`。
- [x] 6.11 增加注销集成测试，覆盖短信/OAuth 复核失败不改数据、成功清理、历史对局匿名展示、进行中对局收口、并发注销幂等和旧 OAuth 不恢复原账号。
- [x] 6.12 在 `app/api/user.js` 增加数据导出和账号注销门面，并在 `app/utils` 实现游标拉取、增量 JSON 组装、完整性校验和失败清理。
- [x] 6.13 实现 App / HarmonyOS 与 MP-WEIXIN 的本地文件保存或分享适配；无法保存的平台必须在开始前给出明确限制。
- [x] 6.14 在设置页增加“导出个人数据”和危险语义“注销账号”流程，包含进度、不可逆说明、二次确认、短信或 OAuth 复核及成功后的全局 logout。
- [x] 6.15 增加设置页与导出 helper测试，覆盖成功、分段失败、不完整文件删除、重复点击、注销取消/失败/成功和会话状态清空。

## 7. 真实加载、空数据与错误反馈

- [x] 7.1 在 `app/utils` 增加最小异步页面状态 helper，覆盖 `idle/loading/ready/empty/error/refreshing`、旧数据保留、错误类别和 auth generation 淘汰测试。
- [x] 7.2 核对并补齐请求层 network/session/permission/not-found/rate-limit/server/business 分类，确保 handled/superseded 错误可被页面无重复副作用地识别。
- [x] 7.3 盘点所有发布范围数据页面，形成“成功有数据、成功空数据、首次失败、刷新失败、权限/会话失败”状态矩阵，并标记现有 catch 后伪空点。
- [x] 7.4 将首页/Bootstrap、对局大厅和排行榜接入状态 helper，首次失败显示可重试错误，已有数据刷新失败保持旧数据。
- [x] 7.5 将对局历史、统计、H2H、对手记录、荣誉与信誉页面接入状态 helper，区分空记录、权限、资源不存在和服务故障。
- [x] 7.6 将通知、好友申请、好友列表、PK 邀约和相关社交列表接入状态 helper，避免 catch 后空数组渲染“暂无消息/暂无邀约”。
- [x] 7.7 将赛事/赛季、球房、规则及其详情页面接入状态 helper，提供与错误类别匹配的重试、返回或权限提示。
- [x] 7.8 为上述页面补纯逻辑或 SFC 挂载测试，证明成功空数据与 network/5xx/403/404/SESSION_INVALID 不会落到同一 UI 状态。
- [x] 7.9 增加静态契约检查或显式页面清单测试，阻止核心页面重新出现“首次 catch 只记录日志后渲染空状态”的模式。

## 8. 全量验证与上线门禁

- [x] 8.1 运行并修复 `backend` 全量 `go test ./...`，同时确认新增热点读取、导出和 WebSocket ticket测试保持查询与依赖操作有界。
- [x] 8.2 在 `app` 运行并修复 `node --test tests/*.test.mjs`，确认登录、WebSocket、扫码、二维码、账号生命周期和错误状态契约全部通过。
- [x] 8.3 运行仓库秘密与安全契约扫描，确认无签名秘密、绝对用户路径、客户端 OpenID 信任、query access token、通配 Origin/CORS 或 `api.qrserver.com`。
- [ ] 8.4 用生产等价配置启动后端，验证正式 HTTP / WebSocket Origin、缺失 Origin 原生策略、Provider 配置和 Redis ticket故障均按设计失败关闭。
- [ ] 8.5 完成 App、HarmonyOS、微信小程序真机矩阵：OAuth、匹配码展示与扫码、裁判扫码、PK 邀约闭环、弱网重连、后台恢复、权限拒绝和错误态。
- [ ] 8.6 完成个人数据导出文件人工抽检与账号注销前后数据库快照核对，确认无认证秘密、直接身份残留或悬挂进行中对局。
- [ ] 8.7 由发布负责人轮换所有曾进入仓库的签名材料，将新材料迁入受控存储，并记录旧材料废弃和新包签名验证结果。
- [ ] 8.8 记录正式允许 Origin、Huawei 服务端配置、Manifest 权限用途、签名轮换和三端真机证据；任一证据缺失时保持 P0 发布阻断。
- [ ] 8.9 运行 `openspec validate fix-client-p0-release-blockers --type change --strict --no-interactive` 并复核所有任务、规范场景与实际实现一一对应后，才将变更交付上线评审。
