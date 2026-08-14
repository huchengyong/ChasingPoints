package friend

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetFriendListUsesConstantQueriesForProfiles(t *testing.T) {
	oneQueries := getFriendListQueryCount(t, 1)
	hundredQueries := getFriendListQueryCount(t, 100)
	if oneQueries != 2 || hundredQueries != 2 {
		t.Fatalf("friend list must use one COUNT and one joined page query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func getFriendListQueryCount(t *testing.T, friendCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", friendCount)), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare friend schema: %v", err)
	}
	if err := db.Create(&[]model.RankConfig{{Level: 1, Name: "新手"}, {Level: 2, Name: "进阶"}}).Error; err != nil {
		t.Fatalf("seed configs: %v", err)
	}
	users := make([]model.User, 0, friendCount+1)
	users = append(users, model.User{Id: 1, Nickname: "我"})
	friends := make([]model.Friend, 0, friendCount)
	rankings := make([]model.UserRanking, 0, friendCount)
	base := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	for i := 0; i < friendCount; i++ {
		friendID := int64(i + 2)
		users = append(users, model.User{Id: friendID, Nickname: fmt.Sprintf("好友%d", friendID), Avatar: fmt.Sprintf("%d.png", friendID)})
		friends = append(friends, model.Friend{Id: friendID, UserId: 1, FriendId: friendID, Status: 1, CreatedAt: base.Add(time.Duration(i) * time.Second)})
		rankings = append(rankings, model.UserRanking{UserId: friendID, GameType: 3, RankLevel: 2})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&friends).Error; err != nil {
		t.Fatalf("seed friends: %v", err)
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}

	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	svcCtx := &svc.ServiceContext{FriendModel: model.NewFriendModel(db.WithContext(ctx))}
	resp, err := NewGetFriendListLogic(ctx, svcCtx).GetFriendList(&types.GetFriendListReq{Page: 1, PageSize: 100})
	if err != nil || !resp.Success || resp.Total != int64(friendCount) || len(resp.List) != friendCount {
		t.Fatalf("get friends: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].RankLevel != 2 || resp.List[0].RankName != "进阶" {
		t.Fatalf("joined rank profile missing: %+v", resp.List[0])
	}
	return int(metrics.Snapshot().SQLCount)
}
