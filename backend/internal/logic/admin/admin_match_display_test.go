package admin

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

func newAdminMatchDisplayTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
}

func intPtr(v int) *int {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestAdminGetMatchListUsesObjectivePlayerNames(t *testing.T) {
	svcCtx := newAdminMatchDisplayTestSvc(t)
	matchTime := time.Date(2026, 4, 9, 16, 41, 0, 0, time.FixedZone("UTC+8", 8*3600))

	if err := svcCtx.UserModel.Create(&model.User{Id: 101, Nickname: "用户1001"}); err != nil {
		t.Fatalf("create player1 user: %v", err)
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:           1,
		UserId:       101,
		OpponentId:   int64Ptr(202),
		GameType:     2,
		OpponentName: "用户1194",
		MyScore:      31,
		OpponentScore: 50,
		Status:       2,
		Result:       intPtr(2),
		MatchTime:    matchTime,
		CreatedAt:    matchTime,
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	logic := NewAdminGetMatchListLogic(context.Background(), svcCtx)
	resp, err := logic.AdminGetMatchList(&types.AdminMatchListReq{Page: 1, PageSize: 20, Status: -1, GameType: -1})
	if err != nil {
		t.Fatalf("get admin match list: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected one match row, got %#v", resp.List)
	}

	row := resp.List[0]
	if row.Player1Name != "用户1001" || row.Player2Name != "用户1194" {
		t.Fatalf("admin list should show real player names, got %#v", row)
	}
	if row.Player1Name == "我" || row.Player2Name == "我" {
		t.Fatalf("admin list should not contain viewer-centric name, got %#v", row)
	}
	if row.WinnerName == "我" {
		t.Fatalf("admin winner should not contain viewer-centric name, got %#v", row)
	}
}

func TestAdminGetRecentMatchesUsesObjectivePlayerNames(t *testing.T) {
	svcCtx := newAdminMatchDisplayTestSvc(t)
	matchTime := time.Date(2026, 4, 9, 16, 9, 0, 0, time.FixedZone("UTC+8", 8*3600))

	if err := svcCtx.UserModel.Create(&model.User{Id: 101, Nickname: "用户1001"}); err != nil {
		t.Fatalf("create recent player1 user: %v", err)
	}

	if err := svcCtx.MatchModel.Create(&model.Match{
		Id:            2,
		UserId:        101,
		OpponentId:    int64Ptr(202),
		GameType:      3,
		OpponentName:  "用户1194",
		MyScore:       7,
		OpponentScore: 0,
		Status:        2,
		Result:        intPtr(1),
		MatchTime:     matchTime,
		CreatedAt:     matchTime,
	}); err != nil {
		t.Fatalf("create recent match: %v", err)
	}

	logic := NewAdminGetRecentMatchesLogic(context.Background(), svcCtx)
	resp, err := logic.AdminGetRecentMatches()
	if err != nil {
		t.Fatalf("get admin recent matches: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected one recent match row, got %#v", resp.List)
	}

	row := resp.List[0]
	if row.Player1Name != "用户1001" || row.Player2Name != "用户1194" {
		t.Fatalf("admin recent matches should show real player names, got %#v", row)
	}
	if row.Player1Name == "我" || row.Player2Name == "我" {
		t.Fatalf("admin recent matches should not contain viewer-centric name, got %#v", row)
	}
}
