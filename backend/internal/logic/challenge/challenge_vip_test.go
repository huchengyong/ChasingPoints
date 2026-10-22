package challenge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newChallengeVipTestSvc 在通用约球测试环境上补齐会员相关的表与 model。
// 默认成长门槛 10/40/120/300，所以成长值 45 对应 Lv3。
func newChallengeVipTestSvc(t *testing.T, withGrowthProfiles bool) *svc.ServiceContext {
	t.Helper()
	svcCtx := newChallengeLogicTestSvc(t)
	models := []any{&model.MemberRightsConfig{}}
	if withGrowthProfiles {
		models = append(models, &model.MemberGrowthProfile{})
	}
	if err := svcCtx.DB.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare member schema: %v", err)
	}
	svcCtx.MemberGrowthProfileModel = model.NewMemberGrowthProfileModel(svcCtx.DB)
	svcCtx.MemberRightsConfigModel = model.NewMemberRightsConfigModel(svcCtx.DB)
	return svcCtx
}

// seedVipChallengeUsers：1001 是 Lv3 会员，2002 是免费用户。
func seedVipChallengeUsers(t *testing.T, svcCtx *svc.ServiceContext, growthPoints int) {
	t.Helper()
	future := time.Now().Add(24 * time.Hour)
	seedChallengeUsers(t, svcCtx, model.User{Id: 1001, Nickname: "小李", MemberExpiresAt: &future}, model.User{Id: 2002, Nickname: "小王"})
	if growthPoints > 0 {
		if err := svcCtx.DB.Create(&model.MemberGrowthProfile{UserId: 1001, GrowthPoints: growthPoints, GrowthLevel: 1}).Error; err != nil {
			t.Fatalf("seed growth profile: %v", err)
		}
	}
}

func assertChallengeVip(t *testing.T, label string, info *types.ChallengeInfo, from, to int) {
	t.Helper()
	if info == nil {
		t.Fatalf("%s: challenge is nil", label)
	}
	if info.FromVipLevel != from || info.ToVipLevel != to {
		t.Fatalf("%s: from_vip_level=%d to_vip_level=%d want %d/%d", label, info.FromVipLevel, info.ToVipLevel, from, to)
	}
}

func TestChallengeDetailCarriesVipLevelsOfBothSides(t *testing.T) {
	svcCtx := newChallengeVipTestSvc(t, true)
	seedVipChallengeUsers(t, svcCtx, 45)
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	// 发起方是会员、接收方是免费用户；从任一方看到的都是同一组档位。
	for _, viewer := range []int64{1001, 2002} {
		resp, err := NewGetChallengeDetailLogic(challengeTestCtx(viewer), svcCtx).GetChallengeDetail(&types.GetChallengeDetailReq{Id: challenge.Id})
		if err != nil || !resp.Success {
			t.Fatalf("detail for %d: resp=%#v err=%v", viewer, resp, err)
		}
		assertChallengeVip(t, fmt.Sprintf("detail viewer %d", viewer), resp.Challenge, 3, 0)
	}
}

func TestChallengeSummaryAndBootstrapCarryVipLevels(t *testing.T) {
	svcCtx := newChallengeVipTestSvc(t, true)
	seedVipChallengeUsers(t, svcCtx, 45)
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	summary, err := NewGetChallengeSummaryLogic(challengeTestCtx(2002), svcCtx).GetChallengeSummary()
	if err != nil || !summary.Success {
		t.Fatalf("summary: resp=%#v err=%v", summary, err)
	}
	assertChallengeVip(t, "summary current challenge", summary.CurrentChallenge, 3, 0)

	info, err := BuildChallengeInfoForBootstrap(context.Background(), svcCtx, 2002, challenge, time.Now())
	if err != nil {
		t.Fatalf("bootstrap info: %v", err)
	}
	assertChallengeVip(t, "bootstrap current challenge", info, 3, 0)
}

func TestChallengeListsCarryVipLevelsPerRow(t *testing.T) {
	svcCtx := newChallengeVipTestSvc(t, true)
	seedVipChallengeUsers(t, svcCtx, 45)
	future := time.Now().Add(24 * time.Hour)
	seedChallengeUsers(t, svcCtx, model.User{Id: 3003, Nickname: "小张", MemberExpiresAt: &future})
	// 3003 是无成长档案的会员（Lv1）；2002 向他们二人各发一次邀请，1001 发给 3003 的一条已被拒绝，进入历史。
	now := time.Now()
	if err := svcCtx.DB.Create(&[]model.Challenge{
		{Id: 1, FromUserId: 1001, ToUserId: 2002, GameType: 3, Status: model.ChallengeStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Id: 2, FromUserId: 3003, ToUserId: 2002, GameType: 3, Status: model.ChallengeStatusPending, ExpiresAt: now.Add(time.Hour)},
		{Id: 3, FromUserId: 2002, ToUserId: 3003, GameType: 3, Status: model.ChallengeStatusRejected, ExpiresAt: now.Add(time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}

	pending, err := NewGetPendingChallengesLogic(challengeTestCtx(2002), svcCtx).GetPendingChallenges()
	if err != nil || !pending.Success || len(pending.List) < 2 {
		t.Fatalf("pending: resp=%#v err=%v", pending, err)
	}
	for i := range pending.List {
		info := &pending.List[i]
		switch info.Id {
		case 1:
			assertChallengeVip(t, "pending #1", info, 3, 0)
		case 2:
			assertChallengeVip(t, "pending #2", info, 1, 0)
		}
	}

	history, err := NewGetChallengeHistoryLogic(challengeTestCtx(2002), svcCtx).GetChallengeHistory(&types.GetChallengeHistoryReq{PageSize: 20})
	if err != nil || !history.Success {
		t.Fatalf("history: resp=%#v err=%v", history, err)
	}
	found := false
	for i := range history.List {
		if history.List[i].Id == 3 {
			found = true
			assertChallengeVip(t, "history #3", &history.List[i], 0, 1)
		}
	}
	if !found {
		t.Fatalf("rejected challenge must appear in history: %+v", history.List)
	}
}

func TestChallengeStillSucceedsWhenVipLookupFails(t *testing.T) {
	// 不迁移成长档案表：有效会员的成长值查询必然失败。
	svcCtx := newChallengeVipTestSvc(t, false)
	seedVipChallengeUsers(t, svcCtx, 0)
	challenge := seedAcceptedChallengeForEnter(t, svcCtx, 1001)

	resp, err := NewGetChallengeDetailLogic(challengeTestCtx(2002), svcCtx).GetChallengeDetail(&types.GetChallengeDetailReq{Id: challenge.Id})
	if err != nil || !resp.Success || resp.Challenge == nil || resp.Challenge.FromNickname != "小李" {
		t.Fatalf("detail must still be returned: resp=%#v err=%v", resp, err)
	}
	assertChallengeVip(t, "degraded detail", resp.Challenge, 0, 0)
}

func TestChallengeVipQueriesDoNotGrowWithRows(t *testing.T) {
	counts := map[int]int{}
	for _, challengeCount := range []int{1, 100} {
		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", challengeCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&model.User{}, &model.Challenge{}, &model.Match{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{}); err != nil {
			t.Fatalf("prepare schema: %v", err)
		}
		future := time.Now().Add(24 * time.Hour)
		users := []model.User{{Id: 1, Nickname: "我"}}
		challenges := make([]model.Challenge, 0, challengeCount)
		profiles := make([]model.MemberGrowthProfile, 0, challengeCount)
		for i := range challengeCount {
			fromUserID := int64(i + 2)
			users = append(users, model.User{Id: fromUserID, Nickname: fmt.Sprintf("挑战者%d", fromUserID), MemberExpiresAt: &future})
			profiles = append(profiles, model.MemberGrowthProfile{UserId: fromUserID, GrowthPoints: i + 1})
			challenges = append(challenges, model.Challenge{Id: fromUserID, FromUserId: fromUserID, ToUserId: 1, GameType: 3, Status: 0, ExpiresAt: time.Now().Add(time.Hour)})
		}
		if err := db.CreateInBatches(&users, 100).Error; err != nil {
			t.Fatalf("seed users: %v", err)
		}
		if err := db.CreateInBatches(&profiles, 100).Error; err != nil {
			t.Fatalf("seed profiles: %v", err)
		}
		if err := db.CreateInBatches(&challenges, 100).Error; err != nil {
			t.Fatalf("seed challenges: %v", err)
		}
		metrics := observability.NewRequestMetrics(time.Now())
		ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
		resp, err := NewGetPendingChallengesLogic(ctx, &svc.ServiceContext{DB: db}).GetPendingChallenges()
		if err != nil || !resp.Success || len(resp.List) != challengeCount || resp.List[0].FromVipLevel == 0 {
			t.Fatalf("pending challenges: resp=%#v err=%v", resp, err)
		}
		counts[challengeCount] = int(metrics.Snapshot().SQLCount)
	}
	// 联表列表 1 次，会员档位再加用户、成长值、成长规则配置各 1 次。
	if counts[1] != 4 || counts[100] != 4 {
		t.Fatalf("vip lookups must be a fixed number of queries: one=%d hundred=%d", counts[1], counts[100])
	}
}
