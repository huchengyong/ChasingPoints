package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
)

const testInviteSecret = "test-match-invite-secret"

var inviteTestDBSeq int

func newMatchInviteTestContext(t *testing.T, status int) (context.Context, *svc.ServiceContext, string) {
	t.Helper()
	ctx := context.Background()

	inviteTestDBSeq++
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), inviteTestDBSeq)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	inviter := &model.User{Id: 42, Nickname: "邀请人", Avatar: "avatar.png", Status: status}
	if err := db.Create(inviter).Error; err != nil {
		t.Fatalf("create inviter: %v", err)
	}

	mr := miniredis.RunT(t)
	store := matchinvite.NewCodeStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Security: config.SecurityConfig{
				MatchInvite: config.MatchInviteConfig{SigningSecret: testInviteSecret, TTLSeconds: 300},
			},
		},
		UserModel:            model.NewUserModel(db),
		MatchInviteCodeStore: store,
	}

	signer, err := matchinvite.NewSigner(testInviteSecret, 5*time.Minute)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	token, _, err := signer.Issue(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	code, err := store.Save(ctx, token, 5*time.Minute)
	if err != nil {
		t.Fatalf("save code: %v", err)
	}
	return ctx, svcCtx, code
}

func TestResolveMatchInviteCodeReturnsTokenAndInviter(t *testing.T) {
	ctx, svcCtx, code := newMatchInviteTestContext(t, 1)

	resolved, message := ResolveMatchInviteCode(ctx, svcCtx, code)
	if message != "" {
		t.Fatalf("unexpected rejection: %s", message)
	}
	if resolved.Inviter.Id != 42 || resolved.Inviter.Nickname != "邀请人" {
		t.Fatalf("unexpected inviter: %+v", resolved.Inviter)
	}
	if resolved.Token == "" || resolved.ExpiresAt.Before(time.Now()) {
		t.Fatalf("unexpected token result: %+v", resolved)
	}
}

func TestResolveMatchInviteCodeReportsOneMessageForEveryFailure(t *testing.T) {
	ctx, svcCtx, code := newMatchInviteTestContext(t, 1)

	if _, message := ResolveMatchInviteCode(ctx, svcCtx, "Ab3dEf6hIj9-"); message != MatchInviteExpiredMessage {
		t.Fatalf("invalid format must look expired, got %q", message)
	}
	if _, message := ResolveMatchInviteCode(ctx, svcCtx, "Ab3dEf6hIj9k"); message != MatchInviteExpiredMessage {
		t.Fatalf("unknown code must look expired, got %q", message)
	}
	if _, message := ResolveMatchInviteCode(ctx, svcCtx, code[:11]); message != MatchInviteExpiredMessage {
		t.Fatalf("short code must look expired, got %q", message)
	}

	// 邀请人停用与短码失效使用同一句提示，调用方无法区分原因。
	disabledCtx, disabledCtxStore, disabledCode := newMatchInviteTestContext(t, 2)
	if _, message := ResolveMatchInviteCode(disabledCtx, disabledCtxStore, disabledCode); message != MatchInviteExpiredMessage {
		t.Fatalf("disabled inviter must look expired, got %q", message)
	}
}

func TestResolveMatchInviteCodeReportsUnavailableWhenStoreIsMissing(t *testing.T) {
	ctx, svcCtx, _ := newMatchInviteTestContext(t, 1)
	svcCtx.MatchInviteCodeStore = nil

	if _, message := ResolveMatchInviteCode(ctx, svcCtx, "Ab3dEf6hIj9k"); message != MatchInviteUnavailableMessage {
		t.Fatalf("missing store must report unavailable, got %q", message)
	}
}
