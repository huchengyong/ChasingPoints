# RC8：战绩海报与邀请链接

## 本轮要读的小程序源码

- `uniapp/subPages/match/shareResult.vue`、`uniapp/utils/posterGenerator.js`（672 行，海报的全部版式与文案）、`uniapp/api/share.js`（`GET /api/share/match`、`GetMatchShareDataReq/Resp`）
- `uniapp/subPages/match/linkEntry.vue`、`uniapp/utils/link-entry-flow.js`、`entry-funnel.js` 及测试 `link-entry-flow.test.mjs`、`entry-funnel.test.mjs`；已迁的 `EntryLink`、`QuickScanFlow.handoffResolved`
- 规格：`openspec/changes/add-wechat-scan-to-join-and-share/`（了解入口链接的格式与域名）、`openspec/specs/quick-match-invites/spec.md`

## 已有代码（先读，直接复用，不得另建同名或同职责的类型）

- `iOS/iOS/Features/Entry/`：欢迎页 / 登录对齐那一轮已把 `entry-funnel.js` 里与手机号登录、欢迎页、协议相关的导出迁成 `EntryFunnelRules`。本轮只补与链接入口相关、尚未迁移的导出，**加到同一个类型 / 文件里**，并把对应 JS 用例补进已有的 `EntryFunnelRulesTests.swift`。
- 本分支还没有 `.shareResult` 路由：新增 `case shareResult(matchId: Int64)`（标题「战绩海报」）。另一条线的赛果页也会加同名 case（指向占位页），合并由监工处理，你不用管。
- `iOS/iOS/Features/PKReport/PKReportPoster.swift` + `PKReportView.swift`：海报用 SwiftUI 视图 + `ImageRenderer` + `ShareLink` 的现成写法，照它的结构做战绩海报；二维码用 `iOS/iOS/Shared/QRCodeImage.swift`。

## 任务

1. **战绩海报**（`iOS/iOS/Features/Match/Share/`）：`.shareResult(matchId:)` → `ShareResultView`，标题「战绩海报」。读取分享数据；海报用 SwiftUI 视图实现（版式、文案、配色与 `posterGenerator.js` 画出来的一致；小程序码 / 二维码区域：如果分享数据里有可用的入口链接，就用 `QRCodeImage` 生成该链接的二维码，否则不显示二维码区）；用 `ImageRenderer` 渲染成高清图片；提供「保存到相册」（`PHPhotoLibrary` 仅添加权限）与「分享」（系统分享面板）。依赖微信的分享入口不显示。失败提示文案照搬。
2. **邀请链接承接**：把 `link-entry-flow.js`、`entry-funnel.js` 的规则迁移并搬运测试；新增路由 `case linkEntry(url: String)`（标题「追分邀请」）→ `LinkEntryView`，以 `linkEntry.vue` 为准：解析入口链接 → 未登录先登录再继续 → 兑换邀请码 → 交接给确认页（`QuickScanFlow.handoffResolved`）或进入观赛。
3. **应用入口**：在 `iOSApp.swift` 的根视图上处理 `.onOpenURL` 与 `.onContinueUserActivity(NSUserActivityTypeBrowsingWeb)`：把收到的链接交给 `EntryLink.parse`，认识的链接打开 `.linkEntry(url:)`，不认识的忽略。只加这两个处理，不改其他启动逻辑；Universal Links 所需的关联域名（entitlements、服务器上的 apple-app-site-association）不在本轮范围，在交付说明里列出需要工程侧配置的清单。

## 测试

搬运 JS 用例；分享数据读取；海报视图渲染出非空图片、尺寸正确、关键文案出现在视图模型里；邀请链接流程（未登录 → 登录后继续、兑换失败的各提示、身份变化作废、交接成功）；URL 入口只接受认识的链接；快照：`share-result-poster-win`、`share-result-poster-lose`、`share-result-page`、`link-entry-loading`、`link-entry-error`。

## 并行开发提醒

另一条开发线正在修改：`iOS/iOS/Features/Match/`（`Detail/`、`Gear/`、`Replay/`、`Rules/` 下的文件）、`MatchReportView.swift`、`MatchReportModel.swift`、`Features/Gear/`、`Shared/GearModels.swift`、`Features/H2H/`、`Features/Performance/`、`Features/Home/`、`Features/Spectate/` 及其测试。本轮不要碰这些文件；新代码放在 `Features/Match/Share/` 与 `Features/Entry/`（或新建 `Features/LinkEntry/`）。`iOSApp.swift` 只加 URL 入口处理，`AppRoute.swift` 只加 `shareResult`、`linkEntry` 两个 case 及其标题、页面分支。
