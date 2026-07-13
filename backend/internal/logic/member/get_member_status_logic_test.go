package member

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberStatusTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.MemberGrowthProfile{}, &model.MemberGrowthLog{}); err != nil {
		t.Fatalf("prepare member status schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
		MemberGrowthLogModel:     model.NewMemberGrowthLogModel(db),
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func memberStatusCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestGetMemberStatusReturnsGrowthSnapshot(t *testing.T) {
	svcCtx := newMemberStatusTestSvc(t)
	now := time.Now().In(time.FixedZone("UTC+8", 8*3600))
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	expiresAt := time.Date(2099, 4, 11, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))

	if err := svcCtx.UserModel.Create(&model.User{
		Id:              1001,
		Nickname:        "会员用户",
		MemberExpiresAt: &expiresAt,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.MemberGrowthProfileModel.CreateWithTx(nil, &model.MemberGrowthProfile{
		UserId:           1001,
		GrowthPoints:     78,
		GrowthLevel:      3,
		TodayGrowthCount: 3,
		TodayGrowthDate:  timePtr(day),
		LastGrowthAt:     timePtr(day.Add(12 * time.Hour)),
	}); err != nil {
		t.Fatalf("create growth profile: %v", err)
	}

	logic := NewGetMemberStatusLogic(memberStatusCtx(1001), svcCtx)
	resp, err := logic.GetMemberStatus()
	if err != nil {
		t.Fatalf("get member status: %v", err)
	}
	if !resp.Success || !resp.IsActive {
		t.Fatalf("expected active member status, got %#v", resp)
	}
	if resp.GrowthLevel != 3 || resp.GrowthPoints != 78 || resp.TodayGrowthCount != 3 || resp.GrowthDailyCap != 5 {
		t.Fatalf("unexpected growth payload: %#v", resp)
	}
	if resp.GrowthFrozen {
		t.Fatalf("expected active member growth not frozen: %#v", resp)
	}
	if resp.NextGrowthLevel != 4 || resp.NextGrowthLevelPoints != 260 || resp.RemainingGrowthPoints != 182 {
		t.Fatalf("unexpected next growth payload: %#v", resp)
	}
}

func TestGetMemberStatusReturnsFrozenGrowthForExpiredMember(t *testing.T) {
	svcCtx := newMemberStatusTestSvc(t)
	expiresAt := time.Date(2000, 4, 11, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))

	if err := svcCtx.UserModel.Create(&model.User{
		Id:              1002,
		Nickname:        "过期会员",
		MemberExpiresAt: &expiresAt,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.MemberGrowthProfileModel.CreateWithTx(nil, &model.MemberGrowthProfile{
		UserId:       1002,
		GrowthPoints: 78,
		GrowthLevel:  3,
	}); err != nil {
		t.Fatalf("create growth profile: %v", err)
	}

	logic := NewGetMemberStatusLogic(memberStatusCtx(1002), svcCtx)
	resp, err := logic.GetMemberStatus()
	if err != nil {
		t.Fatalf("get member status: %v", err)
	}
	if !resp.Success || resp.IsActive {
		t.Fatalf("expected expired member status, got %#v", resp)
	}
	if !resp.GrowthFrozen || resp.GrowthLevel != 3 || resp.GrowthPoints != 78 {
		t.Fatalf("unexpected frozen growth payload: %#v", resp)
	}
}
