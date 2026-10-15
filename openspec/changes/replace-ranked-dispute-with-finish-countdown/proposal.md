## Why

[PRD V1.4](../../../docs/prd/familiar-player-quick-start.md) 让约球排位一方滑动即完成、积分延后10分钟生效并可被事后异议作废，已于 2026-10-14 随 `add-quick-match-ranked-dispute-window` 上线。产品改为 PRD V1.5：结束前给另一方30秒确认或异议的机会，发起方也能撤销误操作；30秒无人处理按冻结比分结束，此后不再申诉。这样既不会被对方拖住，也不再有“比赛已结束、积分待定、事后作废”的中间状态。

## What Changes

- 无裁判排位对局（约球与扫码开局一致）的结束，包括参赛方滑动结束和赛制达标自动触发，统一进入30秒结束确认：比分冻结、不能记分；发起方可撤回；另一方可确认（立即结束）或提出异议（恢复进行中、比分不变）；30秒无人处理由服务端按冻结比分完成结算，之后不再接受撤回、异议或确认。
- 复用现有结束请求流程（申请/确认/异议/撤回与 `finish_state=pending_confirmation`）：约球排位开局时也启用确认；请求有效期由24小时改为30秒；到期由“撤销请求、恢复进行中”改为“完成结算”，新增完成来源 `finish_timeout` 与系统动作 `finish_timeout`；过期 worker 改为秒级扫描。
- 当前对局、详情、同步快照与 WebSocket 消息返回服务端计算的确认截止时刻与剩余秒数；到期未处理的请求如实显示为“正在结束”，不再被读取当作已取消。
- **BREAKING**：回退 `add-quick-match-ranked-dispute-window` 的实现：删除 `POST /api/match/result-dispute`、`RankSettlementWorker`、待生效/作废结算与读模型的延后应用，以及 `rank_settlement_status`、`rank_settlement_due_at`、`can_dispute_result` 字段；新后端上线后另行发布迁移删除对应三列与索引。项目尚未正式上线，不保留兼容入口。
- UniApp（App 与微信小程序共用一套代码）：进行中页为发起方显示倒计时与“撤销结束”，为对方弹出确认框（滑动确认、有异议继续比赛）；滑块说明按 V1.5 改写；赛果页写明结束方式；移除结算状态与异议框。
- 原生 iOS 不在本变更范围：原生开发尚未开始，相关调整归入 iOS 迁移工作。后端删除字段后，`451a324` 中的 iOS 解码在字段缺省时按普通排位展示，不影响使用。
- 练习对局滑动即结束、裁判结束路径不变。

## Capabilities

### New Capabilities
- `ranked-finish-countdown`: 无裁判排位对局的30秒结束确认——发起、撤回、确认、异议、超时完成、并发与幂等、读取字段与客户端倒计时交互。

### Modified Capabilities
- `pool-match-formats`: “练习赛与排位赛使用各自既有结束流程”中，排位赛（约球与非约球）达到结束条件后进入30秒结束确认，期满按冻结比分完成，不再无限等待双方确认。
- `quick-match-invites`（由 `implement-quick-match-invites` 引入，须其归档后本变更才能归档）：约球排位不再由一方直接完成结算，改走30秒结束确认；约球比赛的结束滑块对排位发起确认、对练习直接结束。

## Impact

- **后端**：`internal/logic/match/` 的 `finish_action_logic.go`、`finish_request_expiry_worker.go`、`finish_match_logic.go`、`end_round_logic.go`、`snooker_action_logic.go`、`match_viewer_context.go` 与读取快照；`internal/logic/challenge/enter_challenge_logic.go`；`internal/model/match.go`（TTL、到期完成）；`internal/pkg/ws/hub.go`；`chasing_points.api` 与 goctl 生成物；回退 `99fc878` 引入的结算延后、异议接口、worker 与读模型改动。
- **UniApp（App 与微信小程序）**：`subPages/match/playing.vue`/`.scss`、`subPages/match/matchResult.vue`/`.scss`、`api/match.js`、`utils/rank-settlement.js`（删除）及对应测试。
- **原生 iOS**：本变更不修改。`451a324` 留下的待生效展示代码，由 iOS 迁移工作（如 `migrate-native-ios-match-history`）清理。
- **数据与发布**：先发布不再读写结算三列的后端，再发布删除三列与索引的迁移（`deploy-prod.sh` 会在切换二进制前执行迁移，同一版本删列会让旧进程报错）。
- **文档**：PRD V1.5、低保真 W1.5 与 HTML 预览已同步；删除被取代且未归档的 `add-quick-match-ranked-dispute-window` 变更目录。
