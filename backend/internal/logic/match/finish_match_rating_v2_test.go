package match

import (
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// ratingV2TestConfig 把 V2 起点设为本月初（offsetMonths=0，已生效）或之后某个月初（未生效），赛季按自然月。
func ratingV2TestConfig(base config.Config, offsetMonths int) config.Config {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(shanghai)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghai).AddDate(0, offsetMonths, 0)
	base.SeasonLifecycle = config.SeasonLifecycleConfig{
		Enabled:       true,
		AnchorDate:    start.AddDate(0, -6, 0).Format(time.DateOnly),
		InitialNumber: 1,
		CycleMonths:   1,
		Timezone:      "Asia/Shanghai",
	}
	base.Ranking.RatingV2StartDate = start.Format(time.DateOnly)
	return base
}

// newRatingV2FinishFixture 准备一场抢 5 的 5:3 排位：甲是会员、旧分 2600 且有 V2 种子 1400/12 场；乙旧分 300、没有种子。
func newRatingV2FinishFixture(t *testing.T, offsetMonths int) (*svc.ServiceContext, int64) {
	t.Helper()
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.RankRatingSeed{}); err != nil {
		t.Fatalf("prepare rating seed schema: %v", err)
	}
	svcCtx.Config = ratingV2TestConfig(svcCtx.Config, offsetMonths)
	memberUntil := time.Now().Add(30 * 24 * time.Hour)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲", MemberExpiresAt: &memberUntil},
		model.User{Id: 2002, Nickname: "选手乙"},
	)
	for _, row := range []model.UserRanking{
		{UserId: 1001, GameType: 3, RankScore: 2600, RankLevel: 6, TotalWins: 80, TotalLosses: 40, MaxStreak: 7},
		{UserId: 2002, GameType: 3, RankScore: 300, RankLevel: 1, TotalWins: 3, TotalLosses: 9},
	} {
		item := row
		if err := svcCtx.DB.Create(&item).Error; err != nil {
			t.Fatalf("seed legacy ranking: %v", err)
		}
	}
	if err := svcCtx.DB.Create(&model.RankRatingSeed{UserId: 1001, GameType: 3, SeedScore: 1400, SeedMatches: 12, CutoffAt: time.Now().Add(-24 * time.Hour)}).Error; err != nil {
		t.Fatalf("seed rating seed: %v", err)
	}

	opponentID := int64(2002)
	match := model.Match{
		Id: 7101, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		MatchFormat: model.MatchFormatRaceTo, TargetWins: 5,
		Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
		MatchTime: time.Now().Add(-40 * time.Minute),
	}
	if err := svcCtx.MatchModel.Create(&match); err != nil {
		t.Fatalf("create match: %v", err)
	}
	// 甲有 2 局炸清：旧规则下会员会额外得到特殊战绩分，V2 不得计入。
	rounds := []struct {
		winner  int
		winType string
	}{{1, "break_clear"}, {2, "normal"}, {1, "break_clear"}, {2, "normal"}, {1, "normal"}, {2, "normal"}, {1, "normal"}, {1, "normal"}}
	for index, item := range rounds {
		winner := item.winner
		if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: index + 1, Winner: &winner, WinType: item.winType}).Error; err != nil {
			t.Fatalf("create round: %v", err)
		}
	}
	return svcCtx, match.Id
}

func findRanking(t *testing.T, svcCtx *svc.ServiceContext, userID int64) model.UserRanking {
	t.Helper()
	var row model.UserRanking
	if err := svcCtx.DB.Where("user_id = ? AND game_type = ?", userID, 3).First(&row).Error; err != nil {
		t.Fatalf("load ranking %d: %v", userID, err)
	}
	return row
}

func TestRankedFinishSettlesWithRatingV2AfterStart(t *testing.T) {
	svcCtx, matchID := newRatingV2FinishFixture(t, 0)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: matchID, ClientActionId: "rating-v2-finish"})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("finish match: resp=%#v err=%v", resp, err)
	}

	// 甲用种子起步（已定级），乙从 1250 定级；抢 5，甲胜，会员高光不计。
	want1, want2 := ranking.Settle(ranking.Input{
		Player1:          ranking.PlayerState{Rating: 1400, RatedMatches: 12},
		Player2:          ranking.PlayerState{Rating: ranking.InitialRating, RatedMatches: 0},
		Player1Won:       true,
		RaceLength:       5,
		DailyPositiveCap: ranking.DefaultDailyPositiveCap,
	})
	player1, player2 := findRanking(t, svcCtx, 1001), findRanking(t, svcCtx, 2002)
	if player1.RatingVersion != model.RatingVersionV2 || player1.RankScore != want1.After || player1.RatedMatches != 13 {
		t.Fatalf("winner must be switched to the seed and settled with V2: row=%+v want=%+v", player1, want1)
	}
	if player2.RatingVersion != model.RatingVersionV2 || player2.RankScore != want2.After || player2.RatedMatches != 1 {
		t.Fatalf("loser without seed must start from the initial rating: row=%+v want=%+v", player2, want2)
	}
	if want1.OpponentDamping >= 0 || want2.Placement >= 0 {
		t.Fatalf("fixture must exercise placement adjustments: %+v %+v", want1, want2)
	}
	if player1.TotalWins != 81 || player1.MaxStreak != 7 || player2.TotalLosses != 10 {
		t.Fatalf("lifetime counters must keep accumulating across the switch: %+v %+v", player1, player2)
	}

	var resets []model.RankChangeLog
	if err := svcCtx.DB.Where("change_type = ?", model.RankChangeTypeRatingReset).Order("user_id ASC").Find(&resets).Error; err != nil {
		t.Fatalf("load reset logs: %v", err)
	}
	if len(resets) != 2 || resets[0].BeforeScore != 2600 || resets[0].AfterScore != 1400 || resets[1].BeforeScore != 300 || resets[1].AfterScore != ranking.InitialRating {
		t.Fatalf("switch must leave one auditable reset per ranking: %+v", resets)
	}
	if remark, ok := ranking.ParseRemark(resets[0].Remark); !ok || remark.SeedMatches != 12 {
		t.Fatalf("reset remark must keep the seed matches: %q", resets[0].Remark)
	}

	var logs []model.RankChangeLog
	if err := svcCtx.DB.Where("match_id = ? AND change_type = ?", matchID, model.RankChangeTypeMatchResult).Order("user_id ASC").Find(&logs).Error; err != nil || len(logs) != 2 {
		t.Fatalf("load match logs: logs=%+v err=%v", logs, err)
	}
	if logs[0].FinalChange != want1.Change || logs[0].AchievementScore != 0 || logs[0].BaseScore != want1.Standard || logs[1].FinalChange != want2.Change {
		t.Fatalf("match logs must carry the V2 outcome without achievement points: %+v", logs)
	}

	detail, err := NewGetMatchDetailLogic(finishReputationCtx(1001), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: matchID})
	if err != nil || detail == nil || !detail.Success {
		t.Fatalf("get match detail: resp=%#v err=%v", detail, err)
	}
	if detail.Match.RankRulesVersion != ranking.RulesVersionV2 || detail.Match.MyRankChange != want1.Change {
		t.Fatalf("match detail must expose the V2 settlement: %+v", detail.Match)
	}
	if len(detail.Match.MyRankDetails) < 2 || !strings.HasPrefix(detail.Match.MyRankDetails[0].Label, "胜负结算") || detail.Match.MyRankDetails[1].Label != "对手定级中减半" {
		t.Fatalf("V2 details must explain the result and the placement damping: %+v", detail.Match.MyRankDetails)
	}
	sum := 0
	for _, item := range detail.Match.MyRankDetails {
		sum += item.Value
		if strings.Contains(item.Label, "VIP") || strings.Contains(item.Label, "特殊战绩") {
			t.Fatalf("V2 details must not mention member highlights: %+v", detail.Match.MyRankDetails)
		}
	}
	if sum != want1.Change {
		t.Fatalf("details must add up to the change: sum=%d change=%d", sum, want1.Change)
	}
}

func TestRankedFinishKeepsLegacyRulesBeforeV2Start(t *testing.T) {
	svcCtx, matchID := newRatingV2FinishFixture(t, 1)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: matchID, ClientActionId: "legacy-finish"})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("finish match: resp=%#v err=%v", resp, err)
	}
	player1 := findRanking(t, svcCtx, 1001)
	if player1.RatingVersion != 1 || player1.RankScore <= 2600 {
		t.Fatalf("before the V2 start the legacy rules must still apply: %+v", player1)
	}
	var resets int64
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("change_type = ?", model.RankChangeTypeRatingReset).Count(&resets).Error; err != nil || resets != 0 {
		t.Fatalf("legacy settlement must not switch rankings: resets=%d err=%v", resets, err)
	}
	detail, err := NewGetMatchDetailLogic(finishReputationCtx(1001), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: matchID})
	if err != nil || detail == nil || detail.Match.RankRulesVersion != ranking.RulesVersionLegacy {
		t.Fatalf("legacy match detail must report rules version 1: resp=%#v err=%v", detail, err)
	}
}
