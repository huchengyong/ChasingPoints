package challenge

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetPendingChallengesReturnsLinkedMatchID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db, UserModel: model.NewUserModel(db), ChallengeModel: model.NewChallengeModel(db)}
	for _, user := range []model.User{{Id: 1001, Nickname: "发起人"}, {Id: 2002, Nickname: "对手"}} {
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	linkedMatchID := int64(88)
	for _, challenge := range []model.Challenge{
		{Id: 1, FromUserId: 1001, ToUserId: 2002, GameType: 3, Status: 1, ExpiresAt: time.Now().Add(time.Hour)},
		{Id: 2, FromUserId: 1001, ToUserId: 2002, GameType: 3, Status: 1, MatchId: &linkedMatchID, ExpiresAt: time.Now().Add(time.Hour)},
	} {
		if err := svcCtx.ChallengeModel.Create(&challenge); err != nil {
			t.Fatalf("create challenge: %v", err)
		}
	}
	ctx := context.WithValue(context.Background(), "user_id", int64(1001))
	resp, err := NewGetPendingChallengesLogic(ctx, svcCtx).GetPendingChallenges()
	if err != nil || !resp.Success || len(resp.List) != 2 {
		t.Fatalf("get challenges: resp=%#v err=%v", resp, err)
	}
	matchIDs := map[int64]int64{}
	for _, item := range resp.List {
		matchIDs[item.Id] = item.MatchId
	}
	if matchIDs[1] != 0 || matchIDs[2] != linkedMatchID {
		t.Fatalf("unexpected challenge match ids: %#v", matchIDs)
	}
}
