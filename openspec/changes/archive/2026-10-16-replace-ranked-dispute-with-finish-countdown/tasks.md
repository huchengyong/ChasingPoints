## 1. 回退10分钟异议期

- [x] 1.1 逐文件核对 `99fc878`，删除以下内容：
  - 延后结算：`planRankSettlement` 的 pending 分支、`applyDeferredRankStatsWithTx`、`ApplyDeferredRankStatsWithTx` 与按生效时刻的赛季归属；
  - `RankSettlementWorker` 及其在 `chasing_points.go` 的注册；
  - `DisputeRankResultLogic` 与对应 handler；
  - 模型中的 `rank_settlement_*` 字段与 `RankExcluded`，以及 `rank_excluded` 的写入和排位口径过滤；
  - 旧 `matches` 直读统计中对 pending/voided 的排除；
  - 相关通知文案与测试。

  与延后结算无关、且仍有调用方的重构可以保留，否则一并删除。
- [x] 1.2 补回归测试：排位比赛完成时，段位、排位日志、排位口径统计、H2H、趋势与赛季归属在完成流程中一次生效；旧统计查询与异议期之前一致。

## 2. 后端30秒结束确认

- [x] 2.1 `enter_challenge_logic.go` 让约球排位以 `FinishConfirmationRequired=true` 开局，并删除 `shouldRequestRankedFinish` 与 `match_viewer_context.go` 中对约球比赛的直接结束特判。测试：约球排位的手动结束与赛制达标自动结束都进入待确认；约球练习、扫码练习与裁判局行为不变。
- [x] 2.2 `model.FinishRequestTTL` 改为30秒，新增完成来源 `finish_timeout` 与系统动作 `finish_timeout`。`ExpireStaleFinishRequest` 改为：锁行复核待确认与截止时刻后，复用确认路径按冻结比分完成结算（结束方为发起方、actor 0），并在同一事务内同步完成约球；提交后的广播、会员成长、成就、`rank_info_updated` 与缓存失效与确认路径一致。测试：结算结果与确认路径一致、恰好一次、多实例与重试不重复、约球完成。
- [x] 2.3 确认、异议、撤回在锁内遇到已到期的请求时，先按超时完成，再拒绝原动作并返回权威状态。测试：29秒与30秒边界、确认/撤回/异议/超时两两竞争只产生一种结果、`client_action_id` 幂等重放、发起方不能确认或异议自己的请求、非参赛者被拒。
- [x] 2.4 `FinishRequestExpiryWorker` 间隔改为2秒（批量100，沿用 `idx_matches_finish_request_expiry`），记录积压数量与最老截止时刻。测试：单次运行完成到期请求，停摆后恢复能补处理。

## 3. 读取与契约

- [x] 3.1 在 `backend/chasing_points.api` 中：当前对局、`MatchDetailData` 与同步快照新增 `finish_request_expires_at` 和 `finish_request_remaining_seconds`；删除 `rank_settlement_status`、`rank_settlement_due_at`、`can_dispute_result` 与 `/api/match/result-dispute` 路由。**紧接着**在 `backend/` 运行 `goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default`，检查生成 diff，删除不再引用的 handler。
- [x] 3.2 删除 `ws/hub.go` 等处“已过期请求视为无请求”的读取改写：到期未处理的请求返回待确认、剩余0秒、各操作权限为 false。读取保持纯读，补到期边界与 query-count 测试。
- [x] 3.3 发起、撤回、异议、超时完成的 WebSocket 广播都携带上述字段；超时完成广播 `match_end` 与权威快照。
- [x] 3.4 发起时沿用现有通知通道告知另一方“X申请结束对局 · 比分，30秒内确认或提出异议”，超时完成后双方收到结束状态；删除异议期相关文案。

## 4. UniApp

- [x] 4.1 删除 `utils/rank-settlement.js`、`api/match.js` 的 `disputeRankResult`、`matchResult.vue` 中的结算状态与异议框，以及对应测试。新增 `utils/finish-countdown.js`：按服务端剩余秒数与本地经过时间计算倒计时，并给出角色、文案与滑块说明。
- [x] 4.2 `subPages/match/playing.vue` 实现两个视图：
  - 发起方面板：倒计时与“撤销结束”；
  - 对方确认弹窗：复用结束滑块的防误触实现做“向右滑动确认结束”，另有“有异议，继续比赛”。

  倒计时归零后显示“正在结束…”，3秒内没有收到结束消息就回源。操作失败保留入口并以权威状态刷新，异步结果按 `userId + authGeneration` 隔离。滑块说明按 PRD V1.5 改写。
- [x] 4.3 `subPages/match/matchResult.vue`：排位赛按 `completion_source`（`player_confirmed` / `finish_timeout`）与 `completed_by_user_id`（确认时为确认方，超时时为发起方）显示“X发起 · Y确认”或“X发起 · 30秒未处理，自动结束”，练习赛不显示。
- [x] 4.4 补 `node --test` 用例：倒计时以服务端剩余为准、不受手机时钟偏移影响；归零后回源；确认滑块误触不发送；发起方没有确认入口；切号后旧响应不回写；结束方式文案正确。运行 `node --test tests/*.test.mjs`。

## 5. 验证、发布与文档

原生 iOS 不在本变更范围，相关调整归入 iOS 迁移工作。

- [x] 5.1 后端运行 `go test ./...`，本变更运行 `openspec validate --strict`，并核对 goctl 生成物。
- [x] 5.2 在 HBuilderX 分别编译 App 与微信小程序，真机走查以下内容，并记录实际覆盖的平台（用户人工确认通过，平台明细见 `docs/testing/replace-ranked-finish-countdown-acceptance.md`）：
  - 发起方倒计时与撤销；
  - 对方确认弹窗与确认滑块的防误触；
  - 归零后显示“正在结束…”并回源；
  - 排位赛果页的结束方式。
- [x] 5.3 双账号走查 PRD V1.5 第9节的新增行：确认、撤销、异议、30秒超时与离线、并发、练习不受影响。结果记录到 `docs/testing/`，与自动化结果分开。（用户人工确认通过；逐项明细待补充。）
- [x] 5.4 按“后端（本版不含迁移）→ UniApp”的顺序发布，确认测试服与生产稳定。（后端已发布并稳定；UniApp 发版与稳定性由用户人工确认。）
- [x] 5.5 新后端稳定后另行提交迁移，删除 `idx_matches_rank_settlement`、`matches.rank_settlement_status`、`matches.rank_settlement_due_at` 与 `match_participant_results.rank_excluded`（先查 `information_schema`，含 Down）。先在测试服实跑 Up/Down，再随下一次生产发布执行。（2026-10-15：迁移 `20261015170000` 已在测试服实跑 Up/Down/Up，并在生产执行 Up，执行前均已备份。）
- [x] 5.6 删除被取代且未归档的 `add-quick-match-ranked-dispute-window` 变更目录；在 `implement-quick-match-invites` 归档后归档本变更。（变更目录已删除；归档待 `implement-quick-match-invites` 归档后进行。）
