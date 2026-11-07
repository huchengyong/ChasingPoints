package match

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type memberIdentityDetailEnv struct {
	svcCtx    *svc.ServiceContext
	db        *gorm.DB
	miniRedis *miniredis.Miniredis
}

// newMemberIdentityDetailEnv 准备一场已完成的中式八球对局：1 号为创建者，2 号为对手。
// 默认成长门槛 10/40/120/300，所以成长值 45 对应 Lv3。
func newMemberIdentityDetailEnv(t *testing.T, withGrowthProfiles bool) *memberIdentityDetailEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	models := []any{
		&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{},
		&model.UserCompetitiveStats{}, &model.UserRanking{}, &model.RankChangeLog{}, &model.MemberRightsConfig{},
	}
	if withGrowthProfiles {
		models = append(models, &model.MemberGrowthProfile{})
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare member identity detail schema: %v", err)
	}
	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// 同一用户在不同球种的段位不同，用来确认只取本场球种（3）的段位。
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankLevel: 2},
		{UserId: 2, GameType: 3, RankLevel: 4},
		{UserId: 2, GameType: 1, RankLevel: 6},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	opponentID := int64(2)
	result := 1
	completedAt := time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{
		Id: 20, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		Status: 2, Result: &result, MyScore: 3, OpponentScore: 1, MatchTime: completedAt, CompletedAt: &completedAt, AchievementSyncedAt: &completedAt,
	}).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	seedCurrentProfiles(t, db, 5, 10, 8, 4, 10, 7)

	return &memberIdentityDetailEnv{
		db:        db,
		miniRedis: miniRedis,
		svcCtx: &svc.ServiceContext{
			DB:                       db,
			Redis:                    redisClient,
			UserModel:                model.NewUserModel(db),
			MatchModel:               model.NewMatchModel(db),
			RankingModel:             model.NewRankingModel(db),
			CompetitiveReadModel:     model.NewCompetitiveReadModel(db),
			MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
			MemberRightsConfigModel:  model.NewMemberRightsConfigModel(db),
			Config:                   config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
		},
	}
}

func (e *memberIdentityDetailEnv) setMember(t *testing.T, userID int64, expiresAt time.Time, growthPoints int) {
	t.Helper()
	if err := e.db.Model(&model.User{}).Where("id = ?", userID).Update("member_expires_at", expiresAt).Error; err != nil {
		t.Fatalf("set member expiry: %v", err)
	}
	if growthPoints > 0 {
		if err := e.db.Create(&model.MemberGrowthProfile{UserId: userID, GrowthPoints: growthPoints, GrowthLevel: 1}).Error; err != nil {
			t.Fatalf("seed growth profile: %v", err)
		}
	}
}

func (e *memberIdentityDetailEnv) detailFor(t *testing.T, viewerID int64) types.MatchDetailData {
	t.Helper()
	resp, err := NewGetMatchDetailLogic(context.WithValue(context.Background(), "user_id", viewerID), e.svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: 20})
	if err != nil || !resp.Success {
		t.Fatalf("get match detail for user %d: resp=%#v err=%v", viewerID, resp, err)
	}
	return resp.Match
}

func TestGetMatchDetailReturnsMemberIdentityFromViewerPerspective(t *testing.T) {
	env := newMemberIdentityDetailEnv(t, true)
	env.setMember(t, 2, time.Now().Add(24*time.Hour), 45)

	asPlayer1 := env.detailFor(t, 1)
	if asPlayer1.MyVipLevel != 0 || asPlayer1.MyRankLevel != 0 {
		t.Fatalf("free viewer must not carry vip identity: %+v", asPlayer1)
	}
	// 对手是 Lv3 会员，本场球种（3）的段位是 4，而不是中式八球以外球种的 6。
	if asPlayer1.OpponentVipLevel != 3 || asPlayer1.OpponentRankLevel != 4 {
		t.Fatalf("opponent identity must come from the viewer perspective: vip=%d rank=%d", asPlayer1.OpponentVipLevel, asPlayer1.OpponentRankLevel)
	}

	asPlayer2 := env.detailFor(t, 2)
	if asPlayer2.MyVipLevel != 3 || asPlayer2.MyRankLevel != 4 || asPlayer2.OpponentVipLevel != 0 || asPlayer2.OpponentRankLevel != 0 {
		t.Fatalf("perspective must swap for the opponent viewer: %+v", asPlayer2)
	}
}

func TestGetMatchDetailOmitsRankLevelForNonMembers(t *testing.T) {
	env := newMemberIdentityDetailEnv(t, true)

	detail := env.detailFor(t, 1)
	if detail.MyVipLevel != 0 || detail.OpponentVipLevel != 0 || detail.MyRankLevel != 0 || detail.OpponentRankLevel != 0 {
		t.Fatalf("non-members must not expose vip level or frame rank level: %+v", detail)
	}
}

func TestGetMatchDetailRendersCurrentMembershipOnReopenedCompletedMatch(t *testing.T) {
	env := newMemberIdentityDetailEnv(t, true)
	env.setMember(t, 2, time.Now().Add(24*time.Hour), 45)

	first := env.detailFor(t, 1)
	if first.OpponentVipLevel != 3 || first.MyScore != 3 || first.OpponentScore != 1 {
		t.Fatalf("member opponent must be shown on the first load: %+v", first)
	}
	if len(env.miniRedis.Keys()) == 0 {
		t.Fatal("completed match core must be cached for this test to prove anything")
	}

	// 对手随后到期；核心摘要已缓存，但会员状态必须按请求时刻重新读取。
	if err := env.db.Model(&model.User{}).Where("id = ?", 2).Update("member_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire opponent: %v", err)
	}
	reopened := env.detailFor(t, 1)
	if reopened.OpponentVipLevel != 0 || reopened.OpponentRankLevel != 0 {
		t.Fatalf("reopened match must reflect the opponent's current (expired) membership: %+v", reopened)
	}
	if reopened.MyScore != 3 || reopened.OpponentScore != 1 {
		t.Fatalf("immutable match score must still come from the cached core: %+v", reopened)
	}
}

func TestGetMatchDetailStillSucceedsWhenMemberIdentityLookupFails(t *testing.T) {
	// 不迁移成长档案表：有效会员的成长值查询必然失败。
	env := newMemberIdentityDetailEnv(t, false)
	env.setMember(t, 2, time.Now().Add(24*time.Hour), 0)

	detail := env.detailFor(t, 1)
	if detail.MyScore != 3 || detail.OpponentScore != 1 || detail.OpponentName != "乙" {
		t.Fatalf("match detail must still be returned: %+v", detail)
	}
	if detail.OpponentVipLevel != 0 || detail.OpponentRankLevel != 0 {
		t.Fatalf("failed identity lookup must degrade to non-member: %+v", detail)
	}
}

func TestGetMatchDetailHandlesOpponentWithoutAccount(t *testing.T) {
	env := newMemberIdentityDetailEnv(t, true)
	env.setMember(t, 1, time.Now().Add(24*time.Hour), 45)
	if err := env.db.Model(&model.Match{}).Where("id = ?", 20).Updates(map[string]any{"opponent_id": nil, "opponent_name": "访客"}).Error; err != nil {
		t.Fatalf("detach opponent account: %v", err)
	}

	detail := env.detailFor(t, 1)
	if detail.MyVipLevel != 3 || detail.MyRankLevel != 2 || detail.OpponentVipLevel != 0 || detail.OpponentRankLevel != 0 {
		t.Fatalf("guest opponent must have no identity while the viewer keeps theirs: %+v", detail)
	}
}
