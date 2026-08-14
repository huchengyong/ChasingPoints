package user

import (
	"context"
	"strconv"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProfileWritesInvalidateAllLeaderboardGameTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 7, Nickname: "旧昵称", Avatar: "old.png"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	svcCtx := &svc.ServiceContext{Redis: redisClient, UserModel: model.NewUserModel(db)}
	ctx := context.WithValue(context.Background(), "user_id", int64(7))

	profile, err := NewUpdateUserProfileLogic(ctx, svcCtx).UpdateUserProfile(&types.UpdateUserProfileReq{Nickname: "新昵称", Avatar: "new.png"})
	if err != nil || !profile.Success {
		t.Fatalf("update profile: resp=%#v err=%v", profile, err)
	}
	assertLeaderboardVersions(t, miniRedis, 1)
	assertRedisVersion(t, miniRedis, "season:leaderboard:profile-version", 1)

	nickname, err := NewUpdateNicknameLogic(ctx, svcCtx).UpdateNickname(&types.UpdateNicknameReq{Nickname: "再次更新"})
	if err != nil || !nickname.Success {
		t.Fatalf("update nickname: resp=%#v err=%v", nickname, err)
	}
	assertLeaderboardVersions(t, miniRedis, 2)
	assertRedisVersion(t, miniRedis, "season:leaderboard:profile-version", 2)
}

func assertRedisVersion(t *testing.T, miniRedis *miniredis.Miniredis, key string, expected int) {
	t.Helper()
	value, err := miniRedis.Get(key)
	if err != nil || value != strconv.Itoa(expected) {
		t.Fatalf("unexpected Redis version %s: value=%q err=%v", key, value, err)
	}
}

func assertLeaderboardVersions(t *testing.T, miniRedis *miniredis.Miniredis, expected int) {
	t.Helper()
	for gameType := 1; gameType <= 4; gameType++ {
		assertRedisVersion(t, miniRedis, "leaderboard:version:"+strconv.Itoa(gameType), expected)
	}
}
