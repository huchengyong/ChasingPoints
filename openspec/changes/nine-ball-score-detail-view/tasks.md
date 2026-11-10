## 1. 后端：得分构成构建函数

- [x] 1.0 只读抽样存量九球追分对局（先确认目标环境与授权），核对"有效动作累计总分 = 最终比分"的通过率，重点看 2026-10-13 之前的对局；通过率偏低时先调整降级方案再继续（2026-11-10 生产库只读统计：已完成的九球追分对局仅 1 场，创建于 2026-10-27，对账通过；24 个胜局动作的胜法与分值全部为 4/7/10 标准值，无 `score` 动作，无缺失局记录。降级方案无需调整。）
- [x] 1.1 在 `backend/internal/logic/` 新增九球追分得分构成构建函数：输入对局、已完成局记录、有效动作，输出选手1 / 选手2 的统计、明细流水、是否可用、是否截断（胜法归一、`score` 与未知胜法计入其他得分、明细上限 200）
- [x] 1.2 实现对账：动作累计总分与 `matches` 最终比分不一致时标记不可用
- [x] 1.3 单元测试覆盖：完整对局统计、尾部犯规、撤销动作、旧胜法代码、早期 `score` 动作、对账失败、明细截断

## 2. 后端：契约与详情

- [x] 2.1 在 `backend/chasing_points.api` 新增 `ChasingScoreSide`、`ChasingScoreEvent`、`ChasingScoreBreakdown`，并给 `MatchDetailData`、`PublicMatchDetailData` 加可选的 `score_breakdown`；给 `ReplaySummary` 加可选的 `score_breakdown`
- [x] 2.2 核对 goctl 版本与模板后在 `backend/` 运行 goctl 生成，检查生成 diff 无无关改写
- [x] 2.3 参与者详情在九球追分时填充 `score_breakdown`（复用已加载的局记录与动作）
- [x] 2.4 公开详情在九球追分时填充 `score_breakdown`：已完成对局复用核心摘要的动作，进行中对局先计数再按对局 id 读取有效动作；动作总数超过 5000 条时不读列表并标记不可用
- [x] 2.5 详情测试：九球追分返回得分构成、其他球种不返回；公开详情补 query-count 测试，确认九球追分查询数固定、其他球种查询数不变

## 3. 后端：总结与复盘

- [x] 3.1 调整 `buildNineBallSummary`：统计条目去掉与对照表重复的项（本人普胜 / 小金 / 大金次数），亮点不变；更新 `match_summary_helper_test.go`
- [x] 3.2 调整 `match_replay.go` 的九球追分分支：返回单个整场片段（走势、最大连续得分、最大落后分差、用时），`summary` 填 `score_breakdown`、不填赢下局数，`frame_count` 与锁定返回不变，不标决胜局
- [x] 3.3 更新 `match_replay_test.go`：整场口径、尾部犯规计入、对账失败与超长降级、中八与美九复盘不变
- [x] 3.4 修订 `openspec/changes/revamp-vip-benefits/specs/member-benefits/spec.md` 中"单场复盘必须按球种从局记录与动作日志生成"的九球追分条目：由"每局得分走势"改为引用 `chasing-points-match-record` 的整场口径；只改九球追分几行，并对该变更运行 `openspec validate`
- [x] 3.5 在 `backend/` 运行 `go test ./...` 与 `go build .`

## 4. UniApp：数据归一与展示模型

- [x] 4.1 `utils/match-detail.js` 归一 `score_breakdown`：按现有视角交换规则映射为"我 / 对手"或"选手1 / 选手2"，生成对照表行与明细流水的展示模型；不可用时给出降级状态
- [x] 4.2 `utils/member-replay-view.js` 增加九球追分整场口径的展示模型（走势、连续得分、得分构成行），不再生成"赢下局数"与逐局卡片
- [x] 4.3 更新 `tests/match-detail.test.mjs`、`tests/member-replay-view.test.mjs`，覆盖视角交换、降级、其他球种不变

## 5. UniApp：页面

- [x] 5.1 `subPages/match/playing.vue`：九球追分不显示"第 N 局"
- [x] 5.2 `subPages/match/matchDetail.vue`：九球追分用对照表与得分明细流水取代"总局数 / 当前局 / 局记录"；进行中对局收到 `score_update`、`round_end` 后合并刷新详情
- [x] 5.3 `subPages/match/matchResult.vue`：九球追分展示双方对照表，核对与总结条目不重复
- [x] 5.4 `subPages/match/matchReplay.vue`：九球追分展示整场走势与得分构成，锁定页不展示局数
- [x] 5.5 新增样式遵循 `DESIGN.md`，检查深色模式与可访问性文本
- [x] 5.6 更新页面挂载测试（`match-result-page`、`match-replay-page-mount` 等）并在 `uniapp/` 运行测试

## 6. 验收与收尾

- [ ] 6.1 用 1.0 抽样到的对局核对接口返回的得分构成与最终比分一致（未完成：需先部署到测试环境，再用真实后端核对。）
- [ ] 6.2 H5 或开发者工具目检：选手、裁判、观战三种视角下的记分页、详情页、结算页、复盘页，含深色模式（部分完成：2026-11-10 用 H5 + 模拟接口目检了详情页（观战、选手 2、不可用、深色）、结算页（浅色、深色）、复盘页（会员、锁定）。未完成：裁判视角、记分页、微信开发者工具与真机。见 `docs/testing/nine-ball-score-detail-view-acceptance.md`。）
- [x] 6.3 回归中式八球、美式九球、斯诺克的详情与复盘展示未变（后端与 UniApp 全量测试通过；H5 目检中式八球详情未变。）
- [x] 6.4 检查 diff 与 `git diff --check`，在 `docs/testing/` 记录验收结果，区分自动化执行、人工确认与未执行项（见 `docs/testing/nine-ball-score-detail-view-acceptance.md`。）
