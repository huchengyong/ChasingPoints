## Context

- 线上结算在 `internal/logic/match/finish_match_logic.go` 调用 `RankSettlementService.SettleWithPolicy`：胜方基础分只看完成局数（1 局 +8 … 20 局 +40），败方扣一半；唯一的对手因素是"对手剩余分不够扣时按比例缩减"。每场向系统净注入 B/2，保本胜率 1/3；会员赢球时额外计入特殊战绩分（×1.0–1.4，日上限 200）。
- `user_ranking` 按 `(user_id, game_type)` 保存当前分、段位与生涯胜负/连胜；`rank_change_logs` 按场记录明细（`remark` JSON 存调整项），段位趋势、赛季报告与战报明细都从它读取；`rank_config` 与 `RankingModel.CalculateLevel` 各存一份 0/500/1000/1500/2000/2500 阈值。
- 赛季记录由竞技投影在同一事务写入；赛季榜与 `final_rank` 按 `end_rank_score DESC, wins DESC, id DESC`，`idx_season_game_rank` 支撑无 filesort 的榜单与 `my_rank` 计数；前 10 名发称号，没有参赛门槛。赛季为自然月，`rank_reset_ratio`（默认 0.7）存在但规格禁止应用。
- `cmd/rank_rebuild` 会清空 `rank_change_logs`/`user_ranking`/`season_records` 后用缺参的 policy 重放（胜 +2、负 −4，战绩分人人有份），称号只增不删；会员身份历史不可回溯，旧规则本就无法忠实重放。
- 约束：读取纯读、有界、SQL 次数固定；契约以 `.api` 为准并由 goctl 生成；不在线 `AutoMigrate`；DDL 不指定字符集；原生 iOS 只按 label/value 渲染排位明细。

## Goals / Non-Goals

**Goals:**
- 排位分反映实力：期望驱动、近似零和、赛制感知，合谋只能转移分数不能凭空造分。
- 规则在一个确定的时刻整体切换，切换前的对局、赛季、称号与日志保持原样。
- 用户在切换时得到有意义的初始分（V2 规则重算其历史），而不是所有人冷启动。
- 赛季名次与称号需要真实参与；赛季榜与 `my_rank` 继续走索引、无 filesort。
- 线上结算、初始分计算与审计重放共用同一个纯函数，可以证明"线上 = 重放"。

**Non-Goals:**
- 匹配/推荐对手、Glicko 不确定度、按局比分（FargoRate 式）计分。
- 改变赛季周期（仍为自然月）或段位阈值与名称。
- 删除旧规则代码与 VIP 排位配置（V2 稳定运行一个赛季后另起清理变更）。
- 全站排行榜的活跃度衰减、段位小段（I/II/III）。
- 原生 iOS 新增段位/赛季页面。

## Decisions

### D1：期望驱动的等级分，按赛制换算整场胜率
- 单局胜率 `p = 1 / (1 + 10^((R_opp − R_me) / 2500))`：分差 2500 对应单局 10:1，一个段位（500 分）约 1.58:1。
- 抢 N 整场胜率 `E = Σ_{k=0}^{N−1} C(N−1+k, k)·p^N·(1−p)^k`；变化 `K·(S − E)`，`S` 为 1/0，`K = 40·√N`。
- N 的取值：抢 N 赛制取目标局数；自由局/旧赛制取双方各自赢局数的较大值；限制在 1–9。这样 5:0 与 5:4 同 K，整场认输按赛制计 N，不再出现"惜败比被剃光头扣得多""落后就早认输"的激励。
- 舍入四舍五入（对称），胜方至少 +1、负方至少 −1；无其他调整时双方变化严格相反。
- 参数取值来自模拟（真实实力标准差 750 分、六种赛制混合）：K₁=40 时抢 5 同分 ±45，稳态中位误差约 120 分（约 1/4 段位）；K₁=24 收敛明显更慢，K₁≥44 噪声收益不再增加。
- 备选：保留"胜 +B / 负 −B/2"只乘对手系数。否决：仍是正和，段位继续随场次膨胀。备选：按局比分（赢局数 − 期望局数）计分。否决：会出现"赢了还掉分"，与段位产品心智冲突。

### D2：定级赛、对手定级减半、同对手对称衰减、每日上限、0 分保护
- 本人 V2 已计分场次 < 10 时 K×2；本人已定级而对手仍在定级中时 K×0.5（限制强号新开、小号喂分对已定级者的冲击）。
- 同一对手当日（业务时区）此前已完成排位 0–1 场 100%、第 2 场 80%、第 3–5 场 30%、≥6 场不计分且不计入胜负统计——阈值沿用旧规则，但双方同比例缩放，避免频繁对练的朋友双双被动扣分。
- 每日上涨上限沿用会员权益配置中的 `daily_positive_cap`（默认 500），只截正向；分数不低于 0。
- 明细顺序固定：胜负结算 → 定级赛加倍 / 对手定级中减半 → 同对手衰减 → 每日封顶 → 0 分保护，各项之和等于最终变化，写入 `rank_change_logs.remark`（`{"v":2,...}`），`base_score` 存"胜负结算"，`achievement_score` 恒为 0。
- 平局、练习赛、任一方非注册用户：不计排位分。

### D3：会员高光不再计入排位分
- V2 结算不读取会员身份与特殊战绩奖励；特殊战绩照常写入战报、生涯成就与统计。
- 旧规则在切换前继续按原配置执行；VIP 排位配置保留到清理变更，管理后台页面标注 V2 后仅"每日上涨上限"生效。
- 备选：对所有人开放战绩分。否决：战绩是双方自报，按局记分即可伪造；排位分应只由对局结果决定。

### D4：按配置的切换时刻整体切换，结算按完成时间选择规则
- 两种配置二选一：`Ranking.RatingV2StartDate`（`YYYY-MM-DD`，按 `SeasonLifecycle.Timezone` 解释的 00:00，启用赛季生命周期时必须是某个赛季窗口起点）或 `Ranking.RatingV2StartAt`（RFC3339 时刻，如 `2026-10-20T18:00:00+08:00`，即时生效，不要求对齐赛季）。两项都留空表示不启用；同时配置或格式错误时启动日志报错并视为未配置（安全回退旧规则）。
- 选择即时生效时，切换时刻所在的当前赛季因开始日早于切换时刻，按旧赛季口径结算（无名次门槛）；切换后的新赛季才启用达标门槛。
- 完成时间（`completed_at`，即结算时刻）早于起点的对局走旧规则，之后走 V2；完成时间等于结算时刻，不存在跨规则的迟到写入。
- 切换动作 = 把行的 `rating_version` 从 1 改为 2，分数改为该用户在该球种的初始分（种子，缺失时 1250），`rated_matches` 改为种子场次，并写一条 `change_type = rating_reset` 的日志（`before` 为旧分，`after` 为初始分，`remark` 记种子场次）。三个入口共用一个函数：
  1. V2 结算时对锁定的行就地切换（保证正确性）；
  2. `RankingRulesWorker` 每分钟检查 `rating_version < 2` 的行并分批切换（保证无人对局的用户在一分钟内看到新分）；
  3. 起点之后新建的段位行直接按 V2 默认值创建。
- 两次切换竞争同一行时，行锁 + `WHERE rating_version < 2` + 日志唯一键 `(user_id, match_id=0, change_type, game_type)` 保证只生效一次。
- 备选：部署即切换。否决：切在赛季中间会让该赛季前后混用两套分数。备选：所有人从 1250 冷启动。模拟显示冷启动后 20 场的中位误差仍有约 250 分（半个段位），首个 V2 赛季的榜单与称号噪声过大。

### D5：初始分 = 用 V2 规则重放全部历史排位赛
- `cmd/rank_rebuild -mode=seed` 以 `(completed_at, id)` 顺序、固定批次读取起点之前已完成的排位赛，从 1250 起按 V2 纯函数重放（跳过平局、练习、访客对手；同对手计数与每日上限在重放中维护），输出段位分布与变化最大的用户；加 `-apply` 时整表替换 `rank_rating_seeds`。
- 只允许在起点之前执行；起点之后拒绝（切换已开始，再写种子会让部分用户用旧种子、部分用新种子）。最后一天的对局晚于计算时间则不进入种子，属于可接受的信息损失；建议在旧赛季最后一晚执行。
- 种子只决定 V2 起点分数，不改写任何历史日志、赛季记录或称号。
- 备选：在切换时由 Worker 重放。否决：重放需要批量读取对局与局记录，不适合放在请求或每分钟任务里，且起点后首批结算会拿不到种子。

### D6：替换 `cmd/rank_rebuild`，审计取代改写
- 删除清空并改写全站段位的旧重放、旧服务及其测试，同时删除没有调用方的 `RankingModel.UpdateAfterMatch`（第三套 +20/−10 公式）。
- `-mode=audit`（只读）：从每位用户的 `rating_reset` 日志恢复 V2 起点分与种子场次，按同一纯函数重放起点之后的对局，逐场比较 `rank_change_logs.final_change` 并比较 `user_ranking.rank_score/rated_matches`，输出差异条数与样例。差异为 0 是 V2 线上结算正确的验收标准。
- 重放与线上共用：`internal/ranking` 纯计分函数、业务时区日界、赛制 N 的计算与"同对手/每日上限"口径。

### D7：赛季达标门槛只作用于 V2 赛季
- V2 赛季（开始日期不早于起点）的记录同时维护 `distinct_opponents` 与 `qualified`：投影写入赛季记录时，用 `idx_mpr_user_opponent_game_completed` 判断该对手是否本季首次交手；累加后单独一条 `UPDATE` 按 `matches_played ≥ 5 AND distinct_opponents ≥ 3` 刷新 `qualified`（避免 MySQL 与 SQLite 对 upsert 赋值顺序的语义差异）。
- 榜单、`my_rank` 与最终名次统一排序 `qualified DESC, end_rank_score DESC, wins DESC, id DESC`，新索引 `idx_season_game_qualified_rank (season_id, game_type, qualified, end_rank_score, wins, id)` 替换 `idx_season_game_rank`，整体反向扫描仍无 filesort；旧赛季所有记录 `qualified = 0`，排序与原来一致。
- 达标者名次 = 达标组内排在其前的人数 + 1；未达标者 `rank = 0`、`my_rank = 0`。换季时 V2 赛季只为达标者写 `final_rank`，未达标者为 0、不发称号；旧赛季保持全员排名。
- 门槛写在 `SeasonInfo.qualify_min_matches/qualify_min_opponents`（旧赛季为 0），客户端据此展示进度，不硬编码。
- 门槛为代码常量而非配置：`qualified` 是持久化列，配置变化会让存量标记失真；调整门槛需要迁移重算。

### D8：读取按规则版本组织
- 段位信息：V2 生效后 `placement_remaining = max(0, 10 − rated_matches)`；没有段位行的用户按 1250 合成（纯读，不建行）。
- 全站榜：V2 生效后只收 `rating_version = 2 AND rated_matches > 0` 的行，榜单、计数与本人名次同一条件；未上榜本人的段位按 1250 展示。
- 段位趋势只取起点之后的 `match_result` 日志；强弱对手分析在 V2 后只读 `rating_stronger/even/weaker` 三个相对分档（分差阈值 ±250），旧的绝对分档保留但不再展示。
- `GetRankConfigsResp` 增加 `rules_version`（当前生效版本）与 `rules_effective_at`（V2 起点，未配置为空），并计入配置版本号；`MatchDetailData.rank_rules_version` 表示该场按哪套规则结算（0 未计分）。
- `rank_reset_ratio` 迁移为 1（不做软重置，与规格一致）。

## Risks / Trade-offs

- [切换后全员分数变化，部分旧"王者"会明显下降] → 以 `rating_reset` 日志留痕、段位说明页提前展示生效日期与新规则；上线前用 `-mode=seed` 的分布输出评估影响并对外说明。
- [会员排位权益停止] → 属于付费权益变更，需在切换前告知会员；VIP 配置保留，便于回滚到旧规则。
- [阈值是基于假设分布校准的] → 种子计算输出各段位占比；若分布明显偏离，再单独调整 `rank_config` 与 `CalculateLevel`（两处必须同步）。
- [种子不含计算之后、起点之前的对局] → 在旧赛季最后一晚执行；这些对局仍完整计入旧赛季。
- [V2 赛季早期达标人数少，榜单上方空] → 未达标者仍列在榜单后部并显示进度，赛季页给出门槛说明。
- [同对手衰减与每日上限只缩正向，零和被轻微打破] → 只在刷分场景触发，可接受；审计重放会覆盖这些分支。
- [`my_rank` 与 30 秒缓存的首屏榜单短暂不一致] → 沿用既有取舍。
- [切换 Worker 与就地切换并发] → 行锁、条件更新与日志唯一键保证幂等；Worker 用 `idx_user_ranking_rating_version` 做稳定状态快路径。

## Migration Plan

1. 执行迁移：`user_ranking` 增加 `rating_version`、`rated_matches` 与索引；新建 `rank_rating_seeds`；`season_records` 增加 `distinct_opponents`、`qualified`，新建 `idx_season_game_qualified_rank` 后删除 `idx_season_game_rank`；`seasons.rank_reset_ratio` 默认值与存量改为 1。均以 `information_schema` 判断，可重复执行。
2. 发布后端（两项配置都留空，行为与现在一致）：生产用 `backend/deploy-prod.sh`，开发用 `backend/deploy-dev.sh`。`deploy-prod.sh` 只替换二进制与迁移，线上 `.env` 不会被改动。
3. 切换前准备（须在切换时刻 T 之前完成）：在服务器 `.env` 写入 `RANKING_RATING_V2_START_AT=T`（T 为当前时间之后数分钟的 RFC3339 时刻），重启服务（仍按旧规则结算）；执行 `rank-rebuild -mode=seed` 预览分布，确认后加 `-apply` 写入 `rank_rating_seeds`。种子 cutoff 为执行时刻，因此 cutoff 到 T 之间不应有排位完成；发布后用 SQL 核对该区间内无已完成的排位赛，否则需在 T 之前重新计算种子。
4. 切换：到达 T 后，后台任务在 1 分钟内把所有段位行切换为种子分（无种子的为 1250），每个段位行写一条 `rating_reset` 记录；T 之后的结算按 V2。客户端段位说明页依据 `rules_effective_at` 显示生效提示。
5. 验证：`rank-rebuild -mode=audit` 差异为 0；抽查战报明细、赛季页达标进度与全站榜；客户端升级后核对段位说明与会员中心文案。
6. 回滚：清空 `RANKING_RATING_V2_START_AT` 并重启即可恢复旧规则结算；已切换的段位行保留 V2 版本号与种子分，旧规则下将以种子分继续结算，`rating_reset` 记录的 `before_score` 可用于人工还原。客户端可独立回退，新增字段对旧客户端透明。

## Open Questions

- 已确认（2026-10-19）：未上线，排位规则 V2 与会员高光不计分立即生效，不补偿。切换时刻 T 由发布负责人在执行前确定。
- 段位阈值是否在首个 V2 赛季结束后按实际分布重新校准。
