## Context

现状：

- 匹配二维码：`GET /api/match/qrcode` 用 `matchinvite.Signer` 签发 `base64url(JSON 内容).base64url(HMAC)` 形式的 token，`qrcode_data` 就是这个 token，有效期 `Security.MatchInvite.TTLSeconds`（300 秒）。token 内容里有明文的 `inviter_user_id`。客户端用 `uqrcodejs` 在本地画码。
- 端内扫码：`quick-scan.js` → `match-scan.js` 解析扫码内容（裸 token 或裁判码 JSON）→ 调用需要登录的 `POST /api/match/invite/preview` → 把凭证放在内存里交给 `scanConfirm`。确认页校验「交接时的身份 = 当前身份」，凭证不进路由和存储。扫码前必须已登录，否则先跳登录页。
- 登录页：小程序里同意协议后走 `getPhoneNumber` 快捷登录；登录成功后页面栈多于一页就 `navigateBack`。
- 分享：整个 UniApp 没有 `onShareAppMessage`。战报海报（`posterGenerator.js`）只能存相册，上面没有二维码。交手报告页（`pkReport.vue`）有海报和复制文案。
- 公开读取：`GET /api/public/match/detail` 不需要登录，只返回可见性为公开的对局；`matchDetail.vue` 在 `mode=spectate` 时用它展示观战视角。观战时的 WebSocket ticket 需要登录。
- 官网：Nuxt 3，静态生成（`npm run generate`），域名 `www.zhuifen.cn`，已备案。下载页目前写的是 iOS/Android。
- 平台：UniApp 输出微信小程序和鸿蒙 App；原生 iOS 没有扫码功能。产品尚未上线，没有外部老客户端。
- 后端 Redis 用 `go-redis/v9`，已有 `wsticket.NewRedisStore` 这种短期凭证存储；没有现成的限流中间件；`utils.GetClientIP` 优先取 `X-Forwarded-For`。

## Goals / Non-Goals

**Goals:**
- 没装追分的人用微信扫匹配码，最少步骤进入开局：扫码 → 看到邀请人 → 快捷登录 → 确认页开局。
- 同一个码在微信、追分小程序端内、鸿蒙 App 端内都能扫；用其他工具扫也有去处。
- 比赛结果能以小程序卡片形式转进微信群，海报也能扫码进入。
- 签名 token 的格式、校验和开局规则不变；确认页的身份校验不变。

**Non-Goals:**
- 微信圆形小程序码（`wxacode.getUnlimited`）、分享到朋友圈、自定义分享卡片图片。
- 鸿蒙 App Linking（系统相机直接拉起 App）、鸿蒙系统分享。
- 公开交手记录页、裁判码改为链接、渠道来源统计、分享奖励。
- 调整 `hide_match_record` 对公开对局详情的作用范围。

## Decisions

**1. 用普通链接二维码 + 微信「扫普通链接二维码打开小程序」，不用圆形小程序码。**
- 圆形小程序码只有微信能识别，鸿蒙 App 的端内扫码（`uni.scanCode`）扫不了，而鸿蒙和小程序是同一套 UniApp 代码。
- 普通二维码可以继续在客户端本地画，不需要后端每次刷新都调用微信接口、管理 access_token、传图片。
- 用其他工具扫码时会打开官网，可以做兜底页；圆形码在微信以外完全打不开。
- 代价：需要在微信公众平台配置规则、放校验文件，且规则要在小程序发布后才能正式发布（上线前用体验版测试链接）。
- 备选：圆形小程序码。否决，理由同上，另外 `scene` 最多 32 个字符，放不下现有 token。

**2. 一个路径、用查询参数区分类型：`/go/?j=<短码>`、`/go/?m=<对局 ID>`、`/go/`。**
- 官网是静态生成，`/go/` 生成一个页面就能承接所有类型，不需要服务器配置路由重写。
- 微信只需要配置一条规则（前缀 `https://www.zhuifen.cn/go/`），对应一个承接页。
- 备选：按路径区分（`/go/j/<短码>`）。否决：静态托管需要额外的重写规则，微信也要配置多条规则或多个承接页。

**3. 链接里放短码，不放完整 token；短码存在 Redis，映射到 token。**
- 完整 token 约 130 个字符，放进链接后二维码要到 9 版左右（53×53）；短码链接约 45 个字符，3 版左右（29×29）。球房灯光暗、对着手机屏幕扫，模块越大扫得越快。
- token 内容里有明文的邀请人用户 ID。链接会经过微信的 `q` 参数、官网访问日志和浏览器历史，放短码只暴露一个不透明的随机串。
- 短码：12 位 base62（约 71 位熵），每次获取匹配二维码都生成新的；Redis 键 `match_invite_code:<短码>`，值为 token，TTL = token 剩余有效期。写入失败时接口返回失败，不返回不可兑换的码。
- 实现参照 `wsticket` 的 Redis 存储，放在 `backend/internal/pkg/matchinvite/` 旁边，便于单测。
- 备选：完整 token 进链接、不引入 Redis 状态。否决，理由如上。备选：把 token 压缩成二进制再截短签名。否决：要改签名格式，违背「token 规则不变」。

**4. 匿名预览放在 `/api/public` 组，只返回展示信息，并按 IP 限流。**
- 新接口 `POST /api/public/match-invite/preview`，请求 `{invite_code}`，响应 `{success, message, preview: {inviter_name, inviter_avatar, expires_at}}`。不返回 token 和用户 ID。
- 不存在、过期、token 校验失败、邀请人停用都返回同一句提示，不让调用者区分原因。
- 先校验短码格式（12 位 base62），再查限流，再查 Redis。
- 限流：Redis 固定窗口，键 `rl:invite_preview:<ip>`，每分钟 30 次。`INCR` 后第一次设置 `EXPIRE`。同一个球房的人可能共用一个出口 IP，30 次/分钟足够。
- Redis 出错时返回「匹配二维码服务暂不可用」，不放行。短码本身也存在 Redis 里，限流放行也查不到。
- 备选：不做限流。否决：这是新开放的匿名入口，按仓库规则必须做信任边界防护。短码的熵已经让枚举不可行，限流主要防止有人刷接口压 Redis。

**5. 登录后兑换用新接口 `POST /api/match/invite/resolve`，不扩展现有预览接口。**
- 请求 `{invite_code}`，响应 `{success, message, invite_token, preview: MatchInvitePreviewInfo}`。校验与现有 `PreviewMatchInvite` 一致：查短码 → `verifyMatchInvite` → 查邀请人状态。把共用部分抽成函数，两个接口一起用。
- 兑换不消耗短码，也不消耗 token。开局仍由现有开局接口凭 token 校验，不改开局逻辑。
- 备选：让现有预览接口同时接受 `invite_token` 和 `invite_code`。否决：一个接口两种输入、只在一种输入下返回 token，语义混乱，也更容易在测试里漏掉分支。

**6. 新增承接页 `subPages/match/linkEntry`，纯逻辑放在 `utils/entry-link.js`；确认页 `scanConfirm` 不改。**
- `entry-link.js`：`buildEntryLink({type, value})`、`parseEntryLink(raw)`、`decodeLaunchQuery(q)`。只认 `https://www.zhuifen.cn/go`，不读写全局状态，可以直接用 node 测试。
- 承接页 `onLoad` 读取 `q` 并分发：对局链接 → `redirectTo` 观战详情；首页或解析失败 → `switchTab` 首页；匹配码链接 → 进入扫码即入流程。
- 扫码即入流程：
  - 未登录：调用匿名预览，展示邀请人、剩余时间和「登录并开局」按钮，点击后 `navigateTo` 登录页。
  - 已登录（进入时就已登录，或在 `onShow` 时发现登录状态从无到有）：调用兑换接口，再用 `quick-scan.js` 新增的 `handoffResolved({identity, inviteToken, opponent})` 写入现有的内存交接，然后 `redirectTo` 确认页。
  - 用一个「本页是否已发起兑换」的标志，避免 `onShow` 重复兑换。
- 确认页还是从内存交接里取凭证、校验身份，路由里没有凭证。短码只在承接页内存里，不写存储、日志和埋点。
- 备选：让 `scanConfirm` 直接接收 `q`。否决：会打破「凭证不进路由」的现有约束，确认页也要多处理未登录状态。
- 冷启动时页面栈只有承接页，跳到登录页后栈里是两页，登录成功后登录页会 `navigateBack` 回到承接页，现有逻辑不用改。

**7. 端内扫码：`match-scan.js` 先识别入口链接，再走原有分支。**
- `parseMatchScanPayload` 的顺序：入口链接（匹配码 → `resolve_match_invite_code`；对局 → `open_match`；首页或其他网址 → 无效）→ 裁判码 JSON → 裸 token。
- `resolveMatchScanPayload` 多注入一个 `resolveMatchInviteCode`；兑换成功后返回和裸 token 一样的 `start_match` 结构（多带 `inviteToken`），所以 `quick-scan.js` 里「再来一局只接受原对手」的检查不用改。
- 新增 `open_match` 结果：`quick-scan.js` 直接 `navigateTo` 观战详情。
- 裸 token 分支保留。这样客户端可以先发布，后端后切换，顺序无关。

**8. 分享卡片用纯函数生成，只在微信小程序注册。**
- `utils/share-card.js`：`buildMatchShareCard(detail)` 同时接受参与者视角（`is_player1` + `my_*`/`opponent_*`）和公开视角（`player1_*`/`player2_*`），统一成选手 1/选手 2 的顺序；昵称超过 8 个字符截断加「…」。可见性不是公开、状态是已取消或数据为空时，返回通用卡片。`buildHomeShareCard()` 返回通用卡片。
- 页面用 `onShareAppMessage(() => buildMatchShareCard(当前数据))`，外面包 `// #ifdef MP-WEIXIN`。分享回调同步返回，用页面当时已经有的数据，不在分享时发请求。
- 卡片图片先不设置 `imageUrl`，用微信默认的页面截图；赛果页和观战页顶部本来就是比分区。
- 备选：分享回调里临时生成卡片图片。否决：要处理异步与失败，本期先看默认截图的效果。

**9. 观战页的未登录引导直接放在 `matchDetail.vue` 里。**
- 条件：`isSpectateMode && !userStore.isLoggedIn`。点击 `navigateTo` 登录页，登录后返回同一页；`onShow` 时登录状态变化，引导条自动隐藏。
- 观战页现有数据请求本来就走公开接口，未登录也能加载，不需要改接口。

**10. 海报二维码在本地画到海报画布上，不请求后端。**
- 用 `uqrcodejs` 生成模块矩阵（`make()`），在海报画布底部指定位置逐格绘制，四周留白，旁边配说明文字。不新增依赖。
- 二维码内容由 `buildEntryLink` 生成：公开对局 → 对局链接；私密对局和交手报告 → 首页链接。
- `MatchShareData` 新增 `visibility` 字段（取 `NormalizeMatchVisibility` 的结果）。海报页只有 `match_id`，可能从多个页面进入，由接口返回可见性最可靠。字段缺失时按私密处理，指向首页。
- 备选：从赛果页通过路由参数把可见性带给海报页。否决：入口不止一个，容易漏传；而且即使印错了链接，服务端的公开详情也会拒绝私密对局，但用户会扫到一个打不开的码。
- 二维码绘制用 `try/catch` 包住，失败时跳过二维码、照常生成海报，并 `console.error` 记录。

**11. 官网 `/go/` 页面在客户端读查询参数，不请求后端。**
- `website/pages/go/index.vue`：静态生成时渲染通用文案；客户端挂载后读 `route.query` 切换成匹配码或对局文案。页面加 `noindex`，不放进 `sitemap.xml`。
- 微信给的校验文件放在 `website/public/`，随官网一起发布。校验文件是公开内容，不是密钥。
- 下载页与首页常见问题改为「微信小程序 + 鸿蒙」：小程序展示小程序码图片（由用户提供）；鸿蒙在上架前显示「即将上线」。

## Risks / Trade-offs

- [微信扫码规则要等小程序发布后才能正式发布] → 上线前用体验版和测试链接验收（测试链接数量有限）；发布小程序当天发布规则，并列入上线清单。
- [聊天里长按海报能否通过规则打开小程序，需要真机确认] → 列入真机验收；如果不行，海报旁注改为「微信扫一扫」，不影响扫码开局主流程。
- [`X-Forwarded-For` 可以伪造，攻击者能绕过 IP 限流] → 短码有约 71 位熵，枚举不可行；限流只用来防刷接口。上线前核对生产反向代理是否覆盖 `X-Forwarded-For`，若不覆盖则记录为后续事项。
- [Redis 不可用时无法签发或兑换匹配码] → 和 WebSocket ticket 的依赖相同；接口明确返回失败，客户端显示「服务暂不可用」，不会生成扫不了的码。
- [手机号快捷登录按次计费，且要求小程序主体已认证] → 用户量小的时候费用可以忽略；上线前确认主体状态（见 Open Questions）。
- [未登录观战进行中的对局时没有实时更新（WebSocket ticket 需要登录）] → 赛果页分享的都是已结束的对局；进行中的对局显示打开时的比分，可以下拉刷新。本期不放开匿名 ticket。
- [分享卡片把公开对局的双方昵称和比分发进群，而对手可能设置了 `hide_match_record`] → 公开大厅和现有海报已经展示同样的信息，本期不改变；私密练习对局只分享通用卡片。是否让 `hide_match_record` 作用于公开详情，列为后续产品决定。
- [`qrcode_data` 改为链接后，旧版本客户端扫不了新码] → 没有外部用户；先发布能解析链接的客户端，再切换后端；回滚后端时客户端仍能解析裸 token。
- [昵称截断后分享标题仍然过长被微信截断] → 截断到 8 个字符，比分和球种放在昵称之后，被截断时优先保住比分。

## Migration Plan

1. 先发布客户端（开发环境）：端内扫码能解析入口链接，同时保留裸 token。此时后端还返回裸 token，行为不变。
2. 后端发布开发环境（`backend/deploy-dev.sh`）：新增两个接口，`qrcode_data` 改为链接。不改表结构，没有迁移。
3. 官网发布 `/go/`、校验文件和下载页修改。
4. 用户在微信公众平台配置「扫普通链接二维码打开小程序」：前缀 `https://www.zhuifen.cn/go/`，承接页 `subPages/match/linkEntry`，添加测试链接，测试范围选体验版。
5. 真机验收（见 tasks）：新用户微信扫码开局、已登录用户微信扫码、小程序端内扫码、鸿蒙端内扫码、系统相机扫码打开官网、另一个微信号打开分享卡片、长按识别海报二维码。
6. 用户确认后发布生产后端（`backend/deploy-prod.sh`）与官网；小程序正式发布后，在公众平台发布扫码规则。
7. 回滚：回退后端，`qrcode_data` 恢复为裸 token，客户端照常解析；Redis 里的短码最多 5 分钟后自然过期。

## Open Questions

- 小程序主体是否已经完成企业认证？手机号快捷登录依赖这一点。代码已在用，大概率没问题，需要用户确认。
- 官网目前怎么发布（静态托管位置）？需要确认能提供 `/go/` 和根目录的校验文件。
- 下载页上的小程序码图片需要用户从公众平台下载后提供。
