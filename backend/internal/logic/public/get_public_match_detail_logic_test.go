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

func TestGetPublicMatchDetailCurrentRoundMatchesParticipantDetail(t *testing.T) {
	svcCtx, db := newPublicScoreBreakdownSvc(t)
	seedPublicMatch(t, db, 605, 3, 2, 4, 2)
	// 已结束对局在生产库里 current_frame_started 可能仍为 true，不能再用它判断“没有进行中的局”。
	if err := db.Model(&model.Match{}).Where("id = ?", 605).Update("current_frame_started", true).Error; err != nil {
		t.Fatalf("settle completed match frame state: %v", err)
	}
	for roundNo := 1; roundNo <= 4; roundNo++ {
		winner := 1
		if roundNo%2 == 0 {
			winner = 2
		}
		if err := db.Create(&model.MatchRound{
			MatchId: 605, RoundNo: roundNo, MyScore: roundNo, OpponentScore: 4 - roundNo, Winner: &winner, WinType: "normal",
		}).Error; err != nil {
			t.Fatalf("seed completed round: %v", err)
		}
	}

	completed := loadPublicDetail(t, svcCtx, 605)
	if completed.TotalRounds != 4 || completed.CurrentRound != completed.TotalRounds {
		t.Fatalf("completed match must not point at a next round: current=%d total=%d", completed.CurrentRound, completed.TotalRounds)
	}

	seedPublicMatch(t, db, 606, 3, 1, 2, 0)
	for roundNo := 1; roundNo <= 2; roundNo++ {
		winner := 1
		if err := db.Create(&model.MatchRound{
			MatchId: 606, RoundNo: roundNo, MyScore: roundNo, OpponentScore: 0, Winner: &winner, WinType: "normal",
		}).Error; err != nil {
			t.Fatalf("seed ongoing round: %v", err)
		}
	}
	ongoing := loadPublicDetail(t, svcCtx, 606)
	if ongoing.TotalRounds != 2 || ongoing.CurrentRound != 3 {
		t.Fatalf("ongoing match must keep pointing at the next round: current=%d total=%d", ongoing.CurrentRound, ongoing.TotalRounds)
	}
}
