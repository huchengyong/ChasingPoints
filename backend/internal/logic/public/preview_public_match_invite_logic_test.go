package public

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"chasing_points/internal/config"
	logic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func TestPreviewRateLimitAllowsBurstThenBlocks(t *testing.T) {
	mr := miniredis.RunT(t)
	svcCtx := &svc.ServiceContext{Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	logic := NewPreviewPublicMatchInviteLogic(context.Background(), svcCtx)

	for i := 0; i < previewRateLimitPerMinute; i++ {
		allowed, err := logic.allowRequest()
		if err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if !allowed {
			t.Fatalf("request %d must be allowed", i+1)
		}
	}

	allowed, err := logic.allowRequest()
	if err != nil {
		t.Fatalf("request over the limit: %v", err)
	}
	if allowed {
		t.Fatalf("request over the limit must be blocked")
	}
	if !mr.Exists(previewRateLimitKeyPrefix + "unknown") {
		t.Fatalf("rate limit key must be set")
	}
}

func TestPreviewRateLimitFailsClosedWithoutRedis(t *testing.T) {
	logic := NewPreviewPublicMatchInviteLogic(context.Background(), &svc.ServiceContext{})

	allowed, err := logic.allowRequest()
	if err == nil {
		t.Fatalf("missing redis must return an error")
	}
	if allowed {
		t.Fatalf("missing redis must not allow the request")
	}
}

func TestPreviewRateLimitWindowAlwaysExpires(t *testing.T) {
	mr := miniredis.RunT(t)
	svcCtx := &svc.ServiceContext{Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	logic := NewPreviewPublicMatchInviteLogic(context.Background(), svcCtx)
	key := previewRateLimitKeyPrefix + "unknown"

	if _, err := logic.allowRequest(); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if ttl := mr.TTL(key); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("rate limit key must expire within a minute, ttl=%v", ttl)
	}

	// 模拟上次设置过期时间失败：键存在但没有 TTL，下一次请求必须补上。
	mr.SetTTL(key, 0)
	if err := mr.Set(key, "99"); err != nil {
		t.Fatalf("seed counter: %v", err)
	}
	if mr.TTL(key) != 0 {
		t.Fatalf("seeded key must have no ttl")
	}
	if allowed, err := logic.allowRequest(); err != nil || allowed {
		t.Fatalf("over-limit request must be blocked without error: allowed=%v err=%v", allowed, err)
	}
	if ttl := mr.TTL(key); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("missing ttl must be repaired, ttl=%v", ttl)
	}

	mr.FastForward(time.Minute + time.Second)
	if allowed, err := logic.allowRequest(); err != nil || !allowed {
		t.Fatalf("a new window must allow requests again: allowed=%v err=%v", allowed, err)
	}
}

func newPublicPreviewTestSvc(t *testing.T) (*svc.ServiceContext, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	if err := db.Create(&model.User{Id: 42, Nickname: "邀请人", Avatar: "avatar.png", Status: 1}).Error; err != nil {
		t.Fatalf("create inviter: %v", err)
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := matchinvite.NewCodeStore(client)
	signer, err := matchinvite.NewSigner("public-preview-secret", 5*time.Minute)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	token, _, err := signer.Issue(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	code, err := store.Save(context.Background(), token, 5*time.Minute)
	if err != nil {
		t.Fatalf("save code: %v", err)
	}
	return &svc.ServiceContext{
		Config: config.Config{Security: config.SecurityConfig{
			MatchInvite: config.MatchInviteConfig{SigningSecret: "public-preview-secret", TTLSeconds: 300},
		}},
		Redis:                client,
		UserModel:            model.NewUserModel(db),
		MatchInviteCodeStore: store,
	}, code
}

func TestPublicPreviewReturnsOnlyDisplayFields(t *testing.T) {
	svcCtx, code := newPublicPreviewTestSvc(t)

	resp, err := NewPreviewPublicMatchInviteLogic(context.Background(), svcCtx).
		PreviewPublicMatchInvite(&types.PublicMatchInvitePreviewReq{InviteCode: code})
	if err != nil || !resp.Success || resp.Preview == nil {
		t.Fatalf("preview: resp=%+v err=%v", resp, err)
	}
	if resp.Preview.InviterName != "邀请人" || resp.Preview.InviterAvatar != "avatar.png" || resp.Preview.ExpiresAt == "" {
		t.Fatalf("unexpected preview: %+v", resp.Preview)
	}
	// 序列化后的响应里不能出现 token 或用户 ID 字段（过期时间里可能含数字，只按字段名检查）。
	body, _ := json.Marshal(resp)
	for _, forbidden := range []string{"token", "inviter_id", "user_id", "opponent_id"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("public preview leaked %q: %s", forbidden, body)
		}
	}
}

func TestPublicPreviewOverLimitDoesNotLookUpCode(t *testing.T) {
	svcCtx, code := newPublicPreviewTestSvc(t)
	previewLogic := NewPreviewPublicMatchInviteLogic(context.Background(), svcCtx)
	for i := 0; i < previewRateLimitPerMinute; i++ {
		if allowed, err := previewLogic.allowRequest(); err != nil || !allowed {
			t.Fatalf("warm-up request %d: allowed=%v err=%v", i+1, allowed, err)
		}
	}

	// 短码存储指向已关闭的 Redis：若超限后仍去查短码，会得到“服务暂不可用”而不是频率提示。
	broken := miniredis.RunT(t)
	svcCtx.MatchInviteCodeStore = matchinvite.NewCodeStore(redis.NewClient(&redis.Options{Addr: broken.Addr()}))
	broken.Close()

	resp, err := previewLogic.PreviewPublicMatchInvite(&types.PublicMatchInvitePreviewReq{InviteCode: code})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if resp.Success || resp.Message != previewRateLimitedMessage {
		t.Fatalf("over-limit request must be rejected before the code lookup: %+v", resp)
	}
	if resp.Message == logic.MatchInviteUnavailableMessage {
		t.Fatalf("over-limit request must not reach the code store")
	}
}
