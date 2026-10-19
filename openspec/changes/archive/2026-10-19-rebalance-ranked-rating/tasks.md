## 1. 纯计分与规则配置

- [x] 1.1 新增 `backend/internal/ranking`：单局胜率、抢 N 整场胜率、`K = 40·√N`、赛制 N 的计算（抢 N 取目标、自由/旧赛制取较大赢局数、限制 1–9）与对称舍入；先写表驱动测试覆盖同分 ±45、差 500 分的 +68/−21、5:0 与 5:4 同分、自由局与整场认输。
- [x] 1.2 在同一包实现 V2 单场结算纯函数（定级 ×2、对手定级 ×0.5、同对手 100/100/80/30/30/30/0、每日封顶、0 分保护、胜负最小 ±1），输出明细与 `remark`；测试明细之和等于最终变化、零和、第 7 场不计分不计统计。
- [x] 1.3 新增 `Ranking.RatingV2StartDate` 配置与规则解析（业务时区 00:00、必须是赛季窗口起点、错误时回退旧规则并记录），提供 `V2ActiveAt`、`IsV2Season`、业务日界；`model` 提供包级段位计算供两套规则共用。

## 2. 数据库与模型

- [x] 2.1 新增迁移：`user_ranking.rating_version`、`rated_matches` 与 `idx_user_ranking_rating_version`；`rank_rating_seeds` 表；`season_records.distinct_opponents`、`qualified`，新建 `idx_season_game_qualified_rank` 后删除 `idx_season_game_rank`；`seasons.rank_reset_ratio` 默认值与存量改为 1。全部用 `information_schema` 判断并提供 Down，不指定字符集。
- [x] 2.2 同步 Gorm 模型与测试 schema（`model.UserRanking`、`model.SeasonRecord`、`model.Season` 默认值、`testsupport` 荣誉墙 schema），新增种子模型。
- [x] 2.3 `RankingModel`：带行锁的 V2 查找/创建、按默认值初始化四球种段位、切换单行与分批读取未切换行、V2 全站榜资格条件；删除无人调用的 `UpdateAfterMatch`。
- [x] 2.4 `SeasonRecordModel`：投影时判断本季首次对手、累加不同对手数并刷新达标状态；榜单/计数/结算排序改为达标优先；`CountAheadInSeason` 限定同一达标组并强制新索引。

## 3. 结算与切换

- [x] 3.1 实现切换函数（种子或 1250、`rating_reset` 日志、幂等），供 V2 结算、Worker 与段位初始化共用；测试并发与重复切换只生效一次。
- [x] 3.2 `finish_match_logic` 按完成时间选择规则：V2 分支按用户 ID 顺序锁定双方行、就地切换、读取同对手与当日涨分、调用纯函数、写日志与快照，不再计算会员战绩分；测试起点前后两场分别走两套规则、会员 147 与普通用户一致。
- [x] 3.3 新增 `RankingRulesWorker` 并在启动时注册：起点后用索引快路径分批切换未切换行；测试起点前不动、起点后全部切换、重复执行无副作用。
- [x] 3.4 段位初始化入口（注册/登录初始化、修复命令）按规则版本取默认值。

## 4. 赛季达标与对手强度

- [x] 4.1 竞技投影：V2 赛季维护不同对手数与达标状态；V2 后对手强度按结算前分差写相对分档；测试再次遇到同一对手不重复计数、重复投影不重复累加、旧赛季不标达标。
- [x] 4.2 赛季结算：V2 赛季只为达标记录写 `final_rank`，称号只发给前 10 名达标者；旧赛季保持全员排名；测试只有两人达标时只发两个称号、未达标高分者无称号。

## 5. 接口契约与读取

- [x] 5.1 修改 `.api`：`RankInfo.placement_remaining`、`GetRankConfigsResp.rules_version/rules_effective_at`、`MatchDetailData.rank_rules_version`、`SeasonInfo.qualify_min_matches/qualify_min_opponents`、`SeasonRecordInfo.qualified/distinct_opponents`、`SeasonLeaderboardItem.qualified`；立即运行 goctl 并检查生成 diff。
- [x] 5.2 段位读取：V2 下无段位行按 1250 合成、返回剩余定级场次；段位配置返回规则版本与起点并计入版本号。
- [x] 5.3 全站榜、摘要与本人名次按规则版本使用资格条件；未上榜本人按初始分段位展示。
- [x] 5.4 赛季 overview / 榜单 / 本人记录 / 历届 / 报告返回门槛、达标与不同对手数；榜单名次按达标组计算；未达标 `my_rank = 0` 且不计数；更新固定 SQL 数测试。
- [x] 5.5 段位趋势只取 V2 起点后的对局日志；强弱对手分析 V2 后只读相对分档；战报明细解析 V2 `remark` 并返回 `rank_rules_version`。
- [x] 5.6 更新 `migrations/mysql_explain_test.go`：赛季榜与名次计数改用新索引且无 filesort，新增切换 Worker 快路径用例。

## 6. 种子与审计工具

- [x] 6.1 实现共享重放引擎：按 `(completed_at, id)` 分批读取对局与局记录、维护同对手与当日涨分、复用纯函数；测试与线上结算逐场一致。
- [x] 6.2 重写 `cmd/rank_rebuild`：`-mode=seed`（默认预览分布与变化概览，`-apply` 整表替换种子，起点后拒绝）与 `-mode=audit`（只读，从 `rating_reset` 恢复起点，比较逐场变化与当前分数）；删除旧重放服务与测试。

## 7. UniApp

- [x] 7.1 段位说明：按 `rules_version` 展示 V2 规则（期望计分、赛制分值示例、定级赛、同对手衰减、每日上限、高光不计分、赛季达标门槛），V2 已排期时预告生效日期；修正旧规则文案中不存在的"失败减免"；补测试。
- [x] 7.2 赛季页：达标进度、本季净变化、按服务端名次渲染榜单、未达标标注、加载更多按用户去重；历届赛季标注未达标；补测试。
- [x] 7.3 战报：V2 对局的高光卡片说明高光不计入排位分；会员中心排位权益区按规则版本改为说明；补测试。

## 8. 管理后台

- [x] 8.1 VIP 排位权益配置页增加说明：V2 生效后只有"每日上涨上限"生效，其余项仅影响切换前的对局。

## 9. 验证与收尾

- [x] 9.1 在 `backend/` 运行改动包测试后运行 `go test ./...` 与 `go build .`，清理构建产物。
- [x] 9.2 在 `uniapp/` 运行 `node --test tests/*.test.mjs`；在 `admin/` 运行测试与 `npm run build`。
- [x] 9.3 有授权的隔离 MySQL 时运行 EXPLAIN 回归；否则记录为未执行。
- [x] 9.4 核对原生 iOS 解码不受影响、`git diff --check` 通过、相关指南仍准确；整理上线步骤（配置、种子预览与写入、首日审计）。
