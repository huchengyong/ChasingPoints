package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
	"gorm.io/gorm"
)

func TestCancelMatchBlocksStaleFinishOverwritingTerminalState(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	challengeId, opponentId := int64(77), int64(2002)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id: challengeId, FromUserId: 1001, ToUserId: opponentId,
		Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	match := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponentId, OpponentName: "B", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		MyScore: 1, Status: 1, MatchTime: time.Now(), ChallengeId: &challengeId,
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("seed match: %v", err)
	}
	// 结束请求在事务外读到旧比赛对象（与真实时序一致）。
	staleMatch, err := svcCtx.MatchModel.FindById(match.Id)
	if err != nil {
		t.Fatalf("load stale match: %v", err)
	}

	cancel, err := NewCancelMatchLogic(startReputationCtx(1001), svcCtx).CancelMatch(&types.CancelMatchReq{MatchId: match.Id})
	if err != nil || !cancel.Success {
		t.Fatalf("cancel match: resp=%#v err=%v", cancel, err)
	}

	// 取消提交后，携带旧 revision 的结束结算必须失败，不得覆盖终态。
	err = svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		_, settleErr := NewFinishMatchLogic(startReputationCtx(1001), svcCtx).settleMatchWithTx(tx, staleMatch, 1001, &types.FinishMatchReq{
			MatchId: match.Id, ClientActionId: "stale-finish-review", BaseRevision: staleMatch.SyncRevision,
		})
		return settleErr
	})
	if err == nil {
		t.Fatal("stale finish settlement must fail with revision conflict after cancel")
	}
	after, _ := svcCtx.MatchModel.FindById(match.Id)
	invite, _ := svcCtx.ChallengeModel.FindById(challengeId)
	if after.Status != 3 {
		t.Fatalf("match must stay cancelled: status=%d", after.Status)
	}
	if invite.Status != model.ChallengeStatusMatchCancelled {
		t.Fatalf("challenge must stay match-cancelled: status=%d", invite.Status)
	}
}

func TestChallengePlayerCanFinishWhenRefereeBound(t *testing.T) {
	opponentId, refereeId, challengeId := int64(2002), int64(3003), int64(77)
	m := &model.Match{
		UserId: 1001, OpponentId: &opponentId, RefereeUserId: &refereeId, ChallengeId: &challengeId,
		GameType: 3, MatchFormat: "free", MyScore: 1, Status: 1,
		MatchMode: model.MatchModeRanked, FinishState: model.FinishStateNone,
	}
	caps := resolveMatchViewerCapabilities(m, 1001)
	if !caps.CanFinish {
		t.Fatalf("challenge participant must finish despite referee binding: %+v", caps)
	}
	if caps.CanScore || caps.CanUndo {
		t.Fatalf("only finish is relaxed for referee-bound challenge matches: %+v", caps)
	}
	// 非约球比赛不受放宽影响。
	m.ChallengeId = nil
	if caps := resolveMatchViewerCapabilities(m, 1001); caps.CanFinish {
		t.Fatalf("non-challenge referee-bound match must block player finish: %+v", caps)
	}
}
