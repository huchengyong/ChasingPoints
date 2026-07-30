package match

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRefereeFlowTestSvc(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare referee flow schema: %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	return &svc.ServiceContext{
		DB:         db,
		Redis:      rdb,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}, mr
}

func refereeTestCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedRefereeFlowUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname string) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{
		Id:       id,
		Nickname: nickname,
		Status:   1,
	}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func seedRefereeFlowMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
}

func TestGetMatchRefereeQRCodeCreatesShortLivedJoinPayload(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:           91,
		UserId:       1001,
		OpponentId:   &opponentID,
		OpponentName: "选手乙",
		GameType:     3,
		Status:       1,
		MatchTime:    time.Now(),
	})

	resp, err := NewGetMatchRefereeQRCodeLogic(refereeTestCtx(1001), svcCtx).GetMatchRefereeQRCode(&types.GetMatchRefereeQRCodeReq{
		MatchId: 91,
	})
	if err != nil {
		t.Fatalf("get referee qrcode: %v", err)
	}
	if !resp.Success || resp.QrcodeData == "" {
		t.Fatalf("expected qrcode payload, got %#v", resp)
	}
	if resp.ExpiresInSeconds <= 0 {
		t.Fatalf("expected positive ttl, got %#v", resp)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(resp.QrcodeData), &payload); err != nil {
		t.Fatalf("unmarshal qrcode payload: %v", err)
	}
	if payload["type"] != "match_referee" {
		t.Fatalf("expected referee payload type, got %#v", payload)
	}
	if int64(payload["match_id"].(float64)) != 91 {
		t.Fatalf("expected match_id=91, got %#v", payload)
	}
	token, _ := payload["join_token"].(string)
	if token == "" {
		t.Fatalf("expected join token in payload, got %#v", payload)
	}
	if !mr.Exists(buildMatchRefereeJoinTokenKey(91)) {
		t.Fatalf("expected redis join token key to exist")
	}
}

func TestJoinMatchRefereeBindsUserAndReturnsRefereeView(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "助教丙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:           92,
		UserId:       1001,
		OpponentId:   &opponentID,
		OpponentName: "选手乙",
		GameType:     3,
		Status:       1,
		MatchTime:    time.Now(),
	})

	if err := svcCtx.Redis.Set(context.Background(), buildMatchRefereeJoinTokenKey(92), "token-92", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	resp, err := NewJoinMatchRefereeLogic(refereeTestCtx(refereeID), svcCtx).JoinMatchReferee(&types.JoinMatchRefereeReq{
		MatchId:   92,
		JoinToken: "token-92",
	})
	if err != nil {
		t.Fatalf("join match referee: %v", err)
	}
	if !resp.Success || resp.Match == nil {
		t.Fatalf("expected success response with match, got %#v", resp)
	}
	assertStructFieldEqual(t, resp.Match, "ViewerRole", "referee")
	assertStructFieldEqual(t, resp.Match, "CanScore", true)
	assertStructFieldEqual(t, resp.Match, "RefereeBound", true)
	assertStructFieldEqual(t, resp.Match, "RefereeUserId", refereeID)

	stored, err := svcCtx.MatchModel.FindById(92)
	if err != nil || stored == nil {
		t.Fatalf("reload match: %v %+v", err, stored)
	}
	if stored.RefereeUserId == nil || *stored.RefereeUserId != refereeID {
		t.Fatalf("expected stored referee id %d, got %+v", refereeID, stored.RefereeUserId)
	}
	if mr.Exists(buildMatchRefereeJoinTokenKey(92)) {
		t.Fatalf("expected join token to be consumed after referee binding")
	}
}

func TestJoinMatchRefereeRejectsPendingFinishConfirmation(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	requesterID := int64(1001)
	refereeID := int64(3003)
	requestedAt := time.Now()
	seedRefereeFlowUser(t, svcCtx, requesterID, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id: 93, UserId: requesterID, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, FinishConfirmationRequired: true,
		FinishState: model.FinishStatePendingConfirmation, FinishRequestedBy: &requesterID, FinishRequestedAt: &requestedAt,
		Status: 1, MatchTime: time.Now(),
	})
	if err := svcCtx.Redis.Set(context.Background(), buildMatchRefereeJoinTokenKey(93), "token-93", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	resp, err := NewJoinMatchRefereeLogic(refereeTestCtx(refereeID), svcCtx).JoinMatchReferee(&types.JoinMatchRefereeReq{MatchId: 93, JoinToken: "token-93"})
	if err != nil || resp.Success || resp.Message != "请先处理当前结束请求，再加入裁判" {
		t.Fatalf("expected pending finish rejection: resp=%#v err=%v", resp, err)
	}
	stored, _ := svcCtx.MatchModel.FindById(93)
	if stored == nil || stored.RefereeUserId != nil || !mr.Exists(buildMatchRefereeJoinTokenKey(93)) {
		t.Fatalf("pending finish rejection mutated referee state: stored=%+v", stored)
	}
}

func TestJoinMatchRefereeRejectsExpiredTokenWithoutBinding(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id: 94, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		Status: 1, MatchTime: time.Now(),
	})

	resp, err := NewJoinMatchRefereeLogic(refereeTestCtx(refereeID), svcCtx).JoinMatchReferee(&types.JoinMatchRefereeReq{
		MatchId: 94, JoinToken: "expired-token",
	})
	if err != nil || resp.Success || resp.Message != "裁判二维码已失效" {
		t.Fatalf("expected expired token rejection: resp=%#v err=%v", resp, err)
	}
	stored, findErr := svcCtx.MatchModel.FindById(94)
	if findErr != nil || stored == nil || stored.RefereeUserId != nil {
		t.Fatalf("expired join mutated referee state: match=%+v err=%v", stored, findErr)
	}
}

func TestCancelMatchClearsCompletionAttribution(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	legacyCompletedBy := refereeID
	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id: 95, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		RefereeUserId: &refereeID, CompletedByUserId: &legacyCompletedBy,
		CompletionSource: model.CompletionSourceReferee, Status: 1, MatchTime: time.Now(),
	})

	resp, err := NewCancelMatchLogic(refereeTestCtx(1001), svcCtx).CancelMatch(&types.CancelMatchReq{MatchId: 95})
	if err != nil || !resp.Success {
		t.Fatalf("cancel match: resp=%#v err=%v", resp, err)
	}
	stored, findErr := svcCtx.MatchModel.FindById(95)
	if findErr != nil || stored == nil {
		t.Fatalf("reload cancelled match: match=%+v err=%v", stored, findErr)
	}
	if stored.Status != 3 || stored.EndTime == nil || stored.CompletedByUserId != nil || stored.CompletionSource != model.CompletionSourceUnknown {
		t.Fatalf("cancelled match kept completion attribution: %+v", stored)
	}
}

func TestGetMatchDetailRejectsPrivateMatchForNonParticipant(t *testing.T) {
	svcCtx, mr := newRefereeFlowTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id: 96, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now(),
	})
	resp, err := NewGetMatchDetailLogic(refereeTestCtx(4004), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 96})
	if err != nil || resp.Success {
		t.Fatalf("private match detail leaked to non-participant: resp=%#v err=%v", resp, err)
	}
}
