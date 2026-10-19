package logic

import (
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
)

// qualificationTestConfig：赛季按自然月从 2026-08 开始，排位规则 V2 从 2026-11-01（S4）生效。
func qualificationTestConfig() config.Config {
	return config.Config{
		CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"},
		SeasonLifecycle: config.SeasonLifecycleConfig{
			Enabled: true, AnchorDate: "2026-08-01", InitialNumber: 1, CycleMonths: 1, Timezone: "Asia/Shanghai",
		},
		Ranking: config.RankingConfig{RatingV2StartDate: "2026-11-01"},
	}
}

func v2ProjectionInput(match *model.Match, player1Score, player2Score int) CompetitiveProjectionInput {
	return CompetitiveProjectionInput{
		Match:         match,
		Player1Before: &model.UserRanking{UserId: match.UserId, GameType: match.GameType, RankScore: player1Score},
		Player1After:  &model.UserRanking{UserId: match.UserId, GameType: match.GameType, RankScore: player1Score + 20},
		Player2Before: &model.UserRanking{UserId: *match.OpponentId, GameType: match.GameType, RankScore: player2Score},
		Player2After:  &model.UserRanking{UserId: *match.OpponentId, GameType: match.GameType, RankScore: player2Score - 20},
	}
}

func TestProjectorMaintainsSeasonQualificationOnlyForV2Seasons(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	ctx.Config = qualificationTestConfig()
	users := []model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}, {Id: 3, Nickname: "three"}, {Id: 4, Nickname: "four"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.Season{
		{Id: 3, Name: "S3", StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), Status: 2},
		{Id: 4, Name: "S4", StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC), Status: 1},
	}).Error; err != nil {
		t.Fatalf("seed seasons: %v", err)
	}
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	projector := NewCompetitiveProjector(ctx)
	project := func(match *model.Match, player1Score, player2Score int) {
		t.Helper()
		if _, err := projector.ProjectWithTx(db, v2ProjectionInput(match, player1Score, player2Score)); err != nil {
			t.Fatalf("project match %d: %v", match.Id, err)
		}
	}

	// 旧赛季（S3）：不维护不同对手数，也不标记达标。
	project(completedRankedMatch(10, 1, 2, 3, 1, time.Date(2026, 10, 20, 20, 0, 0, 0, shanghai)), 1500, 1100)
	// V2 赛季（S4）：1 号与 2、3、4 号交手，再与 2 号打两场——5 场、3 位对手刚好达标；2 号只遇到 1 号。
	opponents := []int64{2, 3, 4, 2, 2}
	for index, opponent := range opponents {
		completedAt := time.Date(2026, 11, 2+index, 20, 0, 0, 0, shanghai)
		project(completedRankedMatch(int64(100+index), 1, opponent, 3, 1, completedAt), 1500, 1100)
	}
	// 重复投影同一场比赛不得重复累加。
	project(completedRankedMatch(104, 1, 2, 3, 1, time.Date(2026, 11, 6, 20, 0, 0, 0, shanghai)), 1500, 1100)

	var legacy, qualified, opponent model.SeasonRecord
	db.Where("season_id = 3 AND user_id = 1 AND game_type = 3").First(&legacy)
	db.Where("season_id = 4 AND user_id = 1 AND game_type = 3").First(&qualified)
	db.Where("season_id = 4 AND user_id = 2 AND game_type = 3").First(&opponent)
	if legacy.DistinctOpponents != 0 || legacy.Qualified != 0 || legacy.MatchesPlayed != 1 {
		t.Fatalf("legacy seasons must not track qualification: %+v", legacy)
	}
	if qualified.MatchesPlayed != 5 || qualified.DistinctOpponents != 3 || qualified.Qualified != 1 {
		t.Fatalf("five matches against three opponents must qualify exactly once: %+v", qualified)
	}
	if opponent.MatchesPlayed != 3 || opponent.DistinctOpponents != 1 || opponent.Qualified != 0 {
		t.Fatalf("repeated opponent must not count as new: %+v", opponent)
	}

	var buckets []model.UserOpponentStrengthBucket
	if err := db.Where("user_id IN ? AND game_type = 3", []int64{1, 2}).Order("user_id ASC, rank_bucket ASC").Find(&buckets).Error; err != nil {
		t.Fatalf("load strength buckets: %v", err)
	}
	seen := map[string]bool{}
	for _, bucket := range buckets {
		seen[bucket.RankBucket] = true
	}
	if !seen["score_2001_plus"] && !seen["score_1001_2000"] {
		t.Fatalf("legacy match must keep absolute buckets: %+v", buckets)
	}
	if !seen[ranking.StrengthBucketWeaker] || !seen[ranking.StrengthBucketStronger] {
		t.Fatalf("V2 matches must use relative buckets (1500 vs 1100): %+v", buckets)
	}
}
