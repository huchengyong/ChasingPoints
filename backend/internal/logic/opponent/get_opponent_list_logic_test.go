package opponent

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOpponentLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.Friend{}); err != nil {
		t.Fatalf("prepare opponent logic schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:          db,
		UserModel:   model.NewUserModel(db),
		MatchModel:  model.NewMatchModel(db),
		FriendModel: model.NewFriendModel(db),
	}
}

func opponentLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedOpponentLogicUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname string) {
	t.Helper()

	if err := svcCtx.UserModel.Create(&model.User{
		Id:       id,
		Nickname: nickname,
		Status:   1,
	}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func seedOpponentLogicMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()

	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
}

func TestGetOpponentListUsesCurrentUserByDefault(t *testing.T) {
	svcCtx := newOpponentLogicTestSvc(t)
	seedOpponentLogicUser(t, svcCtx, 101, "查看者")
	seedOpponentLogicUser(t, svcCtx, 303, "默认对手")

	win := 1
	opponentID := int64(303)
	seedOpponentLogicMatch(t, svcCtx, &model.Match{
		Id:            1,
		UserId:        101,
		OpponentId:    &opponentID,
		OpponentName:  "默认对手",
		GameType:      3,
		MyScore:       7,
		OpponentScore: 5,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
	})

	logic := NewGetOpponentListLogic(opponentLogicCtx(101), svcCtx)
	resp, err := logic.GetOpponentList(&types.GetOpponentListReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get opponent list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if len(resp.List) != 1 || resp.List[0].Id != 303 {
		t.Fatalf("expected current user opponent 303, got %#v", resp.List)
	}
}

func TestGetOpponentListAllowsViewingFriendTargetUser(t *testing.T) {
	svcCtx := newOpponentLogicTestSvc(t)
	seedOpponentLogicUser(t, svcCtx, 101, "查看者")
	seedOpponentLogicUser(t, svcCtx, 202, "好友")
	seedOpponentLogicUser(t, svcCtx, 404, "好友对手")

	if err := svcCtx.FriendModel.AddFriend(101, 202); err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	win := 1
	friendID := int64(202)
	opponentID := int64(404)
	seedOpponentLogicMatch(t, svcCtx, &model.Match{
		Id:            11,
		UserId:        202,
		OpponentId:    &opponentID,
		OpponentName:  "好友对手",
		GameType:      3,
		MyScore:       9,
		OpponentScore: 7,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
	})
	seedOpponentLogicMatch(t, svcCtx, &model.Match{
		Id:            12,
		UserId:        404,
		OpponentId:    &friendID,
		OpponentName:  "好友",
		GameType:      3,
		MyScore:       9,
		OpponentScore: 8,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC),
	})

	logic := NewGetOpponentListLogic(opponentLogicCtx(101), svcCtx)
	resp, err := logic.GetOpponentList(&types.GetOpponentListReq{
		Page:         1,
		PageSize:     20,
		TargetUserId: 202,
	})
	if err != nil {
		t.Fatalf("get friend target opponent list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 opponent entry, got %#v", resp.List)
	}
	if resp.List[0].Id != 404 || resp.List[0].Name != "好友对手" {
		t.Fatalf("expected friend opponent 404, got %#v", resp.List[0])
	}
	if resp.TotalOpponents != 1 || resp.TotalWins != 1 {
		t.Fatalf("expected friend summary 1 opponent / 1 win, got %#v", resp)
	}
}

func TestGetOpponentListRejectsNonFriendTargetUser(t *testing.T) {
	svcCtx := newOpponentLogicTestSvc(t)
	seedOpponentLogicUser(t, svcCtx, 101, "查看者")
	seedOpponentLogicUser(t, svcCtx, 909, "陌生人")

	logic := NewGetOpponentListLogic(opponentLogicCtx(101), svcCtx)
	resp, err := logic.GetOpponentList(&types.GetOpponentListReq{
		Page:         1,
		PageSize:     20,
		TargetUserId: 909,
	})
	if err != nil {
		t.Fatalf("get stranger target opponent list: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response for non-friend target, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestGetOpponentListRejectsMissingTargetUserWithGenericMessage(t *testing.T) {
	svcCtx := newOpponentLogicTestSvc(t)
	seedOpponentLogicUser(t, svcCtx, 101, "查看者")

	logic := NewGetOpponentListLogic(opponentLogicCtx(101), svcCtx)
	resp, err := logic.GetOpponentList(&types.GetOpponentListReq{
		Page:         1,
		PageSize:     20,
		TargetUserId: 9999,
	})
	if err != nil {
		t.Fatalf("get missing target opponent list: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response for missing target, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestGetOpponentListReturnsHiddenStateForFriendTargetWhoHidesMatchRecord(t *testing.T) {
	svcCtx := newOpponentLogicTestSvc(t)
	seedOpponentLogicUser(t, svcCtx, 101, "查看者")
	if err := svcCtx.UserModel.Create(&model.User{
		Id:              202,
		Nickname:        "隐藏好友",
		Status:          1,
		HideMatchRecord: true,
	}); err != nil {
		t.Fatalf("create hidden friend: %v", err)
	}

	if err := svcCtx.FriendModel.AddFriend(101, 202); err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	logic := NewGetOpponentListLogic(opponentLogicCtx(101), svcCtx)
	resp, err := logic.GetOpponentList(&types.GetOpponentListReq{
		Page:         1,
		PageSize:     20,
		TargetUserId: 202,
	})
	if err != nil {
		t.Fatalf("get hidden friend target opponent list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response for hidden friend target, got %#v", resp)
	}
	if !resp.Hidden {
		t.Fatalf("expected hidden state to be returned, got %#v", resp)
	}
	if resp.Message != "对方已隐藏战绩" {
		t.Fatalf("unexpected hidden state message: %#v", resp)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list for hidden friend target, got %#v", resp.List)
	}
}
