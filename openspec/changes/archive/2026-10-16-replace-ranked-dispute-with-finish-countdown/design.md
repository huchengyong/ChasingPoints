## Context

依据 [PRD V1.5](../../../docs/prd/familiar-player-quick-start.md) 第7–9节、[低保真 W1.5](../../../docs/prd/familiar-player-quick-start-wireframes.md) 与 [HTML 预览](../../../docs/previews/quick-match-ui/README.md)。

现状：
- 非约球、无裁判的排位赛开局时 `FinishConfirmationRequired=true`。参赛方结束（`/api/match/finish` 经 `shouldRequestRankedFinish`，或赛制达标时 `end_round_logic.go` / `snooker_action_logic.go` 自动转入）后，比赛进入 `finish_state=pending_confirmation`，比分冻结。另一方可确认（`handleConfirmedFinishAction` 结算，完成来源 `player_confirmed`）或异议（清除请求、恢复进行中），发起方可撤回。
- 请求有效期 `model.FinishRequestTTL = 24h`。到期由每分钟一次的 `FinishRequestExpiryWorker` 清除请求、恢复进行中，并写 `finish_expired` 动作；读取侧（如 `ws/hub.go`）把已过期请求视为无请求。索引 `idx_matches_finish_request_expiry (status, finish_state, finish_requested_at, id)` 已存在。
- 约球比赛在 `enter_challenge_logic.go` 以 `FinishConfirmationRequired=false` 开局，`shouldRequestRankedFinish` 也排除约球，由任一方直接完成。
- 2026-10-14 上线的 `add-quick-match-ranked-dispute-window`（提交 `99fc878`、`7a34736`、`451a324`）在此基础上加入：排位延后10分钟生效、`/api/match/result-dispute`、`RankSettlementWorker`、`rank_settlement_*` 字段，以及读模型 `rank_excluded` 延后应用。生产库现有比赛均为 `none`，没有待生效数据。

## Goals / Non-Goals

**Goals:**
- 无裁判排位对局结束前，另一方有30秒确认或异议，发起方可撤回。
- 30秒无人处理时，由服务端可靠、恰好一次地按冻结比分完成结算。
- 期满后不再有任何申诉入口。
- UniApp（App 与微信小程序）显示以服务端时间为准的倒计时和结束方式。
- 回退10分钟异议期。

**Non-Goals:**
- 改写段位或积分算法。
- 练习对局或裁判局的确认期。
- 限制同一场反复发起和异议的次数。
- 原生 iOS 的任何改动：原生开发尚未开始，iOS 归入迁移工作。
- 修改 HTML 预览（已随 V1.5 完成）。

## Decisions

1. **复用现有结束请求流程，不新建状态机。** 约球排位在 `enter_challenge_logic.go` 改为 `FinishConfirmationRequired = (match_mode == ranked)`，并删除 `shouldRequestRankedFinish` 与 `match_viewer_context.go` 中对约球比赛的直接结束特判。这样约球与扫码排位在手动结束和达标自动结束时，都走同一条 `pending_confirmation` 路径。练习赛（`FinishConfirmationRequired=false`）与裁判局的结束路径不变。*备选*：为约球单独写一套30秒请求。放弃原因：申请、确认、异议、撤回、幂等、冻结记分和 WebSocket 广播都已存在并有测试，重写只会制造两套规则。

2. **有效期改为30秒，全局统一。** `model.FinishRequestTTL = 30 * time.Second`，同时作用于约球与扫码排位。旧的24小时是“等双方都确认”的设计，而 V1.5 明确不允许被对方拖住。*备选*：约球30秒、扫码保留24小时。放弃原因：同一种比赛出现两种结束规则，客户端也要维护两套文案。

3. **到期即完成结算，复用确认路径。** 到期处理改为：
   - 锁比赛行；
   - 复核 `status=1`、`finish_state=pending_confirmation`，且 `finish_requested_at + TTL <= now`；
   - 调用与确认相同的结算函数完成比赛，完成来源记为新增的 `finish_timeout`，`completed_by_user_id` 记为发起方；
   - 写系统动作 `finish_timeout`（actor 0），并在同一事务内同步完成关联约球。

   提交后的副作用与确认路径一致：`match_end` 广播、会员成长、成就、`rank_info_updated`、缓存失效。确认、异议、撤回三个动作如果在锁内发现请求已到期，同样先按到期完成，再以“已超时结束”拒绝原动作，并返回权威状态。比分沿用请求时冻结的比分，确认期间记分接口已被拒绝。

4. **过期扫描改为秒级。** `FinishRequestExpiryWorker` 的间隔由1分钟改为2秒，批量100，沿用现有索引。服务端完成时刻最多晚于截止约2秒，加上处理耗时；客户端在归零后显示“正在结束…”。任意写动作也会在锁内完成到期请求，worker 只是保底，不依赖任何人在线。*备选*：发起请求时挂进程内定时器。放弃原因：进程重启会丢失定时器，多实例还要额外协调；扫描本身幂等，代价很低。

5. **并发由行锁串行，先提交者决定结果。** 确认、撤回、异议与到期都在同一比赛行锁内复核状态。后到的请求看到 `finish_state` 已变化或比赛已完成，就以 `errFinishActionInvalid` 或“状态已变化”拒绝，并返回最新快照。`client_action_id` 的幂等重放规则不变。是否到期只按服务端 `now` 判断。

6. **读取返回倒计时权威字段，到期未处理不再伪装成已取消。** 当前对局、`MatchDetailData`、同步快照与相关 WebSocket 消息新增两个字段：
   - `finish_request_expires_at`（RFC3339）；
   - `finish_request_remaining_seconds`（按服务端 `now` 计算，最小为 0）。

   删除 `ws/hub.go` 等处“已过期请求视为无请求”的读取改写：到期但尚未结算的比赛仍返回 `pending_confirmation`、剩余0秒，各操作权限为 false，客户端据此显示“正在结束…”。读取保持纯读，不在 GET 中结算。修改 `.api` 后立即运行 goctl。

7. **结束方式由完成来源区分。** 排位赛的 `completion_source` 为 `player_confirmed`（对方确认）或 `finish_timeout`（30秒未处理）。`completed_by_user_id` 沿用“实际完成人”的语义：确认时为确认方，所以发起方是另一名参赛者；超时时为发起方。客户端据此显示“X发起 · Y确认”或“X发起 · 30秒未处理，自动结束”。练习赛的手动结束和达标自动结束都记为 `player_direct`，无法区分，因此赛果页不显示结束方式，改由结束提醒“X结束了对局 · 比分”告知。本变更不新增区分标记。

8. **回退10分钟异议期，删列分两步。** 删除以下内容：
   - `settleOptions.manualFinish` 的 pending 分支、`planRankSettlement` / `applyDeferredRankStatsWithTx` / `RankSettlementWorker`（及其在 `chasing_points.go` 的注册）；
   - `DisputeRankResultLogic`、路由与 handler；
   - `rank_settlement_*` 与 `can_dispute_result` 的读取字段；
   - `rank_excluded` 的写入和排位口径过滤；
   - 旧 `matches` 直读统计里对 `pending/voided` 的排除；
   - 相关通知文案与测试。

   与延后结算无关、且仍有调用方的通用重构可以保留，否则一并删除。迁移 `20261015100000` 已在测试与生产执行，文件保留。新后端稳定上线后，另行提交迁移删除 `idx_matches_rank_settlement`、`matches.rank_settlement_status`、`matches.rank_settlement_due_at` 与 `match_participant_results.rank_excluded`，并单独发布。原因是 `deploy-prod.sh` 先执行迁移再切换二进制，同一版本删列会让仍在运行的旧进程报错。

9. **UniApp 只展示与发起操作，时间以服务端为准。**
   - **两端共用：** 代码由 App 与微信小程序共用，不引入单平台 API 或条件编译分叉；确认滑块复用现有结束滑块的实现。
   - **UniApp：** 新增纯函数工具，按“收到快照时的剩余秒数 − 本地经过时间”倒数，只依赖本地时钟的流逝速度，不依赖手机时钟的绝对值。归零后显示“正在结束…”，3秒内没有收到 `match_end` 就回源读取。`playing.vue` 用发起方面板（倒计时 + 撤销结束）和对方确认弹窗（复用结束滑块的防误触实现 + “有异议，继续比赛”）替换现有的确认面板，调用现有的 confirm/dispute/withdraw 门面。异步回写按 `userId + authGeneration` 隔离身份。`matchResult.vue` 删除结算状态与异议框，排位赛显示结束方式。删除 `utils/rank-settlement.js` 与 `disputeRankResult` 门面。

## Risks / Trade-offs

- **[对方锁屏或离线，30秒内来不及处理]** 这是有意设计，期满按冻结比分结束。发起时仍走现有通知，但30秒内的确认以 App 前台的实时弹窗为主。
- **[反复“发起—异议”拖延比赛]** 双方当面打球，可以当面沟通；本版不限次数，在 Open Questions 跟踪。
- **[worker 停摆，比赛停在“正在结束”]** 读取如实显示剩余0秒。任一方的写动作会在锁内完成结算，worker 恢复后补处理。日志记录积压数量和最老的截止时刻。
- **[旧版 App 不显示倒计时]** 已发布的客户端如果仍显示旧确认面板，能确认或异议，只是没有倒计时；如果不显示任何入口，也会在30秒后由服务端结束，结算正确。
- **[App 与小程序的编译和渲染差异]** Node 测试覆盖不到 → 验收时在 HBuilderX 分别编译 App 与微信小程序，并真机走查倒计时、确认弹窗与滑块。
- **[回退范围大，可能误删有用改动]** 按 `99fc878` 逐文件核对。保留与延后结算无关的修复，补“确认 / 超时 / 撤回 / 异议 / 并发 / 练习不受影响 / 统计口径恢复”的回归测试。

## Migration Plan

1. 后端：回退异议期，并实现第1–6点；修改 `.api` 后运行 goctl；`go test ./...`。
2. 接入 UniApp，运行 `node --test tests/*.test.mjs`，并在 HBuilderX 分别编译 App 与微信小程序。
3. 部署顺序：后端（本版不含迁移）→ 客户端。回滚时重新部署 `451a324` 即可：结算三列在第4步之前仍然存在，旧二进制可以正常运行。
4. 新后端稳定后，另行提交删除三列与索引的迁移，先在测试服执行 Up/Down，再随下一次生产发布执行。

## Open Questions

- 是否需要限制同一场的“发起—异议”次数或加冷却时间：本版不限，视实际使用再提。
- 练习对局是否也要30秒确认：本版不需要，如需只要在开局时打开确认开关，改动很小。
