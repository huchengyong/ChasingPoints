package logic

import (
	"testing"

	"chasing_points/internal/model"
)

func TestCalculateMemberAchievementRankingScoreUsesConfiguredMultipliers(t *testing.T) {
	cases := []struct {
		level int
		want  int
	}{
		{1, 16},
		{2, 17},
		{3, 19},
		{4, 20},
		{5, 22},
	}

	for _, tc := range cases {
		got := CalculateMemberAchievementRankingScore(16, true, true, tc.level)
		if got != tc.want {
			t.Fatalf("level %d expected %d, got %d", tc.level, tc.want, got)
		}
	}
}

func TestCalculateMemberAchievementRankingScoreSkipsNonMembersAndLosses(t *testing.T) {
	if got := CalculateMemberAchievementRankingScore(16, true, false, 5); got != 0 {
		t.Fatalf("expected inactive member to get 0, got %d", got)
	}
	if got := CalculateMemberAchievementRankingScore(16, false, true, 5); got != 0 {
		t.Fatalf("expected losing member to get 0, got %d", got)
	}
}

func TestRankSettlementServiceCapsMemberAchievementScoreByDailyCap(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:    1,
		RankScore: 480,
		RankLevel: 1,
	}, true, 42, RankSettlementPolicy{
		TodayMemberAchievementGain: 190,
		DailyMemberAchievementCap:  200,
	})

	if result.AchievementScore != 10 {
		t.Fatalf("expected capped achievement score 10, got %d", result.AchievementScore)
	}
	if result.MemberAchievementCapAdjustment != -32 {
		t.Fatalf("expected member achievement cap adjustment -32, got %d", result.MemberAchievementCapAdjustment)
	}
	if result.FinalChange != 30 {
		t.Fatalf("expected final change 30, got %d", result.FinalChange)
	}
}
