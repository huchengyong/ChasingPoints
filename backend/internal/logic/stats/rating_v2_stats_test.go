package stats

import (
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/types"
)

func statsV2Config(base config.Config, offsetMonths int) (config.Config, time.Time) {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(shanghai)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghai).AddDate(0, offsetMonths, 0)
	base.SeasonLifecycle = config.SeasonLifecycleConfig{Enabled: true, AnchorDate: start.AddDate(0, -6, 0).Format(time.DateOnly), InitialNumber: 1, CycleMonths: 1, Timezone: "Asia/Shanghai"}
	base.Ranking = config.RankingConfig{RatingV2StartDate: start.Format(time.DateOnly)}
	return base, start
}

func TestRankScoreTrendOnlyShowsV2ChangesAfterStart(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.RankChangeLog{}); err != nil {
		t.Fatalf("prepare rank log schema: %v", err)
	}
	cfg, start := statsV2Config(svcCtx.Config, 0)
	svcCtx.Config = cfg
	if err := svcCtx.DB.Create(&[]model.RankChangeLog{
		{UserId: 101, MatchId: 1, ChangeType: model.RankChangeTypeMatchResult, GameType: 3, Result: "win", FinalChange: 20, BeforeScore: 2580, AfterScore: 2600, Remark: "{}", EffectiveAt: start.Add(-time.Hour)},
		{UserId: 101, MatchId: 0, ChangeType: model.RankChangeTypeRatingReset, GameType: 3, Result: "reset", FinalChange: -1200, BeforeScore: 2600, AfterScore: 1400, Remark: "{}", EffectiveAt: start},
		{UserId: 101, MatchId: 2, ChangeType: model.RankChangeTypeMatchResult, GameType: 3, Result: "win", FinalChange: 32, BeforeScore: 1400, AfterScore: 1432, Remark: "{}", EffectiveAt: start.Add(2 * time.Hour)},
		{UserId: 101, MatchId: 3, ChangeType: model.RankChangeTypeMatchResult, GameType: 3, Result: "lose", FinalChange: -40, BeforeScore: 1432, AfterScore: 1392, Remark: "{}", EffectiveAt: start.Add(3 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed rank logs: %v", err)
	}
	resp, err := NewGetRankScoreTrendLogic(statsLogicCtx(101), svcCtx).GetRankScoreTrend(&types.GetRankScoreTrendReq{GameType: 3, Limit: 30})
	if err != nil || !resp.Success || len(resp.List) != 2 || resp.List[0].RankScore != 1392 || resp.List[1].RankScore != 1432 {
		t.Fatalf("V2 trend must not mix legacy scores or the reset: resp=%+v err=%v", resp, err)
	}

	svcCtx.Config, _ = statsV2Config(svcCtx.Config, 1)
	legacy, err := NewGetRankScoreTrendLogic(statsLogicCtx(101), svcCtx).GetRankScoreTrend(&types.GetRankScoreTrendReq{GameType: 3, Limit: 30})
	if err != nil || len(legacy.List) != 3 {
		t.Fatalf("before the V2 start the trend keeps every match log: resp=%+v err=%v", legacy, err)
	}
}

func TestOpponentStrengthShowsOnlyRelativeBucketsAfterV2(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	svcCtx.Config, _ = statsV2Config(svcCtx.Config, 0)
	if err := svcCtx.DB.Create(&[]model.UserOpponentStrengthBucket{
		{UserId: 101, GameType: 3, RankBucket: "score_1001_2000", Matches: 9, Wins: 5},
		{UserId: 101, GameType: 3, RankBucket: ranking.StrengthBucketWeaker, Matches: 4, Wins: 4},
		{UserId: 101, GameType: 3, RankBucket: ranking.StrengthBucketEven, Matches: 2, Wins: 1},
		{UserId: 101, GameType: 3, RankBucket: ranking.StrengthBucketStronger, Matches: 4, Wins: 1},
	}).Error; err != nil {
		t.Fatalf("seed buckets: %v", err)
	}
	resp, err := NewGetOpponentStrengthLogic(statsLogicCtx(101), svcCtx).GetOpponentStrength(&types.GetOpponentStrengthReq{GameType: 3})
	if err != nil || !resp.Success || len(resp.List) != 3 {
		t.Fatalf("V2 analysis must only list relative buckets: resp=%+v err=%v", resp, err)
	}
	want := []string{"强于我", "实力相当", "弱于我"}
	for index, item := range resp.List {
		if item.RankRange != want[index] {
			t.Fatalf("relative buckets must be ordered strong to weak: %+v", resp.List)
		}
	}
	if resp.List[0].Matches != 4 || resp.List[0].Wins != 1 {
		t.Fatalf("unexpected stronger bucket: %+v", resp.List[0])
	}
}

func TestOpponentStrengthFallbackUsesRelativeGapAfterV2(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	cfg, start := statsV2Config(config.Config{}, 0)
	svcCtx.Config = cfg // 竞技读模型未切读，走回退查询
	opponentStrong, opponentWeak := int64(202), int64(303)
	won, lost := 1, 2
	for _, match := range []*model.Match{
		{Id: 1, UserId: 101, OpponentId: &opponentStrong, GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &lost, MatchTime: start.Add(time.Hour), EndTime: timePtr(start.Add(2 * time.Hour))},
		{Id: 2, UserId: opponentWeak, OpponentId: int64Ptr(101), GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &lost, MatchTime: start.Add(time.Hour), EndTime: timePtr(start.Add(3 * time.Hour))},
		{Id: 3, UserId: 101, OpponentId: &opponentWeak, GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &won, MatchTime: start.Add(-3 * time.Hour), EndTime: timePtr(start.Add(-2 * time.Hour))},
	} {
		seedStatsMatch(t, svcCtx, match)
	}
	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: 101, GameType: 3, RankScore: 1300, RatingVersion: model.RatingVersionV2})
	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: opponentStrong, GameType: 3, RankScore: 1700, RatingVersion: model.RatingVersionV2})
	seedStatsRanking(t, svcCtx, &model.UserRanking{UserId: opponentWeak, GameType: 3, RankScore: 900, RatingVersion: model.RatingVersionV2})

	resp, err := NewGetOpponentStrengthLogic(statsLogicCtx(101), svcCtx).GetOpponentStrength(&types.GetOpponentStrengthReq{GameType: 3})
	if err != nil || !resp.Success || len(resp.List) != 2 {
		t.Fatalf("fallback must only count V2 matches with relative labels: resp=%+v err=%v", resp, err)
	}
	if resp.List[0].RankRange != "强于我" || resp.List[0].Wins != 0 || resp.List[1].RankRange != "弱于我" || resp.List[1].Wins != 1 {
		t.Fatalf("unexpected fallback buckets: %+v", resp.List)
	}
}

func timePtr(value time.Time) *time.Time { return &value }

func int64Ptr(value int64) *int64 { return &value }
