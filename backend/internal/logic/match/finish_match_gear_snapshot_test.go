package match

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"
)

// enableGearSnapshots 给测试用的 ServiceContext 接上装备相关的表与模型。
func enableGearSnapshots(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()
	if err := testsupport.PrepareGearSchema(svcCtx.DB); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	svcCtx.GearModel = model.NewGearModel(svcCtx.DB)
}

// seedMainCue 为用户登记一根带皮头、巧克的球杆，并设为该球种的主力球杆，返回装备 ID。
func seedMainCue(t *testing.T, svcCtx *svc.ServiceContext, userID int64, gameType int, tip, chalk int64) int64 {
	t.Helper()
	item := model.UserGearItem{
		UserId: userID, Category: model.GearCategoryCue, ProductId: 1, TipProductId: tip, ChalkProductId: chalk,
		Status: model.GearItemStatusActive,
	}
	if err := svcCtx.GearModel.CreateItemWithTx(nil, &item); err != nil {
		t.Fatalf("seed cue: %v", err)
	}
	if err := svcCtx.GearModel.UpsertDefaultWithTx(nil, &model.UserGearDefault{UserId: userID, GameType: gameType, Slot: model.GearSlotCue, ItemId: item.Id}); err != nil {
		t.Fatalf("seed main cue: %v", err)
	}
	return item.Id
}

func seedMainBreak(t *testing.T, svcCtx *svc.ServiceContext, userID int64, gameType int) int64 {
	t.Helper()
	item := model.UserGearItem{UserId: userID, Category: model.GearCategoryBreakJump, ProductId: 2, Status: model.GearItemStatusActive}
	if err := svcCtx.GearModel.CreateItemWithTx(nil, &item); err != nil {
		t.Fatalf("seed break cue: %v", err)
	}
	if err := svcCtx.GearModel.UpsertDefaultWithTx(nil, &model.UserGearDefault{UserId: userID, GameType: gameType, Slot: model.GearSlotBreak, ItemId: item.Id}); err != nil {
		t.Fatalf("seed main break: %v", err)
	}
	return item.Id
}

func gearSnapshotsByUser(t *testing.T, svcCtx *svc.ServiceContext, matchID int64) map[int64]model.MatchGearSnapshot {
	t.Helper()
	rows, err := svcCtx.GearModel.FindSnapshotsByMatch(matchID)
	if err != nil {
		t.Fatalf("find gear snapshots: %v", err)
	}
	byUser := map[int64]model.MatchGearSnapshot{}
	for _, row := range rows {
		byUser[row.UserId] = row
	}
	return byUser
}

func requireBothPlayersRecorded(t *testing.T, svcCtx *svc.ServiceContext, matchID, user1, user2, cue1, cue2 int64) {
	t.Helper()
	rows := gearSnapshotsByUser(t, svcCtx, matchID)
	if len(rows) != 2 {
		t.Fatalf("expected a gear record per registered player after completion, got %+v", rows)
	}
	if rows[user1].CueItemId != cue1 || rows[user1].TipProductId != 11 || rows[user1].ChalkProductId != 21 || rows[user1].Source != model.GearSnapshotSourceAuto {
		t.Fatalf("player 1 record should hold the combination at completion: %+v", rows[user1])
	}
	if rows[user2].CueItemId != cue2 || rows[user2].TipProductId != 12 {
		t.Fatalf("player 2 record should hold the combination at completion: %+v", rows[user2])
	}
}

func TestDirectFinishRecordsGearForBothPlayers(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9701, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	cue1 := seedMainCue(t, svcCtx, 1001, 3, 11, 21)
	cue2 := seedMainCue(t, svcCtx, 2002, 3, 12, 0)
	break1 := seedMainBreak(t, svcCtx, 1001, 3)

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9701, ClientActionId: "gear-direct-9701", BaseRevision: 0,
	})
	if err != nil || !resp.Success {
		t.Fatalf("finish: resp=%#v err=%v", resp, err)
	}
	requireBothPlayersRecorded(t, svcCtx, 9701, 1001, 2002, cue1, cue2)
	if rows := gearSnapshotsByUser(t, svcCtx, 9701); rows[1001].BreakItemId != break1 || rows[2002].BreakItemId != 0 {
		t.Fatalf("the break cue is recorded for the player who set one: %+v", rows)
	}

	// 重放同一次结束请求：仍然只有每人一条
	if replay, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9701, ClientActionId: "gear-direct-9701", BaseRevision: 0,
	}); err != nil || !replay.Success {
		t.Fatalf("replay finish: resp=%#v err=%v", replay, err)
	}
	if rows := gearSnapshotsByUser(t, svcCtx, 9701); len(rows) != 2 {
		t.Fatalf("a replayed finish must not add rows: %+v", rows)
	}
}

func TestRankedConfirmedFinishRecordsGear(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9702, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModeRanked, FinishConfirmationRequired: true, Visibility: model.MatchVisibilityPublic,
		FinishState: model.FinishStateNone, Status: 1, MyScore: 2, OpponentScore: 2, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	cue1 := seedMainCue(t, svcCtx, 1001, 3, 11, 21)
	cue2 := seedMainCue(t, svcCtx, 2002, 3, 12, 0)

	request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9702, ClientActionId: "gear-request-9702", BaseRevision: 0,
	})
	if err != nil || !request.Success {
		t.Fatalf("request finish: resp=%#v err=%v", request, err)
	}
	if rows := gearSnapshotsByUser(t, svcCtx, 9702); len(rows) != 0 {
		t.Fatalf("a pending finish request must not record anything yet: %+v", rows)
	}
	confirm, err := NewConfirmFinishMatchLogic(finishReputationCtx(2002), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9702, ClientActionId: "gear-confirm-9702", BaseRevision: request.ServerRevision,
	})
	if err != nil || !confirm.Success {
		t.Fatalf("confirm finish: resp=%#v err=%v", confirm, err)
	}
	requireBothPlayersRecorded(t, svcCtx, 9702, 1001, 2002, cue1, cue2)
}

func TestFinishTimeoutRecordsGear(t *testing.T) {
	svcCtx := newFinishTimeoutTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	seedPendingRankedFinish(t, svcCtx, 9703, model.FinishRequestTTL+time.Second)
	cue1 := seedMainCue(t, svcCtx, timeoutRequesterID, 3, 11, 21)
	cue2 := seedMainCue(t, svcCtx, timeoutOpponentID, 3, 12, 0)

	if _, err := NewConfirmFinishMatchLogic(finishReputationCtx(timeoutOpponentID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{
		MatchId: 9703, ClientActionId: "late-confirm", BaseRevision: 3,
	}); err != nil {
		t.Fatalf("late confirm: %v", err)
	}
	stored, err := svcCtx.MatchModel.FindById(9703)
	if err != nil || stored.Status != 2 || stored.CompletionSource != model.CompletionSourceFinishTimeout {
		t.Fatalf("the late action must first complete the match by timeout: %+v err=%v", stored, err)
	}
	requireBothPlayersRecorded(t, svcCtx, 9703, timeoutRequesterID, timeoutOpponentID, cue1, cue2)
}

func TestSnookerAutoCompletionRecordsGearWithoutBreakCue(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	seedSnookerActionMatch(t, svcCtx, 9704, 1, 303)
	cue1 := seedMainCue(t, svcCtx, 101, 1, 11, 21)
	cue2 := seedMainCue(t, svcCtx, 202, 1, 12, 0)
	// 斯诺克没有开球冲杆，即便数据库里有这条设置也不会被记录
	seedMainBreak(t, svcCtx, 101, 1)

	resp, err := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: 9704, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1, Reason: "裁判判局",
		ClientActionId: "gear-award", BaseRevision: 0,
	})
	if err != nil || !resp.Success || resp.Snapshot.Status != 2 {
		t.Fatalf("auto finish by frame award: resp=%+v err=%v", resp, err)
	}
	requireBothPlayersRecorded(t, svcCtx, 9704, 101, 202, cue1, cue2)
	if rows := gearSnapshotsByUser(t, svcCtx, 9704); rows[101].BreakItemId != 0 {
		t.Fatalf("snooker never records a break cue: %+v", rows[101])
	}
}

func TestPoolRoundEndAutoCompletionRecordsGear(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	match := seedPoolFormatMatch(t, svcCtx, 9705, func(m *model.Match) {
		m.MatchFormat = model.MatchFormatRaceTo
		m.TargetWins = 1
	})
	cue1 := seedMainCue(t, svcCtx, 101, 3, 11, 21)
	cue2 := seedMainCue(t, svcCtx, 202, 3, 12, 0)

	// 球员只能记录对手的胜局：101 记下 202 赢了这一局，达到抢 1 后自动结束。
	resp := endPoolRound(t, svcCtx, match.Id, 101, 2, "normal", 1, "gear-race-to-1")
	if !resp.Success || resp.Snapshot.Status != 2 {
		t.Fatalf("reaching the race target should complete the practice match: %+v", resp)
	}
	requireBothPlayersRecorded(t, svcCtx, match.Id, 101, 202, cue1, cue2)
}

func TestGearRecordFailureDoesNotBlockCompletion(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	enableGearSnapshots(t, svcCtx)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9706, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	seedMainCue(t, svcCtx, 1001, 3, 11, 21)
	if err := svcCtx.DB.Exec("DROP TABLE match_gear_snapshots").Error; err != nil {
		t.Fatalf("break the snapshot table: %v", err)
	}

	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9706, ClientActionId: "gear-failure-9706", BaseRevision: 0,
	})
	if err != nil || resp == nil || !resp.Success || resp.Result != 1 {
		t.Fatalf("a failing gear write must not fail the completion: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9706)
	if err != nil || stored.Status != 2 || stored.Result == nil || *stored.Result != 1 {
		t.Fatalf("the match must be completed with its result intact: %+v err=%v", stored, err)
	}
}

func TestFinishWithoutGearModelStillCompletes(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t) // 没有注入 GearModel
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9707, UserId: 1001, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone,
		Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now(),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	resp, err := NewFinishMatchLogic(context.WithValue(context.Background(), "user_id", int64(1001)), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9707, ClientActionId: "gear-none-9707", BaseRevision: 0,
	})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("finish without gear support: resp=%#v err=%v", resp, err)
	}
}
