package match

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

func newMatchLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.Friend{}); err != nil {
		t.Fatalf("prepare match logic schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:          db,
		UserModel:   model.NewUserModel(db),
		MatchModel:  model.NewMatchModel(db),
		FriendModel: model.NewFriendModel(db),
	}
}

func matchLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedMatchLogicUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname string) {
	t.Helper()

	if err := svcCtx.UserModel.Create(&model.User{
		Id:       id,
		Nickname: nickname,
		Status:   1,
	}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func seedMatchLogicRecord(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()

	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
}

func seedTargetH2HFixtures(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()

	seedMatchLogicUser(t, svcCtx, 101, "查看者")
	seedMatchLogicUser(t, svcCtx, 202, "好友")
	seedMatchLogicUser(t, svcCtx, 303, "我的对手")
	seedMatchLogicUser(t, svcCtx, 404, "好友对手")
	seedMatchLogicUser(t, svcCtx, 909, "陌生人")

	if err := svcCtx.FriendModel.AddFriend(101, 202); err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	win := 1
	viewerOpponentID := int64(303)
	friendID := int64(202)
	friendOpponentID := int64(404)

	seedMatchLogicRecord(t, svcCtx, &model.Match{
		Id:            1,
		UserId:        101,
		OpponentId:    &viewerOpponentID,
		OpponentName:  "我的对手",
		GameType:      3,
		MyScore:       7,
		OpponentScore: 5,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
	})

	seedMatchLogicRecord(t, svcCtx, &model.Match{
		Id:            11,
		UserId:        202,
		OpponentId:    &friendOpponentID,
		OpponentName:  "好友对手",
		GameType:      3,
		MyScore:       9,
		OpponentScore: 7,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
	})
	seedMatchLogicRecord(t, svcCtx, &model.Match{
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
}

func TestGetH2HStatsUsesCurrentUserByDefault(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HStats(&types.H2HStatsReq{
		OpponentId: 303,
	})
	if err != nil {
		t.Fatalf("get h2h stats: %v", err)
	}
	if !resp.Success || resp.Stats == nil {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.Stats.TotalMatches != 1 || resp.Stats.MyWins != 1 || resp.Stats.OpponentWins != 0 {
		t.Fatalf("unexpected default stats response: %#v", resp)
	}
}

func TestGetH2HStatsAllowsViewingFriendTargetUser(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HStats(&types.H2HStatsReq{
		TargetUserId: 202,
		OpponentId:   404,
	})
	if err != nil {
		t.Fatalf("get friend target h2h stats: %v", err)
	}
	if !resp.Success || resp.Stats == nil {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.Opponent == nil || resp.Opponent.Id != 404 || resp.Opponent.Name != "好友对手" {
		t.Fatalf("unexpected opponent response: %#v", resp.Opponent)
	}
	if resp.Stats.TotalMatches != 2 || resp.Stats.MyWins != 1 || resp.Stats.OpponentWins != 1 {
		t.Fatalf("unexpected friend target stats response: %#v", resp)
	}
}

func TestGetH2HStatsRejectsNonFriendTargetUser(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HStats(&types.H2HStatsReq{
		TargetUserId: 909,
		OpponentId:   404,
	})
	if err != nil {
		t.Fatalf("get stranger target h2h stats: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestGetH2HStatsRejectsMissingTargetUserWithGenericMessage(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HStats(&types.H2HStatsReq{
		TargetUserId: 9999,
		OpponentId:   404,
	})
	if err != nil {
		t.Fatalf("get missing target h2h stats: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestGetH2HHistoryAllowsViewingFriendTargetUser(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HHistoryLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HHistory(&types.H2HHistoryReq{
		TargetUserId: 202,
		OpponentId:   404,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("get friend target h2h history: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.Total != 2 || len(resp.List) != 2 {
		t.Fatalf("expected 2 history items, got %#v", resp)
	}
	if resp.List[0].Id != 12 || resp.List[1].Id != 11 {
		t.Fatalf("expected newest-first history ids [12 11], got %#v", resp.List)
	}
}

func TestGetH2HHistoryRejectsNonFriendTargetUser(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HHistoryLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HHistory(&types.H2HHistoryReq{
		TargetUserId: 909,
		OpponentId:   404,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("get stranger target h2h history: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestGetH2HHistoryRejectsMissingTargetUserWithGenericMessage(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	logic := NewGetH2HHistoryLogic(matchLogicCtx(101), svcCtx)
	resp, err := logic.GetH2HHistory(&types.H2HHistoryReq{
		TargetUserId: 9999,
		OpponentId:   404,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("get missing target h2h history: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response, got %#v", resp)
	}
	if resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}
