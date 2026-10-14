## 1. 数据与结算拆分

- [x] 1.1 新增 goose 迁移：`matches` 增加 `rank_settlement_status`、`rank_settlement_due_at` 及 `(status, due_at)` 索引（先查 `information_schema`，含 Down），存量已完成比赛回填 `settled`/`none`；同步 Gorm 模型与 `internal/testsupport` schema，验证 Up/Down（本机仅做文本与 sqlite 校验；MySQL 实跑见 6.1，需授权隔离库）。
- [x] 1.2 把 `settleMatchResultTx` 的段位快照、`rank_change_logs` 与排位投影抽为 `applyRankSettlementTx(tx, match, effectiveAt)`；即时路径行为不变，补回归测试覆盖练习、平局、自动达标、裁判、认输与非约球双确认结果一致。
- [x] 1.3 为结算增加调用方显式传入的手动结束标记：仅 `/api/match/finish` 参赛方手动结束 + 约球 + 排位 + 胜负时写 `pending` 与 `due_at = 结束时刻 + 10 分钟`，仍完成对局、约球、结束方、`match_end` action 与参与者投影；其余路径写 `settled`/`none`。测试 `end_round` 自动结束与斯诺克路径不进入 `pending`。

## 2. 延后结算与读模型

- [x] 2.1 `CompetitiveProjector` 支持延后应用：`pending` 完成时只写参与者行，生效时补段位快照并一次性应用总场次/胜负/连胜、H2H、对手强度与赛季记录增量；`voided` 只更新参与者行状态。测试重复执行不重复增量、跨赛季按生效时刻归属、作废不计入任何排位口径统计。
- [x] 2.2 实现 `RankSettlementWorker`（每分钟、批量有界，在 `chasing_points.go` 注册）：锁比赛行复核 `pending` 与 `due_at`，按用户 ID 升序锁段位后调用 `applyRankSettlementTx` 并置 `settled`；测试多实例/重试恰好一次、`effectiveAt` 为生效时刻、待生效期间可正常约球与开局。
- [x] 2.3 把 `rank_info_updated` 广播、会员成长与成就进度对 `pending` 比赛推迟到 `settled`/`voided` 提交后执行（作废按非排位处理），核对依赖排位胜负的成就与成长条目；补日志或指标记录积压数量与最老 `due_at`。

## 3. 异议接口与契约

- [x] 3.1 在 `backend/chasing_points.api` 增加 `POST /api/match/result-dispute`，并在当前对局、`MatchDetailData` 与同步快照增加 `rank_settlement_status`、`rank_settlement_due_at`、`can_dispute_result`；**紧接着**在 `backend/` 运行 `goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default`，检查生成 diff，不留 `todo` 空壳。
- [x] 3.2 实现异议 logic：锁比赛行，校验约球、排位、`pending`、服务端时间早于 `due_at`、请求者为非结束方的参赛方，置 `voided` 并写 `result_dispute` action；`client_action_id` 幂等重放。测试结束方/裁判/旁观者/非参赛者被拒、超时与重复被拒、异议与 worker 竞争只产生一种终态、对局与约球不被重开。
- [x] 3.3 读取逻辑按身份与服务端时间计算 `can_dispute_result`，`my_rank_change` 仅在 `settled` 时返回，到期未处理仍如实返回 `pending`；保持纯读并补 query-count 测试。
- [x] 3.4 手动结束提交后通知另一参赛方「X 结束了对局 · 比分，HH:MM 前可提出异议」（练习只发前半句），异议提交后通知结束方；只携带比赛 ID，走现有 `DispatchNotification` 与 user WS。

## 4. UniApp

- [x] 4.1 `uniapp/api/match.js` 增加异议门面；`subPages/match/matchResult.vue` 按权威字段显示结束方（「你滑动结束」/「X滑动结束」）、排位结算「HH:MM 生效 / 已生效 / 有异议 · 不计排位积分」，`can_dispute_result` 时在比分下方显示异议框，一次确认后提交、提交中防重、失败保留入口并回源。
- [x] 4.2 `subPages/match/playing.vue` 对约球排位的滑块说明改为「松手即结束，无需对方确认。按当前比分结算排位，对方10分钟内可提出异议。」，练习保留第一句；结束与异议成功后按 `userId + authGeneration` 精确失效相关缓存。
- [x] 4.3 补 `node --test` 用例：异议框仅对非结束方且在期限内显示、确认后才请求、失败复位、待生效不显示 +0、切号后旧响应不回写；运行 `node --test tests/*.test.mjs`。

## 5. 原生 iOS

- [x] 5.1 `MatchHistoryModels.swift` 解码 `rank_settlement_status` / `rank_settlement_due_at`（缺省按旧行为），`MatchReportView.swift` 排位区块对待生效显示「HH:MM 生效」、对作废显示「不计排位积分」，不显示 +0、不提供异议操作。
- [x] 5.2 用 `MockURLProtocol` 补待生效、已生效、作废与字段缺失的解码/展示测试，运行 `xcodebuild test` 与 Debug/Release 构建。

## 6. 验证与文档

- [x] 6.1 后端 `go test ./...`；运行本变更 `openspec validate --strict`，核对 goctl 生成物与新路由、迁移文本校验。
- [ ] 6.2 双账号走 PRD 第9节新增两行：A 滑动结束后 B 收到提醒并在期限内异议、期限后生效、结束方无入口、练习与自动达标无异议期、待生效期间再约正常；记录环境与结果到 `docs/testing/`，与自动化结果分开。
- [ ] 6.3 若实现与 PRD V1.4 / W1.4 有差异，先说明再同步文档；在 `implement-quick-match-invites` 归档后再归档本变更。
- [ ] 6.4 在明确授权的隔离 MySQL 上实跑迁移 Up/Down，并用代表性数据对 worker 查询（`rank_settlement_status = 'pending' AND rank_settlement_due_at <= ?`）做 EXPLAIN，确认走 `idx_matches_rank_settlement`。
