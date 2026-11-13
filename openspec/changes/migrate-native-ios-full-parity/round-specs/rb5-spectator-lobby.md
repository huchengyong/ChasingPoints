# RB5：观赛大厅

本轮迁移观赛大厅（`uniapp/pages/match/index.vue`）。

## 本轮要读的小程序源码

- `uniapp/pages/match/index.vue`（+ 同目录 scss）、`uniapp/utils/spectator-lobby.js` + `uniapp/tests/spectator-lobby.test.mjs`
- `uniapp/utils/match-core-flow.js` 的 `buildFinishedMatchDetailRoute`、`uniapp/utils/ongoing-match-guard.js` 的 `buildPlayingRoute`（只为了解点击后去哪）
- `uniapp/api/match.js` 的 `getPublicMatches`；契约：`GET /api/public/matches`（`PublicMatchListReq/Resp` 及列表项类型），访客可读。服务端对隐藏战绩用户、本人对局的处理读 `backend/internal/logic/public/` 确认。

## 任务

1. `spectator-lobby.js` 的规则逐条迁移（`iOS/iOS/Features/Spectate/SpectatorLobbyRules.swift`）并搬测试。
2. `.spectatorLobby` → `SpectatorLobbyView`，标题「观赛大厅」。状态筛选、球种筛选（含「全部球种」）、列表卡片内容、「我」的标记、下拉刷新、分页、空态、失败态以 `index.vue` 为准。访客可看。
3. 点击对局：
   - 进行中的对局 → 新增路由 `case liveMatch(matchId: Int64)`（标题「对局」，目的地先用占位页；实时计分 / 观赛页后面迁移）。
   - 已结束的对局 → 小程序走公开对局详情。新增路由 `case publicMatchDetail(matchId: Int64)`（标题「对局详情」，目的地先用占位页）。本人参与的已结束对局若小程序走的是本人战报，则 → `.matchReport(matchId:)`。具体分支以 `index.vue` 的点击处理为准。

## 测试

搬运 JS 用例；接口门面的路径 / 参数、访客不带 Authorization；页面模型：筛选切换重读且丢弃旧结果、分页、刷新失败保留数据、空、失败重试、登录 / 退出后重读（「我」的标记变化）。快照：`spectator-lobby`（进行中 3 场 + 已结束 2 场）、`spectator-lobby-empty`、`spectator-lobby-error`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Settings/**`、`Features/Reputation/**`、`Features/Opponents/**`、`Features/Stats/**`、`Features/News/**`、`iOS/iOS/SettingsView.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。
