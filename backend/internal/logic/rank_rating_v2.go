package logic

import (
	"errors"
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

// RatingV2MatchFacts 是一场按排位规则 V2 结算的对局事实。
type RatingV2MatchFacts struct {
	MatchId     int64
	GameType    int
	Player1Id   int64
	Player2Id   int64
	Player1Won  bool
	RaceLength  int
	CompletedAt time.Time
}

// RatingV2Settlement 是 V2 结算写库后的双方快照，供竞技投影与响应使用。
type RatingV2Settlement struct {
	Player1Before  *model.UserRanking
	Player1After   *model.UserRanking
	Player2Before  *model.UserRanking
	Player2After   *model.UserRanking
	Player1Outcome ranking.Outcome
	Player2Outcome ranking.Outcome
}

// RatingV2RaceLength 按对局赛制与已完成局计算 V2 的赛制长度 N。
func RatingV2RaceLength(match *model.Match, rounds []model.MatchRound) int {
	if match == nil {
		return 1
	}
	targetWins := 0
	switch {
	case match.GameType == 1:
		if format, target, ok := model.NormalizeSnookerFormat(match.SnookerFormat, match.SnookerTargetWins, match.BestOfFrames); ok && format == model.SnookerFormatRaceTo {
			targetWins = target
		}
	case model.IsPoolMatchFormatGameType(match.GameType):
		if format, target, ok := model.NormalizePoolMatchFormat(match.GameType, match.MatchFormat, match.TargetWins); ok && format == model.MatchFormatRaceTo {
			targetWins = target
		}
	}
	player1Racks, player2Racks := 0, 0
	for _, round := range rounds {
		if round.Winner == nil {
			continue
		}
		switch *round.Winner {
		case 1:
			player1Racks++
		case 2:
			player2Racks++
		}
	}
	return ranking.RaceLength(targetWins, player1Racks, player2Racks)
}

// SettleRatingV2WithTx 在调用方事务内按 V2 结算一场双方均为注册用户的有效排位：
// 按用户 ID 顺序锁定段位行、必要时就地切换到 V2、计算并写入快照与明细。
func SettleRatingV2WithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, facts RatingV2MatchFacts) (RatingV2Settlement, error) {
	var out RatingV2Settlement
	if svcCtx == nil || svcCtx.RankingModel == nil || svcCtx.MatchModel == nil {
		return out, errors.New("rating v2 settlement infrastructure is unavailable")
	}
	if facts.Player1Id <= 0 || facts.Player2Id <= 0 || facts.Player1Id == facts.Player2Id {
		return out, errors.New("rating v2 settlement requires two registered players")
	}
	rules := ranking.RulesFor(svcCtx.Config)
	rows, err := lockRatingV2RowsWithTx(tx, svcCtx, rules, facts.GameType, facts.Player1Id, facts.Player2Id)
	if err != nil {
		return out, err
	}
	player1, player2 := rows[facts.Player1Id], rows[facts.Player2Id]

	dayStart, dayEnd := rules.DayRange(facts.CompletedAt)
	prior, err := svcCtx.MatchModel.CountCompletedMatchesBetweenUsersByGameTypeBetween(tx, facts.Player1Id, facts.Player2Id, facts.GameType, dayStart, dayEnd, facts.MatchId)
	if err != nil {
		return out, err
	}
	gain1, err := svcCtx.RankingModel.SumPositiveRankChangesByUserAndGameTypeBetween(tx, facts.Player1Id, facts.GameType, dayStart, dayEnd)
	if err != nil {
		return out, err
	}
	gain2, err := svcCtx.RankingModel.SumPositiveRankChangesByUserAndGameTypeBetween(tx, facts.Player2Id, facts.GameType, dayStart, dayEnd)
	if err != nil {
		return out, err
	}

	outcome1, outcome2 := ranking.Settle(ranking.Input{
		Player1:               ranking.PlayerState{Rating: player1.RankScore, RatedMatches: player1.RatedMatches, TodayPositiveGain: gain1},
		Player2:               ranking.PlayerState{Rating: player2.RankScore, RatedMatches: player2.RatedMatches, TodayPositiveGain: gain2},
		Player1Won:            facts.Player1Won,
		RaceLength:            facts.RaceLength,
		PriorPairMatchesToday: int(prior),
		DailyPositiveCap:      ratingV2DailyPositiveCap(svcCtx),
	})

	out.Player1Before, out.Player1After = applyRatingV2Outcome(player1, outcome1, facts.Player1Won)
	out.Player2Before, out.Player2After = applyRatingV2Outcome(player2, outcome2, !facts.Player1Won)
	out.Player1Outcome, out.Player2Outcome = outcome1, outcome2
	for _, row := range []*model.UserRanking{player1, player2} {
		if err := svcCtx.RankingModel.UpdateRankingSnapshot(tx, row); err != nil {
			return out, err
		}
	}
	logs := []model.RankChangeLog{
		buildRatingV2ChangeLog(facts, facts.Player1Id, facts.Player1Won, out.Player1Before, out.Player1After, outcome1),
		buildRatingV2ChangeLog(facts, facts.Player2Id, !facts.Player1Won, out.Player2Before, out.Player2After, outcome2),
	}
	if err := svcCtx.RankingModel.CreateRankChangeLogs(tx, logs); err != nil {
		return out, err
	}
	return out, nil
}

// lockRatingV2RowsWithTx 按用户 ID 升序加锁读取（缺失则按 V2 初始值创建）双方段位行，并把仍是旧规则的行切换到 V2。
func lockRatingV2RowsWithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, rules ranking.Rules, gameType int, userIds ...int64) (map[int64]*model.UserRanking, error) {
	ordered := append([]int64(nil), userIds...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	defaults := rules.DefaultsAt(rules.V2StartAt())
	rows := make(map[int64]*model.UserRanking, len(ordered))
	pending := make([]model.UserRanking, 0, len(ordered))
	for _, userId := range ordered {
		row, err := svcCtx.RankingModel.FindOrCreateForUpdateWithTx(tx, userId, gameType, defaults)
		if err != nil {
			return nil, err
		}
		rows[userId] = row
		if row.RatingVersion < model.RatingVersionV2 {
			pending = append(pending, *row)
		}
	}
	if len(pending) == 0 {
		return rows, nil
	}
	seeds, err := svcCtx.RankingModel.FindRatingSeedsWithTx(tx, pending)
	if err != nil {
		return nil, err
	}
	for _, userId := range ordered {
		row := rows[userId]
		if row.RatingVersion >= model.RatingVersionV2 {
			continue
		}
		if _, err := SwitchRankingToRatingV2WithTx(tx, svcCtx, rules, row, seeds); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// SwitchRankingToRatingV2WithTx 把已加锁、仍是旧规则的段位行切换到 V2：分数取种子（缺失为初始分），
// 已计分场次取种子场次，并按唯一键幂等写入 rating_reset 记录。行已被切换时返回 false 且不改动内存中的行。
func SwitchRankingToRatingV2WithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, rules ranking.Rules, row *model.UserRanking, seeds map[[2]int64]model.RankRatingSeed) (bool, error) {
	if row == nil || row.RatingVersion >= model.RatingVersionV2 {
		return false, nil
	}
	score, seedMatches := ranking.InitialRating, 0
	if seed, ok := seeds[[2]int64{row.UserId, int64(row.GameType)}]; ok {
		score, seedMatches = max(seed.SeedScore, 0), max(seed.SeedMatches, 0)
	}
	level := model.RankLevelForScore(score)
	switched, err := svcCtx.RankingModel.SwitchRatingVersionWithTx(tx, row.Id, model.RatingVersionV2, score, level, seedMatches)
	if err != nil || !switched {
		return false, err
	}
	if _, err := svcCtx.RankingModel.CreateRankChangeLogsIfAbsent(tx, []model.RankChangeLog{{
		UserId:      row.UserId,
		MatchId:     0,
		ChangeType:  model.RankChangeTypeRatingReset,
		GameType:    row.GameType,
		Result:      "reset",
		FinalChange: score - row.RankScore,
		BeforeScore: row.RankScore,
		AfterScore:  score,
		BeforeLevel: max(row.RankLevel, 1),
		AfterLevel:  level,
		Remark:      ranking.ResetRemark(seedMatches),
		EffectiveAt: rules.V2StartAt(),
	}}); err != nil {
		return false, err
	}
	row.RankScore, row.RankLevel = score, level
	row.RatingVersion, row.RatedMatches = model.RatingVersionV2, seedMatches
	return true, nil
}

func applyRatingV2Outcome(row *model.UserRanking, outcome ranking.Outcome, won bool) (*model.UserRanking, *model.UserRanking) {
	before := &model.UserRanking{UserId: row.UserId, GameType: row.GameType, RankScore: row.RankScore, RankLevel: model.RankLevelForScore(row.RankScore)}
	row.RankScore = outcome.After
	row.RankLevel = model.RankLevelForScore(outcome.After)
	if outcome.Rated {
		row.RatedMatches++
		if won {
			row.TotalWins++
			if row.CurrentStreak >= 0 {
				row.CurrentStreak++
			} else {
				row.CurrentStreak = 1
			}
			row.MaxStreak = max(row.MaxStreak, row.CurrentStreak)
		} else {
			row.TotalLosses++
			if row.CurrentStreak <= 0 {
				row.CurrentStreak--
			} else {
				row.CurrentStreak = -1
			}
		}
	}
	after := &model.UserRanking{UserId: row.UserId, GameType: row.GameType, RankScore: row.RankScore, RankLevel: row.RankLevel}
	return before, after
}

func buildRatingV2ChangeLog(facts RatingV2MatchFacts, userId int64, won bool, before, after *model.UserRanking, outcome ranking.Outcome) model.RankChangeLog {
	return model.RankChangeLog{
		UserId:      userId,
		MatchId:     facts.MatchId,
		ChangeType:  model.RankChangeTypeMatchResult,
		GameType:    facts.GameType,
		Result:      resultLabel(won, false),
		BaseScore:   outcome.Standard,
		FinalChange: outcome.Change,
		BeforeScore: before.RankScore,
		AfterScore:  after.RankScore,
		BeforeLevel: before.RankLevel,
		AfterLevel:  after.RankLevel,
		Remark:      ranking.RemarkFor(outcome),
		EffectiveAt: facts.CompletedAt,
	}
}

// ratingV2DailyPositiveCap 读取每日上涨上限；V2 只沿用会员权益配置中的这一项。
func ratingV2DailyPositiveCap(svcCtx *svc.ServiceContext) int {
	config, err := NewMemberRightsConfigService(svcCtx).GetConfig()
	if err != nil || config.RankingRights.DailyPositiveCap <= 0 {
		return ranking.DefaultDailyPositiveCap
	}
	return config.RankingRights.DailyPositiveCap
}

// RatingV2SwitchBatchSize 是后台切换任务每批处理的段位行数。
const RatingV2SwitchBatchSize = 500

// SwitchPendingRankingsToRatingV2 分批把仍是旧规则的段位行切换到 V2，返回切换行数；V2 未生效时不做任何事。
func SwitchPendingRankingsToRatingV2(svcCtx *svc.ServiceContext, now time.Time) (int, error) {
	if svcCtx == nil || svcCtx.DB == nil || svcCtx.RankingModel == nil {
		return 0, errors.New("rating v2 switch infrastructure is unavailable")
	}
	rules := ranking.RulesFor(svcCtx.Config)
	if !rules.V2ActiveAt(now) {
		return 0, nil
	}
	total := 0
	for {
		switched, fetched := 0, 0
		err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			rows, err := svcCtx.RankingModel.ListRankingsBeforeRatingVersionForUpdateWithTx(tx, model.RatingVersionV2, RatingV2SwitchBatchSize)
			fetched = len(rows)
			if err != nil || fetched == 0 {
				return err
			}
			seeds, err := svcCtx.RankingModel.FindRatingSeedsWithTx(tx, rows)
			if err != nil {
				return err
			}
			for i := range rows {
				ok, err := SwitchRankingToRatingV2WithTx(tx, svcCtx, rules, &rows[i], seeds)
				if err != nil {
					return err
				}
				if ok {
					switched++
				}
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += switched
		if fetched < RatingV2SwitchBatchSize {
			return total, nil
		}
	}
}
