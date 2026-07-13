package eventnews

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	eventNewsCacheVersionKey   = "eventnews:version"
	eventNewsListCachePrefix   = "eventnews:list"
	eventNewsViewCachePrefix   = "eventnews:view"
	eventNewsListCacheTTL      = 5 * time.Minute
	eventNewsViewCacheTTL      = 3 * time.Minute
)

type eventNewsListCacheParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	GameType int    `json:"game_type"`
	Status   int    `json:"status"`
	City     string `json:"city"`
	From     string `json:"from"`
	To       string `json:"to"`
}

func loadCachedEventNewsListResp(ctx context.Context, svcCtx *svc.ServiceContext, req *types.GetEventNewsListReq, window eventNewsDateWindow) (*types.GetEventNewsListResp, bool) {
	version := getEventNewsCacheVersion(ctx, svcCtx)
	key := buildEventNewsListCacheKey(req, window, version)

	var resp types.GetEventNewsListResp
	if !loadCachedJSON(ctx, svcCtx, key, &resp) {
		return nil, false
	}
	return &resp, true
}

func storeCachedEventNewsListResp(ctx context.Context, svcCtx *svc.ServiceContext, req *types.GetEventNewsListReq, window eventNewsDateWindow, resp *types.GetEventNewsListResp) {
	if resp == nil || !resp.Success {
		return
	}

	version := getEventNewsCacheVersion(ctx, svcCtx)
	key := buildEventNewsListCacheKey(req, window, version)
	storeCachedJSON(ctx, svcCtx, key, resp, eventNewsListCacheTTL)
}

func loadCachedEventNewsViewResp(ctx context.Context, svcCtx *svc.ServiceContext, eventID int64) (*types.GetEventNewsViewResp, bool) {
	version := getEventNewsCacheVersion(ctx, svcCtx)
	key := buildEventNewsViewCacheKey(eventID, version)

	var resp types.GetEventNewsViewResp
	if !loadCachedJSON(ctx, svcCtx, key, &resp) {
		return nil, false
	}
	return &resp, true
}

func storeCachedEventNewsViewResp(ctx context.Context, svcCtx *svc.ServiceContext, eventID int64, resp *types.GetEventNewsViewResp) {
	if resp == nil || !resp.Success {
		return
	}

	version := getEventNewsCacheVersion(ctx, svcCtx)
	key := buildEventNewsViewCacheKey(eventID, version)
	storeCachedJSON(ctx, svcCtx, key, resp, eventNewsViewCacheTTL)
}

func BumpEventNewsCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext) error {
	if svcCtx == nil || svcCtx.Redis == nil {
		return nil
	}

	if err := svcCtx.Redis.Incr(ctx, eventNewsCacheVersionKey).Err(); err != nil {
		return err
	}
	return nil
}

func getEventNewsCacheVersion(ctx context.Context, svcCtx *svc.ServiceContext) int64 {
	if svcCtx == nil || svcCtx.Redis == nil {
		return 0
	}

	version, err := svcCtx.Redis.Get(ctx, eventNewsCacheVersionKey).Int64()
	if err == nil {
		return version
	}
	if err == redis.Nil {
		return 0
	}

	logx.WithContext(ctx).Errorf("读取赛讯缓存版本失败: err=%v", err)
	return 0
}

func buildEventNewsListCacheKey(req *types.GetEventNewsListReq, window eventNewsDateWindow, version int64) string {
	page, pageSize := normalizeEventNewsPage(req.Page, req.PageSize)
	payload := eventNewsListCacheParams{
		Page:     page,
		PageSize: pageSize,
		GameType: req.GameType,
		Status:   req.Status,
		City:     strings.TrimSpace(req.City),
		From:     strings.TrimSpace(window.From),
		To:       strings.TrimSpace(window.To),
	}
	return fmt.Sprintf("%s:v%d:%s", eventNewsListCachePrefix, version, hashEventNewsCachePayload(payload))
}

func buildEventNewsViewCacheKey(eventID, version int64) string {
	return fmt.Sprintf("%s:v%d:%d", eventNewsViewCachePrefix, version, eventID)
}

func hashEventNewsCachePayload(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "fallback"
	}
	sum := sha1.Sum(payload)
	return hex.EncodeToString(sum[:])
}

func loadCachedJSON(ctx context.Context, svcCtx *svc.ServiceContext, key string, target any) bool {
	if svcCtx == nil || svcCtx.Redis == nil || strings.TrimSpace(key) == "" || target == nil {
		return false
	}

	payload, err := svcCtx.Redis.Get(ctx, key).Bytes()
	if err == nil {
		if err := json.Unmarshal(payload, target); err == nil {
			return true
		}
		logx.WithContext(ctx).Errorf("解析赛讯缓存失败: key=%s err=%v", key, err)
		_ = svcCtx.Redis.Del(ctx, key).Err()
		return false
	}
	if err == redis.Nil {
		return false
	}

	logx.WithContext(ctx).Errorf("读取赛讯缓存失败: key=%s err=%v", key, err)
	return false
}

func storeCachedJSON(ctx context.Context, svcCtx *svc.ServiceContext, key string, value any, ttl time.Duration) {
	if svcCtx == nil || svcCtx.Redis == nil || strings.TrimSpace(key) == "" || value == nil {
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		logx.WithContext(ctx).Errorf("序列化赛讯缓存失败: key=%s err=%v", key, err)
		return
	}
	if err := svcCtx.Redis.Set(ctx, key, payload, ttl).Err(); err != nil {
		logx.WithContext(ctx).Errorf("写入赛讯缓存失败: key=%s err=%v", key, err)
	}
}
