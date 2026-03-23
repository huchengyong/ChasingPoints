package logic

import (
	"testing"

	"billiard_master/internal/model"
)

func TestRankSettlementServiceLossAtZeroScoreDoesNotDeduct(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.Settle(&model.UserRanking{
		UserId:        1,
		RankScore:     0,
		RankLevel:     1,
		CurrentStreak: 2,
		MaxStreak:     4,
	}, false, 0)

	if result.BaseScore != 0 {
		t.Fatalf("expected base score 0, got %d", result.BaseScore)
	}
	if result.FinalChange != 0 {
		t.Fatalf("expected final change 0, got %d", result.FinalChange)
	}
	if result.AfterScore != 0 {
		t.Fatalf("expected after score 0, got %d", result.AfterScore)
	}
	if result.TotalLosses != 1 {
		t.Fatalf("expected total losses 1, got %d", result.TotalLosses)
	}
	if result.CurrentStreak != -1 {
		t.Fatalf("expected current streak -1, got %d", result.CurrentStreak)
	}
}

func TestRankSettlementServiceWinAddsAchievementScore(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.Settle(&model.UserRanking{
		UserId:        1,
		RankScore:     480,
		RankLevel:     1,
		TotalWins:     3,
		CurrentStreak: 1,
		MaxStreak:     2,
	}, true, 15)

	if result.BaseScore != 20 {
		t.Fatalf("expected base score 20, got %d", result.BaseScore)
	}
	if result.AchievementScore != 15 {
		t.Fatalf("expected achievement score 15, got %d", result.AchievementScore)
	}
	if result.FinalChange != 35 {
		t.Fatalf("expected final change 35, got %d", result.FinalChange)
	}
	if result.AfterScore != 515 {
		t.Fatalf("expected after score 515, got %d", result.AfterScore)
	}
	if result.AfterLevel != 2 {
		t.Fatalf("expected after level 2, got %d", result.AfterLevel)
	}
	if len(result.Details) != 2 {
		t.Fatalf("expected 2 detail rows, got %d", len(result.Details))
	}
}

func TestRankSettlementServicePromotesAcrossThreshold(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.Settle(&model.UserRanking{
		UserId:      1,
		RankScore:   1490,
		RankLevel:   3,
		TotalWins:   9,
		TotalLosses: 4,
	}, true, 0)

	if result.AfterScore != 1510 {
		t.Fatalf("expected after score 1510, got %d", result.AfterScore)
	}
	if result.AfterLevel != 4 {
		t.Fatalf("expected after level 4, got %d", result.AfterLevel)
	}
}

func TestRankSettlementServiceDemotesAfterLoss(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.Settle(&model.UserRanking{
		UserId:      1,
		RankScore:   500,
		RankLevel:   2,
		TotalWins:   6,
		TotalLosses: 3,
	}, false, 0)

	if result.BaseScore != -10 {
		t.Fatalf("expected base score -10, got %d", result.BaseScore)
	}
	if result.AfterScore != 490 {
		t.Fatalf("expected after score 490, got %d", result.AfterScore)
	}
	if result.AfterLevel != 1 {
		t.Fatalf("expected after level 1, got %d", result.AfterLevel)
	}
}

func TestRankSettlementServiceLossUsesNegativeTwoFloorWhenAchievementsOffsetLoss(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:      1,
		RankScore:   120,
		RankLevel:   1,
		TotalWins:   6,
		TotalLosses: 3,
	}, false, 15, RankSettlementPolicy{})

	if result.FinalChange != -2 {
		t.Fatalf("expected final change -2, got %d", result.FinalChange)
	}
	if result.AfterScore != 118 {
		t.Fatalf("expected after score 118, got %d", result.AfterScore)
	}
	if result.LossFloorAdjustment != -7 {
		t.Fatalf("expected loss floor adjustment -7, got %d", result.LossFloorAdjustment)
	}
}

func TestRankSettlementServiceLossAtZeroScoreWithAchievementDoesNotIncreaseRank(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:      1,
		RankScore:   0,
		RankLevel:   1,
		TotalWins:   2,
		TotalLosses: 3,
	}, false, 15, RankSettlementPolicy{})

	if result.FinalChange != 0 {
		t.Fatalf("expected final change 0, got %d", result.FinalChange)
	}
	if result.AfterScore != 0 {
		t.Fatalf("expected after score 0, got %d", result.AfterScore)
	}
}

func TestRankSettlementServiceThirdSameOpponentWinUsesEightyPercentGain(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:      1,
		RankScore:   480,
		RankLevel:   1,
		TotalWins:   3,
		TotalLosses: 1,
	}, true, 15, RankSettlementPolicy{
		SameOpponentMatchesToday: 2,
	})

	if result.FinalChange != 28 {
		t.Fatalf("expected final change 28, got %d", result.FinalChange)
	}
	if result.AfterScore != 508 {
		t.Fatalf("expected after score 508, got %d", result.AfterScore)
	}
	if result.SameOpponentAdjustment != -7 {
		t.Fatalf("expected same opponent adjustment -7, got %d", result.SameOpponentAdjustment)
	}
}

func TestRankSettlementServiceDailyGainCapTruncatesPositiveGain(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:      1,
		RankScore:   480,
		RankLevel:   1,
		TotalWins:   3,
		TotalLosses: 1,
	}, true, 15, RankSettlementPolicy{
		TodayPositiveGain: 295,
		DailyPositiveCap:  300,
	})

	if result.FinalChange != 5 {
		t.Fatalf("expected final change 5, got %d", result.FinalChange)
	}
	if result.AfterScore != 485 {
		t.Fatalf("expected after score 485, got %d", result.AfterScore)
	}
	if result.DailyCapAdjustment != -30 {
		t.Fatalf("expected daily cap adjustment -30, got %d", result.DailyCapAdjustment)
	}
}

func TestRankSettlementServiceSeventhSameOpponentMatchSkipsRankAndStats(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:        1,
		RankScore:     480,
		RankLevel:     1,
		TotalWins:     3,
		TotalLosses:   1,
		CurrentStreak: 2,
		MaxStreak:     4,
	}, true, 15, RankSettlementPolicy{
		SameOpponentMatchesToday: 6,
	})

	if result.FinalChange != 0 {
		t.Fatalf("expected final change 0, got %d", result.FinalChange)
	}
	if result.AfterScore != 480 {
		t.Fatalf("expected after score 480, got %d", result.AfterScore)
	}
	if result.TotalWins != 3 || result.TotalLosses != 1 {
		t.Fatalf("expected stats unchanged, got wins=%d losses=%d", result.TotalWins, result.TotalLosses)
	}
	if result.CurrentStreak != 2 || result.MaxStreak != 4 {
		t.Fatalf("expected streaks unchanged, got current=%d max=%d", result.CurrentStreak, result.MaxStreak)
	}
}

func TestRankSettlementServiceSeventhSameOpponentLossAlsoSkipsRankAndStats(t *testing.T) {
	service := NewRankSettlementService(&model.RankingModel{})

	result := service.SettleWithPolicy(&model.UserRanking{
		UserId:        1,
		RankScore:     480,
		RankLevel:     1,
		TotalWins:     3,
		TotalLosses:   1,
		CurrentStreak: -2,
		MaxStreak:     4,
	}, false, 0, RankSettlementPolicy{
		SameOpponentMatchesToday: 6,
	})

	if result.FinalChange != 0 {
		t.Fatalf("expected final change 0, got %d", result.FinalChange)
	}
	if result.AfterScore != 480 {
		t.Fatalf("expected after score 480, got %d", result.AfterScore)
	}
	if result.TotalWins != 3 || result.TotalLosses != 1 {
		t.Fatalf("expected stats unchanged, got wins=%d losses=%d", result.TotalWins, result.TotalLosses)
	}
	if result.CurrentStreak != -2 || result.MaxStreak != 4 {
		t.Fatalf("expected streaks unchanged, got current=%d max=%d", result.CurrentStreak, result.MaxStreak)
	}
	if result.SameOpponentAdjustment != 10 {
		t.Fatalf("expected same opponent relief 10, got %d", result.SameOpponentAdjustment)
	}
}
