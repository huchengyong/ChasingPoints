// rank_rebuild 是排位规则 V2 的离线工具，与线上结算共用 internal/ranking 纯函数：
//
//	-mode=seed  （默认，只预览）在 V2 起点之前用 V2 规则重放全部历史排位，输出初始分分布；加 -apply 整表写入 rank_rating_seeds。
//	-mode=audit （只读）在 V2 生效后从 rating_reset 记录恢复起点，重放起点之后的排位并与线上记录、当前分数逐项比较。
//
// 工具不会改写任何对局记录、段位日志、赛季记录或称号。
package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"gorm.io/gorm"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	mode       = flag.String("mode", "seed", "seed: 计算 V2 初始分；audit: 审计 V2 线上结算")
	apply      = flag.Bool("apply", false, "seed 模式下确认写入 rank_rating_seeds")
	samples    = flag.Int("samples", 10, "输出的样例条数")
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	svcCtx := svc.NewServiceContext(c)
	rules := ranking.RulesFor(c)
	if err := rules.Err(); err != nil {
		log.Fatalf("排位规则 V2 配置无效: %v", err)
	}
	if !rules.V2Configured() {
		log.Fatal("未配置 Ranking.RatingV2StartDate，无法计算种子或审计")
	}

	switch *mode {
	case "seed":
		runSeed(svcCtx, rules)
	case "audit":
		runAudit(svcCtx, rules)
	default:
		log.Fatalf("未知模式 %q，可选 seed / audit", *mode)
	}
}

func runSeed(svcCtx *svc.ServiceContext, rules ranking.Rules) {
	cutoff := time.Now()
	if !cutoff.Before(rules.V2StartAt()) {
		log.Fatalf("排位规则 V2 已于 %s 生效，切换可能已经开始，拒绝再计算种子", rules.V2StartAt().Format(time.RFC3339))
	}
	if missing, err := svcCtx.MatchModel.CountCompletedWithoutCompletedAt(); err != nil {
		log.Fatalf("检查 completed_at 失败: %v", err)
	} else if missing > 0 {
		log.Fatalf("有 %d 场已完成比赛缺少 completed_at，请先执行竞技读模型回建补齐后再计算种子", missing)
	}

	dailyCap := logic.RatingV2DailyPositiveCap(svcCtx)
	replay := logic.NewRatingReplay(rules, dailyCap, nil)
	if err := logic.ReplayCompletedMatches(svcCtx, replay, time.Time{}, cutoff, nil); err != nil {
		log.Fatalf("重放历史排位失败: %v", err)
	}
	players := replay.Players()

	current := map[logic.RatingReplayKey]int{}
	legacyLevels := map[int]int{}
	if err := eachRanking(svcCtx, func(row model.UserRanking) {
		current[logic.RatingReplayKey{UserId: row.UserId, GameType: row.GameType}] = row.RankScore
		if row.TotalWins+row.TotalLosses > 0 {
			legacyLevels[model.RankLevelForScore(row.RankScore)]++
		}
	}); err != nil {
		log.Fatalf("读取当前段位失败: %v", err)
	}

	seedLevels := map[int]int{}
	type mover struct {
		key            logic.RatingReplayKey
		before, after  int
		matches, delta int
	}
	movers := make([]mover, 0, len(players))
	seeds := make([]model.RankRatingSeed, 0, len(players))
	for key, player := range players {
		seedLevels[model.RankLevelForScore(player.Rating)]++
		before := current[key]
		movers = append(movers, mover{key: key, before: before, after: player.Rating, matches: player.RatedMatches, delta: player.Rating - before})
		seeds = append(seeds, model.RankRatingSeed{UserId: key.UserId, GameType: key.GameType, SeedScore: player.Rating, SeedMatches: player.RatedMatches, CutoffAt: cutoff})
	}
	sort.Slice(movers, func(i, j int) bool { return abs(movers[i].delta) > abs(movers[j].delta) })

	fmt.Printf("rating v2 seed: v2_start=%s cutoff=%s daily_cap=%d replayed_matches=%d skipped=%d seeded_rankings=%d\n",
		rules.V2StartAt().Format(time.RFC3339), cutoff.Format(time.RFC3339), dailyCap, replay.Applied, replay.Skipped, len(seeds))
	fmt.Println("段位分布（旧规则有对局的段位行 → V2 初始分）：")
	for level := 1; level <= 6; level++ {
		fmt.Printf("  level %d: legacy=%d seed=%d\n", level, legacyLevels[level], seedLevels[level])
	}
	fmt.Println("变化最大的段位行（旧分 → 初始分，重放场次）：")
	for index := 0; index < len(movers) && index < *samples; index++ {
		item := movers[index]
		fmt.Printf("  user=%d game=%d %d → %d (%+d) matches=%d\n", item.key.UserId, item.key.GameType, item.before, item.after, item.delta, item.matches)
	}

	if !*apply {
		fmt.Println("预览完成，数据库未写入；确认无误后加 -apply 写入 rank_rating_seeds。")
		return
	}
	if err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		return svcCtx.RankingModel.ReplaceRatingSeedsWithTx(tx, seeds)
	}); err != nil {
		log.Fatalf("写入初始分失败: %v", err)
	}
	fmt.Printf("已写入 %d 条初始分。\n", len(seeds))
}

func runAudit(svcCtx *svc.ServiceContext, rules ranking.Rules) {
	upper := time.Now()
	if !rules.V2ActiveAt(upper) {
		log.Fatalf("排位规则 V2 尚未生效（%s），没有可审计的数据", rules.V2StartAt().Format(time.RFC3339))
	}
	resets, err := svcCtx.RankingModel.ListRatingResetLogsWithTx(nil)
	if err != nil {
		log.Fatalf("读取规则切换记录失败: %v", err)
	}
	start := make(map[logic.RatingReplayKey]logic.RatingReplayPlayer, len(resets))
	for _, reset := range resets {
		seedMatches := 0
		if remark, ok := ranking.ParseRemark(reset.Remark); ok {
			seedMatches = remark.SeedMatches
		}
		start[logic.RatingReplayKey{UserId: reset.UserId, GameType: reset.GameType}] = logic.RatingReplayPlayer{Rating: reset.AfterScore, RatedMatches: seedMatches}
	}

	replay := logic.NewRatingReplay(rules, logic.RatingV2DailyPositiveCap(svcCtx), start)
	compared, logDiffs := 0, 0
	var batch []*logic.RatingReplayMatchResult
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(batch))
		for _, result := range batch {
			ids = append(ids, result.MatchId)
		}
		logs, err := svcCtx.RankingModel.ListMatchRankChangesByMatchIdsWithTx(nil, ids)
		if err != nil {
			return err
		}
		byKey := make(map[[2]int64]model.RankChangeLog, len(logs))
		for _, item := range logs {
			byKey[[2]int64{item.MatchId, item.UserId}] = item
		}
		for _, result := range batch {
			for _, side := range []struct {
				userId  int64
				outcome ranking.Outcome
			}{{result.Player1Id, result.Player1}, {result.Player2Id, result.Player2}} {
				compared++
				actual, ok := byKey[[2]int64{result.MatchId, side.userId}]
				if ok && actual.FinalChange == side.outcome.Change && actual.AfterScore == side.outcome.After {
					continue
				}
				logDiffs++
				if logDiffs <= *samples {
					fmt.Printf("  对局差异 match=%d user=%d expected=%+d→%d actual_found=%t actual=%+d→%d\n",
						result.MatchId, side.userId, side.outcome.Change, side.outcome.After, ok, actual.FinalChange, actual.AfterScore)
				}
			}
		}
		batch = batch[:0]
		return nil
	}
	if err := logic.ReplayCompletedMatches(svcCtx, replay, rules.V2StartAt(), upper, func(result *logic.RatingReplayMatchResult) error {
		batch = append(batch, result)
		if len(batch) >= logic.RatingReplayBatchSize {
			return flush()
		}
		return nil
	}); err != nil {
		log.Fatalf("重放 V2 排位失败: %v", err)
	}
	if err := flush(); err != nil {
		log.Fatalf("比较 V2 排位记录失败: %v", err)
	}

	players := replay.Players()
	rows, rowDiffs := 0, 0
	if err := eachRanking(svcCtx, func(row model.UserRanking) {
		// 审计开始后才更新的行可能包含尚未重放的对局，跳过避免误报。
		if row.RatingVersion < model.RatingVersionV2 || !row.UpdatedAt.Before(upper) {
			return
		}
		rows++
		expected, ok := players[logic.RatingReplayKey{UserId: row.UserId, GameType: row.GameType}]
		if !ok {
			expected = logic.RatingReplayPlayer{Rating: ranking.InitialRating}
		}
		if row.RankScore == expected.Rating && row.RatedMatches == expected.RatedMatches {
			return
		}
		rowDiffs++
		if rowDiffs <= *samples {
			fmt.Printf("  段位差异 user=%d game=%d expected=%d/%d actual=%d/%d\n", row.UserId, row.GameType, expected.Rating, expected.RatedMatches, row.RankScore, row.RatedMatches)
		}
	}); err != nil {
		log.Fatalf("读取当前段位失败: %v", err)
	}

	fmt.Printf("rating v2 audit: v2_start=%s upper=%s resets=%d replayed_matches=%d compared_logs=%d log_differences=%d compared_rankings=%d ranking_differences=%d differences=%d\n",
		rules.V2StartAt().Format(time.RFC3339), upper.Format(time.RFC3339), len(resets), replay.Applied, compared, logDiffs, rows, rowDiffs, logDiffs+rowDiffs)
}

func eachRanking(svcCtx *svc.ServiceContext, visit func(model.UserRanking)) error {
	afterID := int64(0)
	for {
		rows, err := svcCtx.RankingModel.ListRankingsPageWithTx(nil, afterID, 500)
		if err != nil {
			return err
		}
		for _, row := range rows {
			visit(row)
		}
		if len(rows) < 500 {
			return nil
		}
		afterID = rows[len(rows)-1].Id
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
