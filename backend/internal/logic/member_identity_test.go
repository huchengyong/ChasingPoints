package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func memberIdentityTestSvc(db *gorm.DB) (*svc.ServiceContext, func() int64) {
	scoped, metrics := vipLevelSvcWithMetrics(db)
	scoped.RankingModel = model.NewRankingModel(scoped.DB)
	return scoped, func() int64 { return metrics.Snapshot().SQLCount }
}

func seedMemberIdentityUsers(t *testing.T, db *gorm.DB, now time.Time) {
	t.Helper()
	future := now.Add(24 * time.Hour)
	if err := db.Create(&[]model.User{
		{Id: 1, Nickname: "Lv3会员", MemberExpiresAt: &future},
		{Id: 2, Nickname: "无排位记录的会员", MemberExpiresAt: &future},
		{Id: 3, Nickname: "免费用户"},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&model.MemberGrowthProfile{UserId: 1, GrowthPoints: 45, GrowthLevel: 1}).Error; err != nil {
		t.Fatalf("seed growth profile: %v", err)
	}
	// 免费用户也有段位，但头像框只给会员，所以不应被读取。同一用户其他球种的段位不得串入。
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankLevel: 4},
		{UserId: 1, GameType: 1, RankLevel: 6},
		{UserId: 3, GameType: 3, RankLevel: 5},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
}

func TestLoadMemberIdentitiesReadsRankLevelOnlyForMembersInTheGameType(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{}, &model.UserRanking{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	seedMemberIdentityUsers(t, db, now)
	svcCtx, queries := memberIdentityTestSvc(db)

	identities, err := LoadMemberIdentities(svcCtx, []int64{1, 2, 3, 3, 0, 99}, 3, now)
	if err != nil {
		t.Fatalf("load identities: %v", err)
	}
	if got := identities[1]; got.VipLevel != 3 || got.RankLevel != 4 {
		t.Fatalf("member must carry vip level and the game's rank level: %+v", got)
	}
	if got := identities[2]; got.VipLevel != 1 || got.RankLevel != 0 {
		t.Fatalf("member without a ranking row keeps vip level and has no frame rank: %+v", got)
	}
	for _, id := range []int64{3, 99} {
		if got, ok := identities[id]; ok || got != (MemberIdentity{}) {
			t.Fatalf("non-member %d must stay out of the result: %+v ok=%v", id, got, ok)
		}
	}
	// 用户、成长值、段位各 1 次，不随用户数量增长；这里没有注入成长规则配置，所以没有配置查询。
	if got := queries(); got != 3 {
		t.Fatalf("expected 3 fixed queries, got %d", got)
	}
}

func TestLoadMemberIdentitiesSkipsRankQueryWithoutMembers(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{}, &model.UserRanking{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	seedMemberIdentityUsers(t, db, now)
	svcCtx, queries := memberIdentityTestSvc(db)

	identities, err := LoadMemberIdentities(svcCtx, []int64{3}, 3, now)
	if err != nil || len(identities) != 0 {
		t.Fatalf("free users have no identity: %v err=%v", identities, err)
	}
	if got := queries(); got != 1 {
		t.Fatalf("a match between free users must cost one query, got %d", got)
	}

	empty, err := LoadMemberIdentities(svcCtx, nil, 3, now)
	if err != nil || len(empty) != 0 || queries() != 1 {
		t.Fatalf("empty input must not query: %v err=%v queries=%d", empty, err, queries())
	}
}

func TestLoadMemberIdentitiesKeepsWhatItCouldReadWhenOnePartFails(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)

	// 排位表不存在：会员档位保留，段位按 0 展示，并返回错误。
	noRanking := openNamedVipLevelTestDB(t, t.Name()+"-no-ranking", &model.User{}, &model.MemberGrowthProfile{})
	future := now.Add(24 * time.Hour)
	if err := noRanking.Create(&model.User{Id: 1, Nickname: "会员", MemberExpiresAt: &future}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svcCtx, _ := memberIdentityTestSvc(noRanking)
	identities, err := LoadMemberIdentities(svcCtx, []int64{1}, 3, now)
	if err == nil || identities[1].VipLevel != 1 || identities[1].RankLevel != 0 {
		t.Fatalf("rank failure must keep the vip level and report the error: %+v err=%v", identities, err)
	}

	// 排位模型缺失：同样保留会员档位。
	svcCtx.RankingModel = nil
	identities, err = LoadMemberIdentities(svcCtx, []int64{1}, 3, now)
	if err == nil || identities[1].VipLevel != 1 {
		t.Fatalf("missing ranking model must keep the vip level and report the error: %+v err=%v", identities, err)
	}

	// 成长档案表不存在：无法确认会员档位，全部按非会员返回并报错。
	noGrowth := openNamedVipLevelTestDB(t, t.Name()+"-no-growth", &model.User{}, &model.UserRanking{})
	if err := noGrowth.Create(&model.User{Id: 1, Nickname: "会员", MemberExpiresAt: &future}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svcCtx, _ = memberIdentityTestSvc(noGrowth)
	identities, err = LoadMemberIdentities(svcCtx, []int64{1}, 3, now)
	if err == nil || len(identities) != 0 {
		t.Fatalf("vip failure must yield no identities and report the error: %+v err=%v", identities, err)
	}
}
