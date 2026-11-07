package member

import (
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
)

var performanceTestNow = time.Date(2026, 10, 22, 12, 0, 0, 0, logicx.UTC8Location)

func v2Remark(rated bool, raceLength int, expected float64) string {
	return ranking.RemarkFor(ranking.Outcome{Rated: rated, RaceLength: raceLength, Expected: expected})
}

func testSample(matchID int64, won bool, expected float64, playedAt time.Time) performanceSample {
	return performanceSample{
		performanceCandidate: performanceCandidate{matchID: matchID, won: won, expected: expected, raceLength: 5, playedAt: playedAt},
		opponentUserID:       matchID + 1000,
		opponentRating:       1500,
	}
}

func TestParsePerformanceRowsKeepsOnlyRatedV2ResultsAndTracksMonthEnd(t *testing.T) {
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, logicx.UTC8Location) }
	rows := []model.RankChangeLog{ // 最新在前
		{MatchId: 5, Result: "win", AfterScore: 1530, EffectiveAt: at(21, 20), Remark: v2Remark(false, 5, 0.5)},                                 // V2 不计分：不进样本，但分数有效
		{MatchId: 4, Result: "lose", AfterScore: 1490, EffectiveAt: at(21, 10), Remark: v2Remark(true, 5, 0.62)},                                // 样本
		{MatchId: 3, Result: "win", AfterScore: 1520, EffectiveAt: at(20, 10), Remark: v2Remark(true, 7, 0.3)},                                  // 样本
		{MatchId: 2, Result: "draw", AfterScore: 1500, EffectiveAt: at(19, 10), Remark: v2Remark(true, 5, 0.5)},                                 // 平局不进样本
		{MatchId: 1, Result: "win", AfterScore: 2600, EffectiveAt: time.Date(2026, 9, 30, 10, 0, 0, 0, logicx.UTC8Location), Remark: `{"v":1}`}, // 旧规则：口径不同，完全忽略
	}

	candidates, monthEnd := parsePerformanceRows(rows)
	if len(candidates) != 2 || candidates[0].matchID != 4 || candidates[0].won || candidates[0].raceLength != 5 || candidates[0].expected != 0.62 ||
		candidates[1].matchID != 3 || !candidates[1].won || candidates[1].raceLength != 7 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	if monthEnd["2026-10"] != 1530 {
		t.Fatalf("month end must come from the newest V2 row even when it was not rated: %v", monthEnd)
	}
	if _, ok := monthEnd["2026-09"]; ok {
		t.Fatalf("legacy rows must not provide a month-end rating: %v", monthEnd)
	}
}

func TestParsePerformanceRowsUsesBusinessTimezoneForMonths(t *testing.T) {
	// UTC 9 月 30 日 17:00 已是业务时区 10 月 1 日 01:00。
	rows := []model.RankChangeLog{{MatchId: 1, Result: "win", AfterScore: 1510, EffectiveAt: time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC), Remark: v2Remark(true, 5, 0.5)}}
	candidates, monthEnd := parsePerformanceRows(rows)
	if monthEnd["2026-10"] != 1510 || len(candidates) != 1 || candidates[0].playedAt.Format("2006-01") != "2026-10" {
		t.Fatalf("months must follow the business timezone: monthEnd=%v candidates=%+v", monthEnd, candidates)
	}
}

func TestAttachOpponentsDropsSamplesWithoutOpponentRecord(t *testing.T) {
	candidates := []performanceCandidate{{matchID: 1}, {matchID: 2}, {matchID: 3}}
	samples, dropped := attachOpponents(candidates, []model.OpponentRatingRef{{MatchId: 1, UserId: 11, BeforeScore: 1400}, {MatchId: 3, UserId: 13, BeforeScore: 1600}})
	if dropped != 1 || len(samples) != 2 || samples[0].opponentUserID != 11 || samples[0].opponentRating != 1400 || samples[1].opponentRating != 1600 {
		t.Fatalf("unexpected attach result: samples=%+v dropped=%d", samples, dropped)
	}
}

func TestSummaryWindowIsLimitedTo90DaysAnd200Samples(t *testing.T) {
	samples := make([]performanceSample, 0, 260)
	for i := 0; i < 250; i++ {
		samples = append(samples, testSample(int64(i+1), true, 0.5, performanceTestNow.Add(-time.Duration(i+1)*time.Minute)))
	}
	samples = append(samples, testSample(900, true, 0.5, performanceTestNow.AddDate(0, 0, -91)))
	if got := summaryWindow(samples, performanceTestNow); len(got) != 200 || got[0].matchID != 1 || got[199].matchID != 200 {
		t.Fatalf("window must keep the newest 200 samples: len=%d", len(got))
	}

	few := []performanceSample{
		testSample(1, true, 0.5, performanceTestNow.AddDate(0, 0, -89)),
		testSample(2, true, 0.5, performanceTestNow.AddDate(0, 0, -91)),
		testSample(3, true, 0.5, performanceTestNow.AddDate(0, 0, -120)),
	}
	if got := summaryWindow(few, performanceTestNow); len(got) != 1 || got[0].matchID != 1 {
		t.Fatalf("samples older than 90 days must be left out: %+v", got)
	}
}

func TestSummarizeReportsWinsExpectedWinsBestWinAndWorstLoss(t *testing.T) {
	// 从新到旧；最佳胜利是期望最低的胜局，爆冷失利是期望最高的负局，期望相同取较近的一场。
	window := []performanceSample{
		testSample(1, true, 0.21, performanceTestNow.Add(-1*time.Hour)),
		testSample(2, true, 0.21, performanceTestNow.Add(-2*time.Hour)),
		testSample(3, true, 0.64, performanceTestNow.Add(-3*time.Hour)),
		testSample(4, false, 0.88, performanceTestNow.Add(-4*time.Hour)),
		testSample(5, false, 0.88, performanceTestNow.Add(-5*time.Hour)),
		testSample(6, false, 0.35, performanceTestNow.Add(-6*time.Hour)),
	}
	summary, bestWin, worstLoss := summarize(window)
	if summary.Wins != 3 || summary.ExpectedWins != 3.17 || summary.WinsDiff != -0.17 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if bestWin == nil || bestWin.matchID != 1 || worstLoss == nil || worstLoss.matchID != 4 {
		t.Fatalf("ties must keep the more recent match: best=%+v worst=%+v", bestWin, worstLoss)
	}
	if summary.Rating == nil {
		t.Fatal("six samples are enough for a performance rating")
	}

	none, noBest, noWorst := summarize(nil)
	if none.Wins != 0 || none.Rating != nil || noBest != nil || noWorst != nil {
		t.Fatalf("empty window must have no rating or notable matches: %+v", none)
	}

	allWins := []performanceSample{testSample(1, true, 0.5, performanceTestNow)}
	if _, best, worst := summarize(allWins); best == nil || worst != nil {
		t.Fatalf("a window without losses has no upset loss: best=%+v worst=%+v", best, worst)
	}
}

func TestSummarizeWithFourSamplesHidesOnlyTheRating(t *testing.T) {
	window := []performanceSample{
		testSample(1, true, 0.5, performanceTestNow), testSample(2, true, 0.5, performanceTestNow),
		testSample(3, false, 0.5, performanceTestNow), testSample(4, false, 0.5, performanceTestNow),
	}
	summary, _, _ := summarize(window)
	if summary.Rating != nil || summary.Wins != 2 || summary.ExpectedWins != 2 || summary.WinsDiff != 0 {
		t.Fatalf("four samples must still report win counts: %+v", summary)
	}
}

func TestBuildMonthsCoversSixBusinessMonthsOldestFirst(t *testing.T) {
	var samples []performanceSample // 10 月 3 场、9 月 2 场、5 月（超出 6 个月）不应出现
	for i := 0; i < 3; i++ {
		samples = append(samples, testSample(int64(i+1), i == 0, 0.5, time.Date(2026, 10, 20-i, 10, 0, 0, 0, logicx.UTC8Location)))
	}
	for i := 0; i < 2; i++ {
		samples = append(samples, testSample(int64(i+10), true, 0.5, time.Date(2026, 9, 10-i, 10, 0, 0, 0, logicx.UTC8Location)))
	}
	months := buildMonths(samples, map[string]int{"2026-10": 1555, "2026-09": 1480}, performanceTestNow, false, time.Time{})

	if len(months) != 6 || months[0].Month != "2026-05" || months[5].Month != "2026-10" {
		t.Fatalf("expected six months oldest first: %+v", months)
	}
	oct, sep := months[5], months[4]
	if oct.SampleCount != 3 || oct.Rating == nil || oct.EndRankScore != 1555 || oct.Incomplete {
		t.Fatalf("October has three samples and must carry a rating: %+v", oct)
	}
	if sep.SampleCount != 2 || sep.Rating != nil || sep.EndRankScore != 1480 {
		t.Fatalf("September has only two samples: no rating but still an end rating: %+v", sep)
	}
	if empty := months[0]; empty.SampleCount != 0 || empty.Rating != nil || empty.EndRankScore != 0 || empty.Incomplete {
		t.Fatalf("a month without matches is empty and not incomplete: %+v", empty)
	}
}

func TestBuildMonthsMarksTruncatedMonthsIncompleteWithoutRating(t *testing.T) {
	var samples []performanceSample
	for i := 0; i < 4; i++ { // 9 月只读到最新的 4 场；读取上限在 9 月内被触发
		samples = append(samples, testSample(int64(i+1), true, 0.5, time.Date(2026, 9, 28-i, 10, 0, 0, 0, logicx.UTC8Location)))
	}
	for i := 0; i < 3; i++ {
		samples = append(samples, testSample(int64(i+10), true, 0.5, time.Date(2026, 10, 20-i, 10, 0, 0, 0, logicx.UTC8Location)))
	}
	oldest := time.Date(2026, 9, 25, 10, 0, 0, 0, logicx.UTC8Location)
	months := buildMonths(samples, map[string]int{"2026-10": 1600, "2026-09": 1500}, performanceTestNow, true, oldest)

	for _, item := range months {
		wantIncomplete := item.Month <= "2026-09"
		if item.Incomplete != wantIncomplete {
			t.Fatalf("%s: incomplete=%v want %v", item.Month, item.Incomplete, wantIncomplete)
		}
		if item.Incomplete && item.Rating != nil {
			t.Fatalf("%s: an incomplete month must not report a rating computed from partial samples", item.Month)
		}
	}
	if oct := months[5]; oct.Rating == nil || oct.Incomplete {
		t.Fatalf("months newer than the cut-off are complete: %+v", oct)
	}
	if sep := months[4]; sep.EndRankScore != 1500 {
		t.Fatalf("the truncated month still keeps its newest end rating: %+v", sep)
	}
}

func TestBuildMonthPreviewMatchesTheCurveValueForTheCurrentMonth(t *testing.T) {
	var samples []performanceSample
	for i := 0; i < 6; i++ {
		samples = append(samples, testSample(int64(i+1), i < 3, 0.5, time.Date(2026, 10, 20-i, 10, 0, 0, 0, logicx.UTC8Location)))
	}
	samples = append(samples, testSample(99, true, 0.5, time.Date(2026, 9, 15, 10, 0, 0, 0, logicx.UTC8Location)))

	preview := buildMonthPreview(samples, performanceTestNow)
	months := buildMonths(samples, nil, performanceTestNow, false, time.Time{})
	current := months[len(months)-1]
	if preview.Month != "2026-10" || preview.SampleCount != 6 || preview.Rating == nil || current.Rating == nil || *preview.Rating != *current.Rating {
		t.Fatalf("preview must equal the curve's current month: preview=%+v current=%+v", preview, current)
	}

	few := buildMonthPreview(samples[:2], performanceTestNow)
	if few.SampleCount != 2 || few.Rating != nil {
		t.Fatalf("two samples this month must not produce a preview rating: %+v", few)
	}
}
