# RC7b：赛果 / 对局详情修正（合并后去重 + 与小程序对齐）

上一轮（赛果、对局详情、单场复盘、本场装备）已提交。之后主分支合并进了本分支，带来了「我的装备」模块和排位权益规则，**当前构建失败**。本轮先修好构建，再补齐与小程序不一致的地方。先 `git log --oneline -3`、`git show --stat HEAD~1` 了解上一轮改了什么。

## 1. 装备类型去重（先做，做完构建必须通过）

主分支已有完整的装备模块，以它为准，删掉上一轮在 `Features/Match/` 下重复建的类型：

- 已有（保留、复用）：`iOS/iOS/Shared/GearModels.swift`（`GearProductInfo`、`GearItemInfo`、`GearDefaultInfo`、`GearMineResponse` 等）、`iOS/iOS/Features/Gear/GearAPI.swift`（`GearAPI`）、`iOS/iOS/Features/Gear/GearDisplayRules.swift`（`GearCategory`、`GearItemStatus`、`GearImageSource` / `GearImageURLs`、`GearDisplayRules`）、`GearSharedViews.swift`（`GearImageView` 等）、`GearStore.swift`。
- 重复（删除）：`iOS/iOS/Features/Match/Gear/GearAPI.swift`、`iOS/iOS/Features/Match/Gear/GearModels.swift`、`iOS/iOS/Features/Match/Rules/GearDisplay.swift`。
- 迁移：`GET /api/gear/match`、`POST /api/gear/match/update` 两个方法加到已有的 `GearAPI` 里；`GearMatchSide`、`GearMatchResponse` 放进 `Shared/GearModels.swift`（字段以 `backend/chasing_points.api` 为准）。已有模型缺本场装备需要的字段时，按 `.api` 补成可选字段，不要另建平行类型。图片、状态、分类文案一律用 `GearDisplayRules` 已有的函数；确实没有的再加到 `GearDisplayRules` 里。
- `GearMatchRules.swift`、`MatchGearCardModel.swift`、`MatchGearCardView.swift` 及其测试改为使用上述已有类型。`GearMatchLoader` 若只是包了一层请求，可以并进 `MatchGearCardModel`。
- 改完 `grep -rn "^struct Gear\|^enum Gear\|^final class Gear" iOS/iOS` 确认每个名字只出现一次。

## 2. 本场装备：皮头 / 巧克选择

`uniapp/components/matchGearCard.vue` 的编辑器里，皮头 / 巧克的「选择 / 更换」会以 `mode=pick` 打开 `subPages/gear/add`，选中后通过事件回传（约 240–300 行）。原生端已有同样机制：`AppRoute.gearAdd(category:pickToken:)` + `GearStore.beginPick(token:handler:)` / `finishPick` / `cancelPick`（看 `GearAddView.swift` 里的用法）。

- 编辑器里补上「选择」/「更换」按钮（文案照搬），点了以后生成一个 token、调 `beginPick`，再打开 `.gearAdd(category: "tip" 或 "chalk", pickToken: token)`，回传的产品写进草稿。
- 编辑器如果是 sheet：在 sheet 里放自己的 `NavigationStack`，用 `navigationDestination(for: AppRoute.self)` 推入 `GearAddView`（同一个 `GearStore` 环境对象），不要关闭 sheet 再推页面。编辑器关闭时调用 `cancelPick`。
- 测试：选择回传后草稿更新、保存请求体里带上新的皮头 / 巧克；取消选择不改草稿。

## 3. 对局详情的三种进入方式

小程序 `subPages/match/matchDetail.vue` 的 `onLoad` 读取三个参数：`mode`（`spectate`）、`source`（`history`）、`perspective_user_id`。`match-detail.js` 的 `shouldUsePublicMatchDetail({ mode, source })` 决定走公开接口还是参与者接口，`normalizeMatchDetailPayload(match, { perspectiveUserId })` 决定是否左右互换。标题：观战模式「观战」，否则「对局详情」，裁判视角「执裁详情」。

- 新增路由 `case matchDetail(matchId: Int64, source: String?, perspectiveUserId: Int64?)`（标题「对局详情」）→ `MatchDetailView`，非观战模式。`.publicMatchDetail(matchId:)` 保持为观战模式。
- `MatchDetailModel` 接收 `mode` / `source` / `perspectiveUserId` 三个参数，**视角用路由带进来的 `perspectiveUserId`（没有就是 0），不要用当前登录用户的 id**。
- 按小程序的跳转改原生调用处（只改路由值）：
  - `uniapp/subPages/user/h2hRecord.vue` 约 612 行：`matchDetail?match_id=…&source=history&perspective_user_id=<主体用户>` → 原生 `H2HRecordView` 里对局行的跳转改成 `.matchDetail(matchId:, source: "history", perspectiveUserId: <主体用户 id>)`。
  - `uniapp/subPages/user/performance.vue` 约 267 行：`matchDetail?match_id=…` → 原生 `PerformanceView` 的跳转改成 `.matchDetail(matchId:, source: nil, perspectiveUserId: nil)`。
  - `uniapp/utils/match-core-flow.js` 的 `buildFinishedMatchDetailRoute`：本人参与的已结束对局进参与者详情 → 原生 `HomeView` 与 `SpectatorLobbyView` 里 `case .report:` 返回的路由改成 `.matchDetail(matchId:, source: nil, perspectiveUserId: nil)`（先核对 `SpectatorLobbyRules.destination` 的 `.report` 确实对应这个函数的参与者分支；不一致就以 JS 为准并在交付说明里写明）。
  - 对局记录、消息中心、实时计分结束后的去向保持进赛果页（小程序就是 `matchResult`），不要改。
- 测试：三种进入方式各自调用哪个接口；`source=history` 时按 `perspectiveUserId` 互换左右；登录用户是选手 2 但没有传 `perspectiveUserId` 时不互换。

## 4. 对局详情的实时更新

`matchDetail.vue` 在对局进行中（`status === 1`）会连接对局 WebSocket（观战模式 `allowAnonymous`），处理比分推送与对局结束，并在收到推送后 400–1000 毫秒内静默刷新得分构成（`refreshScoreBreakdown`：不切整页加载态，失败保留旧数据，过期响应丢弃）。约 460–600 行。

- 用已有的 `iOS/iOS/Shared/Realtime/MatchRealtime.swift`（及 `RealtimeConnection` 的 `allowAnonymous`）实现同样的行为：页面出现且对局进行中时连接、离开页面时断开；收到比分推送更新比分；收到结束消息后重新读取详情。把「推送后延迟刷新」的随机延迟做成可注入的参数，测试里用固定值。
- 去掉「进行中只给『进入实时对局』入口」这一原生自创的替代做法，除非小程序也有同样的入口（以 `matchDetail.vue` 模板为准）。
- 测试：进行中连接、已结束不连接；推送更新比分；静默刷新失败保留旧数据；身份变化或页面离开后的推送不再改状态。

## 5. 失败与空状态

- `MatchDetailModel` 的失败不要直接显示 `error.localizedDescription`：网络 / 服务错误走 `LoadFailure.feedback(resource: "对局")`，用 `LoadableContent` / `StateMessageView` 显示并支持重试；接口返回 `success=false` 的提示保持现有文案。赛果页、复盘页同样检查一遍。

## 6. 补齐快照

上一轮规格要求而没有做的快照全部补上（亮暗各一张，按 `_common.md` 的「先让模型读完数据再传给页面」做法）：
`match-result-win`、`match-result-lose-ranked-pending`、`match-result-nine-ball-detail`、`match-result-gear-edit`（编辑器展开、皮头已选）、`match-detail-public`（进行中）、`match-detail-public-finished`、`match-detail-history-perspective`（`source=history` 且主体是选手 2）、`match-result-error`、`match-detail-error`、`match-replay-error`。已有的 `match-result-player` 若与 `match-result-win` 重复，可以改名为后者。

## 并行开发提醒

另一条开发线正在改 `iOS/iOS/Features/Entry/`、`LoginView.swift`、`ContentView.swift` 及其测试，本轮不要碰这些文件。`AppRoute.swift` 只加 `matchDetail` 一个 case 和它的标题、页面分支。
