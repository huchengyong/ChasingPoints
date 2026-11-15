# RC7：赛果、对局详情、单场复盘、本场装备

实时计分页已迁完。本轮迁移对局结束后的页面。

## 本轮要读的小程序源码

- `uniapp/subPages/match/matchResult.vue`（751 行，自定义导航栏）、`matchDetail.vue`（661 行）、`matchReplay.vue`（248 行）及各自样式
- 组件：`uniapp/components/matchChasingScoreCard.vue`、`matchGearCard.vue`（574 行）
- 规则：`match-result-page.js`、`match-detail.js`、`match-referee-view.js`、`match-core-flow.js`、`gear-match.js`、`member-ranking-rights.js`、`share-card.js` 及各自测试；已迁的 `ChasingScore`、`FinishCountdown`、`MatchReplayEntry`、`MemberReplayView`、`AppFormat`、`Features/Achievement/` 里的 `honor-wall` 规则（如已存在则复用）
- 原生现有的只读战报：`iOS/iOS/MatchReportView.swift`、`MatchReportModel.swift`、`MatchHistoryModels.swift`（`.matchReport(matchId:)` 目前指向它）
- 接口与契约：`GET /api/match/detail`、`GET /api/public/match/detail`、`GET /api/match/reward-summary`、`GET /api/member/match-replay`（`GetMemberMatchReplayReq/Resp`）、`GET /api/gear/match`、`POST /api/gear/match/update`、`GET /api/share/match`；规格 `openspec/specs/match-gear-record/spec.md`、`ranked-finish-countdown/spec.md`，以及 `openspec/changes/nine-ball-score-detail-view/`。

## 任务

`iOS/iOS/Features/Match/Result/`：

1. 规则迁移：上面列出的未迁规则逐条迁移并搬运测试（放 `Features/Match/Rules/`）。
2. **赛果页**：新增路由 `case matchResult(matchId: Int64, ...)`（参数以 `matchResult.vue` 的 onLoad 为准：来源 `from`、角色 `role` 等），标题按小程序。内容以 `matchResult.vue` 为准：胜负头图、比分、排位分变化与待生效 / 作废状态（排位结束倒计时）、奖励与成就、追分明细卡（`MatchChasingScoreCard`）、本场装备卡（`MatchGearCard`：查看与编辑本场用的装备，`gear-match.js` 规则）、裁判视角差异、操作（再来一局 → `QuickScanFlow.start(rematch:)`、查看复盘、生成海报 → 占位路由 `.shareResult(matchId:)`、交锋记录、荣誉墙、返回首页）。实时计分页结束后的跳转、对局记录里点对局的跳转改为进这个页面（`LiveMatchModel` 的结束去向、`MatchHistoryView` 的行点击——只改路由值）。
3. **对局详情（公开 / 观赛）**：`.publicMatchDetail(matchId:)` → `MatchDetailView`，以 `matchDetail.vue` 为准（它同时承担「已结束对局的公开详情」与观赛入口：进行中对局的去向按页面逻辑转到实时计分页）。
4. 现有的 `MatchReportView`：赛果页覆盖了它的全部内容之后，把 `.matchReport(matchId:)` 也指向新赛果页（以「来自对局记录」的参数进入），旧的 `MatchReportView` / `MatchReportModel` 及其测试保留不删（避免本轮改动过大），在交付说明里注明它已不再被路由引用。
5. **单场复盘**：`.matchReplay(matchId:)` → `MatchReplayView`，标题「单场复盘」，以 `matchReplay.vue` 和 `MemberReplayView` 规则为准；非会员锁定态的开通入口只由 `AppFlags.canShowPaidMembershipEntry` 控制。
6. 新增占位路由 `case shareResult(matchId: Int64)`（标题「战绩海报」），下一轮实现。

## 测试

搬运 JS 用例；赛果模型（选手 / 裁判 / 来自记录三种进入方式、排位待生效与作废、奖励 pending / ready / 失败、本场装备读取与保存、失败重试、身份变化）；对局详情模型；复盘模型（会员 / 锁定）；快照：`match-result-win`、`match-result-lose-ranked-pending`、`match-result-referee`、`match-result-nine-ball-detail`、`match-result-gear-edit`、`match-detail-public`、`match-replay-member`、`match-replay-locked`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/`）、`iOS/iOS/App/ActivityStore.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`MatchHistoryViews.swift` 只改行点击的路由值。
