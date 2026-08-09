package auth

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/sms"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSmsBindTestSvc(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phone TEXT UNIQUE,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 1,
			push_token TEXT NOT NULL DEFAULT '',
			member_expires_at DATETIME,
			hide_match_record BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		);
		CREATE TABLE user_oauth (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			open_id TEXT NOT NULL,
			union_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, open_id)
		);
	`).Error; err != nil {
		t.Fatalf("prepare sms bind schema: %v", err)
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "sms-bind-test-secret"
	cfg.Auth.AccessExpire = 3600
	cfg.Auth.RefreshExpire = 7200

	return &svc.ServiceContext{
		DB:          db,
		Config:      cfg,
		UserModel:   model.NewUserModel(db),
		OauthModel:  model.NewUserOauthModel(db),
		CodeManager: sms.NewCodeManager(rdb),
	}, mr
}

func TestSmsBindPhoneMergesIntoPhoneAccountAndReturnsTargetSession(t *testing.T) {
	phone := "13800138000"
	svcCtx, _ := newSmsBindTestSvc(t)
	ctx := context.Background()

	sourceUser := &model.User{Nickname: "OAuth用户", Status: 1}
	if err := svcCtx.UserModel.Create(sourceUser); err != nil {
		t.Fatalf("create source user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   sourceUser.Id,
		Provider: "oauth_sms",
		OpenId:   "sms-openid-merge",
	}); err != nil {
		t.Fatalf("create source oauth: %v", err)
	}

	targetUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(targetUser); err != nil {
		t.Fatalf("create target user: %v", err)
	}
	if err := svcCtx.CodeManager.SaveCode(ctx, phone, "123456"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), "user_id", int64(sourceUser.Id))
	resp, err := NewBindPhoneLogic(reqCtx, svcCtx).BindPhone(&types.BindPhoneReq{Phone: phone, SmsCode: "123456"})
	if err != nil {
		t.Fatalf("bind phone merge: %v", err)
	}
	if !resp.Success || !resp.MergedAccount || resp.NeedBindPhone {
		t.Fatalf("unexpected merge response: %#v", resp)
	}
	if resp.Message != "账号已合并" {
		t.Fatalf("replacement session must not ask the user to log in again: %q", resp.Message)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.UserInfo == nil || resp.UserInfo.Id != targetUser.Id {
		t.Fatalf("expected target user session, got %#v", resp)
	}

	refreshUserID, err := parseRefreshTokenUserId(resp.RefreshToken, svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		t.Fatalf("decode refresh token: %v", err)
	}
	if refreshUserID != targetUser.Id {
		t.Fatalf("refresh token user_id = %d, want %d", refreshUserID, targetUser.Id)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId("oauth_sms", "sms-openid-merge")
	if err != nil {
		t.Fatalf("find migrated oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != targetUser.Id {
		t.Fatalf("expected oauth moved to target user, got %#v", oauth)
	}

	deletedUser, err := svcCtx.UserModel.FindById(sourceUser.Id)
	if err != nil {
		t.Fatalf("find deleted source user: %v", err)
	}
	if deletedUser != nil {
		t.Fatalf("expected source user deleted, got %#v", deletedUser)
	}

	activeTarget, err := svcCtx.UserModel.FindById(targetUser.Id)
	if err != nil {
		t.Fatalf("find target user: %v", err)
	}
	if activeTarget == nil || activeTarget.Status != 1 {
		t.Fatalf("expected target user still active, got %#v", activeTarget)
	}
}

func TestSmsBindPhoneRejectsDisabledMergeTargetWithoutChangingSource(t *testing.T) {
	phone := "13500135000"
	svcCtx, _ := newSmsBindTestSvc(t)
	ctx := context.Background()

	sourceUser := &model.User{Nickname: "OAuth用户", Status: 1}
	if err := svcCtx.UserModel.Create(sourceUser); err != nil {
		t.Fatalf("create source user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   sourceUser.Id,
		Provider: "oauth_sms",
		OpenId:   "sms-openid-disabled-target",
	}); err != nil {
		t.Fatalf("create source oauth: %v", err)
	}

	targetUser := &model.User{Phone: &phone, Nickname: "停用手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(targetUser); err != nil {
		t.Fatalf("create target user: %v", err)
	}
	if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", targetUser.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("disable target user: %v", err)
	}
	if err := svcCtx.CodeManager.SaveCode(ctx, phone, "222222"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), "user_id", int64(sourceUser.Id))
	resp, err := NewBindPhoneLogic(reqCtx, svcCtx).BindPhone(&types.BindPhoneReq{Phone: phone, SmsCode: "222222"})
	if err != nil {
		t.Fatalf("bind phone with disabled target: %v", err)
	}
	if resp.Success || resp.MergedAccount || resp.AccessToken != "" || resp.RefreshToken != "" {
		t.Fatalf("expected disabled target merge rejection, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId("oauth_sms", "sms-openid-disabled-target")
	if err != nil {
		t.Fatalf("find source oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != sourceUser.Id {
		t.Fatalf("oauth ownership changed for disabled target: %#v", oauth)
	}
	sourceAfter, err := svcCtx.UserModel.FindById(sourceUser.Id)
	if err != nil {
		t.Fatalf("find retained source user: %v", err)
	}
	if sourceAfter == nil {
		t.Fatal("source user was deleted for a disabled merge target")
	}
}

func TestSmsBindPhoneWithoutMergeKeepsCurrentUserAndSession(t *testing.T) {
	phone := "13900139000"
	svcCtx, _ := newSmsBindTestSvc(t)
	ctx := context.Background()

	user := &model.User{Nickname: "当前用户", Status: 1}
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.CodeManager.SaveCode(ctx, phone, "654321"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), "user_id", int64(user.Id))
	resp, err := NewBindPhoneLogic(reqCtx, svcCtx).BindPhone(&types.BindPhoneReq{Phone: phone, SmsCode: "654321"})
	if err != nil {
		t.Fatalf("bind phone: %v", err)
	}
	if !resp.Success || resp.MergedAccount || resp.NeedBindPhone {
		t.Fatalf("unexpected normal binding response: %#v", resp)
	}
	if resp.AccessToken != "" || resp.RefreshToken != "" || resp.UserInfo != nil {
		t.Fatalf("normal binding must not replace session: %#v", resp)
	}

	boundUser, err := svcCtx.UserModel.FindById(user.Id)
	if err != nil {
		t.Fatalf("find bound user: %v", err)
	}
	if boundUser == nil || boundUser.Phone == nil || *boundUser.Phone != phone {
		t.Fatalf("expected the same user to stay bound, got %#v", boundUser)
	}
}

func TestSmsBindPhoneRollsBackMergeOnDeleteFailure(t *testing.T) {
	phone := "13600136000"
	svcCtx, _ := newSmsBindTestSvc(t)
	ctx := context.Background()

	sourceUser := &model.User{Nickname: "OAuth用户", Status: 1}
	if err := svcCtx.UserModel.Create(sourceUser); err != nil {
		t.Fatalf("create source user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{
		UserId:   sourceUser.Id,
		Provider: "oauth_sms",
		OpenId:   "sms-openid-rollback",
	}); err != nil {
		t.Fatalf("create source oauth: %v", err)
	}

	targetUser := &model.User{Phone: &phone, Nickname: "手机号用户", Status: 1}
	if err := svcCtx.UserModel.Create(targetUser); err != nil {
		t.Fatalf("create target user: %v", err)
	}
	if err := svcCtx.CodeManager.SaveCode(ctx, phone, "111111"); err != nil {
		t.Fatalf("save sms code: %v", err)
	}
	if err := svcCtx.DB.Exec(`
		CREATE TRIGGER reject_sms_user_soft_delete
		BEFORE UPDATE OF deleted_at ON users
		WHEN NEW.deleted_at IS NOT NULL
		BEGIN
			SELECT RAISE(ABORT, 'soft delete blocked');
		END;
	`).Error; err != nil {
		t.Fatalf("create soft delete trigger: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), "user_id", int64(sourceUser.Id))
	resp, err := NewBindPhoneLogic(reqCtx, svcCtx).BindPhone(&types.BindPhoneReq{Phone: phone, SmsCode: "111111"})
	if err != nil {
		t.Fatalf("bind phone with forced delete failure: %v", err)
	}
	if resp.Success || resp.MergedAccount {
		t.Fatalf("expected merge failure without a partial write, got %#v", resp)
	}

	oauth, err := svcCtx.OauthModel.FindByProviderAndOpenId("oauth_sms", "sms-openid-rollback")
	if err != nil {
		t.Fatalf("find unchanged oauth: %v", err)
	}
	if oauth == nil || oauth.UserId != sourceUser.Id {
		t.Fatalf("oauth migration was not rolled back: %#v", oauth)
	}

	retainedUser, err := svcCtx.UserModel.FindById(sourceUser.Id)
	if err != nil {
		t.Fatalf("find retained source user: %v", err)
	}
	if retainedUser == nil {
		t.Fatal("source user was deleted despite a failed merge")
	}
}
