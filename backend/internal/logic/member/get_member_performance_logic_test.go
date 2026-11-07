package member

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type performanceFixture struct {
	t      *testing.T
	db     *gorm.DB
	svcCtx *svc.ServiceContext
	now    time.Time
}

func newPerformanceFixture(t *testing.T) *performanceFixture {
	t.Helper()
	return newPerformanceFixtureNamed(t, t.Name())
}

// seedUser 创建用户；memberFor 为 0 表示从未开通，负数表示已到期。
func (f *performanceFixture) seedUser(id int64, nickname string, memberFor time.Duration) {
	f.t.Helper()
	user := model.User{Id: id, Nickname: nickname, Avatar: nickname + ".png"}
	if memberFor != 0 {
		expires := f.now.Add(memberFor)
		user.MemberExpiresAt = &expires
	}
	if err := f.db.Create(&user).Error; err != nil {
		f.t.Fatalf("seed user %d: %v", id, err)
	}
}

type seededMatch struct {
	id         int64
	self, opp  int64
	won        bool
	expected   float64
	raceLength int
	selfBefore int
	oppBefore  int
	at         time.Time
	rated      bool
	remark     string // 非空时覆盖默认的 V2 明细
	gameType   int
}

func (f *performanceFixture) matchRows(m seededMatch) []model.RankChangeLog {
	if m.gameType == 0 {
		m.gameType = 3
	}
	if m.raceLength == 0 {
		m.raceLength = 5
	}
	delta, selfResult, oppResult := 10, "win", "lose"
	if !m.won {
		delta, selfResult, oppResult = -10, "lose", "win"
	}
	selfRemark, oppRemark := m.remark, m.remark
	if m.remark == "" {
		selfRemark, oppRemark = v2Remark(m.rated, m.raceLength, m.expected), v2Remark(m.rated, m.raceLength, 1-m.expected)
	}
	return []model.RankChangeLog{
		{UserId: m.self, MatchId: m.id, ChangeType: model.RankChangeTypeMatchResult, GameType: m.gameType, Result: selfResult, BeforeScore: m.selfBefore, AfterScore: m.selfBefore + delta, Remark: selfRemark, EffectiveAt: m.at},
		{UserId: m.opp, MatchId: m.id, ChangeType: model.RankChangeTypeMatchResult, GameType: m.gameType, Result: oppResult, BeforeScore: m.oppBefore, AfterScore: m.oppBefore - delta, Remark: oppRemark, EffectiveAt: m.at},
	}
}

func (f *performanceFixture) seedMatches(matches ...seededMatch) {
	f.t.Helper()
	var rows []model.RankChangeLog
	for _, m := range matches {
		rows = append(rows, f.matchRows(m)...)
	}
	if err := f.db.CreateInBatches(&rows, 200).Error; err != nil {
		f.t.Fatalf("seed rank change logs: %v", err)
	}
}

// seedEvenMatches 写入 count 场用户 1 对赛前分 1500 对手、期望 0.5 的抢 5 排位：前 wins 场（最新的）获胜，每场隔一小时。
func (f *performanceFixture) seedEvenMatches(count, wins int) {
	f.t.Helper()
	matches := make([]seededMatch, 0, count)
	for i := 1; i <= count; i++ {
		matches = append(matches, seededMatch{id: int64(i), self: 1, opp: int64(100 + i), won: i <= wins, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: f.now.Add(-time.Duration(i) * time.Hour), rated: true})
	}
	f.seedMatches(matches...)
}

func (f *performanceFixture) get(userID int64, gameType int) (*types.GetMemberPerformanceResp, int64) {
	f.t.Helper()
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userID), metrics)
	l := NewGetMemberPerformanceLogic(ctx, f.svcCtx)
	l.now = func() time.Time { return f.now }
	resp, err := l.GetMemberPerformance(&types.GetMemberPerformanceReq{GameType: gameType})
	if err != nil {
		f.t.Fatalf("get performance: %v", err)
	}
	return resp, metrics.Snapshot().SQLCount
}

func TestMemberPerformanceReturnsFullAnalysisForActiveMember(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	for i := 1; i <= 5; i++ {
		f.seedUser(int64(100+i), fmt.Sprintf("对手%d", i), 0)
	}
	f.seedEvenMatches(5, 3) // 比赛 1–3 胜、4–5 负，1 最新

	resp, _ := f.get(1, 3)
	if !resp.Success || resp.Locked || resp.GameType != 3 || resp.SampleCount != 5 || resp.MonthPreview != nil {
		t.Fatalf("active member must get the full analysis: %+v", resp)
	}
	s := resp.Summary
	if s == nil || s.Wins != 3 || s.ExpectedWins != 2.5 || s.WinsDiff != 0.5 || s.Rating == nil || s.Rating.Rating != 1678 || s.Rating.Bound != "" {
		t.Fatalf("race-to-5 vs five 1500 opponents, 3 wins: want 3 vs 2.5 expected and rating 1678: %+v", s)
	}
	// 期望相同取较近的一场：最佳胜利是最新的胜局（1），爆冷失利是最新的负局（4）。
	if resp.BestWin == nil || resp.BestWin.MatchId != 1 || resp.BestWin.OpponentName != "对手1" || resp.BestWin.OpponentAvatar != "对手1.png" ||
		resp.BestWin.OpponentBeforeScore != 1500 || resp.BestWin.ExpectedWinRate != 50 {
		t.Fatalf("unexpected best win: %+v", resp.BestWin)
	}
	if resp.WorstLoss == nil || resp.WorstLoss.MatchId != 4 || resp.WorstLoss.OpponentName != "对手4" {
		t.Fatalf("unexpected worst loss: %+v", resp.WorstLoss)
	}
	if len(resp.Months) != 6 || resp.Months[5].Month != "2026-10" || resp.Months[5].SampleCount != 5 || resp.Months[5].Rating == nil || resp.Months[5].Rating.Rating != 1678 {
		t.Fatalf("current month must carry the same rating as the summary: %+v", resp.Months)
	}
	if resp.Months[5].EndRankScore != 1510 { // 最新一场（比赛 1）赛前 1500 赢 10 分
		t.Fatalf("month end rating must come from the newest match: %+v", resp.Months[5])
	}
}

func TestMemberPerformanceMarksAllWinsAsALowerBound(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(5, 5)

	resp, _ := f.get(1, 3)
	if resp.Summary == nil || resp.Summary.Rating == nil || resp.Summary.Rating.Rating != 2415 || resp.Summary.Rating.Bound != "at_least" {
		t.Fatalf("five straight wins must be 2415 and a lower bound: %+v", resp.Summary)
	}
	if resp.WorstLoss != nil {
		t.Fatalf("no losses means no upset loss: %+v", resp.WorstLoss)
	}
	if resp.BestWin == nil || resp.BestWin.OpponentName != "对手" {
		t.Fatalf("a missing opponent profile must fall back to a neutral label: %+v", resp.BestWin)
	}
}

func TestMemberPerformanceKeepsWinCountsButHidesRatingBelowFiveSamples(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(4, 2)

	resp, _ := f.get(1, 3)
	if resp.Summary == nil || resp.Summary.Rating != nil || resp.Summary.Wins != 2 || resp.Summary.ExpectedWins != 2 {
		t.Fatalf("four samples: counts stay, 90-day rating hidden: %+v", resp.Summary)
	}
	if resp.Months[5].Rating == nil {
		t.Fatalf("the monthly threshold is three samples, so the current month still has a rating: %+v", resp.Months[5])
	}
}

func TestMemberPerformanceIgnoresRowsThatAreNotRatedV2SamplesOfTheGameType(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(5, 3)
	at := f.now.Add(-30 * time.Minute)
	f.seedMatches(
		seededMatch{id: 50, self: 1, opp: 150, won: true, remark: `{"v":1}`, selfBefore: 1500, oppBefore: 1500, at: at},                       // 旧规则
		seededMatch{id: 51, self: 1, opp: 151, won: true, expected: 0.1, selfBefore: 1500, oppBefore: 1500, at: at, rated: false},             // V2 不计分
		seededMatch{id: 52, self: 1, opp: 152, won: true, expected: 0.1, selfBefore: 1500, oppBefore: 1500, at: at, rated: true, gameType: 1}, // 其他球种
	)
	// 规则切换记录不是对局结算，不能进样本。
	if err := f.db.Create(&model.RankChangeLog{UserId: 1, MatchId: 0, ChangeType: model.RankChangeTypeRatingReset, GameType: 3, Result: "reset", Remark: v2Remark(true, 5, 0.01), EffectiveAt: at}).Error; err != nil {
		t.Fatalf("seed reset log: %v", err)
	}
	// 别人的记录不能混入。
	f.seedMatches(seededMatch{id: 60, self: 2, opp: 3, won: true, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: at, rated: true})

	resp, _ := f.get(1, 3)
	if resp.SampleCount != 5 || resp.Summary.Wins != 3 || resp.Summary.ExpectedWins != 2.5 {
		t.Fatalf("only the five rated V2 matches of this game type may count: %+v summary=%+v", resp, resp.Summary)
	}
	// 不计分的 V2 记录分数仍然有效，可以作为月末排位分（它是最新的一条）。
	if resp.Months[5].EndRankScore != 1510 {
		t.Fatalf("unexpected month end rating: %+v", resp.Months[5])
	}
}

func TestMemberPerformanceDropsSamplesWhoseOpponentRecordIsMissing(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(5, 3)
	if err := f.db.Where("match_id = ? AND user_id <> ?", 2, 1).Delete(&model.RankChangeLog{}).Error; err != nil {
		t.Fatalf("delete opponent row: %v", err)
	}

	resp, _ := f.get(1, 3)
	if resp.SampleCount != 4 || resp.Summary.Wins != 2 || resp.Summary.Rating != nil {
		t.Fatalf("a match without an opponent record cannot be analysed: %+v summary=%+v", resp, resp.Summary)
	}
}

func TestMemberPerformanceSummaryUsesNinetyDaysWhileCurveUsesSixMonths(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(5, 3)
	old := f.now.AddDate(0, 0, -120) // 6 月 24 日：在 6 个自然月（5 月起）内、90 天外
	tooOld := time.Date(2026, 4, 20, 10, 0, 0, 0, f.now.Location())
	f.seedMatches(
		seededMatch{id: 70, self: 1, opp: 170, won: true, expected: 0.5, selfBefore: 1400, oppBefore: 1400, at: old, rated: true},
		seededMatch{id: 71, self: 1, opp: 171, won: true, expected: 0.5, selfBefore: 1300, oppBefore: 1300, at: tooOld, rated: true},
	)

	resp, _ := f.get(1, 3)
	if resp.SampleCount != 5 || resp.Summary.Wins != 3 {
		t.Fatalf("the 90-day summary must not include older matches: %+v", resp)
	}
	var june *types.PerformanceMonthItem
	for i := range resp.Months {
		if resp.Months[i].Month == "2026-06" {
			june = &resp.Months[i]
		}
		if resp.Months[i].Month == "2026-04" {
			t.Fatalf("matches before the six-month window must not be read")
		}
	}
	if june == nil || june.SampleCount != 1 || june.EndRankScore != 1410 || june.Rating != nil {
		t.Fatalf("the curve covers older months with their own end rating: %+v", june)
	}
}

func TestMemberPerformanceLocksFreeAndExpiredUsersWithoutAnyAnalysisValue(t *testing.T) {
	for name, memberFor := range map[string]time.Duration{"free": 0, "expired": -time.Hour} {
		t.Run(name, func(t *testing.T) {
			f := newPerformanceFixture(t)
			f.seedUser(1, "我", memberFor)
			f.seedEvenMatches(23, 12) // 23 场都在本月

			resp, _ := f.get(1, 3)
			if !resp.Success || !resp.Locked || resp.SampleCount != 23 {
				t.Fatalf("locked state with the sample count: %+v", resp)
			}
			if resp.MonthPreview == nil || resp.MonthPreview.Month != "2026-10" || resp.MonthPreview.SampleCount != 23 || resp.MonthPreview.Rating == nil {
				t.Fatalf("locked users get the current month preview: %+v", resp.MonthPreview)
			}
			if resp.Summary != nil || resp.BestWin != nil || resp.WorstLoss != nil || resp.Months != nil {
				t.Fatalf("locked response must not carry any analysis value: %+v", resp)
			}

			// 序列化后的响应里也不能出现任何分析值。
			raw, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for _, key := range []string{"summary", "best_win", "worst_loss", "months"} {
				if decoded[key] != nil {
					t.Fatalf("locked JSON must have %s empty, got %v", key, decoded[key])
				}
			}
		})
	}
}

func TestMemberPerformanceUnlocksAgainAfterRenewalFromTheSameHistory(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", -time.Hour)
	f.seedEvenMatches(5, 3)

	locked, _ := f.get(1, 3)
	if !locked.Locked || locked.Summary != nil {
		t.Fatalf("expired member must be locked: %+v", locked)
	}
	renewed := f.now.Add(30 * 24 * time.Hour)
	if err := f.db.Model(&model.User{}).Where("id = ?", 1).Update("member_expires_at", renewed).Error; err != nil {
		t.Fatalf("renew: %v", err)
	}
	full, _ := f.get(1, 3)
	if full.Locked || full.Summary == nil || full.Summary.Rating == nil || full.Summary.Rating.Rating != 1678 {
		t.Fatalf("renewal must restore the full analysis from the same history: %+v", full)
	}
}

func TestMemberPerformanceMonthPreviewMatchesTheMembersCurveAndNeedsThreeSamples(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "免费用户", 0)
	f.seedUser(2, "会员", 30*24*time.Hour)
	for _, self := range []int64{1, 2} {
		var matches []seededMatch
		for i := 1; i <= 6; i++ {
			matches = append(matches, seededMatch{id: int64(self*100) + int64(i), self: self, opp: self*100 + 50 + int64(i), won: i <= 4, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: f.now.Add(-time.Duration(i) * time.Hour), rated: true})
		}
		f.seedMatches(matches...)
	}

	free, _ := f.get(1, 3)
	member, _ := f.get(2, 3)
	if free.MonthPreview.Rating == nil || member.Months[5].Rating == nil || *free.MonthPreview.Rating != *member.Months[5].Rating {
		t.Fatalf("preview must equal the curve's current month: preview=%+v curve=%+v", free.MonthPreview, member.Months[5])
	}

	few := newPerformanceFixtureNamed(t, t.Name()+"-few")
	few.seedUser(1, "免费用户", 0)
	few.seedEvenMatches(2, 1)
	resp, _ := few.get(1, 3)
	if resp.MonthPreview == nil || resp.MonthPreview.SampleCount != 2 || resp.MonthPreview.Rating != nil || resp.SampleCount != 2 {
		t.Fatalf("two matches this month must not produce a preview rating: %+v", resp)
	}
}

func TestMemberPerformanceCapsLockedSampleCountAt200(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "免费用户", 0)
	var matches []seededMatch
	for i := 1; i <= 205; i++ {
		matches = append(matches, seededMatch{id: int64(i), self: 1, opp: int64(1000 + i), won: i%2 == 0, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: f.now.Add(-time.Duration(i) * time.Minute), rated: true})
	}
	f.seedMatches(matches...)

	resp, _ := f.get(1, 3)
	if !resp.Locked || resp.SampleCount != 200 {
		t.Fatalf("sample count is capped at 200: %+v", resp)
	}
}

func TestMemberPerformanceNeverWritesRankingData(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	f.seedEvenMatches(5, 3)
	if err := f.db.Create(&model.UserRanking{UserId: 1, GameType: 3, RankScore: 1510, RankLevel: 4, TotalWins: 3, TotalLosses: 2, RatingVersion: model.RatingVersionV2, RatedMatches: 5}).Error; err != nil {
		t.Fatalf("seed ranking: %v", err)
	}
	snapshot := func() string {
		var logs []model.RankChangeLog
		var rankings []model.UserRanking
		f.db.Order("id ASC").Find(&logs)
		f.db.Order("id ASC").Find(&rankings)
		return fmt.Sprintf("%+v|%+v", logs, rankings)
	}
	before := snapshot()

	resp, _ := f.get(1, 3)
	if resp.Summary == nil || resp.Summary.Rating == nil {
		t.Fatalf("expected a performance rating: %+v", resp)
	}
	if after := snapshot(); after != before {
		t.Fatalf("the performance rating must never be written into ranking data:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestMemberPerformanceUsesAFixedNumberOfQueries(t *testing.T) {
	counts := map[string]int64{}
	for _, matchCount := range []int{5, 200} {
		f := newPerformanceFixtureNamed(t, fmt.Sprintf("%s-%d", t.Name(), matchCount))
		f.seedUser(1, "会员", 30*24*time.Hour)
		f.seedUser(2, "免费用户", 0)
		var matches []seededMatch
		for i := 1; i <= matchCount; i++ {
			for _, self := range []int64{1, 2} {
				matches = append(matches, seededMatch{id: int64(self)*10000 + int64(i), self: self, opp: self*100000 + int64(i), won: i%2 == 0, expected: 0.4 + float64(i%5)/100, selfBefore: 1500, oppBefore: 1500, at: f.now.Add(-time.Duration(i) * time.Minute), rated: true})
			}
		}
		f.seedMatches(matches...)
		_, memberQueries := f.get(1, 3)
		_, lockedQueries := f.get(2, 3)
		counts[fmt.Sprintf("member-%d", matchCount)] = memberQueries
		counts[fmt.Sprintf("locked-%d", matchCount)] = lockedQueries
	}
	// 会员：用户、本人记录、对手赛前分、对手昵称各 1 次；免费用户没有昵称查询。
	if counts["member-5"] != 4 || counts["member-200"] != 4 || counts["locked-5"] != 3 || counts["locked-200"] != 3 {
		t.Fatalf("queries must not grow with the number of matches: %v", counts)
	}
}

func TestMemberPerformanceWithoutAnyMatchesStillSucceeds(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)

	resp, queries := f.get(1, 3)
	if !resp.Success || resp.Locked || resp.SampleCount != 0 || resp.Summary == nil || resp.Summary.Rating != nil || resp.BestWin != nil || resp.WorstLoss != nil || len(resp.Months) != 6 {
		t.Fatalf("a member without matches gets an empty analysis: %+v summary=%+v", resp, resp.Summary)
	}
	if queries != 2 {
		t.Fatalf("no samples means no opponent or profile lookups, got %d queries", queries)
	}
}

func TestMemberPerformanceReadLimitMarksTruncatedMonthsIncomplete(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)
	var matches []seededMatch
	// 10 月 400 场，9 月 250 场：只读最新的 600 条，9 月最早的 50 场被截断。
	for i := 0; i < 400; i++ {
		matches = append(matches, seededMatch{id: int64(1000 + i), self: 1, opp: int64(5000 + i), won: i%2 == 0, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: f.now.Add(-time.Duration(i) * time.Minute), rated: true})
	}
	for i := 0; i < 250; i++ {
		matches = append(matches, seededMatch{id: int64(2000 + i), self: 1, opp: int64(6000 + i), won: true, expected: 0.5, selfBefore: 1500, oppBefore: 1500, at: time.Date(2026, 9, 28, 10, 0, 0, 0, f.now.Location()).Add(-time.Duration(i) * time.Minute), rated: true})
	}
	f.seedMatches(matches...)

	resp, _ := f.get(1, 3)
	oct, sep := resp.Months[5], resp.Months[4]
	if oct.Incomplete || oct.Rating == nil || oct.SampleCount != 400 {
		t.Fatalf("the newest month is read in full: %+v", oct)
	}
	if !sep.Incomplete || sep.Rating != nil || sep.SampleCount != 200 {
		t.Fatalf("the month cut by the read limit must be incomplete and have no rating: %+v", sep)
	}
	if resp.SampleCount != 200 {
		t.Fatalf("summary stays capped at 200: %+v", resp.SampleCount)
	}
}

func TestMemberPerformanceRejectsUnknownGameTypeAndMissingUser(t *testing.T) {
	f := newPerformanceFixture(t)
	f.seedUser(1, "我", 30*24*time.Hour)

	if resp, _ := f.get(1, 9); resp.Success {
		t.Fatalf("unknown game type must be rejected: %+v", resp)
	}
	if resp, _ := f.get(404, 3); resp.Success {
		t.Fatalf("unknown user must not receive an analysis: %+v", resp)
	}
	l := NewGetMemberPerformanceLogic(context.Background(), f.svcCtx)
	if resp, _ := l.GetMemberPerformance(&types.GetMemberPerformanceReq{GameType: 3}); resp.Success {
		t.Fatalf("a request without a user must fail: %+v", resp)
	}
}

func newPerformanceFixtureNamed(t *testing.T, name string) *performanceFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankChangeLog{}); err != nil {
		t.Fatalf("prepare performance schema: %v", err)
	}
	// 生产里 ServiceContext 持有长期的 RankingModel，game_type 列只在首次使用时探测一次；这里预热，避免把一次性探测算进每次请求。
	rankingModel := model.NewRankingModel(db)
	if _, err := rankingModel.ListRankChangesByUserAndGameType(0, 3, 1); err != nil {
		t.Fatalf("warm up ranking model: %v", err)
	}
	return &performanceFixture{t: t, db: db, svcCtx: &svc.ServiceContext{DB: db, RankingModel: rankingModel}, now: time.Date(2026, 10, 22, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))}
}
