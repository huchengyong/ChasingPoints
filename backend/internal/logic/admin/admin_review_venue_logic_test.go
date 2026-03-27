package admin

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminReviewVenueTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFavoriteVenueRewardSchema(db); err != nil {
		t.Fatalf("prepare favorite venue reward schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                             db,
		UserModel:                      model.NewUserModel(db),
		VenueModel:                     model.NewVenueModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
		FavoriteVenueRewardRecordModel: model.NewFavoriteVenueRewardRecordModel(db),
	}
}

func seedReviewRewardConfig(t *testing.T, svcCtx *svc.ServiceContext, enabled bool) {
	t.Helper()

	if err := svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.FavoriteVenueRewardActivityKey,
		Enabled:           enabled,
		PopupEnabled:      enabled,
		RewardDays:        30,
		NewUserWindowDays: 7,
	}); err != nil {
		t.Fatalf("seed reward config: %v", err)
	}
}

func seedReviewUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func seedReviewVenue(t *testing.T, svcCtx *svc.ServiceContext, venue *model.Venue) {
	t.Helper()
	if err := svcCtx.VenueModel.Create(venue); err != nil {
		t.Fatalf("create venue: %v", err)
	}
}

func TestAdminReviewVenueApprovesAndGrantsReward(t *testing.T) {
	svcCtx := newAdminReviewVenueTestSvc(t)
	seedReviewRewardConfig(t, svcCtx, true)
	seedReviewUser(t, svcCtx, &model.User{
		Id:        301,
		Nickname:  "奖励用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})
	seedReviewVenue(t, svcCtx, &model.Venue{
		Id:          401,
		Name:        "南山球馆",
		Address:     "高新南一道",
		City:        "深圳",
		District:    "南山区",
		FullAddress: "深圳南山区高新南一道",
		OwnerUserId: 301,
		Status:      model.VenueStatusPending,
		GeoStatus:   model.VenueGeoStatusSuccess,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
	})

	logic := NewAdminReviewVenueLogic(context.Background(), svcCtx)
	resp, err := logic.AdminReviewVenue(&types.AdminVenueReviewReq{
		VenueId: 401,
		Status:  1,
	})
	if err != nil {
		t.Fatalf("approve venue: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected success approve resp, got %#v", resp)
	}

	venue, err := svcCtx.VenueModel.FindById(401)
	if err != nil {
		t.Fatalf("find updated venue: %v", err)
	}
	if venue == nil || venue.Status != model.VenueStatusPublished {
		t.Fatalf("expected published venue, got %#v", venue)
	}

	user, err := svcCtx.UserModel.FindById(301)
	if err != nil {
		t.Fatalf("find rewarded user: %v", err)
	}
	if user == nil || user.MemberExpiresAt == nil {
		t.Fatalf("expected rewarded member expiry, got %#v", user)
	}

	record, err := svcCtx.FavoriteVenueRewardRecordModel.FindByActivityAndUser(model.FavoriteVenueRewardActivityKey, 301)
	if err != nil {
		t.Fatalf("find reward record: %v", err)
	}
	if record == nil || record.VenueId != 401 {
		t.Fatalf("expected reward record for venue 401, got %#v", record)
	}
}

func TestAdminReviewVenueSkipsRewardWhenConfigDisabled(t *testing.T) {
	svcCtx := newAdminReviewVenueTestSvc(t)
	seedReviewRewardConfig(t, svcCtx, false)
	seedReviewUser(t, svcCtx, &model.User{
		Id:        302,
		Nickname:  "无奖励用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})
	seedReviewVenue(t, svcCtx, &model.Venue{
		Id:          402,
		Name:        "福田球馆",
		Address:     "中心路 8 号",
		City:        "深圳",
		District:    "福田区",
		FullAddress: "深圳福田区中心路8号",
		OwnerUserId: 302,
		Status:      model.VenueStatusPending,
		GeoStatus:   model.VenueGeoStatusSuccess,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
	})

	logic := NewAdminReviewVenueLogic(context.Background(), svcCtx)
	resp, err := logic.AdminReviewVenue(&types.AdminVenueReviewReq{
		VenueId: 402,
		Status:  1,
	})
	if err != nil {
		t.Fatalf("approve venue with disabled reward: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected approval success, got %#v", resp)
	}

	user, err := svcCtx.UserModel.FindById(302)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.MemberExpiresAt != nil {
		t.Fatalf("expected no member expiry when reward disabled, got %#v", user.MemberExpiresAt)
	}

	record, err := svcCtx.FavoriteVenueRewardRecordModel.FindByActivityAndUser(model.FavoriteVenueRewardActivityKey, 302)
	if err != nil {
		t.Fatalf("find reward record: %v", err)
	}
	if record != nil {
		t.Fatalf("expected no reward record, got %#v", record)
	}
}

func TestAdminReviewVenueRejectStoresReason(t *testing.T) {
	svcCtx := newAdminReviewVenueTestSvc(t)
	seedReviewRewardConfig(t, svcCtx, true)
	seedReviewUser(t, svcCtx, &model.User{
		Id:        303,
		Nickname:  "拒绝用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})
	seedReviewVenue(t, svcCtx, &model.Venue{
		Id:          403,
		Name:        "老城球馆",
		Address:     "建设路 2 号",
		City:        "广州",
		District:    "越秀区",
		FullAddress: "广州越秀区建设路2号",
		OwnerUserId: 303,
		Status:      model.VenueStatusPending,
		GeoStatus:   model.VenueGeoStatusFailed,
		CreatedAt:   time.Now().Add(-12 * time.Hour),
	})

	logic := NewAdminReviewVenueLogic(context.Background(), svcCtx)
	resp, err := logic.AdminReviewVenue(&types.AdminVenueReviewReq{
		VenueId:      403,
		Status:       3,
		RejectReason: "地址信息不完整",
	})
	if err != nil {
		t.Fatalf("reject venue: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected reject response success, got %#v", resp)
	}

	venue, err := svcCtx.VenueModel.FindById(403)
	if err != nil {
		t.Fatalf("find rejected venue: %v", err)
	}
	if venue == nil || venue.Status != model.VenueStatusRejected || venue.RejectReason != "地址信息不完整" {
		t.Fatalf("expected rejected venue with reason, got %#v", venue)
	}
}
