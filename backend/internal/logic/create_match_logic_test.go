package logic

import (
	"testing"

	"billiard_master/internal/model"
	"billiard_master/internal/types"
)

func TestCalculateAchievementScoreFromItems(t *testing.T) {
	items := []types.AchievementItem{
		{Type: "break_clear", Count: 1},
		{Type: "small_gold", Count: 2},
		{Type: "unknown", Count: 3},
	}
	rewardMap := map[string]int{
		"break_and_run": 15,
		"golden_break":  10,
	}

	score := calculateAchievementScoreFromItems(items, rewardMap)
	if score != 35 {
		t.Fatalf("expected score 35, got %d", score)
	}
}

func TestApplyHistoricalMatchSeasonSnapshotKeepsExistingRecordWithoutLogs(t *testing.T) {
	recordInfo := &types.SeasonRecordInfo{
		SeasonId:       3,
		SeasonName:     "S3",
		StartRankScore: 600,
		EndRankScore:   710,
		PeakRankScore:  730,
	}
	season := &model.Season{Id: 3, Name: "S3"}

	got := applyHistoricalMatchSeasonSnapshot(recordInfo, season, 0, 0, 0, false)
	if got.StartRankScore != 600 || got.EndRankScore != 710 || got.PeakRankScore != 730 {
		t.Fatalf("expected existing season record to be preserved, got %+v", got)
	}
}

func TestApplyHistoricalMatchSeasonSnapshotUsesComputedValuesWhenLogsExist(t *testing.T) {
	recordInfo := &types.SeasonRecordInfo{
		SeasonId:       4,
		SeasonName:     "S4",
		StartRankScore: 300,
		EndRankScore:   320,
		PeakRankScore:  350,
	}
	season := &model.Season{Id: 4, Name: "S4"}

	got := applyHistoricalMatchSeasonSnapshot(recordInfo, season, 500, 530, 540, true)
	if got.StartRankScore != 500 || got.EndRankScore != 530 || got.PeakRankScore != 540 {
		t.Fatalf("expected computed values to override record, got %+v", got)
	}
}
