package user

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFavoriteVenueRewardUserTestSvc(t *testing.T) *svc.ServiceContext {
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

func seedFavoriteVenueRewardConfig(t *testing.T, svcCtx *svc.ServiceContext, enabled bool) {
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

func seedRewardUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) *model.User {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func seedRewardVenue(t *testing.T, svcCtx *svc.ServiceContext, venue *model.Venue) *model.Venue {
	t.Helper()
	if err := svcCtx.VenueModel.Create(venue); err != nil {
		t.Fatalf("create venue: %v", err)
	}
	return venue
}

func userRewardCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestGetFavoriteVenueRewardStatusReturnsNotStartedForEligibleNewUser(t *testing.T) {
	svcCtx := newFavoriteVenueRewardUserTestSvc(t)
	seedFavoriteVenueRewardConfig(t, svcCtx, true)
	seedRewardUser(t, svcCtx, &model.User{
		Id:        101,
		Nickname:  "新用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})

	logic := NewGetFavoriteVenueRewardStatusLogic(userRewardCtx(101), svcCtx)
	resp, err := logic.GetFavoriteVenueRewardStatus()
	if err != nil {
		t.Fatalf("get reward status: %v", err)
	}
	if !resp.Success || resp.Status != "not_started" {
		t.Fatalf("expected not_started status, got %#v", resp)
	}
}

func TestGetFavoriteVenueRewardStatusReturnsPendingReviewForSubmittedVenue(t *testing.T) {
	svcCtx := newFavoriteVenueRewardUserTestSvc(t)
	seedFavoriteVenueRewardConfig(t, svcCtx, true)
	seedRewardUser(t, svcCtx, &model.User{
		Id:        102,
		Nickname:  "待审核用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})
	seedRewardVenue(t, svcCtx, &model.Venue{
		Id:          201,
		Name:        "星轨台球",
		Address:     "科技园 1 号",
		City:        "深圳",
		District:    "南山区",
		FullAddress: "深圳南山区科技园1号",
		OwnerUserId: 102,
		Status:      model.VenueStatusPending,
		GeoStatus:   model.VenueGeoStatusPending,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
	})

	logic := NewGetFavoriteVenueRewardStatusLogic(userRewardCtx(102), svcCtx)
	resp, err := logic.GetFavoriteVenueRewardStatus()
	if err != nil {
		t.Fatalf("get reward status: %v", err)
	}
	if !resp.Success || resp.Status != "pending_review" || resp.SubmittedVenueId != 201 {
		t.Fatalf("expected pending_review status, got %#v", resp)
	}
}

func TestGetFavoriteVenueRewardStatusReturnsRewardGrantedWhenRecordExists(t *testing.T) {
	svcCtx := newFavoriteVenueRewardUserTestSvc(t)
	seedFavoriteVenueRewardConfig(t, svcCtx, true)
	memberExpiresAt := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	seedRewardUser(t, svcCtx, &model.User{
		Id:              103,
		Nickname:        "已到账用户",
		CreatedAt:       time.Now().Add(-2 * 24 * time.Hour),
		MemberExpiresAt: &memberExpiresAt,
	})
	if err := svcCtx.FavoriteVenueRewardRecordModel.Create(&model.FavoriteVenueRewardRecord{
		ActivityKey:          model.FavoriteVenueRewardActivityKey,
		UserId:               103,
		VenueId:              202,
		RewardDays:           30,
		MemberExpiresAtAfter: memberExpiresAt,
		GrantedAt:            time.Now(),
	}); err != nil {
		t.Fatalf("create reward record: %v", err)
	}

	logic := NewGetFavoriteVenueRewardStatusLogic(userRewardCtx(103), svcCtx)
	resp, err := logic.GetFavoriteVenueRewardStatus()
	if err != nil {
		t.Fatalf("get reward status: %v", err)
	}
	if !resp.Success || resp.Status != "reward_granted" || resp.MemberExpiresAt == "" {
		t.Fatalf("expected reward_granted status, got %#v", resp)
	}
}

func TestGetFavoriteVenueRewardStatusReturnsRejectedWithReason(t *testing.T) {
	svcCtx := newFavoriteVenueRewardUserTestSvc(t)
	seedFavoriteVenueRewardConfig(t, svcCtx, true)
	seedRewardUser(t, svcCtx, &model.User{
		Id:        104,
		Nickname:  "被拒用户",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	})
	seedRewardVenue(t, svcCtx, &model.Venue{
		Id:           203,
		Name:         "旧街角台球",
		Address:      "人民路 9 号",
		City:         "广州",
		District:     "天河区",
		FullAddress:  "广州天河区人民路9号",
		OwnerUserId:  104,
		Status:       model.VenueStatusRejected,
		GeoStatus:    model.VenueGeoStatusFailed,
		RejectReason: "地址不完整",
		CreatedAt:    time.Now().Add(-24 * time.Hour),
	})

	logic := NewGetFavoriteVenueRewardStatusLogic(userRewardCtx(104), svcCtx)
	resp, err := logic.GetFavoriteVenueRewardStatus()
	if err != nil {
		t.Fatalf("get reward status: %v", err)
	}
	if !resp.Success || resp.Status != "rejected" || resp.RejectReason != "地址不完整" {
		t.Fatalf("expected rejected status with reason, got %#v", resp)
	}
}

func TestGetFavoriteVenueRewardStatusKeepsVenueRewardAvailableForOlderUsers(t *testing.T) {
	svcCtx := newFavoriteVenueRewardUserTestSvc(t)
	seedFavoriteVenueRewardConfig(t, svcCtx, true)
	seedRewardUser(t, svcCtx, &model.User{
		Id:        105,
		Nickname:  "过期用户",
		CreatedAt: time.Now().Add(-15 * 24 * time.Hour),
	})

	logic := NewGetFavoriteVenueRewardStatusLogic(userRewardCtx(105), svcCtx)
	resp, err := logic.GetFavoriteVenueRewardStatus()
	if err != nil {
		t.Fatalf("get reward status: %v", err)
	}
	if !resp.Success || resp.Status != "not_started" {
		t.Fatalf("expected not_started status, got %#v", resp)
	}
}
