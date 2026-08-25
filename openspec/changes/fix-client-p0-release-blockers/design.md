## Context

本变更只处理 App / 小程序用户端上线前的 P0 阻断项，但会触及客户端、共享后端和发布配置：

- `login-by-oauth` 当前直接使用客户端上传的 `provider/open_id/union_id` 查找或创建账号；HarmonyOS 页面也只上传 SDK 返回的 OpenID，没有服务端 Provider 验证。
- 后端当前以 `rest.WithCors("*", ...)` 开放 HTTP CORS，WebSocket `CheckOrigin` 无条件返回 `true`。
- 用户与对局 WebSocket 均把 access token 放在 URL 查询参数，客户端在固定 5 次失败后永久停止自动重连。
- 匹配码与裁判码的正文会被拼入 `api.qrserver.com` URL；普通匹配码还是可由客户端修改的用户资料 JSON。
- 多个扫码入口只在 `APP-PLUS || APP-HARMONY` 编译，MP-WEIXIN 点击后无法完成同等流程。
- `app/manifest.json` 中 Harmony 默认与 release 签名字段均非空，并含本机绝对路径；Android 权限清单包含多项没有当前用户链路依据的高敏权限。
- 客户端没有个人数据导出与账号注销入口；服务端用户模型虽支持状态和软删除，但没有面向用户的完整数据收口事务。
- 请求层已经能区分部分错误类别，但多处页面 catch 后只记录日志，随后以空数组渲染“暂无数据”。

约束如下：

- API 契约以 `backend/chasing_points.api` 为唯一真源，修改后必须立即运行 goctl；生成文件不得手改。
- 页面只能经 `app/api/*.js` 调用后端；跨端规则应进入 `app/utils/*.js` 并补 `app/tests/*.test.mjs`。
- WebSocket 需要同时支持 App、HarmonyOS、微信小程序，并保留公开对局匿名只读观赛能力。
- 用户级读取和导出必须认证隔离、批量有界，旧身份响应不得写入新会话。
- 本变更不修改 `website/` 或 `admin/` 页面；共享后端 CORS 配置仍需保留正式管理端来源。
- 签名材料轮换属于代码外的发布动作，没有轮换证明时本变更不得被视为上线通过。

## Goals / Non-Goals

**Goals:**

- 关闭报告中全部 11 项 P0：OAuth 信任、CORS、WS Origin、WS URL token、签名材料、过度权限、账号数据生命周期、固定重连上限、第三方二维码、小程序/PK 闭环和错误态伪空。
- 给每项 P0 建立可自动化或可人工留证的上线验收标准。
- 在 App、HarmonyOS 和微信小程序之间提供一致、可信且可恢复的关键链路。
- 保持现有对局、会话恢复、账号合并和公开观赛边界不倒退。

**Non-Goals:**

- 不处理官网或 admin UI，也不重做整体视觉系统。
- 不在本变更内补齐动态 UGC 上传、会员支付、Push 全平台覆盖、战报 `0:0` 或其他 P1/P2 成熟度问题。
- 不引入通用请求缓存、通用页面框架或与 P0 无关的代码重构。
- 不为旧版不安全 OAuth 或 query access token提供生产环境长期兼容模式。

## Decisions

### 1. 以一个发布门禁变更交付，按依赖顺序实施

实施顺序固定为：发布秘密止血与配置骨架 → OAuth / WebSocket 安全边界 → 可信二维码与跨端扫码 → 账号生命周期 → 页面真实状态 → 全量门禁验证。每一阶段均先补可失败的测试，再改实现。

选择统一变更而不是拆成互不关联的多个变更，是因为 OAuth 复核会被账号注销复用，WebSocket 鉴权会被会话注销收口复用，二维码安全又与小程序 PK 闭环共用协议。任务仍按能力分组，便于独立提交与回滚。

### 2. 第三方登录使用服务端 Provider 适配器

`LoginByOauthReq` 改为提交 `provider`、`credential`、`credential_type`、`platform` 及可选展示资料；删除或不再接受作为身份依据的 `open_id/union_id`。后端在 `internal/pkg` 下增加最小的 Provider verifier 接口，并由 `ServiceContext` 注入当前唯一需要的 Huawei 实现。

verifier 返回规范化的 `{provider, subject, unionID}`，只有该结果能参与 OAuth 查找与创建。Huawei 客户端使用环境变量注入的应用配置、受限超时和官方接口；配置缺失、上游异常或验证失败均失败关闭。客户端昵称和头像只作为待清洗的展示建议，不能改变身份映射。

OAuth 绑定写入事务中完成，并通过 migration 为 `(provider, open_id)` 增加唯一约束，处理并发首登。Provider 错误统一分为 `unsupported`、`rejected`、`unavailable` 和 `invalid_response`，完整凭据不进入日志。

替代方案：继续信任客户端 OpenID，改动最小但无法关闭身份伪造；只在客户端校验 Provider 响应也无法建立服务端信任边界，因此不采用。

### 3. WebSocket 使用短期单次 ticket，而不是长期 token 或平台专属 Header

在 `user` 与 `match` API 组分别增加 ticket 申领接口。客户端先通过现有 Bearer REST 请求申领 ticket；服务端验证用户状态、通道类型、目标比赛及权限后生成高熵随机值，只在 Redis 保存其哈希、用户 ID、通道 scope、比赛 ID 和最多 30 秒 TTL。

WebSocket URL 只携带 `ticket` 与非敏感的 `match_id`。Handler 使用原子读取并删除消费 ticket，重放同一 ticket失败。原始 access token与完整 ticket均不得进入 URL 日志、结构化字段或错误响应。公开对局匿名观赛不申领 ticket，仍按公开只读边界连接。

选择 ticket 而不是直接依赖 `Authorization` WebSocket Header，是为了避免 App、HarmonyOS 与 MP-WEIXIN 对自定义握手 Header 的差异，同时让 ticket申领的 401 / 403 / 5xx 可以沿用成熟的 HTTP 错误分类。代价是每次认证连接或重连多一次 REST 与 Redis 操作；P0 安全和跨端确定性优先于这点开销。

### 4. HTTP CORS 与 WebSocket Origin 共用显式安全配置

后端配置新增明确的 HTTP Origin 列表、WebSocket Origin 列表和“是否允许原生客户端缺失 Origin”开关。生产环境启动校验禁止 `*`，并要求至少存在正式来源；本地开发使用独立的 localhost 列表。管理端正式域名通过配置保留，不需要修改 admin 代码。

WebSocket 请求携带 Origin 时必须精确匹配；缺失 Origin 只在原生开关开启且使用安全传输时允许，之后仍执行 ticket、用户、比赛可见性和角色校验。Origin 只防浏览器跨站升级，不被当作身份认证手段。

### 5. 两条 WebSocket 复用生命周期感知的重连策略

从 `MatchWebSocket` 与 `UserWebSocket` 中提取小型纯逻辑重连策略，状态包含连接 generation、连续失败次数、前后台、网络状态、人工断开和目标 scope。退避采用指数增长、约 20% 抖动和 60 秒延迟上限，不设置永久尝试次数上限。

每次认证重连先申领新 ticket。明确的 `SESSION_INVALID` / 401 停止当前 auth generation 的重连并进入全局会话失效；403 或无权访问停止目标通道；网络与 5xx 按退避继续。运行时无法返回握手状态时，客户端通过一次认证隔离的 Bootstrap / 会话验证区分失效会话与瞬时网络故障。

App 进入后台、网络离线、logout、切号或页面离开目标对局时取消定时器和心跳；回前台或网络恢复后立即安排一次受 generation 保护的恢复。连接成功重置退避，对局通道主动请求 snapshot。

### 6. 二维码本地渲染，匹配身份改为服务端可验证邀请

App 增加一个统一二维码渲染 wrapper，使用锁定版本、经过许可证与体积检查的 UniApp 兼容离线编码器（`uqrcodejs` 4.0.7，Apache-2.0，npm 展开约 68.7 KB）。页面只传正文、尺寸和 canvas 标识，不再构造外部图片 URL。匹配码、个人页匹配码和裁判码共用该 wrapper。

普通匹配码不再携带可修改的 `user_id/nickname/avatar/ts`。后端签发短期匹配邀请 token，内容只包括 purpose、邀请用户 ID、签发时间、过期时间和随机标识，并使用独立于 access token的服务端密钥签名。新增邀请预览接口；客户端扫码后先由服务端验证 token并获取对手展示资料，开局请求携带 invite token，服务端从 token和数据库重新确定对手，忽略客户端身份字段。

已接受挑战继续携带 `challenge_id`，并复用现有参与者、球种与原子 `match_id` 关联校验。裁判码沿用已有 join token语义，只替换图像渲染与跨端扫码入口。

替代方案：由后端返回二维码图片仍会增加传输和缓存面，并不能解决第三方依赖；继续使用客户端 JSON 无法防止伪造，因此不采用。

### 7. 统一扫码 helper 并正式支持 MP-WEIXIN

建立共享扫码 helper，负责平台能力检测、调用 `uni.scanCode`、识别匹配邀请与裁判邀请、区分取消/权限/内容错误以及生成后续 API payload。首页、对局页、排行榜和“我的”只负责选择业务上下文并调用 helper。

现有 `APP-PLUS || APP-HARMONY` 条件扩展到 `MP-WEIXIN`。不支持扫码的其他构建目标在渲染入口前显式降级；不保留点击无响应分支。挑战上下文继续通过认证隔离的短期本地状态传递，并在消费、logout、切号或失败后清除。

### 8. 数据导出使用有界游标，客户端组装文件

新增个人数据导出 API，首个请求返回格式版本、生成时间、分类元数据、第一批数据和不透明签名游标；后续请求通过游标按固定上限读取下一批。游标绑定用户 ID、导出快照时间、分类和最后主键，客户端不能修改 scope。所有数据库读取进入对应 model，并使用 keyset / limit，不做 `ListAll` 或 Go 内存分页。

客户端逐批组装 UTF-8 JSON，完成后再写入临时文件并校验条目数；任何批次失败都删除不完整临时文件。App / HarmonyOS 和微信小程序分别通过一个平台文件适配层保存或分享，无法保存的目标在操作前明确说明。

导出 schema 明确排除 token、验证码、Provider 凭据、完整 OAuth 标识、推送 token及其他用户私密字段。与他人相关的记录只保留理解当前用户事实所需的公开快照。

选择客户端组装而不是服务端持久化导出文件，可避免引入含敏感数据的对象存储和清理 worker，同时满足数据库读取有界；代价是客户端实现平台文件适配和进度状态。

### 9. 注销采用强复核、事务去标识化和立即会话失效

设置页新增“导出个人数据”和危险语义的“注销账号”。注销请求必须带不可逆确认；手机号用户使用新增 `delete_account` 短信场景复核，纯 OAuth 用户提交新的 Provider 凭据并走同一服务端 verifier。

注销 service 在用户行锁事务内执行以下收口：

1. 再次确认用户存在且状态有效，并将并发注销收敛为同一结果。
2. 使用明确 `account_deleted` 原因结束或取消进行中对局，避免悬挂状态。
3. 删除 OAuth 关联、推送标识、通知偏好、好友/关注/挑战等可删除关系以及用户可删除的社交内容。
4. 将需要保留的比赛事实和审计事实去标识化，清理冗余昵称、头像和其他直接身份快照。
5. 清空手机号、昵称、头像、会员和其他直接身份字段，设置已注销状态并软删除用户行。

提交后通知相关连接和资源失效，主动断开该用户 WebSocket。现有 ActiveUserSession 中间件会把所有旧 JWT 解释为 `SESSION_INVALID`；客户端收到成功后使用现有全局 logout 清理全部 Store 和持久化键。

实现前先建立一份代码内数据分类清单，逐表标注 `export`、`delete`、`anonymize` 或 `retain-fact`，并用集成测试证明没有直接身份残留。保留比赛事实是为了不破坏另一参赛者的真实记录，但展示身份统一变为“已注销用户”。

### 10. 页面使用最小统一异步状态模型

在 `app/utils` 增加纯逻辑状态 helper，状态只包含 `idle/loading/ready/empty/error/refreshing`、错误类别和是否已有旧数据。请求层继续负责构造稳定错误类别，页面负责选择文案和操作；不新增通用请求缓存。

首屏只有在成功响应且权威数据为空时进入 `empty`；首次失败进入带重试的 `error`；已有数据的刷新失败保留数据并显示非阻塞错误。`SESSION_INVALID` 或 superseded 请求不覆盖页面状态。优先覆盖报告范围内的核心数据页，并为每类页面补状态矩阵测试。

选择轻量 helper 而不是全量重写页面或引入 UI 状态库，能在保持现有组件结构的前提下关闭“失败伪空”P0。

### 11. 发布配置由静态契约和人工证据共同守护

从 `app/manifest.json` 删除 Harmony 签名密码、证书/profile/store 路径和本机绝对路径；私密扩展名与本地签名配置进入 `.gitignore`。发布流程从 IDE 本地安全配置或 CI secret 注入，不在仓库提供可用秘密。

HBuilderX 本地运行与打包通过 `app/harmony-configs/build-profile.json5` 接入本机签名：HBuilderX 会在编译前将 `harmony-configs` 覆盖到临时 Harmony 工程；该文件由受版本控制的安全模板和被忽略的 `.harmony-signing.local.json` 生成。调试运行使用 `default`，本地发布包使用 `release`。安全版 Manifest 保持空 `signingConfigs`，以免 HBuilderX 用 Manifest 中的材料覆盖本地 build profile。

建立 Manifest 契约测试：拒绝非空签名秘密、绝对用户路径、未批准高敏 Android 权限和未启用模块带来的权限。首轮移除没有当前链路依据的 `READ_LOGS`、`GET_ACCOUNTS`、`READ_PHONE_STATE`、`WRITE_SETTINGS`、存储挂载与网络状态修改权限；相机、网络、位置、Push 等保留项必须对应实际入口与运行时申请。

自动化还检查 CORS / WS 不允许通配、源码不含 query access token或 `api.qrserver.com`、OAuth 请求不再接受客户端身份。签名轮换、正式 Origin 值、平台隐私清单和真机矩阵由人工发布记录补齐。

## Risks / Trade-offs

- [旧客户端无法使用新 OAuth、WebSocket 和匹配码协议] → 后端与客户端协同发布；生产不保留不安全降级，开发环境可短期保留仅用于迁移验证的明确开关并默认关闭。
- [WebSocket ticket使 Redis 成为认证连接依赖] → ticket存储接口可测试，Redis 故障返回 503 并由客户端退避；不得在故障时回退 query access token。
- [无限次数重连可能耗电或形成请求风暴] → 后台/离线暂停、指数退避、抖动、60 秒上限和单连接 generation 限制。
- [Origin 严格化误伤原生客户端或正式域名] → 配置区分“缺失 Origin 的原生客户端”和显式浏览器 Origin，上线前用完整平台矩阵验证；不使用自动反射。
- [Huawei 官方凭据形态或上游接口发生变化] → Provider verifier 隔离外部协议，锁定超时和错误映射；客户端请求契约不暴露服务端密钥。
- [离线二维码依赖在某平台 canvas 行为不同] → 通过单一 wrapper隔离依赖，固定版本并做三端真机渲染、长内容和深色主题验证。
- [账号注销跨表清理遗漏或破坏历史对局] → 先完成逐表分类清单，在单事务与外键/唯一约束下执行，并以删除前后快照集成测试验证。
- [个人数据量过大导致客户端内存或文件失败] → 后端游标批量、客户端增量写文件、显示进度和完成前不暴露文件；对单批大小设硬上限。
- [错误态覆盖页面较多造成回归] → 只引入最小状态 helper，按核心页面清单逐页改动和测试，不顺带重构视觉或数据 Store。
- [签名轮换需要仓库外权限] → 代码任务先移除与阻断旧材料，发布清单把轮换证明设为不可绕过的人工门禁。

## Migration Plan

1. 立即从 Manifest 移除签名秘密和绝对路径，补忽略与扫描规则；由发布负责人创建并保存新签名材料，旧材料标记废弃。
2. 增加后端安全配置、OAuth 唯一索引及账号生命周期所需 migration；修改 `.api` 后立即运行 goctl，并保持生成文件与真实 logic 一致。
3. 先部署支持新 OAuth、WS ticket、可信匹配邀请、导出和注销的新后端。生产配置在客户端切换前保持新接口可用，但不开放旧身份信任或 query access token。
4. 发布同步升级的 App / HarmonyOS / 微信小程序客户端，切换 OAuth、ticket、本地二维码、共享扫码和真实页面状态。
5. 开启生产 Origin / CORS 强校验，确认所有正式来源、WebSocket 连接和匿名公开观赛正常。
6. 执行后端全量测试、App 全量测试、静态安全契约和三端真机矩阵；完成账号导出/注销数据快照核验与签名轮换记录。
7. 只有所有 P0 证据齐全后，才解除发布阻断。

回滚时可以关闭尚未稳定的导出、注销或扫码入口并回退同版本客户端，但 MUST NOT 恢复通配 CORS / Origin、客户端 OpenID 信任、query access token、第三方二维码或旧签名材料。若新实时通道故障，应修复 ticket链路或暂时降级为无实时提示的 HTTP 刷新，不得回到泄露长期凭据的协议。

## Open Questions

- 无阻断性的产品或架构问题。实施前需要由发布环境提供正式 HTTP / WebSocket Origin 列表、Huawei 服务端验证配置和新签名材料；这些是上线输入，不改变上述设计。
- 数据保留分类按“直接身份立即删除、可删除关系删除、比赛事实去标识化保留”执行；若法务后续要求不同保留期限，应只调整分类与期限，不得恢复已清除的直接身份信息。
