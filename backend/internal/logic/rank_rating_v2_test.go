package logic

import (
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ratingV2LogicTestConfig 把 V2 起点设为本月初（offsetMonths=0，已生效）或之后某个月初（未生效），赛季按自然月。
func ratingV2LogicTestConfig(offsetMonths int) config.Config {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(shanghai)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghai).AddDate(0, offsetMonths, 0)
	return config.Config{
		SeasonLifecycle: config.SeasonLifecycleConfig{
			Enabled:       true,
			AnchorDate:    start.AddDate(0, -6, 0).Format(time.DateOnly),
			InitialNumber: 1,
			CycleMonths:   1,
			Timezone:      "Asia/Shanghai",
		},
		Ranking: config.RankingConfig{RatingV2StartDate: start.Format(time.DateOnly)},
	}
}

func newRatingV2LogicTestSvc(t *testing.T, offsetMonths int) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Match{}, &model.MatchRound{}, &model.UserRanking{}, &model.RankChangeLog{}, &model.RankRatingSeed{}); err != nil {
		t.Fatalf("prepare rating v2 schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:           db,
		MatchModel:   model.NewMatchModel(db),
		RankingModel: model.NewRankingModel(db),
		Config:       ratingV2LogicTestConfig(offsetMonths),
	}
}

func TestSwitchPendingRankingsToRatingV2UsesSeedsOnceAcrossBatches(t *testing.T) {
	svcCtx := newRatingV2LogicTestSvc(t, 0)
	rows := []model.UserRanking{
		{UserId: 1, GameType: 3, RankScore: 2600, RankLevel: 6, TotalWins: 30, TotalLosses: 10},
		{UserId: 2, GameType: 3, RankScore: 300, RankLevel: 1},
		{UserId: 3, GameType: 1, RankScore: 1300, RankLevel: 3, RatingVersion: model.RatingVersionV2, RatedMatches: 4},
	}
	// 再补一批旧规则行，覆盖第二个批次。
	for userID := int64(100); userID < 100+RatingV2SwitchBatchSize; userID++ {
		rows = append(rows, model.UserRanking{UserId: userID, GameType: 4, RankScore: int(userID), RankLevel: 1})
	}
	if err := svcCtx.DB.CreateInBatches(&rows, 200).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	if err := svcCtx.DB.Create(&model.RankRatingSeed{UserId: 1, GameType: 3, SeedScore: 1400, SeedMatches: 12, CutoffAt: time.Now()}).Error; err != nil {
		t.Fatalf("seed rating seed: %v", err)
	}

	before := newRatingV2LogicTestSvc(t, 1)
	before.DB, before.MatchModel, before.RankingModel = svcCtx.DB, svcCtx.MatchModel, svcCtx.RankingModel
	if switched, err := SwitchPendingRankingsToRatingV2(before, time.Now()); err != nil || switched != 0 {
		t.Fatalf("switch must wait for the V2 start: switched=%d err=%v", switched, err)
	}

	switched, err := SwitchPendingRankingsToRatingV2(svcCtx, time.Now())
	if err != nil || switched != 2+RatingV2SwitchBatchSize {
		t.Fatalf("switch all legacy rows: switched=%d err=%v", switched, err)
	}
	var seeded, unseeded, untouched model.UserRanking
	svcCtx.DB.Where("user_id = 1 AND game_type = 3").First(&seeded)
	svcCtx.DB.Where("user_id = 2 AND game_type = 3").First(&unseeded)
	svcCtx.DB.Where("user_id = 3 AND game_type = 1").First(&untouched)
	if seeded.RankScore != 1400 || seeded.RatedMatches != 12 || seeded.RatingVersion != model.RatingVersionV2 || seeded.RankLevel != model.RankLevelForScore(1400) || seeded.TotalWins != 30 {
		t.Fatalf("seeded row must switch to its seed and keep lifetime counters: %+v", seeded)
	}
	if unseeded.RankScore != ranking.InitialRating || unseeded.RatedMatches != 0 || unseeded.RatingVersion != model.RatingVersionV2 {
		t.Fatalf("unseeded row must start from the initial rating: %+v", unseeded)
	}
	if untouched.RankScore != 1300 || untouched.RatedMatches != 4 {
		t.Fatalf("already switched rows must stay untouched: %+v", untouched)
	}

	again, err := SwitchPendingRankingsToRatingV2(svcCtx, time.Now())
	if err != nil || again != 0 {
		t.Fatalf("second switch must be a no-op: switched=%d err=%v", again, err)
	}
	var resets int64
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("change_type = ?", model.RankChangeTypeRatingReset).Count(&resets).Error; err != nil || resets != int64(2+RatingV2SwitchBatchSize) {
		t.Fatalf("each row must keep exactly one reset log: resets=%d err=%v", resets, err)
	}
	if pending, err := svcCtx.RankingModel.HasRankingsBeforeRatingVersion(nil, model.RatingVersionV2); err != nil || pending {
		t.Fatalf("no legacy rows must remain: pending=%v err=%v", pending, err)
	}
}

func TestRatingV2RaceLengthUsesFormatTargetOrLargerRackCount(t *testing.T) {
	rounds := func(winners ...int) []model.MatchRound {
		out := make([]model.MatchRound, 0, len(winners))
		for _, winner := range winners {
			w := winner
			out = append(out, model.MatchRound{Winner: &w})
		}
		return out
	}
	cases := []struct {
		name   string
		match  model.Match
		rounds []model.MatchRound
		want   int
	}{
		{name: "斯诺克三局两胜", match: model.Match{GameType: 1, BestOfFrames: 3}, rounds: rounds(1, 1), want: 2},
		{name: "斯诺克抢5认输", match: model.Match{GameType: 1, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: 5}, rounds: rounds(2, 1, 1), want: 5},
		{name: "斯诺克自由局", match: model.Match{GameType: 1, SnookerFormat: model.SnookerFormatFree}, rounds: rounds(1, 2, 2, 2), want: 3},
		{name: "中八抢7", match: model.Match{GameType: 3, MatchFormat: model.MatchFormatRaceTo, TargetWins: 7}, rounds: rounds(1, 1, 1, 1, 1, 1, 1), want: 7},
		{name: "中八自由局", match: model.Match{GameType: 3, MatchFormat: model.MatchFormatFree}, rounds: rounds(1, 1, 2, 1, 1, 2, 1, 2, 1), want: 6},
		{name: "九球追分按赢局", match: model.Match{GameType: 2}, rounds: rounds(2, 2, 1, 2), want: 3},
		{name: "长赛制封顶", match: model.Match{GameType: 4, MatchFormat: model.MatchFormatRaceTo, TargetWins: 11}, rounds: rounds(1), want: ranking.MaxRaceLength},
		{name: "无完成局", match: model.Match{GameType: 3}, rounds: nil, want: 1},
	}
	for _, tc := range cases {
		match := tc.match
		if got := RatingV2RaceLength(&match, tc.rounds); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

type ratingV2Step struct {
	player1, player2 int64
	gameType         int
	result           int // 1=player1 胜 2=player2 胜 3=平
	format           string
	target           int
	rounds           []int
	at               time.Duration // 相对 V2 起点
}

// TestRatingV2LiveSettlementMatchesReplay 逐场按线上流程结算，再用离线重放从 rating_reset 起点重算，两者必须完全一致。
func TestRatingV2LiveSettlementMatchesReplay(t *testing.T) {
	svcCtx := newRatingV2LogicTestSvc(t, 0)
	rules := ranking.RulesFor(svcCtx.Config)
	start := rules.V2StartAt()
	for _, row := range []model.UserRanking{
		{UserId: 1, GameType: 3, RankScore: 2600, RankLevel: 6},
		{UserId: 2, GameType: 3, RankScore: 900, RankLevel: 2},
		{UserId: 3, GameType: 3, RankScore: 50, RankLevel: 1},
		{UserId: 1, GameType: 1, RankScore: 700, RankLevel: 2},
	} {
		item := row
		if err := svcCtx.DB.Create(&item).Error; err != nil {
			t.Fatalf("seed legacy ranking: %v", err)
		}
	}
	for _, seed := range []model.RankRatingSeed{
		{UserId: 1, GameType: 3, SeedScore: 1500, SeedMatches: 20},
		{UserId: 2, GameType: 3, SeedScore: 900, SeedMatches: 4},
	} {
		item := seed
		item.CutoffAt = start
		if err := svcCtx.DB.Create(&item).Error; err != nil {
			t.Fatalf("seed rating seed: %v", err)
		}
	}

	day := 34 * time.Hour // 起点后第二天 10:00（业务时区）
	steps := []ratingV2Step{
		{player1: 1, player2: 2, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 5, rounds: []int{1, 2, 1, 1, 2, 1, 1}, at: day},
		{player1: 2, player2: 1, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 3, rounds: []int{1, 1, 2, 1}, at: day + time.Hour},
		{player1: 1, player2: 2, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 1, rounds: []int{1}, at: day + 2*time.Hour},
		{player1: 1, player2: 2, gameType: 3, result: 3, format: model.MatchFormatFree, rounds: []int{1, 2}, at: day + 3*time.Hour},
		{player1: 2, player2: 1, gameType: 3, result: 1, format: model.MatchFormatFree, rounds: []int{1, 1, 2, 1}, at: day + 4*time.Hour},
		{player1: 1, player2: 2, gameType: 3, result: 2, format: model.MatchFormatRaceTo, target: 5, rounds: []int{2, 2, 1, 2, 2, 2}, at: day + 5*time.Hour},
		{player1: 1, player2: 2, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 5, rounds: []int{1, 1, 1, 1, 1}, at: day + 6*time.Hour},
		// 定级中的 3 号连胜抢 9，触发每日封顶；对手 4、5 号是新用户。
		{player1: 3, player2: 4, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 9, rounds: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, at: day + 7*time.Hour},
		{player1: 5, player2: 3, gameType: 3, result: 2, format: model.MatchFormatRaceTo, target: 9, rounds: []int{2, 2, 2, 2, 2, 2, 2, 2, 2}, at: day + 8*time.Hour},
		{player1: 3, player2: 4, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 9, rounds: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, at: day + 9*time.Hour},
		{player1: 3, player2: 5, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 9, rounds: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, at: day + 10*time.Hour},
		{player1: 3, player2: 1, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 9, rounds: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, at: day + 11*time.Hour},
		{player1: 3, player2: 6, gameType: 3, result: 1, format: model.MatchFormatRaceTo, target: 9, rounds: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, at: day + 12*time.Hour},
		{player1: 7, player2: 3, gameType: 3, result: 2, format: model.MatchFormatRaceTo, target: 9, rounds: []int{2, 2, 2, 2, 2, 2, 2, 2, 2}, at: day + 13*time.Hour},
		// 第二天：同一对手计数重新开始；斯诺克是独立的球种。
		{player1: 1, player2: 2, gameType: 3, result: 2, format: model.MatchFormatRaceTo, target: 3, rounds: []int{2, 1, 2, 2}, at: day + 24*time.Hour},
		{player1: 1, player2: 3, gameType: 1, result: 1, format: "", target: 0, rounds: []int{1, 2, 1}, at: day + 25*time.Hour},
	}

	liveOutcomes := map[[2]int64]ranking.Outcome{}
	for index, step := range steps {
		matchID := int64(1000 + index)
		completedAt := start.Add(step.at)
		opponentID := step.player2
		result := step.result
		match := model.Match{
			Id: matchID, UserId: step.player1, OpponentId: &opponentID, OpponentName: "对手", GameType: step.gameType,
			MatchMode: model.MatchModeRanked, MatchFormat: step.format, TargetWins: step.target,
			Status: 2, Result: &result, MatchTime: completedAt.Add(-30 * time.Minute), EndTime: &completedAt, CompletedAt: &completedAt,
		}
		if step.gameType == 1 {
			match.MatchFormat, match.TargetWins, match.BestOfFrames = "", 0, 3
		}
		if err := svcCtx.DB.Create(&match).Error; err != nil {
			t.Fatalf("create match %d: %v", matchID, err)
		}
		rounds := make([]model.MatchRound, 0, len(step.rounds))
		for roundIndex, winner := range step.rounds {
			w := winner
			rounds = append(rounds, model.MatchRound{MatchId: matchID, RoundNo: roundIndex + 1, Winner: &w, WinType: "normal"})
		}
		if err := svcCtx.DB.Create(&rounds).Error; err != nil {
			t.Fatalf("create rounds for match %d: %v", matchID, err)
		}
		if result == 3 {
			continue // 线上平局不计排位分
		}
		err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			settlement, err := SettleRatingV2WithTx(tx, svcCtx, RatingV2MatchFacts{
				MatchId: matchID, GameType: step.gameType, Player1Id: step.player1, Player2Id: step.player2,
				Player1Won: result == 1, RaceLength: RatingV2RaceLength(&match, rounds), CompletedAt: completedAt,
			})
			if err == nil {
				liveOutcomes[[2]int64{matchID, step.player1}] = settlement.Player1Outcome
				liveOutcomes[[2]int64{matchID, step.player2}] = settlement.Player2Outcome
			}
			return err
		})
		if err != nil {
			t.Fatalf("settle match %d: %v", matchID, err)
		}
	}

	var sawDecay, sawUnrated, sawCap, sawPlacement bool
	for _, outcome := range liveOutcomes {
		sawDecay = sawDecay || outcome.SameOpponent != 0
		sawUnrated = sawUnrated || !outcome.Rated
		sawCap = sawCap || outcome.DailyCap != 0
		sawPlacement = sawPlacement || outcome.Placement != 0 || outcome.OpponentDamping != 0
	}
	if !sawDecay || !sawUnrated || !sawCap || !sawPlacement {
		t.Fatalf("scenario must cover decay=%v unrated=%v cap=%v placement=%v", sawDecay, sawUnrated, sawCap, sawPlacement)
	}

	resets, err := svcCtx.RankingModel.ListRatingResetLogsWithTx(nil)
	if err != nil {
		t.Fatalf("list reset logs: %v", err)
	}
	startState := map[RatingReplayKey]RatingReplayPlayer{}
	for _, reset := range resets {
		remark, _ := ranking.ParseRemark(reset.Remark)
		startState[RatingReplayKey{UserId: reset.UserId, GameType: reset.GameType}] = RatingReplayPlayer{Rating: reset.AfterScore, RatedMatches: remark.SeedMatches}
	}
	replay := NewRatingReplay(rules, RatingV2DailyPositiveCap(svcCtx), startState)
	replayed := 0
	err = ReplayCompletedMatches(svcCtx, replay, start, start.Add(30*24*time.Hour), func(result *RatingReplayMatchResult) error {
		replayed++
		for _, side := range []struct {
			userID  int64
			outcome ranking.Outcome
		}{{result.Player1Id, result.Player1}, {result.Player2Id, result.Player2}} {
			live, ok := liveOutcomes[[2]int64{result.MatchId, side.userID}]
			if !ok || live != side.outcome {
				return fmt.Errorf("match %d user %d: live=%+v replay=%+v", result.MatchId, side.userID, live, side.outcome)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("replay must match live settlement: %v", err)
	}
	if replayed != len(liveOutcomes)/2 {
		t.Fatalf("replay must cover every settled match: replayed=%d live=%d", replayed, len(liveOutcomes)/2)
	}
	for key, player := range replay.Players() {
		var row model.UserRanking
		if err := svcCtx.DB.Where("user_id = ? AND game_type = ?", key.UserId, key.GameType).First(&row).Error; err != nil {
			t.Fatalf("load ranking %+v: %v", key, err)
		}
		if row.RankScore != player.Rating || row.RatedMatches != player.RatedMatches {
			t.Fatalf("ranking %+v: live=%d/%d replay=%d/%d", key, row.RankScore, row.RatedMatches, player.Rating, player.RatedMatches)
		}
	}
}
