## Why

现行排位分是"胜 +B、负 −B/2"的正和计分：只看完成局数、不看对手强弱，保本胜率只有 1/3，段位实际衡量的是打了多少场而不是打得多好；赛季又不重置，赛季榜等于"全时榜里本月打过一场的人"，赛季称号没有参赛门槛，会员特殊战绩还能直接买到排位分。闭环工程（结算、榜单、换季、回顾）已经扎实，但它运送的段位与名次缺乏含金量，需要在下一个赛季边界整体换规则。

## What Changes

- **BREAKING** 新增排位规则 V2：排位分改为按对手实力计算期望的等级分——单局胜率由双方分差决定，按赛制换算整场胜率；赢强者多得、赢弱者少得，输给强者少扣；分值随赛制长度（抢 N）按 √N 增长，不再随"实际打了几局"增长。前 10 场为定级赛（涨跌加倍），同一对手当日衰减改为双方对称，保留每日上涨上限与 0 分保护。
- **BREAKING** 排位分只由胜负与对手实力决定：V2 生效后会员与普通用户的特殊战绩都不再计入排位分，高光仅记录。
- V2 在配置的切换时刻整体生效（`Ranking.RatingV2StartAt` 即时生效；或 `Ranking.RatingV2StartDate` 在赛季起点生效）：之前的对局与赛季按旧规则保持不变；生效时每位用户的排位分一次性切换为"用 V2 规则重算其全部历史排位赛"得到的初始分（未重算的用户取 1250），并写入可审计的切换记录。不配置则维持旧规则。
- 自 V2 赛季起，赛季名次与称号增加达标门槛：本季该球种至少 5 场排位且对手不少于 3 人。达标者按排位分排名并参与称号，未达标者在榜单中列于达标者之后、不占名次。
- 强弱对手分析在 V2 后改为相对口径（强于我 / 实力相当 / 弱于我），段位趋势只展示 V2 规则下的变化。
- 替换 `cmd/rank_rebuild`：删除会清空并按另一套公式改写全站段位的重放；改为"计算 V2 初始分（种子）"与"审计 V2 线上结算与重放是否一致"两种模式，二者与线上结算共用同一纯函数。
- `rank_reset_ratio` 改为如实表达"不做赛季软重置"（取值 1）。
- 客户端：段位说明页按规则版本展示（同时修正旧规则文案中并不存在的"失败减免"）、赛季页展示达标进度与本季净变化并按服务端名次渲染榜单、战报与会员中心不再宣传会员高光加排位分；管理后台的 VIP 排位权益配置页标注 V2 后停用的项。

## Capabilities

### New Capabilities

- `ranked-rating`: 排位规则 V2 的计分、定级赛、防刷、规则切换、初始分重算、审计工具与对外展示契约。
- `season-qualification`: V2 赛季的达标门槛、赛季榜排序、最终名次与称号资格。

### Modified Capabilities

- `continuous-season-lifecycle`: "连续换季保护永久资产并保持段位连续"增加例外——V2 生效时允许一次已声明的规则切换；`rank_reset_ratio` 表达不做软重置。
- `competitive-read-models`: 对手强度分档在 V2 后改为结算时双方分差的相对口径；赛季记录增量维护不同对手数与达标状态。
- `read-api-composition`: 赛季 overview 的本人名次只在达标后计算，排序增加达标优先。
- `season-archive`: 进行中赛季展示达标进度与本季净变化，历届赛季标注未达标。

## Impact

- 契约：`backend/chasing_points.api` 新增字段（`RankInfo.placement_remaining`、`GetRankConfigsResp.rules_version/rules_effective_at`、`MatchDetailData.rank_rules_version`、`SeasonInfo.qualify_min_*`、`SeasonRecordInfo.qualified/distinct_opponents`、`SeasonLeaderboardItem.qualified`），均为加法；修改后立即运行 goctl。
- 后端：新增纯计分包 `internal/ranking`；`finish_match_logic` 按规则版本分支；新增规则切换 Worker；竞技投影维护赛季不同对手数与达标状态；赛季结算只为达标者排名；段位、排行榜、赛季、统计读取按规则版本调整；`cmd/rank_rebuild` 重写。
- 数据库：`user_ranking` 增加 `rating_version`、`rated_matches`；新增 `rank_rating_seeds`；`season_records` 增加 `distinct_opponents`、`qualified` 并以 `(season_id, game_type, qualified, end_rank_score, wins, id)` 索引替换 `idx_season_game_rank`；`seasons.rank_reset_ratio` 默认值与存量改为 1。不指定字符集。
- UniApp：段位说明、赛季页、历届赛季、战报、会员中心及对应测试。原生 iOS 只按通用 label/value 渲染排位明细，新增字段不影响解码。
- 管理后台：VIP 排位权益配置页增加 V2 停用说明。
- 运营：上线前需配置生效日期、在旧赛季最后一天执行种子计算；切换后会员排位权益停止，需要对外说明。
