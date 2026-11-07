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

func TestGetFriendRequestsPaginatesInDatabaseWithConstantQueries(t *testing.T) {
	oneQueries := getFriendRequestQueryCount(t, 1)
	hundredQueries := getFriendRequestQueryCount(t, 100)
	if oneQueries != 2 || hundredQueries != 2 {
		t.Fatalf("friend requests must use one COUNT and one joined page query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func TestGetFriendRequestsReturnsDatabasePage(t *testing.T) {
	db, svcCtx := prepareFriendRequestTestSvc(t, 45)
	_ = db
	resp, err := NewGetFriendRequestsLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetFriendRequests(&types.GetFriendRequestsReq{Page: 2, PageSize: 20})
	if err != nil || !resp.Success || resp.Total != 45 || len(resp.List) != 20 {
		t.Fatalf("get request page: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].FromUserId != 26 || resp.List[len(resp.List)-1].FromUserId != 7 {
		t.Fatalf("unexpected descending request page: %#v", resp.List)
	}
}

func getFriendRequestQueryCount(t *testing.T, requestCount int) int {
	t.Helper()
	db, svcCtx := prepareFriendRequestTestSvc(t, requestCount)
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	svcCtx.FriendModel = model.NewFriendModel(db.WithContext(ctx))
	resp, err := NewGetFriendRequestsLogic(ctx, svcCtx).GetFriendRequests(&types.GetFriendRequestsReq{Page: 1, PageSize: 100})
	if err != nil || !resp.Success || resp.Total != int64(requestCount) || len(resp.List) != requestCount {
		t.Fatalf("get requests: resp=%#v err=%v", resp, err)
	}
	return int(metrics.Snapshot().SQLCount)
}

func prepareFriendRequestTestSvc(t *testing.T, requestCount int) (*gorm.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", requestCount)), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.FriendRequest{}); err != nil {
		t.Fatalf("prepare request schema: %v", err)
	}
	users := make([]model.User, 0, requestCount+1)
	users = append(users, model.User{Id: 1, Nickname: "我"})
	requests := make([]model.FriendRequest, 0, requestCount)
	base := time.Date(2026, 8, 13, 13, 0, 0, 0, time.UTC)
	for i := 0; i < requestCount; i++ {
		fromUserID := int64(i + 2)
		users = append(users, model.User{Id: fromUserID, Nickname: fmt.Sprintf("申请人%d", fromUserID), Avatar: fmt.Sprintf("%d.png", fromUserID)})
		requests = append(requests, model.FriendRequest{Id: fromUserID, FromUserId: fromUserID, ToUserId: 1, Status: 0, Message: "加个好友", CreatedAt: base.Add(time.Duration(i) * time.Second)})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&requests).Error; err != nil {
		t.Fatalf("seed requests: %v", err)
	}
	return db, &svc.ServiceContext{FriendModel: model.NewFriendModel(db)}
}
