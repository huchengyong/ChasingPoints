package challenge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetPendingChallengesUsesOneJoinedQueryAndDoesNotExpireOnRead(t *testing.T) {
	oneQueries := getPendingChallengeQueryCount(t, 1)
	hundredQueries := getPendingChallengeQueryCount(t, 100)
	if oneQueries != 1 || hundredQueries != 1 {
		t.Fatalf("challenge list must use one joined query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func getPendingChallengeQueryCount(t *testing.T, challengeCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", challengeCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	users := []model.User{{Id: 1, Nickname: "我"}}
	challenges := make([]model.Challenge, 0, challengeCount+1)
	now := time.Now()
	for i := 0; i < challengeCount; i++ {
		fromUserID := int64(i + 2)
		users = append(users, model.User{Id: fromUserID, Nickname: fmt.Sprintf("挑战者%d", fromUserID)})
		challenges = append(challenges, model.Challenge{Id: fromUserID, FromUserId: fromUserID, ToUserId: 1, GameType: 3, Status: 0, ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(time.Duration(i) * time.Second)})
	}
	challenges = append(challenges, model.Challenge{Id: 1002, FromUserId: 2, ToUserId: 1, GameType: 3, Status: 0, ExpiresAt: now.Add(-time.Hour)})
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&challenges).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	svcCtx := &svc.ServiceContext{ChallengeModel: model.NewChallengeModel(db.WithContext(ctx))}
	resp, err := NewGetPendingChallengesLogic(ctx, svcCtx).GetPendingChallenges()
	if err != nil || !resp.Success || len(resp.List) != challengeCount {
		t.Fatalf("get challenges: resp=%#v err=%v", resp, err)
	}
	var expired model.Challenge
	if err := db.First(&expired, 1002).Error; err != nil || expired.Status != 0 {
		t.Fatalf("GET must not mutate expired challenge: challenge=%+v err=%v", expired, err)
	}
	return int(metrics.Snapshot().SQLCount)
}

func TestGetPendingChallengesDoesNotWriteBusinessRows(t *testing.T) {
	recorder := testsupport.NewSQLWriteRecorder()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: recorder})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "挑战者"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.Challenge{
		{Id: 1, FromUserId: 2, ToUserId: 1, GameType: 3, Status: 0, ExpiresAt: now.Add(time.Hour)},
		{Id: 2, FromUserId: 2, ToUserId: 1, GameType: 3, Status: 0, ExpiresAt: now.Add(-time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}
	recorder.Reset()
	svcCtx := &svc.ServiceContext{ChallengeModel: model.NewChallengeModel(db)}
	resp, err := NewGetPendingChallengesLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetPendingChallenges()
	if err != nil || !resp.Success || len(resp.List) != 1 {
		t.Fatalf("get pending challenges: resp=%#v err=%v", resp, err)
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("challenge GET must not issue writes: %q", writes)
	}
}

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
