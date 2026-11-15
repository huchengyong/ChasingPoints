# R2.7：赛季、赛季报告、历届赛季

## 先修正上一轮的一个行为问题（改完再做本轮任务）

荣誉墙现在加载后直接把最新一条未读的「换季结果」通知（`season_rollover`）标为已读，但没有展示任何提示——用户会永远看不到这次换季结果。小程序的做法（`uniapp/utils/honor-wall.js` 的 `presentLatestSeasonRollover` 与 `buildSeasonRolloverModal`）是：**先弹窗展示，用户关闭弹窗后才标已读**，点弹窗的确认按钮跳到赛季报告。小程序在两处调用它：`pages/user/index.vue`（「我的」页加载完成后）和 `subPages/achievement/index.vue`（荣誉墙）。

要求：

1. 新建 `iOS/iOS/Features/Season/SeasonRolloverPresenter.swift`：`@MainActor final class SeasonRolloverPresenter: ObservableObject`，按 `presentLatestSeasonRollover` 的流程实现——同一时刻只展示一次（对应 `rolloverPresentationPromise`）；没有传入通知时自己读 `GET /api/notification/list?type=season_rollover&page=1&page_size=10` 找最新未读；用 `HonorWallRules.seasonRollover(...)`（已迁的 `buildSeasonRolloverModal`）得到标题、正文、按钮文案；**弹窗关闭后**才调 `POST /api/notification/read`；确认且有报告时导航到本轮新增的赛季报告路由。发布一个 `@Published var alert: SeasonRolloverAlert?` 供视图用 `.alert` 展示。
2. 荣誉墙：删掉现在「加载后直接标已读」的代码，改用这个 presenter。
3. 「我的」页（`iOS/iOS/Features/Profile/MyPage.swift` / `MyPageModel.swift`）：概览读取完成后调用 presenter（对应小程序 `loadHomepageData().finally(showLatestSeasonRollover)`）；通知来源优先用 `ActivityStore.latestSeasonRollover`（如果当前代码里已有 `ActivityStore`），没有则让 presenter 自己读。标已读后调用 `ActivityStore.clearLatestSeasonRollover()` 与 `decrementUnread()`（如存在）。这是本轮唯一允许修改 `Features/Profile/` 的地方。
4. 测试：没有未读换季通知时不弹；有时弹出且此时**尚未**调用已读接口；关闭后调用一次已读接口；确认跳转赛季报告；并发触发只弹一次；身份变化后迟到的结果不弹。

## 本轮要读的小程序源码

- `uniapp/subPages/season/index.vue`（761 行）、`report.vue`（539 行，自定义导航栏）、`history.vue`（350 行）及各自 scss
- `uniapp/utils/season-lifecycle.js`、`season-qualification.js`、`season-history.js` 及测试 `season-lifecycle.test.mjs`、`season-qualification.test.mjs`、`season-history.test.mjs`、`season-page.test.mjs`（能对应到规则 / 页面模型的用例都搬）
- `uniapp/api/season.js`；`index.vue` 里对 `honor-wall.js` 的用法（已在 `iOS/iOS/Features/Achievement/` 迁移过的直接复用）
- 契约：`backend/chasing_points.api` 的 `/api/season/*`（`GetCurrentSeasonResp`、`GetSeasonLeaderboardReq/Resp`、`GetMySeasonRecordReq/Resp`、`GetSeasonOverviewReq/Resp`、`GetSeasonReportReq/Resp`、`GetSeasonHistoryReq/Resp`）及嵌套类型；赛季未开始 / 休赛期 / 无记录时的实际返回读 `backend/internal/logic/season/` 确认。

## 任务

1. **规则迁移**（`iOS/iOS/Features/Season/`）：三个 utils 的全部导出逐条迁移并搬运测试。
2. **赛季**：`.seasonIndex` → `SeasonView`，标题「赛季」。当前赛季卡（阶段、倒计时 / 起止时间）、我的赛季战绩与资格、赛季榜、球种切换、空态 / 休赛期 / 失败态、跳转（历届赛季带当前球种、赛季报告）以 `index.vue` 为准。需要的路由参数：`.seasonHistory` 改为带 `gameType`；新增 `case seasonReport(...)`（参数以小程序跳转时带的查询参数为准），标题「赛季报告」。赛季榜头像用 `MemberAvatarView`。
3. **赛季报告**：→ `SeasonReportView`。小程序是自定义导航栏的沉浸式页面；原生端用系统导航栏（透明背景或 inline 标题），内容区块、数据、分享入口以 `report.vue` 为准。依赖微信分享的入口不显示；如有「保存 / 分享海报」，本轮先不做并在交付说明里列出。
4. **历届赛季**：→ `SeasonHistoryView`，标题「历届赛季」，内容与分页以 `history.vue` 为准。

## 测试

搬运 JS 用例；接口门面（访客可读的接口不带 Authorization）；页面模型（首次加载、切换球种、休赛期 / 未开赛、空、失败重试、未登录时只显示公开部分、身份变化）；快照：`season-active`、`season-offseason`、`season-error`、`season-report`、`season-history`、`season-history-empty`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/News/**`、`Features/Match/**`、`Features/Home/**`、`Features/Messages/**`、`Features/Friends/**`、`iOS/iOS/App/MainTabView.swift` 及其测试，本轮不要碰这些文件。
