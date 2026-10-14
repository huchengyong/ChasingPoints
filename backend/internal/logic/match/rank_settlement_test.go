package match

import (
	"context"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

const (
	rankFixtureMatchID     = int64(9301)
	rankFixtureChallengeID = int64(701)
	rankFixtureCreator     = int64(1001)
	rankFixtureOpponent    = int64(2002)
)

// newRankSettlementFixture 创建约球比赛：小李(1001) 5:3 领先小王(2002)，等待被结束。
func newRankSettlementFixture(t *testing.T, mode string, myScore, opponentScore int) *svc.ServiceContext {
	t.Helper()
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(
		&model.Challenge{},
		&model.MatchParticipantResult{},
		&model.UserCompetitiveStats{},
		&model.UserOpponentStats{},
		&model.UserOpponentStrengthBucket{},
		&model.UserNotificationPreference{},
	); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	svcCtx.ChallengeModel = model.NewChallengeModel(svcCtx.DB)
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	svcCtx.UserNotificationPreferenceModel = model.NewUserNotificationPreferenceModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: rankFixtureCreator, Nickname: "小李"},
		model.User{Id: rankFixtureOpponent, Nickname: "小王"},
	)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: rankFixtureChallengeID, FromUserId: rankFixtureCreator, ToUserId: rankFixtureOpponent,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	opponent, challengeID := rankFixtureOpponent, rankFixtureChallengeID
	visibility := model.MatchVisibilityPublic
	if mode == model.MatchModePractice {
		visibility = model.MatchVisibilityPrivate
	}
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: rankFixtureMatchID, UserId: rankFixtureCreator, OpponentId: &opponent, OpponentName: "小王",
		ChallengeId: &challengeID, GameType: 3, MatchMode: mode, Visibility: visibility,
		Status: 1, MyScore: myScore, OpponentScore: opponentScore, CurrentFrameStarted: true,
		FinishState: model.FinishStateNone, MatchTime: time.Now().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	previousHub := ws.GlobalHub
	ws.GlobalHub = ws.NewHub()
	t.Cleanup(func() { ws.GlobalHub = previousHub })
	return svcCtx
}

func finishRankFixture(t *testing.T, svcCtx *svc.ServiceContext, userID int64, actionID string) *types.FinishMatchResp {
	t.Helper()
	resp, err := NewFinishMatchLogic(finishReputationCtx(userID), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: rankFixtureMatchID, ClientActionId: actionID, BaseRevision: 0,
	})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("finish match: resp=%+v err=%v", resp, err)
	}
	return resp
}

func loadRankFixtureMatch(t *testing.T, svcCtx *svc.ServiceContext) *model.Match {
	t.Helper()
	match, err := svcCtx.MatchModel.FindById(rankFixtureMatchID)
	if err != nil || match == nil {
		t.Fatalf("load match: %v", err)
	}
	return match
}

func countRankChangeLogs(t *testing.T, svcCtx *svc.ServiceContext) int64 {
	t.Helper()
	var count int64
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", rankFixtureMatchID).Count(&count).Error; err != nil {
		t.Fatalf("count rank logs: %v", err)
	}
	return count
}

func disputeRankFixture(svcCtx *svc.ServiceContext, userID int64, actionID string) *types.RankResultDisputeResp {
	resp, _ := NewDisputeRankResultLogic(finishReputationCtx(userID), svcCtx).DisputeRankResult(&types.RankResultDisputeReq{
		MatchId: rankFixtureMatchID, ClientActionId: actionID,
	})
	return resp
}

func setRankFixtureDue(t *testing.T, svcCtx *svc.ServiceContext, due time.Time) {
	t.Helper()
	if err := svcCtx.DB.Model(&model.Match{}).Where("id = ?", rankFixtureMatchID).Update("rank_settlement_due_at", due).Error; err != nil {
		t.Fatalf("move due time: %v", err)
	}
}

func notificationsFor(t *testing.T, svcCtx *svc.ServiceContext, userID int64) []string {
	t.Helper()
	var rows []model.Notification
	if err := svcCtx.DB.Where("user_id = ?", userID).Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	contents := make([]string, 0, len(rows))
	for _, row := range rows {
		contents = append(contents, row.Content)
	}
	return contents
}

func TestChallengeRankedManualFinishDefersRankSettlement(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	before := time.Now()
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-defer")

	match := loadRankFixtureMatch(t, svcCtx)
	if match.Status != 2 || match.Result == nil || *match.Result != 1 {
		t.Fatalf("match must complete immediately: %+v", match)
	}
	if match.CompletedByUserId == nil || *match.CompletedByUserId != rankFixtureCreator {
		t.Fatalf("finisher must be recorded: %+v", match.CompletedByUserId)
	}
	if match.RankSettlementStatus != model.RankSettlementPending || match.RankSettlementDueAt == nil {
		t.Fatalf("settlement must be pending: status=%q due=%v", match.RankSettlementStatus, match.RankSettlementDueAt)
	}
	if due := *match.RankSettlementDueAt; due.Before(before.Add(model.RankDisputeWindow-time.Second)) || due.After(time.Now().Add(model.RankDisputeWindow+time.Second)) {
		t.Fatalf("due time must be end + %v, got %v", model.RankDisputeWindow, due)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 0 {
		t.Fatalf("no rank change may be written before the window ends, got %d logs", got)
	}
	if match.AchievementSyncedAt != nil {
		t.Fatalf("rank achievements must wait for the settlement, not be synced at completion")
	}
	challenge, err := svcCtx.ChallengeModel.FindById(rankFixtureChallengeID)
	if err != nil || challenge == nil || challenge.Status != model.ChallengeStatusCompleted {
		t.Fatalf("challenge must be completed immediately so both can book again: %+v err=%v", challenge, err)
	}
	for _, userID := range []int64{rankFixtureCreator, rankFixtureOpponent} {
		participant, findErr := svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(rankFixtureMatchID, userID)
		if findErr != nil || participant == nil || !participant.RankExcluded {
			t.Fatalf("history row must exist but stay out of rank stats for %d: %+v err=%v", userID, participant, findErr)
		}
		if stats, statsErr := svcCtx.CompetitiveReadModel.FindStats(userID, 0); statsErr != nil || stats != nil {
			t.Fatalf("rank stats must not move before settlement for %d: %+v err=%v", userID, stats, statsErr)
		}
	}
	if !resolveMatchViewerCapabilities(match, rankFixtureOpponent).CanDisputeResult {
		t.Fatalf("the other player must be able to dispute")
	}
	if resolveMatchViewerCapabilities(match, rankFixtureCreator).CanDisputeResult {
		t.Fatalf("the finisher must not be able to dispute")
	}
	if got := notificationsFor(t, svcCtx, rankFixtureOpponent); len(got) != 1 || got[0] != "小李结束了对局 · 3:5，10分钟内可提出异议" {
		t.Fatalf("other player must get exactly one finisher notice with the window: %v", got)
	}
	if got := notificationsFor(t, svcCtx, rankFixtureCreator); len(got) != 1 || !strings.Contains(got[0], "你的对局已结束") {
		t.Fatalf("finisher keeps the normal result notice: %v", got)
	}
}

func TestChallengeFinishesThatAreNotDeferred(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		my, opp    int
		wantStatus string
		wantNotice string
	}{
		{name: "practice", mode: model.MatchModePractice, my: 5, opp: 3, wantStatus: model.RankSettlementNone, wantNotice: "小李结束了对局 · 3:5"},
		{name: "ranked draw", mode: model.MatchModeRanked, my: 4, opp: 4, wantStatus: model.RankSettlementNone, wantNotice: "小李结束了对局 · 4:4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := newRankSettlementFixture(t, tc.mode, tc.my, tc.opp)
			finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-"+tc.name)
			match := loadRankFixtureMatch(t, svcCtx)
			if match.RankSettlementStatus != tc.wantStatus || match.RankSettlementDueAt != nil {
				t.Fatalf("status=%q due=%v", match.RankSettlementStatus, match.RankSettlementDueAt)
			}
			if resolveMatchViewerCapabilities(match, rankFixtureOpponent).CanDisputeResult {
				t.Fatalf("no dispute entry without a window")
			}
			if resp := disputeRankFixture(svcCtx, rankFixtureOpponent, "dispute-"+tc.name); resp.Success {
				t.Fatalf("dispute must be rejected: %+v", resp)
			}
			if got := notificationsFor(t, svcCtx, rankFixtureOpponent); len(got) != 1 || got[0] != tc.wantNotice {
				t.Fatalf("the other player is told who finished, without a dispute hint: %v", got)
			}
		})
	}
}

func TestRankSettlementAppliesExactlyOnceAfterWindow(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-settle")
	match := loadRankFixtureMatch(t, svcCtx)
	due := *match.RankSettlementDueAt

	// 未到期：不处理。
	if _, _, kind, err := settleDueRankMatch(context.Background(), svcCtx, match.Id, due.Add(-time.Second)); err != nil || kind != rankSettlementSkipped {
		t.Fatalf("not due yet must skip: kind=%v err=%v", kind, err)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 0 {
		t.Fatalf("must not settle early: %d", got)
	}

	settled, settlement, kind, err := settleDueRankMatch(context.Background(), svcCtx, match.Id, due.Add(time.Second))
	if err != nil || kind != rankSettlementApplied || settled == nil || settlement.Result != 1 {
		t.Fatalf("due settlement must apply: kind=%v settlement=%+v err=%v", kind, settlement, err)
	}
	stored := loadRankFixtureMatch(t, svcCtx)
	if stored.RankSettlementStatus != model.RankSettlementSettled {
		t.Fatalf("status = %q", stored.RankSettlementStatus)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 2 {
		t.Fatalf("both players need exactly one rank log, got %d", got)
	}
	for _, userID := range []int64{rankFixtureCreator, rankFixtureOpponent} {
		stats, statsErr := svcCtx.CompetitiveReadModel.FindStats(userID, 0)
		if statsErr != nil || stats == nil || stats.TotalMatches != 1 {
			t.Fatalf("rank stats must count the match once for %d: %+v err=%v", userID, stats, statsErr)
		}
		participant, _ := svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(match.Id, userID)
		if participant == nil || participant.RankExcluded {
			t.Fatalf("participant must join rank scope after settlement: %+v", participant)
		}
	}

	// 重复/并发调用不重复结算。
	if _, _, kind, err := settleDueRankMatch(context.Background(), svcCtx, match.Id, due.Add(time.Minute)); err != nil || kind != rankSettlementSkipped {
		t.Fatalf("second run must skip: kind=%v err=%v", kind, err)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 2 {
		t.Fatalf("rank logs must stay at 2, got %d", got)
	}
	if stats, _ := svcCtx.CompetitiveReadModel.FindStats(rankFixtureCreator, 0); stats == nil || stats.TotalMatches != 1 {
		t.Fatalf("stats must not double count: %+v", stats)
	}
}

func TestRankSettlementWorkerSettlesDueMatchesWithoutAnyUserOnline(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-worker")
	worker := NewRankSettlementWorker(svcCtx)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if got := loadRankFixtureMatch(t, svcCtx).RankSettlementStatus; got != model.RankSettlementPending {
		t.Fatalf("worker must leave an open window alone, status=%q", got)
	}

	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Minute))
	for i := 0; i < 2; i++ {
		if err := worker.RunOnce(context.Background()); err != nil {
			t.Fatalf("run worker: %v", err)
		}
	}
	if got := loadRankFixtureMatch(t, svcCtx).RankSettlementStatus; got != model.RankSettlementSettled {
		t.Fatalf("worker must settle the due match, status=%q", got)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 2 {
		t.Fatalf("repeated worker runs must settle once, got %d logs", got)
	}
}

func TestRankDisputeVoidsSettlementAndKeepsRecord(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-dispute")

	if resp := disputeRankFixture(svcCtx, rankFixtureCreator, "d-self"); resp.Success || !strings.Contains(resp.Message, "结束方") {
		t.Fatalf("finisher cannot dispute their own result: %+v", resp)
	}
	if resp := disputeRankFixture(svcCtx, 3003, "d-stranger"); resp.Success || !strings.Contains(resp.Message, "不是本场") {
		t.Fatalf("non participants cannot dispute: %+v", resp)
	}
	if got := loadRankFixtureMatch(t, svcCtx).RankSettlementStatus; got != model.RankSettlementPending {
		t.Fatalf("rejected requests must not change state: %q", got)
	}

	resp := disputeRankFixture(svcCtx, rankFixtureOpponent, "d-ok")
	if !resp.Success || resp.RankSettlementStatus != model.RankSettlementVoided {
		t.Fatalf("dispute must succeed: %+v", resp)
	}
	match := loadRankFixtureMatch(t, svcCtx)
	if match.Status != 2 || match.RankSettlementStatus != model.RankSettlementVoided || match.MyScore != 5 || match.OpponentScore != 3 {
		t.Fatalf("record and score must be kept, match must stay completed: %+v", match)
	}
	if match.AchievementSyncedAt == nil {
		t.Fatalf("a voided match is treated as non-ranked: no rank achievement events, sync marked complete")
	}
	challenge, _ := svcCtx.ChallengeModel.FindById(rankFixtureChallengeID)
	if challenge == nil || challenge.Status != model.ChallengeStatusCompleted {
		t.Fatalf("dispute must not reopen the booking: %+v", challenge)
	}
	var actions int64
	svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "result_dispute").Count(&actions)
	if actions != 1 {
		t.Fatalf("one audit action expected, got %d", actions)
	}

	// 幂等重放成功，不重复写；不同请求标识被拒。
	if replay := disputeRankFixture(svcCtx, rankFixtureOpponent, "d-ok"); !replay.Success {
		t.Fatalf("same client action must replay: %+v", replay)
	}
	if again := disputeRankFixture(svcCtx, rankFixtureOpponent, "d-again"); again.Success || !strings.Contains(again.Message, "已提出异议") {
		t.Fatalf("second dispute must be rejected: %+v", again)
	}
	svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "result_dispute").Count(&actions)
	if actions != 1 {
		t.Fatalf("replay must not write another action, got %d", actions)
	}

	// 到期后 worker 不得再结算作废的比赛，也不计入排位口径。
	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Minute))
	if err := NewRankSettlementWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("worker: %v", err)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 0 {
		t.Fatalf("voided match must never settle, got %d logs", got)
	}
	for _, userID := range []int64{rankFixtureCreator, rankFixtureOpponent} {
		participant, _ := svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(match.Id, userID)
		if participant == nil || !participant.RankExcluded {
			t.Fatalf("voided history row must stay outside rank scope: %+v", participant)
		}
		if stats, _ := svcCtx.CompetitiveReadModel.FindStats(userID, 0); stats != nil {
			t.Fatalf("voided match must not count in rank stats: %+v", stats)
		}
		recent, err := svcCtx.CompetitiveReadModel.ListRecentParticipantResults(userID, 0, 10)
		if err != nil || len(recent) != 0 {
			t.Fatalf("rank trend must exclude the voided match: %+v err=%v", recent, err)
		}
	}
	if got := notificationsFor(t, svcCtx, rankFixtureCreator); len(got) != 2 || got[1] != "小王对本场结果提出异议，本场不计排位积分" {
		t.Fatalf("finisher must be told about the dispute: %v", got)
	}
}

func TestRankDisputeRejectedAfterWindowEvenBeforeWorkerRuns(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-late")
	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Second))

	if resp := disputeRankFixture(svcCtx, rankFixtureOpponent, "late"); resp.Success || !strings.Contains(resp.Message, "异议期已结束") {
		t.Fatalf("a late dispute must be rejected: %+v", resp)
	}
	match := loadRankFixtureMatch(t, svcCtx)
	if match.RankSettlementStatus != model.RankSettlementPending {
		t.Fatalf("rejection must not change state: %q", match.RankSettlementStatus)
	}
	if resolveMatchViewerCapabilities(match, rankFixtureOpponent).CanDisputeResult {
		t.Fatalf("no dispute entry after the window")
	}
	// 之后 worker 正常生效；生效后同样被拒。
	if err := NewRankSettlementWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("worker: %v", err)
	}
	if resp := disputeRankFixture(svcCtx, rankFixtureOpponent, "after-settle"); resp.Success || !strings.Contains(resp.Message, "已生效") {
		t.Fatalf("settled results cannot be disputed: %+v", resp)
	}
}

func TestRankSettlementSkipsAndVoidsWhenAPlayerAccountIsGone(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-gone")
	if err := svcCtx.DB.Delete(&model.User{}, rankFixtureOpponent).Error; err != nil {
		t.Fatalf("delete opponent: %v", err)
	}
	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Minute))
	if err := NewRankSettlementWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("worker: %v", err)
	}
	if got := loadRankFixtureMatch(t, svcCtx).RankSettlementStatus; got != model.RankSettlementVoided {
		t.Fatalf("a missing account must not stall or rewrite ranks, status=%q", got)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 0 {
		t.Fatalf("no rank log for a missing account, got %d", got)
	}
}

func TestMatchDetailReportsRankSettlementWithoutWriting(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-detail")
	recorder := testsupport.NewSQLWriteRecorder()
	readDB := svcCtx.DB.Session(&gorm.Session{Logger: recorder})
	svcCtx.MatchModel = model.NewMatchModel(readDB)
	svcCtx.UserModel = model.NewUserModel(readDB)
	recorder.Reset()

	detail := func(userID int64) types.MatchDetailData {
		t.Helper()
		resp, err := NewGetMatchDetailLogic(finishReputationCtx(userID), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: rankFixtureMatchID})
		if err != nil || resp == nil || !resp.Success {
			t.Fatalf("detail for %d: resp=%+v err=%v", userID, resp, err)
		}
		return resp.Match
	}
	creator, opponent := detail(rankFixtureCreator), detail(rankFixtureOpponent)
	for name, d := range map[string]types.MatchDetailData{"creator": creator, "opponent": opponent} {
		if d.RankSettlementStatus != model.RankSettlementPending || d.RankSettlementDueAt == "" || d.CompletedByUserId != rankFixtureCreator {
			t.Fatalf("%s must see the pending settlement and who finished: %+v", name, d)
		}
		if d.MyRankChange != 0 || d.OpponentRankChange != 0 {
			t.Fatalf("%s must not see an estimated rank change before settlement: %d/%d", name, d.MyRankChange, d.OpponentRankChange)
		}
	}
	if creator.CanDisputeResult || !opponent.CanDisputeResult {
		t.Fatalf("only the player who did not finish may dispute: creator=%v opponent=%v", creator.CanDisputeResult, opponent.CanDisputeResult)
	}
	if _, err := time.Parse(time.RFC3339, opponent.RankSettlementDueAt); err != nil {
		t.Fatalf("due time must be RFC3339: %q", opponent.RankSettlementDueAt)
	}

	// 到期但 worker 尚未处理：读取仍如实返回待生效，且不得顺手结算或改表。
	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Minute))
	recorder.Reset()
	late := detail(rankFixtureOpponent)
	if late.RankSettlementStatus != model.RankSettlementPending || late.CanDisputeResult {
		t.Fatalf("overdue reads must stay pending and close the dispute entry: %+v", late)
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("detail must be a pure read, got writes: %v", writes)
	}
	if got := countRankChangeLogs(t, svcCtx); got != 0 {
		t.Fatalf("a read must not settle the match, got %d logs", got)
	}
}

// 上赛季最后时刻结束的比赛，排位在下一赛季生效：赛季记录只更新生效时刻所在的赛季。
func TestRankSettlementAssignsSeasonByEffectiveTime(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	if err := svcCtx.DB.AutoMigrate(&model.Season{}, &model.SeasonRecord{}); err != nil {
		t.Fatalf("prepare season schema: %v", err)
	}
	svcCtx.SeasonModel = model.NewSeasonModel(svcCtx.DB)
	svcCtx.SeasonRecordModel = model.NewSeasonRecordModel(svcCtx.DB)
	location, err := seasonx.LocationForConfig(svcCtx.Config.SeasonLifecycle)
	if err != nil {
		t.Fatalf("season location: %v", err)
	}
	today := time.Now().In(location)
	day := func(offset int) time.Time {
		return time.Date(today.Year(), today.Month(), today.Day()+offset, 0, 0, 0, 0, time.UTC)
	}
	for _, season := range []model.Season{
		{Id: 1, Name: "上赛季", StartDate: day(-30), EndDate: day(0), Status: 1},
		{Id: 2, Name: "新赛季", StartDate: day(1), EndDate: day(30), Status: 1},
	} {
		s := season
		if err := svcCtx.SeasonModel.Create(&s); err != nil {
			t.Fatalf("seed season %d: %v", s.Id, err)
		}
	}

	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-season")
	// 生效时刻落在次日（新赛季）；完成时刻仍在上赛季。
	effective := time.Date(today.Year(), today.Month(), today.Day()+1, 12, 0, 0, 0, location)
	setRankFixtureDue(t, svcCtx, effective)
	if _, _, kind, err := settleDueRankMatch(context.Background(), svcCtx, rankFixtureMatchID, effective.Add(time.Second)); err != nil || kind != rankSettlementApplied {
		t.Fatalf("settle: kind=%v err=%v", kind, err)
	}
	for _, userID := range []int64{rankFixtureCreator, rankFixtureOpponent} {
		if old, err := svcCtx.SeasonRecordModel.FindBySeasonAndUser(1, userID); err == nil && old != nil && old.MatchesPlayed != 0 {
			t.Fatalf("finished season must stay untouched for %d: %+v", userID, old)
		}
		record, err := svcCtx.SeasonRecordModel.FindBySeasonAndUser(2, userID)
		if err != nil || record == nil || record.MatchesPlayed != 1 {
			t.Fatalf("new season must receive the match for %d: %+v err=%v", userID, record, err)
		}
	}
}

// 异议作废的比赛不得更新任何赛季记录。
func TestRankDisputeKeepsSeasonRecordsUntouched(t *testing.T) {
	svcCtx := newRankSettlementFixture(t, model.MatchModeRanked, 5, 3)
	if err := svcCtx.DB.AutoMigrate(&model.Season{}, &model.SeasonRecord{}); err != nil {
		t.Fatalf("prepare season schema: %v", err)
	}
	svcCtx.SeasonModel = model.NewSeasonModel(svcCtx.DB)
	svcCtx.SeasonRecordModel = model.NewSeasonRecordModel(svcCtx.DB)
	now := time.Now()
	if err := svcCtx.SeasonModel.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}); err != nil {
		t.Fatalf("seed season: %v", err)
	}
	finishRankFixture(t, svcCtx, rankFixtureCreator, "finish-season-void")
	if resp := disputeRankFixture(svcCtx, rankFixtureOpponent, "void-season"); !resp.Success {
		t.Fatalf("dispute: %+v", resp)
	}
	setRankFixtureDue(t, svcCtx, time.Now().Add(-time.Minute))
	if err := NewRankSettlementWorker(svcCtx).RunOnce(context.Background()); err != nil {
		t.Fatalf("worker: %v", err)
	}
	for _, userID := range []int64{rankFixtureCreator, rankFixtureOpponent} {
		if record, err := svcCtx.SeasonRecordModel.FindBySeasonAndUser(1, userID); err == nil && record != nil && record.MatchesPlayed != 0 {
			t.Fatalf("voided match must not enter season records for %d: %+v", userID, record)
		}
	}
}
