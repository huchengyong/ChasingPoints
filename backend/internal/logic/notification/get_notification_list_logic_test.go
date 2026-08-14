package notification

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

func TestNotificationListRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetNotificationListLogic(context.Background(), &svc.ServiceContext{}).GetNotificationList(&types.GetNotificationListReq{})
	if err != nil || resp.Success {
		t.Fatalf("unauthenticated notification list must fail: resp=%#v err=%v", resp, err)
	}
}

func TestNotificationListRejectsUnavailablePrimaryModel(t *testing.T) {
	resp, err := NewGetNotificationListLogic(context.WithValue(context.Background(), "user_id", int64(7)), &svc.ServiceContext{}).GetNotificationList(&types.GetNotificationListReq{})
	if err != nil || resp.Success {
		t.Fatalf("notification list must fail safely without its primary model: resp=%#v err=%v", resp, err)
	}
}

func TestNotificationListIncludesAuthoritativeUnreadCountAcrossFilteredPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("prepare notification schema: %v", err)
	}
	if err := db.Create(&[]model.Notification{{UserId: 7, Type: "match", Title: "未读", IsRead: 0}, {UserId: 7, Type: "friend", Title: "未读", IsRead: 0}, {UserId: 7, Type: "match", Title: "已读", IsRead: 1}, {UserId: 8, Type: "match", Title: "他人", IsRead: 0}}).Error; err != nil {
		t.Fatalf("seed notifications: %v", err)
	}
	resp, err := NewGetNotificationListLogic(context.WithValue(context.Background(), "user_id", int64(7)), &svc.ServiceContext{NotificationModel: model.NewNotificationModel(db)}).GetNotificationList(&types.GetNotificationListReq{Page: 1, PageSize: 20, Type: "match"})
	if err != nil || !resp.Success || resp.Total != 2 || len(resp.List) != 2 || resp.UnreadCount != 2 {
		t.Fatalf("get notification list: resp=%#v err=%v", resp, err)
	}
}

func TestNotificationListUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := notificationListQueryCount(t, 1)
	hundred := notificationListQueryCount(t, 100)
	if one != 3 || hundred != 3 {
		t.Fatalf("notification list must use three fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func notificationListQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("prepare notification query schema: %v", err)
	}
	items := make([]model.Notification, 0, rows)
	for index := 1; index <= rows; index++ {
		items = append(items, model.Notification{Id: int64(index), UserId: 7, Type: "match", Title: "通知", IsRead: index % 2})
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed notifications: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(7)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetNotificationListLogic(ctx, &svc.ServiceContext{NotificationModel: model.NewNotificationModel(requestDB)}).GetNotificationList(&types.GetNotificationListReq{Page: 1, PageSize: 20})
	if err != nil || !resp.Success {
		t.Fatalf("get notification list: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
