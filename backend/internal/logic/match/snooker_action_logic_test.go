package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSnookerActionTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
}

func seedSnookerActionMatch(t *testing.T, svcCtx *svc.ServiceContext, id int64, bestOf int, refereeID int64) *model.Match {
	t.Helper()
	for _, user := range []model.User{{Id: 101, Nickname: "选手1", Status: 1}, {Id: 202, Nickname: "选手2", Status: 1}, {Id: 303, Nickname: "裁判", Status: 1}} {
		if err := svcCtx.DB.FirstOrCreate(&user, user.Id).Error; err != nil {
			t.Fatalf("seed user %d: %v", user.Id, err)
		}
	}
	opponentID := int64(202)
	match := &model.Match{
		Id:                  id,
		UserId:              101,
		OpponentId:          &opponentID,
		OpponentName:        "选手2",
		GameType:            1,
		SnookerRulesVersion: model.SnookerRulesVersionWPBSA,
		BestOfFrames:        bestOf,
		StartingActor:       1,
		MatchMode:           model.MatchModePractice,
		Visibility:          model.MatchVisibilityPrivate,
		CurrentFrameStarted: true,
		Status:              1,
		MatchTime:           time.Now(),
	}
	if refereeID > 0 {
		match.RefereeUserId = &refereeID
		now := time.Now()
		match.RefereeJoinedAt = &now
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("seed match: %v", err)
	}
	return match
}

func TestSnookerStrokeUsesAuthoritativeStateRevisionAndPerspective(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 1, 3, 0)
	logic := NewSnookerStrokeLogic(matchLogicCtx(101), svcCtx)

	resp, err := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        1,
		Actor:          1,
		Outcome:        model.SnookerOutcomePot,
		PottedReds:     1,
		ClientActionId: "stroke-red-1",
		BaseRevision:   0,
	})
	if err != nil || resp == nil || !resp.Success || !resp.Accepted {
		t.Fatalf("stroke failed: resp=%+v err=%v", resp, err)
	}
	if resp.ServerRevision != 1 || resp.Snapshot.SnookerBallOn != model.SnookerBallColorChoice ||
		resp.Snapshot.SnookerStriker != 1 || resp.Snapshot.SnookerCurrentBreak != 1 || resp.Snapshot.SnookerRedsRemaining != 14 {
		t.Fatalf("unexpected snapshot: %+v", resp.Snapshot)
	}

	stored, err := svcCtx.MatchModel.FindById(1)
	if err != nil || stored.CurrentFrameMyScore != 1 || stored.CurrentFrameOpponentScore != 0 {
		t.Fatalf("unexpected stored score: match=%+v err=%v", stored, err)
	}
	actions, _ := svcCtx.MatchModel.ListActiveActions(1)
	if len(actions) != 1 {
		t.Fatalf("expected one action, got %d", len(actions))
	}
	event, err := model.DecodeSnookerEvent(actions[0].ExtraData)
	if err != nil || event.VisitNo != 1 || event.PottedReds != 1 {
		t.Fatalf("unexpected stored event: event=%+v err=%v", event, err)
	}

	player2State, err := loadMatchWriteState(svcCtx, 202, stored)
	if err != nil {
		t.Fatalf("load player2 view: %v", err)
	}
	if player2State.Snapshot.CurrentFrameMyScore != 0 || player2State.Snapshot.CurrentFrameOpponentScore != 1 ||
		player2State.Snapshot.SnookerStriker != 1 || player2State.Snapshot.SnookerBallOn != model.SnookerBallColorChoice {
		t.Fatalf("unexpected player2 snapshot: %+v", player2State.Snapshot)
	}

	replay, _ := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        1,
		Actor:          1,
		Outcome:        model.SnookerOutcomePot,
		PottedReds:     1,
		ClientActionId: "stroke-red-1",
		BaseRevision:   0,
	})
	if !replay.Success || replay.ServerRevision != 1 {
		t.Fatalf("idempotent replay failed: %+v", replay)
	}
	mismatchedReplay, _ := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        1,
		Actor:          1,
		Outcome:        model.SnookerOutcomeNoScore,
		ClientActionId: "stroke-red-1",
		BaseRevision:   0,
	})
	if mismatchedReplay.Success || mismatchedReplay.Accepted || mismatchedReplay.Message != "操作编号已被其他动作使用" {
		t.Fatalf("mismatched idempotent payload should be rejected: %+v", mismatchedReplay)
	}

	conflict, _ := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        1,
		Actor:          1,
		Outcome:        model.SnookerOutcomeNoScore,
		ClientActionId: "stroke-conflict",
		BaseRevision:   0,
	})
	if conflict.Success || conflict.Accepted || conflict.ServerRevision != 1 {
		t.Fatalf("expected revision conflict: %+v", conflict)
	}

	wrongActor, _ := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        1,
		Actor:          2,
		Outcome:        model.SnookerOutcomeNoScore,
		BallOnValue:    7,
		ClientActionId: "stroke-wrong-actor",
		BaseRevision:   1,
	})
	if wrongActor.Success || wrongActor.ServerRevision != 1 {
		t.Fatalf("expected non-striker rejection: %+v", wrongActor)
	}
}

func TestSnookerStrokeNoScoreStartsNewVisit(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 2, 3, 0)
	logic := NewSnookerStrokeLogic(matchLogicCtx(101), svcCtx)
	resp, _ := logic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId:        2,
		Actor:          1,
		Outcome:        model.SnookerOutcomeNoScore,
		ClientActionId: "stroke-miss",
		BaseRevision:   0,
	})
	if !resp.Success || resp.Snapshot.SnookerStriker != 2 || resp.Snapshot.SnookerVisitNo != 2 || resp.Snapshot.SnookerBallOn != model.SnookerBallRed {
		t.Fatalf("unexpected no-score transition: %+v", resp)
	}
}

func TestSnookerFrameAwardEndsRoundAndAlternatesStarter(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 3, 3, 303)
	frameLogic := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx)
	resp, err := frameLogic.SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId:        3,
		Actor:          1,
		Action:         model.SnookerFrameActionAwardFrame,
		Winner:         1,
		Reason:         "裁判判局",
		ClientActionId: "award-frame-1",
		BaseRevision:   0,
	})
	if err != nil || !resp.Success || resp.Snapshot.CurrentFrameStarted || resp.Snapshot.MyScore != 1 || resp.Snapshot.Status != 1 {
		t.Fatalf("award frame failed: resp=%+v err=%v", resp, err)
	}
	rounds, _ := svcCtx.MatchModel.ListCompletedRounds(3)
	if len(rounds) != 1 || rounds[0].Winner == nil || *rounds[0].Winner != 1 || rounds[0].WinType != model.SnookerFrameEndRefereeAward {
		t.Fatalf("unexpected rounds: %+v", rounds)
	}

	nextLogic := NewStartNextRoundLogic(matchLogicCtx(303), svcCtx)
	next, err := nextLogic.StartNextRound(&types.StartNextRoundReq{MatchId: 3, ClientActionId: "round-2", BaseRevision: 1})
	if err != nil || !next.Success || !next.Snapshot.CurrentFrameStarted || next.Snapshot.SnookerStriker != 2 || next.Snapshot.StartingActor != 1 {
		t.Fatalf("start round 2 failed: resp=%+v err=%v", next, err)
	}
}

func TestSnookerBestOfOneAutoCompletesMatch(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 4, 1, 303)
	logic := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx)
	resp, err := logic.SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId:        4,
		Actor:          1,
		Action:         model.SnookerFrameActionAwardFrame,
		Winner:         1,
		Reason:         "裁判判局",
		ClientActionId: "award-match",
		BaseRevision:   0,
	})
	if err != nil || !resp.Success || resp.Snapshot.Status != 2 || resp.Snapshot.MyScore != 1 || resp.ServerRevision != 2 {
		t.Fatalf("auto finish failed: resp=%+v err=%v", resp, err)
	}
	stored, _ := svcCtx.MatchModel.FindById(4)
	if stored.Status != 2 || stored.Result == nil || *stored.Result != 1 || stored.EndTime == nil {
		t.Fatalf("unexpected completed match: %+v", stored)
	}
	if next, _ := NewStartNextRoundLogic(matchLogicCtx(303), svcCtx).StartNextRound(&types.StartNextRoundReq{MatchId: 4, ClientActionId: "too-late", BaseRevision: 2}); next.Success {
		t.Fatalf("completed match should reject next round: %+v", next)
	}

	if err := svcCtx.DB.Model(&model.Match{}).Where("id = ?", 4).Update("achievement_synced_at", nil).Error; err != nil {
		t.Fatalf("clear achievement sync marker: %v", err)
	}
	replayed, err := logic.SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId:        4,
		Actor:          1,
		Action:         model.SnookerFrameActionAwardFrame,
		Winner:         1,
		Reason:         "裁判判局",
		ClientActionId: "award-match",
		BaseRevision:   0,
	})
	if err != nil || !replayed.Success || !replayed.Accepted {
		t.Fatalf("completed action replay failed: resp=%+v err=%v", replayed, err)
	}
	stored, _ = svcCtx.MatchModel.FindById(4)
	if stored.AchievementSyncedAt == nil {
		t.Fatal("completed action replay should reconcile post-commit achievement sync")
	}
}

func TestSnookerV2GenericScoreAndFinishAreRejected(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 5, 3, 0)
	score, _ := NewMatchScoreLogic(matchLogicCtx(101), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId: 5, Actor: 1, Score: 147, ClientActionId: "legacy-score", BaseRevision: 0,
	})
	if score.Success || score.Snapshot.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
		t.Fatalf("generic score should be rejected: %+v", score)
	}
	finish, _ := NewFinishMatchLogic(matchLogicCtx(101), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 5, ClientActionId: "legacy-finish", BaseRevision: 0,
	})
	if finish.Success || finish.ServerRevision != 0 {
		t.Fatalf("generic finish should be rejected: %+v", finish)
	}
}

func TestSnookerV2UndoReplaysStrokeState(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 6, 3, 0)
	strokeLogic := NewSnookerStrokeLogic(matchLogicCtx(101), svcCtx)
	if resp, _ := strokeLogic.SnookerStroke(&types.SnookerStrokeReq{
		MatchId: 6, Actor: 1, Outcome: model.SnookerOutcomePot, PottedReds: 1, ClientActionId: "undo-red", BaseRevision: 0,
	}); !resp.Success {
		t.Fatalf("seed stroke failed: %+v", resp)
	}
	undo, err := NewMatchUndoLogic(matchLogicCtx(101), svcCtx).MatchUndo(&types.MatchUndoReq{
		MatchId: 6, ClientActionId: "undo-action", BaseRevision: 1,
	})
	if err != nil || !undo.Success || undo.Snapshot.SnookerBallOn != model.SnookerBallRed || undo.Snapshot.SnookerRedsRemaining != 15 ||
		undo.Snapshot.SnookerVisitNo != 1 || undo.Snapshot.CurrentFrameMyScore != 0 {
		t.Fatalf("undo failed: resp=%+v err=%v", undo, err)
	}
}

func TestSnookerV2UndoFrameEndingActionDeletesRound(t *testing.T) {
	svcCtx := newSnookerActionTestSvc(t)
	seedSnookerActionMatch(t, svcCtx, 7, 3, 303)
	frameLogic := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx)
	if resp, _ := frameLogic.SnookerFrameAction(&types.SnookerFrameActionReq{
		MatchId: 7, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1, Reason: "判局", ClientActionId: "end-round", BaseRevision: 0,
	}); !resp.Success {
		t.Fatalf("seed frame end failed: %+v", resp)
	}
	undo, err := NewMatchUndoLogic(matchLogicCtx(303), svcCtx).MatchUndo(&types.MatchUndoReq{
		MatchId: 7, ClientActionId: "undo-round", BaseRevision: 1,
	})
	if err != nil || !undo.Success || !undo.Snapshot.CurrentFrameStarted || undo.Snapshot.MyScore != 0 || undo.Snapshot.SnookerStriker != 1 {
		t.Fatalf("undo round failed: resp=%+v err=%v", undo, err)
	}
	rounds, _ := svcCtx.MatchModel.ListCompletedRounds(7)
	if len(rounds) != 0 {
		t.Fatalf("round should be deleted: %+v", rounds)
	}
}
