## Context

依据 [PRD V1.4](../../../docs/prd/familiar-player-quick-start.md) 第7–9节与[低保真 W1.4](../../../docs/prd/familiar-player-quick-start-wireframes.md)。`implement-quick-match-invites` 已让约球比赛（`matches.challenge_id` 非空）的排位 `finish_confirmation_required=false`，参赛方经 `/api/match/finish` → `FinishMatchLogic.settleMatchResultTx` 在一个事务内完成对局、同步约球、写 `completed_by_user_id` / `completion_source`，并立即结算排位：更新 `user_rankings` 快照、写 `rank_change_logs`、调用 `CompetitiveProjector.ProjectWithTx` 投影参与者行与排位口径统计；提交后再发会员成长、成就进度与 `rank_info_updated` 等广播。赛制达标自动结束走 `end_round_logic.go` 的同一结算函数，斯诺克认输/自动完成走 `snooker_action_logic.go`。现有 `DisputeFinishMatch` 只处理非约球排位「结束请求」的异议并恢复记分，约球比赛不进入该流程。`FinishRequestExpiryWorker` 是现成的每分钟批处理 worker 模式。项目未上线，无存量数据转换。

## Goals / Non-Goals

**Goals:** 约球排位手动结束后对局与约球立即完成、再约不受阻；排位结算在 10 分钟异议期后由服务端可靠、恰好一次生效；另一方可提出一次异议使本场不计排位积分；各端如实展示结束方与结算状态。

**Non-Goals:** 改写段位/积分算法；恢复对局或重开约球；人工仲裁或异议理由；为赛制自动结束、裁判结束、认输、练习、非约球比赛增加异议期；在原生 iOS 提供异议操作或实时对局；修改 HTML 预览实现。

## Decisions

1. **把「完成对局」与「排位结算」拆成两段，用比赛行上的状态机衔接。** `matches` 新增 `rank_settlement_status`（`none` / `pending` / `settled` / `voided`）与 `rank_settlement_due_at`，并为 worker 建 `(rank_settlement_status, rank_settlement_due_at)` 索引。`due_at` 在生效或作废后保留，作为实际生效时刻；`pending` 以外的状态 worker 不再扫描。约球排位自动达标结束即时写 `settled` 且不带 `due_at`，客户端据此区分「走过异议期」与「即时结算」。`settleMatchResultTx` 增加调用方显式传入的「手动结束」选项（`settleOptions.manualFinish`）：只有 `/api/match/finish` 的参赛方（非裁判）手动结束、约球排位、胜负结果三者同时成立才写 `pending` 并在此返回（仍完成对局、约球、结束方、`match_end` action、revision 与参与者投影）；其余路径照旧立即结算并写 `settled`/`none`。手动标记由调用方传入而非从 `completion_source` 推断——自动达标路径同样记为 `player_direct`，无法区分。*备选*：先立即结算、异议时冲正。放弃原因：PRD 写的是「生效」，冲正要回滚段位快照、等级与已投影统计，后续比赛叠加后无法精确还原。

2. **排位结算抽成可单独调用的事务函数，由 worker 与即时路径共用。** 把现有段位/日志/投影部分抽为 `applyRankSettlementTx(tx, match, effectiveAt)`，即时路径在结束事务里调用，延后路径由新 `RankSettlementWorker`（每分钟、批量 100，仿 `FinishRequestExpiryWorker`）调用。worker 对每场 `SELECT … FOR UPDATE` 锁比赛行、复核 `status=pending AND due_at<=now`，再锁双方 `user_rankings`（按用户 ID 升序），完成后置 `settled`；锁内复核保证多实例与重试恰好一次。`effectiveAt` 取实际生效时刻，用于段位策略、`rank_change_logs.effective_at` 与赛季归属——等价于「这场比赛晚 10 分钟结束」，避免写入已关闭赛季。

3. **异议是一次条件状态转换，不碰对局与约球。** 新增 `POST /api/match/result-dispute`（`match_id` + `client_action_id`），logic 放 `logic/match/`。事务内锁比赛行，校验：约球比赛、排位、`pending`、服务端 `now < due_at`、请求者是参赛方且 ≠ `completed_by_user_id`；通过则置 `voided`、推进 revision，并写一条 `result_dispute` action 便于审计与幂等重放。到期结算同样推进 revision 并写 `rank_settled` 系统动作（账号已注销时写 `rank_voided`），客户端可凭 revision 感知变化，也避免持有旧副本的写入把状态覆盖回去。与 worker 的竞争由同一行锁串行，先提交者决定 `settled` 或 `voided`。不复用 `DisputeFinishMatch`：其语义是「恢复记分」，与已完成约球、立即再约冲突，且会形成结束—异议—再结束循环。

4. **竞技读模型分两步：完成时写参与者行，生效时才计排位口径。** 完成事务仍按 `match_id + user_id` 幂等写参与者投影（历史立即可见），但对 `pending` 比赛不应用排位口径增量（总场次/胜负/连胜、H2H、对手强度分档、赛季记录）与段位前后快照。`match_participant_results` 新增 `rank_excluded`（默认 0；待生效与作废时为 1），H2H 历史、近期趋势与高分榜三类排位口径读取加 `rank_excluded = 0`，对局历史列表不过滤。`CompetitiveProjector` 增加 `ApplyDeferredRankStatsWithTx`：以参与者行 `rank_excluded` 1→0 的条件更新作为幂等信号，成功才应用增量，赛季按传入的生效时刻归属；`voided` 保持 `rank_excluded = 1`。用 `RankSettlementDeferred()` 统一判断，历史回建与旧的 `matches` 直读统计（用户统计、胜率趋势、对手强度、时长、排位回放）同样排除待生效与已作废。投影 revision 与失效事件在真正应用时推进。

5. **提交后副作用跟随结算终态。** `pending` 比赛在结束提交后照常广播 `match_end`、发通知，但 `rank_info_updated`、会员成长与成就进度推迟到 `settled` / `voided` 提交后执行（`voided` 按非排位处理）。实现时核对成就与会员成长定义中依赖排位胜负的条目。通知「X 结束了对局 · 比分，HH:MM 前可提出异议」与「Y 提出异议，本场不计排位积分」走现有 `DispatchNotification` + user WS，只带比赛 ID，事务提交后发送。

6. **读取纯读、字段权威。** 当前对局、`MatchDetailData` 与同步快照新增 `rank_settlement_status`、`rank_settlement_due_at`、`can_dispute_result`（服务端按身份与 `now` 计算）；结束方沿用已有 `completed_by_user_id`。到期未处理仍返回 `pending`，`my_rank_change` 只在 `settled` 时有值，客户端据状态显示「待生效 / 不计排位积分」而不是 0。修改 `.api` 后立即运行 goctl。

7. **客户端只做展示与一次确认。** UniApp：`api/match.js` 增加异议门面；`matchResult.vue` 依字段显示结束方、结算状态与异议框（一次确认、提交中防重、失败保留入口并回源、成功后按身份与认证代次失效相关缓存）；`playing.vue` 对约球排位的滑块说明写「按当前比分结算，对方10分钟内可提出异议」。原生 iOS：`MatchHistoryModels` 解码新字段，`MatchReportView` 的排位区块按状态展示，不提供异议操作。

## Risks / Trade-offs

- **[同一用户排位结算顺序倒置]** 待生效期间 A 又打完一场即时结算的排位，两场按生效先后而非结束先后结算 → 每次结算都基于当时快照，结果自洽；`effectiveAt` 记录真实生效时刻，可审计。
- **[输方总提异议规避掉分]** → 与旧流程「输方拒不确认」的风险相当；作废记录保留在历史中可见，后续可据异议次数做信誉处理（本变更不做）。
- **[worker 停摆导致长期待生效]** → 读取如实显示待生效，不阻塞任何操作；worker 恢复后补处理；增加积压量与最老 `due_at` 的日志/指标。
- **[投影拆分引入重复增量]** → 增量只在 `pending→settled` 的锁内转换中应用一次；补「重复执行、多实例、异议与到期竞争、自动结束不受影响」的测试。
- **[跨赛季]** → 以生效时刻归属新赛季，已结束赛季不再变更（见 `competitive-read-models` 增量）。

## Migration Plan

1. goose 迁移为 `matches` 增加两列与索引（`information_schema` 检查后 ALTER，含 Down），存量已完成比赛回填 `settled` 或 `none`；同步 Gorm 模型与测试 schema。
2. 修改 `.api` 并运行 goctl；实现 logic、worker 注册（`chasing_points.go`）与投影拆分，后端 `go test ./...`。
3. 接入 UniApp 与 iOS，分别运行 `node --test tests/*.test.mjs` 与 `xcodebuild test`。
4. 部署顺序：迁移 → 后端（含 worker）→ 客户端；旧客户端读到 `pending` 时 `my_rank_change` 为 0，仅展示不准确，不影响结算正确性。回滚：停用异议接口并让 worker 结算全部 `pending`，再回退服务。

## Open Questions

- 异议期时长按 PRD 定为 10 分钟，实现为服务端常量；如需运营可调再另行提出。
