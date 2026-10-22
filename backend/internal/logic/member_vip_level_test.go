package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openVipLevelTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	return openNamedVipLevelTestDB(t, t.Name(), models...)
}

// openNamedVipLevelTestDB 用于同一个测试里需要多个互相独立的内存库的场景（共享缓存按名字区分库）。
func openNamedVipLevelTestDB(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare vip level schema: %v", err)
	}
	return db
}

// vipLevelSvcWithMetrics 返回绑定了请求指标的服务上下文，用于统计 SQL 次数。
func vipLevelSvcWithMetrics(db *gorm.DB) (*svc.ServiceContext, *observability.RequestMetrics) {
	metrics := observability.NewRequestMetrics(time.Now())
	scoped := db.WithContext(observability.WithRequestMetrics(context.Background(), metrics))
	return &svc.ServiceContext{
		DB:                       scoped,
		UserModel:                model.NewUserModel(scoped),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(scoped),
	}, metrics
}

func TestResolveVipLevelsReturnsOnlyActiveMembersWithRuleBasedLevel(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	future := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)
	atNow := now
	if err := db.Create(&[]model.User{
		{Id: 1, Nickname: "无档案会员", MemberExpiresAt: &future},
		{Id: 2, Nickname: "成长值落在 Lv3", MemberExpiresAt: &future},
		{Id: 3, Nickname: "满级会员", MemberExpiresAt: &future},
		{Id: 4, Nickname: "已到期", MemberExpiresAt: &past},
		{Id: 5, Nickname: "从未开通"},
		{Id: 6, Nickname: "恰好到期", MemberExpiresAt: &atNow},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// 用户 2 的 growth_level 列仍是旧值 1：等级必须由成长值按当前规则换算，不能读这一列。
	if err := db.Create(&[]model.MemberGrowthProfile{
		{UserId: 2, GrowthPoints: 45, GrowthLevel: 1},
		{UserId: 3, GrowthPoints: 300, GrowthLevel: 5},
		{UserId: 4, GrowthPoints: 500, GrowthLevel: 5},
	}).Error; err != nil {
		t.Fatalf("seed growth profiles: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
	}

	levels, err := ResolveVipLevels(svcCtx, []int64{1, 2, 3, 4, 5, 6, 99, 2, 0, -1}, now)
	if err != nil {
		t.Fatalf("resolve vip levels: %v", err)
	}
	want := map[int64]int{1: 1, 2: 3, 3: 5}
	if len(levels) != len(want) {
		t.Fatalf("unexpected levels: got=%v want=%v", levels, want)
	}
	for id, level := range want {
		if levels[id] != level {
			t.Fatalf("user %d level: got=%d want=%d (all=%v)", id, levels[id], level, levels)
		}
	}
	for _, id := range []int64{4, 5, 6, 99} {
		if level, ok := levels[id]; ok || level != 0 {
			t.Fatalf("user %d must resolve to 0 and stay out of the result, got %d (ok=%v)", id, level, ok)
		}
	}
}

func TestResolveVipLevelsRestoresGrowthLevelAfterRenewal(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	expired := now.Add(-time.Hour)
	if err := db.Create(&model.User{Id: 1, Nickname: "到期后续开", MemberExpiresAt: &expired}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.MemberGrowthProfile{UserId: 1, GrowthPoints: 130, GrowthLevel: 4}).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
	}

	levels, err := ResolveVipLevels(svcCtx, []int64{1}, now)
	if err != nil || len(levels) != 0 {
		t.Fatalf("expired member must resolve to 0: levels=%v err=%v", levels, err)
	}

	renewed := now.Add(30 * 24 * time.Hour)
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("member_expires_at", renewed).Error; err != nil {
		t.Fatalf("renew member: %v", err)
	}
	levels, err = ResolveVipLevels(svcCtx, []int64{1}, now)
	if err != nil || levels[1] != 4 {
		t.Fatalf("renewed member must get the preserved level 4: levels=%v err=%v", levels, err)
	}
}

func TestResolveVipLevelsFollowsAdjustedGrowthThresholds(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{}, &model.MemberRightsConfig{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	future := now.Add(24 * time.Hour)
	if err := db.Create(&model.User{Id: 1, Nickname: "门槛调低后的会员", MemberExpiresAt: &future}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// 成长值 25 已达到调整后的 Lv3 门槛，但此后没有再获得成长值，growth_level 列仍是 1。
	if err := db.Create(&model.MemberGrowthProfile{UserId: 1, GrowthPoints: 25, GrowthLevel: 1}).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	cfg := model.DefaultMemberRightsConfig()
	growth := model.DefaultMemberGrowthRulesConfig()
	growth.LevelThresholdLv3 = 20
	cfg.SetGrowthRules(growth)
	configModel := model.NewMemberRightsConfigModel(db)
	if err := configModel.Upsert(cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
		MemberRightsConfigModel:  configModel,
	}

	levels, err := ResolveVipLevels(svcCtx, []int64{1}, now)
	if err != nil || levels[1] != 3 {
		t.Fatalf("level must follow the adjusted threshold: levels=%v err=%v", levels, err)
	}
	snapshot, err := NewMemberGrowthService(svcCtx, func() time.Time { return now }).GetSnapshotForUser(1)
	if err != nil || snapshot.GrowthLevel != levels[1] {
		t.Fatalf("vip level must match the member center level: snapshot=%+v levels=%v err=%v", snapshot, levels, err)
	}
}

func TestResolveVipLevelsUsesConstantQueries(t *testing.T) {
	for _, tc := range []struct {
		name        string
		memberCount int
	}{{"one", 1}, {"hundred", 100}} {
		t.Run(tc.name, func(t *testing.T) {
			db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{})
			now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
			future := now.Add(24 * time.Hour)
			users := make([]model.User, 0, tc.memberCount)
			profiles := make([]model.MemberGrowthProfile, 0, tc.memberCount)
			ids := make([]int64, 0, tc.memberCount)
			for i := 1; i <= tc.memberCount; i++ {
				id := int64(i)
				users = append(users, model.User{Id: id, Nickname: fmt.Sprintf("会员%d", i), MemberExpiresAt: &future})
				profiles = append(profiles, model.MemberGrowthProfile{UserId: id, GrowthPoints: i})
				ids = append(ids, id)
			}
			if err := db.CreateInBatches(&users, 100).Error; err != nil {
				t.Fatalf("seed users: %v", err)
			}
			if err := db.CreateInBatches(&profiles, 100).Error; err != nil {
				t.Fatalf("seed profiles: %v", err)
			}

			svcCtx, metrics := vipLevelSvcWithMetrics(db)
			levels, err := ResolveVipLevels(svcCtx, ids, now)
			if err != nil || len(levels) != tc.memberCount {
				t.Fatalf("resolve vip levels: len=%d err=%v", len(levels), err)
			}
			if got := metrics.Snapshot().SQLCount; got != 2 {
				t.Fatalf("expected exactly 2 queries regardless of rows, got %d", got)
			}
		})
	}
}

func TestResolveVipLevelsSkipsQueriesWhenNothingToResolve(t *testing.T) {
	db := openVipLevelTestDB(t, &model.User{}, &model.MemberGrowthProfile{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	if err := db.Create(&model.User{Id: 1, Nickname: "免费用户"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svcCtx, metrics := vipLevelSvcWithMetrics(db)
	levels, err := ResolveVipLevels(svcCtx, nil, now)
	if err != nil || len(levels) != 0 || metrics.Snapshot().SQLCount != 0 {
		t.Fatalf("empty list must not query: levels=%v err=%v sql=%d", levels, err, metrics.Snapshot().SQLCount)
	}

	levels, err = ResolveVipLevels(svcCtx, []int64{1}, now)
	if err != nil || len(levels) != 0 || metrics.Snapshot().SQLCount != 1 {
		t.Fatalf("no active member must stop after the user query: levels=%v err=%v sql=%d", levels, err, metrics.Snapshot().SQLCount)
	}
}

func TestResolveVipLevelsReturnsErrorWhenQueriesFail(t *testing.T) {
	// 不迁移成长档案表，第二次查询必然失败。
	db := openVipLevelTestDB(t, &model.User{})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	future := now.Add(24 * time.Hour)
	if err := db.Create(&model.User{Id: 1, Nickname: "会员", MemberExpiresAt: &future}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
	}

	levels, err := ResolveVipLevels(svcCtx, []int64{1}, now)
	if err == nil || levels != nil {
		t.Fatalf("query failure must be reported with no partial result: levels=%v err=%v", levels, err)
	}

	if _, err := ResolveVipLevels(&svc.ServiceContext{}, []int64{1}, now); err == nil {
		t.Fatal("missing models must be reported as an error")
	}
}

func TestMemberActiveAtTreatsExpiryAsExclusive(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, UTC8Location)
	future, past, atNow := now.Add(time.Second), now.Add(-time.Second), now
	for name, tc := range map[string]struct {
		expiresAt *time.Time
		want      bool
	}{
		"never opened": {nil, false},
		"future":       {&future, true},
		"past":         {&past, false},
		"exactly now":  {&atNow, false},
	} {
		if got := MemberActiveAt(tc.expiresAt, now); got != tc.want {
			t.Fatalf("%s: got %v want %v", name, got, tc.want)
		}
	}
}
