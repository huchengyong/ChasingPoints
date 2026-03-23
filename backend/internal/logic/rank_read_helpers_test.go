package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestBuildTrendPointsFromRankChanges(t *testing.T) {
	logs := []model.RankChangeLog{
		{
			MatchId:     11,
			Result:      "win",
			AfterScore:  540,
			EffectiveAt: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			FinalChange: 25,
		},
		{
			MatchId:     10,
			Result:      "lose",
			AfterScore:  515,
			EffectiveAt: time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC),
			FinalChange: -10,
		},
	}

	points := buildTrendPointsFromRankChanges(logs)
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].MatchId != 11 || points[0].RankScore != 540 || points[0].Result != 1 {
		t.Fatalf("unexpected first point: %+v", points[0])
	}
	if points[0].Date != "2026-03-10" {
		t.Fatalf("expected effective date 2026-03-10, got %s", points[0].Date)
	}
	if points[1].MatchId != 10 || points[1].RankScore != 515 || points[1].Result != 2 {
		t.Fatalf("unexpected second point: %+v", points[1])
	}
}

func TestBuildSeasonRankTrendFromLogs(t *testing.T) {
	logs := []model.RankChangeLog{
		{AfterScore: 500},
		{AfterScore: 530},
		{AfterScore: 520},
	}

	trend := buildSeasonRankTrendFromLogs(logs)
	if len(trend) != 3 {
		t.Fatalf("expected 3 trend points, got %d", len(trend))
	}
	if trend[0] != 500 || trend[1] != 530 || trend[2] != 520 {
		t.Fatalf("unexpected trend: %#v", trend)
	}
}

func TestBuildSeasonSnapshotFromLogs(t *testing.T) {
	before := &model.RankChangeLog{AfterScore: 480}
	inSeason := []model.RankChangeLog{
		{AfterScore: 500},
		{AfterScore: 530},
		{AfterScore: 520},
	}

	start, end, peak := buildSeasonSnapshotFromLogs(before, inSeason)
	if start != 480 {
		t.Fatalf("expected start 480, got %d", start)
	}
	if end != 520 {
		t.Fatalf("expected end 520, got %d", end)
	}
	if peak != 530 {
		t.Fatalf("expected peak 530, got %d", peak)
	}
}

func TestBuildSeasonSnapshotFromLogsUsesFirstBeforeScoreWhenNoPreviousLog(t *testing.T) {
	inSeason := []model.RankChangeLog{
		{BeforeScore: 620, AfterScore: 600},
		{BeforeScore: 600, AfterScore: 590},
	}

	start, end, peak := buildSeasonSnapshotFromLogs(nil, inSeason)
	if start != 620 {
		t.Fatalf("expected start 620, got %d", start)
	}
	if end != 590 {
		t.Fatalf("expected end 590, got %d", end)
	}
	if peak != 620 {
		t.Fatalf("expected peak 620, got %d", peak)
	}
}

func TestRankLogToDetailsIncludesPolicyAdjustmentsFromRemark(t *testing.T) {
	log := &model.RankChangeLog{
		Result:           "win",
		BaseScore:        20,
		AchievementScore: 15,
		Remark:           `{"same_opponent_adjustment":-7,"daily_cap_adjustment":-18}`,
	}

	details := rankLogToDetails(log)
	if !hasRankDetail(details, "基础分", 20) {
		t.Fatalf("expected base score detail, got %#v", details)
	}
	if !hasRankDetail(details, "成就奖励", 15) {
		t.Fatalf("expected achievement detail, got %#v", details)
	}
	if !hasRankDetail(details, "同对手衰减", -7) {
		t.Fatalf("expected same opponent adjustment detail, got %#v", details)
	}
	if !hasRankDetail(details, "每日封顶", -18) {
		t.Fatalf("expected daily cap detail, got %#v", details)
	}
}

func TestRankLogToDetailsUsesLossCopyForAchievementRelief(t *testing.T) {
	log := &model.RankChangeLog{
		Result:           "lose",
		BaseScore:        -10,
		AchievementScore: 15,
		Remark:           `{"loss_floor_adjustment":-7}`,
	}

	details := rankLogToDetails(log)
	if !hasRankDetail(details, "基础分", -10) {
		t.Fatalf("expected base score detail, got %#v", details)
	}
	if !hasRankDetail(details, "特殊战绩减免", 15) {
		t.Fatalf("expected achievement relief detail, got %#v", details)
	}
	if !hasRankDetail(details, "失败保底", -7) {
		t.Fatalf("expected loss floor detail, got %#v", details)
	}
}

func hasRankDetail(details []types.RankDetail, label string, value int) bool {
	for _, detail := range details {
		if detail.Label == label && detail.Value == value {
			return true
		}
	}
	return false
}
