# RC5：实时计分页（一）——页面骨架、同步、结束流程、中式八球与美式九球

实时计分页 `uniapp/subPages/match/playing.vue` 有 2800 行，分两轮迁移。本轮做与球种无关的骨架和「按局计胜」的球种（中式八球、美式九球）；下一轮做九球追分和斯诺克的计分面板。

## 先修正上一轮的一个缺陷（改完再做本轮任务）

`iOS/iOS/Features/Match/ScanConfirmView.swift` 在 `.task` 里执行 `model.apply(handoff: quickScan.takeHandoff(...))`。`.task` 会在页面每次重新出现时重跑——切到别的 Tab 再切回来、重新扫码的 `fullScreenCover` 关闭时都会——而交接只能取一次，第二次取到 `nil`，`apply(handoff: nil)` 把已经填好的确认页清成「扫码信息已失效」。

改法：交接只在页面第一次出现时取一次。在 `ScanConfirmModel` 里加一个「是否已经尝试过取交接」的标记（例如 `func takeInitialHandoff(from flow: QuickScanFlow, identity:)`，第二次调用直接返回），`.task` 改为调用它；重新扫码成功后的 `apply(handoff:)` 路径不受影响。补回归测试：模型取到交接后，再次触发「页面出现」的入口，状态保持不变、`QuickScanFlow` 不被再次消费；重新扫码得到新交接后状态更新为新对手。

## 本轮要读的小程序源码

- `uniapp/subPages/match/playing.vue` **全文**（模板 1–599 行，脚本 601–2816 行，样式在其后）。先通读脚本，弄清：页面状态、`loadMatchInfo`、`applyMatchSnapshot` / `applyWriteResponse` / 修订号（`updateServerRevision`、`shouldApplyIncomingRevision`）、各个 WebSocket 消息处理函数、`applyViewerCapabilities`（选手 / 裁判 / 观赛者各自能做什么）、结束流程（`handleRequestFinish`、`handleConfirmFinish`、`handleDisputeFinish`、`handleWithdrawFinish`、`handleFinishMatch`、倒计时 watch、滑动确认）、`handleBack` / `showMenu`、裁判二维码弹层。
- 已迁移的规则（`iOS/iOS/Features/Match/Rules/`）：`MatchRoleView`、`MatchScoringTarget`、`PoolMatchFormat`、`FinishCountdown`、`MatchAction`、`MatchNavigation` 等——直接用，不要重写。
- `uniapp/utils/slide-to-confirm.js`（滑动确认的阈值规则）+ 对应测试；`uniapp/tests/` 里以 `playing`、`finish-confirmation`、`match-page` 开头或内容涉及 playing 页的测试（`grep -l playing uniapp/tests/*.mjs`）——能对应到页面模型的行为用例都搬。
- 接口（`uniapp/api/match.js` + `backend/chasing_points.api`）：`GET /api/match/current`、`GET /api/match/detail`、`GET /api/public/match/detail`（观赛）、`POST /api/match/score`、`/round/end`、`/round/start`、`/foul`、`/format`、`/undo`、`/finish`、`/finish/request|confirm|dispute|withdraw`、`/cancel`、`GET /api/match/referee/qrcode`。请求里的幂等 / 修订号字段（看 `buildActionRequest`）必须照搬。
- 实时通道：`iOS/iOS/Shared/Realtime/MatchRealtime.swift`（`connect(matchId:allowAnonymous:)`、`setTargetActive`、`requestSync`、`disconnect`、`on` / `off`）；消息负载的结构以后端推送实现为准（`backend/internal/` 里搜各消息类型）。
- 规格：`openspec/specs/ranked-finish-countdown/spec.md`、`pool-match-formats/spec.md`、`referee-bidirectional-scoring/spec.md`、`quick-match-invites/spec.md`。

## 任务

1. **模型与接口**（`iOS/iOS/Features/Match/Live/`）：`LiveMatchAPI`（上述接口）；对局快照模型（`GET /api/match/current` / `detail` 的对局对象，字段以 `.api` 为准，页面用到哪些就解码哪些，但九球追分和斯诺克下一轮要用的字段也一并解码好，避免下一轮改模型）。
2. **页面模型** `LiveMatchModel`（`@MainActor ObservableObject`）——这是本轮的核心，逐函数对应脚本：
   - 进入：按路由参数读取对局（本人对局 / 裁判 / 观赛三种入口的读取方式与失败去向照搬）；对局已结束或不存在时的跳转（去赛果 / 对局记录 / 返回）照搬。
   - 实时同步：连接对局通道、页面可见性（`setTargetActive`）、各消息处理；**修订号规则**（旧修订的推送和响应不得覆盖新状态）一条都不能错；写操作响应与推送谁先到都要收敛到同一状态。
   - 角色能力：`applyViewerCapabilities` → 哪些面板 / 按钮可见、可用。
   - 写操作：统一经一个带「进行中」锁的入口（对应 `buildActionRequest` + 各 handler 的防重复点击），失败提示文案照搬；撤销（`handleUndo`）。
   - 赛制编辑（`toggleMatchFormatEditor` / `saveMatchFormat`，中式八球与美式九球的抢几 / 局数设置，规则在 `PoolMatchFormat`）。
   - 结束流程：直接结束、申请结束 → 对方确认 / 异议 / 撤回、排位赛的结束倒计时（`FinishCountdown`）、约球对局的滑动确认；倒计时用可注入时钟。结束后跳转赛果只触发一次（`navigateToResultOnce`）。
   - 裁判二维码：读取、过期刷新、关闭后丢弃迟到结果（二维码正文是凭证，不进日志）。
   - 返回 / 菜单（`handleBack`、`showMenu`）：取消对局、认输等入口与确认文案照搬。
   - 结算后段位失效（`invalidateRankAfterSettlement`）：原生端调用 `ActivityStore.markDirty()` + `bumpUserDataRevision()`（如果 `ActivityStore` 在当前代码里已存在；不存在则留一个注入的回调并在交付说明里写明）。
3. **界面**：`LiveMatchView`，把占位路由 `.liveMatch(matchId:)` 指向它（如路由还带不了入口类型——本人 / 裁判 / 观赛——给它加关联值并同步引用处）。全屏沉浸式（隐藏系统导航栏，自绘顶部返回与菜单；计分时保持屏幕常亮 `isIdleTimerDisabled`，离开恢复）。
   - 记分板头部（双方头像 `MemberAvatarView`、昵称、比分、局数 / 赛制文案、对局类型、时长、连接状态）。
   - 结束确认面板与倒计时、结束确认弹层（含滑动确认控件）。
   - **按局计胜面板**（中式八球、美式九球；模板里 `isRoundWinMode` 相关的全部分支，选手视角与裁判视角）：记一局胜、对手胜、犯规 / 特殊局（美式九球的普胜 / 小金 / 大金等按模板）、撤销、下一局、赛制编辑。
   - 观赛者只读面板（`readonly-panel`）。
   - 底部操作区（结束 / 申请结束等）。
   - 九球追分（`gameType === 2`）和斯诺克（`gameType === 1`）的计分面板本轮用一个占位区块「该球种的计分面板正在迁移」，但记分板头部、同步、结束流程对这两个球种同样要工作。
4. 滑动确认控件 `SlideToConfirm`（规则迁自 `slide-to-confirm.js`，支持无障碍替代操作：VoiceOver 下可用按钮完成确认）。

## 测试

规则 / 行为用例搬运；`LiveMatchModel`（替身实时通道 + `MockHTTPServer`）：三种入口的读取；推送与响应的各种先后顺序下状态一致；旧修订被丢弃；写操作进行中重复触发被忽略；失败提示；撤销；赛制保存；结束流程全部分支（直接结束、申请 / 确认 / 异议 / 撤回、倒计时到期、对方超时）；`MATCH_END` 只跳转一次；角色变化（被加入裁判后选手面板变化）；页面隐藏 / 恢复时的通道行为；身份变化后停止并丢弃迟到结果；裁判二维码。快照：`live-eight-ball-player`、`live-eight-ball-referee`、`live-american-nine-player`、`live-spectator`、`live-finish-countdown`、`live-finish-confirm-dialog`、`live-referee-qrcode`、`live-load-error`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/`）、`iOS/iOS/App/MainTabView.swift`、`iOS/iOS/App/ActivityStore.swift`、`iOS/iOS/iOSApp.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`AppRoute.swift` 只改 `.liveMatch` 相关的行。这一轮工作量大：先把 `LiveMatchModel` 和它的测试做扎实，再做界面；如果界面细节做不完，宁可在交付说明里列出没做的区块，也不要牺牲同步与修订号规则的正确性。
