# RB10：帮助与反馈、用户协议与隐私政策、PK 报表

## 先修正上一轮约球的三处偏差（改完再做本轮任务）

1. **业务失败要保留完整响应体**。现在 HTTP 2xx + `success:false`（以及 4xx）时 `APIClient` 只把 `message` 放进 `APIError`，后端一起返回的结构化字段（发起约球时的 `pending_challenge_id`、`need_confirm_close_own`、`type_conflict`、`action` 等）全丢了，约球页只好靠匹配文案判断分支——后端改一个字就会走错。小程序的请求层是带着完整响应数据的（`createRequestError({ data })`）。
   - 给 `APIError` 加 `let responseBody: Data?`（`init` 增加带默认值 `nil` 的参数，现有调用不用改），`APIClient` 在业务失败和非 2xx 时都把原始响应体放进去（`description` / 日志里不要打印它）；再加一个便捷方法 `func decodeBody<T: Decodable>(_ type: T.Type) -> T?`（snake_case 解码，失败返回 nil）。
   - `iOS/iOS/Features/Challenge/` 里所有「按后端文案判断分支」的地方改为解码这些字段，分支规则逐条对照 `uniapp/subPages/match/challengeCompose.vue`（约 272–312 行）及其他约球页面的失败处理；删掉 `ChallengeRules` 里为匹配文案而存的常量。补测试：业务失败带 `pending_challenge_id` 时走「查看已有约球」分支、只有 `message` 时提示文案；`APIClientTests` 补一条：业务失败的 `APIError.responseBody` 等于原始响应、`decodeBody` 能取到字段。
2. **发起成功后的去向**：小程序发送成功后提示「约球已发送」并回到首页 Tab（`uni.switchTab` 到首页，约 304–309 行），首页约球卡会显示等待状态。原生现在直接进等待页——这是我上一轮规格写错了，按小程序改：Toast「约球已发送」→ 回到首页 Tab 根页面（`navigator.select(.home, ...)` + `popToRoot`），并 `ActivityStore.markDirty()` 触发补读。更新相应测试。
3. **更换对手要保留已填内容**：发起页「更换」去选择球友后，选完现在会新建一个发起页，已选的球种、时段、附言丢失；小程序是回到原发起页并保留。改为：选择球友页通过回调（或发起页模型上的选择交接，任选其一，但不要新建发起页）把选中的对手交回原发起页，原发起页保留其他字段只更新对手。为此允许最小改动 `iOS/iOS/Features/Match/OpponentSelectorView.swift` / `OpponentSelectorModel.swift`（只加回传选中对手所需的参数与调用，不动其他逻辑）。补测试。

## 本轮要读的小程序源码

- `uniapp/subPages/help/feedback.vue`、`uniapp/utils/feedback-ticket.js` + 测试、`uniapp/api/feedback.js`；契约 `CreateFeedbackTicketReq/Resp`（`POST /api/feedback/create` 访客可用、`POST /api/user/feedback/create` 登录用户——两者的选择规则看 `api/feedback.js` 与页面）
- `uniapp/subPages/agreement/userAgreement.vue`、`privacyPolicy.vue`（协议正文就在模板里）
- `uniapp/subPages/social/pkReport.vue` + 样式、`uniapp/utils/pk-report-view-model.js` + `pk-report-view-model.test.mjs`、相关接口（`uniapp/api/match.js` 里 PK 报表用到的接口）；`posterGenerator.js` 里 PK 报表海报相关的部分（了解海报内容）

## 任务

1. **帮助、投诉与举报**（`iOS/iOS/Features/Help/`）：`.feedback` → `FeedbackView`，标题「帮助、投诉与举报」。类型选择、内容与联系方式字段、校验规则（`feedback-ticket.js` 迁移并搬测试）、提交与成功 / 失败提示，以页面为准；访客可用。
2. **用户协议 / 隐私政策**（`iOS/iOS/Features/Agreement/`）：新增路由 `case userAgreement`（标题「用户协议」）、`case privacyPolicy`（标题「隐私政策」）。把两个小程序页面模板里的协议正文**逐字**搬成应用内页面（标题层级、段落、列表保持一致；正文作为 Swift 里的结构化常量，不走网络）。搬完后用脚本核对：从 `.vue` 模板里抽出全部可见文字，与 Swift 页面里的文字逐段比对，确保一字不差，并把核对方法和结果写进交付说明。然后把设置页和登录页里现有的两个协议外链（`https://www.zhuifen.cn/agreement`、`/privacy`）改为跳转这两个应用内页面——这是本轮唯一允许修改 `iOS/iOS/SettingsView.swift`（或 `Features/Settings/` 里对应的行）与 `iOS/iOS/LoginView.swift` 的地方，各只改链接那几行。
3. **PK 报表**（`iOS/iOS/Features/PKReport/`）：路由里已有好友模块留下的占位 `case friendPkReport(userId: Int64)`。小程序 `pkReport.vue` 的 `onLoad` 读 `opponent_id`、`opponent_name`、`opponent_avatar`，所以把这个 case 改成携带好友目标（复用已有的 `FriendHomepageTarget` 或 `OpponentRecordTarget`，不要新建同类类型），同步好友主页里的引用，目的地改为新页面 `PKReportView`，标题「PK报表」。`pk-report-view-model.js` 迁移并搬测试；页面内容以 `pkReport.vue` 为准。海报：用 SwiftUI 视图 + `ImageRenderer` 渲染成图片（内容与小程序海报一致），通过系统分享面板（`ShareLink` / `UIActivityViewController`）分享或保存到相册；依赖微信的分享入口不显示。保存相册的权限说明已在工程里配置。

## 测试

搬运 JS 用例；反馈表单模型（校验文案、访客与登录用户走的接口、提交中锁、失败保留表单）；协议正文完整性（段落数、首末段文字与 `.vue` 一致的断言）；PK 报表模型；海报渲染出非空图片且尺寸正确；快照：`feedback`、`feedback-errors`、`user-agreement`、`privacy-policy`、`pk-report`、`pk-report-poster`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Achievement/**`、`Features/Season/**`、`Features/Gear/**`、`Features/Member/**`、`Features/Stats/**`、`Features/Match/**`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件（任务 2 点名的链接行除外）。
