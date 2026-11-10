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

func newDurationDetailSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare detail schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}, db
}

func TestGetMatchDetailReturnsDurationSeconds(t *testing.T) {
	svcCtx, db := newDurationDetailSvc(t)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(2)
	now := time.Now()
	if err := db.Create(&model.Match{
		Id: 601, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		Status: 1, CreatedAt: now.Add(-90 * time.Second),
	}).Error; err != nil {
		t.Fatalf("create ongoing match: %v", err)
	}
	endTime := now.Add(-10 * time.Second)
	if err := db.Create(&model.Match{
		Id: 602, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		Status: 2, CreatedAt: endTime.Add(-120 * time.Second), EndTime: &endTime,
	}).Error; err != nil {
		t.Fatalf("create completed match: %v", err)
	}

	load := func(matchID int64) types.MatchDetailData {
		resp, err := NewGetMatchDetailLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).
			GetMatchDetail(&types.GetMatchDetailReq{MatchId: matchID})
		if err != nil || !resp.Success {
			t.Fatalf("get match detail %d: resp=%#v err=%v", matchID, resp, err)
		}
		return resp.Match
	}

	if ongoing := load(601).DurationSeconds; ongoing < 89 || ongoing > 100 {
		t.Fatalf("ongoing duration_seconds = %d, want about 90", ongoing)
	}
	if completed := load(602).DurationSeconds; completed != 120 {
		t.Fatalf("completed duration_seconds = %d, want 120", completed)
	}
}
