# RC6：实时计分页（二）——九球追分与斯诺克计分面板

上一轮已经迁好实时计分页的骨架（`iOS/iOS/Features/Match/Live/`：`LiveMatchModel`、同步与修订号、结束流程、按局计胜面板）。本轮把九球追分（`gameType === 2`）和斯诺克（`gameType === 1`，含旧版 v1 与 V2 两套面板）的占位区块换成真正的计分面板。

## 本轮要读的小程序源码

- `uniapp/subPages/match/playing.vue`：模板里 `snooker-frame-summary`、`snooker-v2-status snooker-format-overview`、`snooker-v2-layout`（`data-snooker-v2-panel`）、`snooker-layout`（`data-snooker-v1-panel`）、`jiuqiu-layout referee-jiuqiu-layout`、`jiuqiu-layout` 各区块；脚本里 `applySnookerRoundState`、`applyServerCurrentFrameScores`、斯诺克赛制编辑（`toggleSnookerFormatEditor` / `saveSnookerFormat`）、`selectSnookerCount`、`isSnookerScoreDisabled`、`getSnookerColorName`、`adjustSnookerFoulReds`、`handleStartRespottedBlack`、`handleOfferSnookerConcession`、`handleSnookerConcessionDecision`、`handleRefereeAwardFrame`、`handleRefereeAcceptedConcession`、`handleAddScore`、`handleFoul`、`handleFoulByScore`、`handleOpponentWin`、`handleNextRound`，以及它们用到的全部 computed。
- 已迁移规则：`SnookerFrame`、`SnookerMatchFormat`、`SnookerV2`、`ChasingScore`、`MatchScoringTarget`、`MatchRoleView`。
- 接口：`POST /api/match/snooker/stroke`、`/snooker/frame-action`、`/snooker/format`、`/score`、`/foul`、`/round/end`、`/round/start`（请求 / 响应类型与修订号字段以 `.api` 和 `buildActionRequest` 为准）。
- 规格：`openspec/specs/snooker-rules-compliance/spec.md`、`referee-bidirectional-scoring/spec.md`，以及 `openspec/changes/` 里与九球追分计分明细、斯诺克自由局数相关的变更说明（按名字找）。
- `uniapp/tests/` 里涉及斯诺克 / 九球计分页行为的测试（`grep -l -i "snooker\|jiuqiu\|nine" uniapp/tests/*.mjs`）。

## 任务

1. `LiveMatchModel` 增补（只增不改已有行为）：九球追分与斯诺克的写操作与状态应用，逐函数对应上面列出的脚本函数；所有可用性判断（哪个球现在能打、哪个按钮禁用）以服务端状态 + 已迁移规则为准，客户端不自行推演比分。
2. **九球追分面板**：选手视角与裁判视角（裁判可为任一方记分 / 记犯规，对应 `participant-switch`）；加分、犯规、对手胜、撤销、下一局；追分明细（`ChasingScore`）如模板里在本页展示则一并迁移。
3. **斯诺克 v1 面板**：记分球选择、连续进球数、犯规罚分与红球调整、裁判切换记分对象。
4. **斯诺克 V2 面板**：当前局状态与赛制概览、击球录入、自由球 / 重置黑球、局内认输与整场认输（提出 / 对方决定）、裁判判局与记录双方已接受的认输、开始下一局的条件（`canStartNextSnookerFrame...`）、赛制编辑。
5. 斯诺克局分摘要（`snooker-frame-summary`）。
6. 文案、按钮禁用条件、二次确认弹窗（认输、判局等破坏性操作）逐条照搬。

## 顺带核对一处文案

裁判视角的快照里，「最近操作」显示的是「我方赢下本局」——对裁判来说「我方」不成立。对照 `playing.vue` 里最近操作文案的来源（服务端字段还是前端按视角拼接），确认选手 / 裁判 / 观赛三种视角下的文案与小程序一致；如果是快照夹具写错了就改夹具，如果是视角处理漏了就修代码并加测试。

## 测试

行为用例搬运；`LiveMatchModel` 新增操作的请求体与状态应用；按钮可用性在典型局面下的断言（红球阶段、彩球阶段、清彩、自由球、重置黑球、局末）；认输 / 判局流程；裁判为双方记分；写操作进行中锁；推送与响应顺序收敛。快照：`live-nine-ball-player`、`live-nine-ball-referee`、`live-snooker-v1`、`live-snooker-v2-reds`、`live-snooker-v2-colors`、`live-snooker-v2-concession`、`live-snooker-v2-referee`、`live-snooker-format-editor`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/`）及其测试，本轮不要碰这些文件。不要重写上一轮的同步、修订号与结束流程；发现上一轮有错，在交付说明里指出并给出最小修正。
