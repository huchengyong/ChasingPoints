package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberGrowthServiceTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.MemberGrowthProfile{}, &model.MemberGrowthLog{}); err != nil {
		t.Fatalf("prepare member growth service schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
		MemberGrowthLogModel:     model.NewMemberGrowthLogModel(db),
	}
}

func seedMemberGrowthUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create user %d: %v", user.Id, err)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestMemberGrowthServiceAwardsActiveMemberBelowDailyCap(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(24 * time.Hour)
	seedMemberGrowthUser(t, svcCtx, &model.User{Id: 1001, Nickname: "会员用户", MemberExpiresAt: &expiresAt})

	result, err := NewMemberGrowthService(svcCtx, func() time.Time { return now }).AwardCompletedMatch(1001, 18)
	if err != nil {
		t.Fatalf("award completed match: %v", err)
	}
	if !result.Granted || result.GrowthPoints != 1 || result.TodayGrowthCount != 1 {
		t.Fatalf("unexpected grant result: %+v", result)
	}
	if result.GrowthLevel != 1 {
		t.Fatalf("expected level 1, got %+v", result)
	}
}

func TestMemberGrowthServiceSkipsExpiredMember(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(-time.Hour)
	seedMemberGrowthUser(t, svcCtx, &model.User{Id: 1002, Nickname: "已过期会员", MemberExpiresAt: &expiresAt})

	result, err := NewMemberGrowthService(svcCtx, func() time.Time { return now }).AwardCompletedMatch(1002, 19)
	if err != nil {
		t.Fatalf("award completed match: %v", err)
	}
	if result.Granted || result.Reason != "membership_inactive" || !result.Frozen {
		t.Fatalf("unexpected inactive member result: %+v", result)
	}
}

func TestMemberGrowthServiceSkipsNonMember(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	seedMemberGrowthUser(t, svcCtx, &model.User{Id: 1003, Nickname: "普通用户"})

	result, err := NewMemberGrowthService(svcCtx, func() time.Time { return now }).AwardCompletedMatch(1003, 20)
	if err != nil {
		t.Fatalf("award completed match: %v", err)
	}
	if result.Granted || result.Reason != "membership_inactive" {
		t.Fatalf("unexpected non-member result: %+v", result)
	}
}

func TestMemberGrowthServiceSkipsSixthMatchInSameDay(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(24 * time.Hour)
	seedMemberGrowthUser(t, svcCtx, &model.User{Id: 1004, Nickname: "高活跃会员", MemberExpiresAt: &expiresAt})
	dayStart := startOfGrowthDay(now)
	profile, err := svcCtx.MemberGrowthProfileModel.FindOrCreate(1004)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	profile.GrowthPoints = 5
	profile.TodayGrowthCount = 5
	profile.TodayGrowthDate = &dayStart
	if err := svcCtx.MemberGrowthProfileModel.Update(profile); err != nil {
		t.Fatalf("update profile: %v", err)
	}

	result, err := NewMemberGrowthService(svcCtx, func() time.Time { return now }).AwardCompletedMatch(1004, 21)
	if err != nil {
		t.Fatalf("award completed match: %v", err)
	}
	if result.Granted || result.Reason != "daily_cap_reached" {
		t.Fatalf("unexpected cap result: %+v", result)
	}
}

func TestMemberGrowthServiceSkipsDuplicateSameMatch(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(24 * time.Hour)
	seedMemberGrowthUser(t, svcCtx, &model.User{Id: 1005, Nickname: "重复对局会员", MemberExpiresAt: &expiresAt})

	service := NewMemberGrowthService(svcCtx, func() time.Time { return now })
	first, err := service.AwardCompletedMatch(1005, 22)
	if err != nil {
		t.Fatalf("first award: %v", err)
	}
	if !first.Granted {
		t.Fatalf("expected first award granted: %+v", first)
	}

	second, err := service.AwardCompletedMatch(1005, 22)
	if err != nil {
		t.Fatalf("second award: %v", err)
	}
	if second.Granted || second.Reason != "duplicate_match" || second.GrowthPoints != 1 {
		t.Fatalf("unexpected duplicate result: %+v", second)
	}
}

func TestResolveMemberGrowthLevelThresholds(t *testing.T) {
	cases := []struct {
		points int
		level  int
	}{
		{0, 1},
		{10, 2},
		{60, 3},
		{260, 4},
		{760, 5},
	}
	for _, tc := range cases {
		if got := ResolveMemberGrowthLevel(tc.points); got != tc.level {
			t.Fatalf("ResolveMemberGrowthLevel(%d)=%d want %d", tc.points, got, tc.level)
		}
	}
}

func TestMemberGrowthServiceBuildSnapshotKeepsFrozenProgress(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(-time.Hour)
	user := &model.User{Id: 1006, Nickname: "冻结会员", MemberExpiresAt: &expiresAt}
	seedMemberGrowthUser(t, svcCtx, user)

	snapshot := NewMemberGrowthService(svcCtx, func() time.Time { return now }).BuildSnapshot(user, &model.MemberGrowthProfile{
		UserId:       1006,
		GrowthPoints: 78,
		GrowthLevel:  3,
	})
	if !snapshot.Frozen || snapshot.GrowthLevel != 3 || snapshot.GrowthPoints != 78 {
		t.Fatalf("unexpected frozen snapshot: %+v", snapshot)
	}
	if snapshot.NextLevel != 4 || snapshot.RemainingPoints != 182 {
		t.Fatalf("unexpected next level snapshot: %+v", snapshot)
	}
}

func TestMemberGrowthServiceBuildSnapshotResetsDisplayedDailyCountAcrossDays(t *testing.T) {
	now := time.Date(2026, 4, 12, 10, 0, 0, 0, UTC8Location)
	svcCtx := newMemberGrowthServiceTestSvc(t)
	expiresAt := now.Add(24 * time.Hour)
	user := &model.User{Id: 1007, Nickname: "跨天会员", MemberExpiresAt: &expiresAt}
	seedMemberGrowthUser(t, svcCtx, user)

	snapshot := NewMemberGrowthService(svcCtx, func() time.Time { return now }).BuildSnapshot(user, &model.MemberGrowthProfile{
		UserId:           1007,
		GrowthPoints:     12,
		GrowthLevel:      2,
		TodayGrowthCount: 5,
		TodayGrowthDate:  timePtr(time.Date(2026, 4, 11, 0, 0, 0, 0, UTC8Location)),
	})
	if snapshot.TodayGrowthCount != 0 {
		t.Fatalf("expected stale daily growth count to reset in snapshot, got %+v", snapshot)
	}
}
