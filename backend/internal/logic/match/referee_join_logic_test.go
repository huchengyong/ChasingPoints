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
		DB:        db,
		Redis:     rdb,
		UserModel: model.NewUserModel(db),
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
