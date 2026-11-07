package public

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

func TestGetPublicMatchDetailShowsRefereeOnlyForPublicMatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}); err != nil {
		t.Fatalf("prepare public detail schema: %v", err)
	}
	refereeID := int64(3003)
	opponentID := int64(2002)
	joinedAt := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()
	for _, user := range []model.User{
		{Id: 1001, Nickname: "选手甲", Avatar: "p1.png", Status: 1},
		{Id: opponentID, Nickname: "选手乙", Avatar: "p2.png", Status: 1},
		{Id: refereeID, Nickname: "裁判丙", Avatar: "referee.png", Status: 1},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	if err := db.Create(&model.Match{
		Id: 301, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, Status: 2,
		RefereeUserId: &refereeID, RefereeJoinedAt: &joinedAt, EndTime: &endTime,
		CompletedByUserId: &refereeID, CompletionSource: model.CompletionSourceReferee,
		MatchTime: joinedAt.Add(-20 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create public match: %v", err)
	}
	if err := db.Create(&model.Match{
		Id: 302, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate, Status: 2,
		RefereeUserId: &refereeID, RefereeJoinedAt: &joinedAt, EndTime: &endTime,
		CompletedByUserId: &refereeID, CompletionSource: model.CompletionSourceReferee,
		MatchTime: joinedAt.Add(-20 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create private match: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db, UserModel: model.NewUserModel(db), MatchModel: model.NewMatchModel(db)}
	logic := NewGetPublicMatchDetailLogic(context.Background(), svcCtx)

	publicResp, err := logic.GetPublicMatchDetail(&types.GetPublicMatchDetailReq{MatchId: 301})
	if err != nil || !publicResp.Success || publicResp.Match == nil {
		t.Fatalf("load public match: resp=%#v err=%v", publicResp, err)
	}
	if publicResp.Match.RefereeName != "裁判丙" || publicResp.Match.RefereeAvatar != "referee.png" || publicResp.Match.CompletedByUserId != refereeID || publicResp.Match.CompletionSource != model.CompletionSourceReferee {
		t.Fatalf("public referee detail missing: %+v", publicResp.Match)
	}

	privateResp, err := logic.GetPublicMatchDetail(&types.GetPublicMatchDetailReq{MatchId: 302})
	if err != nil || privateResp.Success {
		t.Fatalf("private match must not be available publicly: resp=%#v err=%v", privateResp, err)
	}
}
