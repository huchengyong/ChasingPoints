package match

import (
	"context"
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

func newRefereePreviewTestSvc(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}); err != nil {
		t.Fatalf("prepare referee preview schema: %v", err)
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

func refereePreviewCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestRefereePreviewReturnsMatchInfoWithValidToken(t *testing.T) {
	svcCtx, mr := newRefereePreviewTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Now()

	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判丙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:            101,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		Status:        1,
		MyScore:       3,
		OpponentScore: 2,
		MatchTime:     now,
	})

	tokenKey := buildMatchRefereeJoinTokenKey(101)
	if err := svcCtx.Redis.Set(context.Background(), tokenKey, "valid-token", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	resp, err := NewRefereePreviewLogic(refereePreviewCtx(refereeID), svcCtx).RefereePreview(&types.RefereePreviewReq{
		MatchId:   101,
		JoinToken: "valid-token",
	})
	if err != nil {
		t.Fatalf("RefereePreview: %v", err)
	}
	if !resp.Success || resp.Preview == nil {
		t.Fatalf("expected success with preview, got %#v", resp)
	}
	if resp.Preview.Player1Name != "选手甲" {
		t.Fatalf("expected player1 name '选手甲', got '%s'", resp.Preview.Player1Name)
	}
	if resp.Preview.Player2Name != "选手乙" {
		t.Fatalf("expected player2 name '选手乙', got '%s'", resp.Preview.Player2Name)
	}
	if resp.Preview.Player1Score != 3 || resp.Preview.Player2Score != 2 {
		t.Fatalf("expected scores 3:2, got %d:%d", resp.Preview.Player1Score, resp.Preview.Player2Score)
	}
}

func TestRefereePreviewRejectsInvalidToken(t *testing.T) {
	svcCtx, mr := newRefereePreviewTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Now()

	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判丙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:            102,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		Status:        1,
		MyScore:       0,
		OpponentScore: 0,
		MatchTime:     now,
	})

	resp, err := NewRefereePreviewLogic(refereePreviewCtx(refereeID), svcCtx).RefereePreview(&types.RefereePreviewReq{
		MatchId:   102,
		JoinToken: "wrong-token",
	})
	if err != nil {
		t.Fatalf("RefereePreview: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure for wrong token, got %#v", resp)
	}
	if resp.Message != "裁判码已过期或无效，请重新扫码" {
		t.Fatalf("expected token invalid message, got '%s'", resp.Message)
	}
}

func TestRefereePreviewRejectsFinishedMatch(t *testing.T) {
	svcCtx, mr := newRefereePreviewTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	refereeID := int64(3003)
	now := time.Now()
	win := 1

	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判丙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:            103,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		Status:        2,
		Result:        &win,
		MyScore:       5,
		OpponentScore: 1,
		MatchTime:     now,
	})

	tokenKey := buildMatchRefereeJoinTokenKey(103)
	if err := svcCtx.Redis.Set(context.Background(), tokenKey, "old-token", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	resp, err := NewRefereePreviewLogic(refereePreviewCtx(refereeID), svcCtx).RefereePreview(&types.RefereePreviewReq{
		MatchId:   103,
		JoinToken: "old-token",
	})
	if err != nil {
		t.Fatalf("RefereePreview: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure for finished match, got %#v", resp)
	}
	if resp.Message != "对局已结束或已取消" {
		t.Fatalf("expected finished message, got '%s'", resp.Message)
	}
}

func TestRefereePreviewRejectsMatchAlreadyHasReferee(t *testing.T) {
	svcCtx, mr := newRefereePreviewTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	existingRefereeID := int64(4004)
	refereeID := int64(3003)
	now := time.Now()

	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowUser(t, svcCtx, refereeID, "裁判丙")
	seedRefereeFlowUser(t, svcCtx, existingRefereeID, "已有裁判")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:             104,
		UserId:         1001,
		OpponentId:     &opponentID,
		OpponentName:   "选手乙",
		GameType:       3,
		Status:         1,
		RefereeUserId:  &existingRefereeID,
		RefereeJoinedAt: &now,
		MyScore:        1,
		OpponentScore:  1,
		MatchTime:      now,
	})

	tokenKey := buildMatchRefereeJoinTokenKey(104)
	if err := svcCtx.Redis.Set(context.Background(), tokenKey, "fresh-token", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	resp, err := NewRefereePreviewLogic(refereePreviewCtx(refereeID), svcCtx).RefereePreview(&types.RefereePreviewReq{
		MatchId:   104,
		JoinToken: "fresh-token",
	})
	if err != nil {
		t.Fatalf("RefereePreview: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure when match already has referee, got %#v", resp)
	}
}

func TestRefereePreviewRejectsSelfReferee(t *testing.T) {
	svcCtx, mr := newRefereePreviewTestSvc(t)
	defer mr.Close()

	opponentID := int64(2002)
	now := time.Now()

	seedRefereeFlowUser(t, svcCtx, 1001, "选手甲")
	seedRefereeFlowUser(t, svcCtx, opponentID, "选手乙")
	seedRefereeFlowMatch(t, svcCtx, &model.Match{
		Id:            105,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "选手乙",
		GameType:      3,
		Status:        1,
		MyScore:       2,
		OpponentScore: 2,
		MatchTime:     now,
	})

	tokenKey := buildMatchRefereeJoinTokenKey(105)
	if err := svcCtx.Redis.Set(context.Background(), tokenKey, "token-105", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed join token: %v", err)
	}

	// Player1 tries to be referee of their own match
	resp, err := NewRefereePreviewLogic(refereePreviewCtx(1001), svcCtx).RefereePreview(&types.RefereePreviewReq{
		MatchId:   105,
		JoinToken: "token-105",
	})
	if err != nil {
		t.Fatalf("RefereePreview: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure for self-referring, got %#v", resp)
	}
}
