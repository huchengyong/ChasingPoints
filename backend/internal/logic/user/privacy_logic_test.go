package user

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

type userPrivacySchema struct {
	Id              int64          `gorm:"primarykey"`
	Phone           *string        `gorm:"uniqueIndex;size:20"`
	Nickname        string         `gorm:"size:50;not null;default:''"`
	Avatar          string         `gorm:"size:255;not null;default:''"`
	Status          int            `gorm:"not null;default:1"`
	PushToken       string         `gorm:"size:255;not null;default:''"`
	MemberExpiresAt *time.Time     `gorm:"comment:会员到期时间"`
	HideMatchRecord bool           `gorm:"not null;default:false"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (userPrivacySchema) TableName() string {
	return "users"
}

func newUserPrivacyTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&userPrivacySchema{}); err != nil {
		t.Fatalf("prepare user privacy schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:        db,
		UserModel: model.NewUserModel(db),
	}
}

func userPrivacyCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedUserPrivacyUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()

	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func TestGetUserPrivacyReturnsDefaultHiddenState(t *testing.T) {
	svcCtx := newUserPrivacyTestSvc(t)
	seedUserPrivacyUser(t, svcCtx, &model.User{
		Id:       201,
		Nickname: "隐私用户",
	})

	logic := NewGetUserPrivacyLogic(userPrivacyCtx(201), svcCtx)
	resp, err := logic.GetUserPrivacy()
	if err != nil {
		t.Fatalf("get user privacy: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.HideMatchRecord {
		t.Fatalf("expected hide_match_record=false by default, got %#v", resp)
	}
}

func TestUpdateUserPrivacyPersistsHiddenState(t *testing.T) {
	svcCtx := newUserPrivacyTestSvc(t)
	seedUserPrivacyUser(t, svcCtx, &model.User{
		Id:       202,
		Nickname: "切换用户",
	})

	updateLogic := NewUpdateUserPrivacyLogic(userPrivacyCtx(202), svcCtx)
	updateResp, err := updateLogic.UpdateUserPrivacy(&types.UpdateUserPrivacyReq{
		HideMatchRecord: true,
	})
	if err != nil {
		t.Fatalf("update user privacy: %v", err)
	}
	if !updateResp.Success || !updateResp.HideMatchRecord {
		t.Fatalf("expected hidden state to be enabled, got %#v", updateResp)
	}

	getLogic := NewGetUserPrivacyLogic(userPrivacyCtx(202), svcCtx)
	getResp, err := getLogic.GetUserPrivacy()
	if err != nil {
		t.Fatalf("get user privacy after update: %v", err)
	}
	if !getResp.Success || !getResp.HideMatchRecord {
		t.Fatalf("expected hidden state to persist, got %#v", getResp)
	}
}
