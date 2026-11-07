package achievement

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

func TestAchievementDetailRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetAchievementDetailLogic(context.Background(), &svc.ServiceContext{}).GetAchievementDetail(&types.GetAchievementDetailReq{AchievementId: 8})
	if err != nil || resp.Success || resp.Achievement != nil {
		t.Fatalf("unauthenticated achievement detail must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestAchievementDetailReadsOneDefinitionAndViewerProgress(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Achievement{}, &model.UserAchievement{}); err != nil {
		t.Fatalf("prepare achievement schema: %v", err)
	}
	if err := db.Create(&model.Achievement{Id: 8, Key: "eight", Name: "八杆", Status: 1, Threshold: 8}).Error; err != nil {
		t.Fatalf("seed achievement: %v", err)
	}
	unlockedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := db.Create(&[]model.UserAchievement{
		{UserId: 7, AchievementId: 8, Progress: 6, Unlocked: 1, UnlockedAt: &unlockedAt},
		{UserId: 9, AchievementId: 8, Progress: 2, Unlocked: 0},
	}).Error; err != nil {
		t.Fatalf("seed user achievements: %v", err)
	}
	svcCtx := &svc.ServiceContext{AchievementModel: model.NewAchievementModel(db), UserAchievementModel: model.NewUserAchievementModel(db)}
	resp, err := NewGetAchievementDetailLogic(context.WithValue(context.Background(), "user_id", int64(7)), svcCtx).GetAchievementDetail(&types.GetAchievementDetailReq{AchievementId: 8})
	if err != nil || !resp.Success || resp.Achievement == nil || resp.Achievement.Id != 8 || resp.Achievement.Progress != 6 || !resp.Achievement.Unlocked {
		t.Fatalf("get achievement detail: resp=%#v err=%v", resp, err)
	}
	other, otherErr := NewGetAchievementDetailLogic(context.WithValue(context.Background(), "user_id", int64(9)), svcCtx).GetAchievementDetail(&types.GetAchievementDetailReq{AchievementId: 8})
	if otherErr != nil || !other.Success || other.Achievement == nil || other.Achievement.Progress != 2 || other.Achievement.Unlocked {
		t.Fatalf("achievement detail must be scoped to the requesting user: resp=%#v err=%v", other, otherErr)
	}
}

func TestAchievementDetailDoesNotReturnInactiveDefinition(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Achievement{}, &model.UserAchievement{}); err != nil {
		t.Fatalf("prepare achievement schema: %v", err)
	}
	if err := db.Create(&model.Achievement{Id: 9, Key: "hidden", Name: "隐藏", Status: 2}).Error; err != nil {
		t.Fatalf("seed inactive achievement: %v", err)
	}
	svcCtx := &svc.ServiceContext{AchievementModel: model.NewAchievementModel(db), UserAchievementModel: model.NewUserAchievementModel(db)}
	resp, err := NewGetAchievementDetailLogic(context.WithValue(context.Background(), "user_id", int64(7)), svcCtx).GetAchievementDetail(&types.GetAchievementDetailReq{AchievementId: 9})
	if err != nil || resp.Success {
		t.Fatalf("inactive achievement must not be returned: resp=%#v err=%v", resp, err)
	}
}

func TestAchievementDetailUsesTwoQueriesRegardlessOfOtherProgressRows(t *testing.T) {
	one := achievementDetailQueryCount(t, 1)
	hundred := achievementDetailQueryCount(t, 100)
	if one != 2 || hundred != 2 {
		t.Fatalf("achievement detail must use two fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func achievementDetailQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Achievement{}, &model.UserAchievement{}); err != nil {
		t.Fatalf("prepare achievement query schema: %v", err)
	}
	if err := db.Create(&model.Achievement{Id: 8, Key: "eight", Name: "八杆", Status: 1, Threshold: 8}).Error; err != nil {
		t.Fatalf("seed achievement: %v", err)
	}
	progress := make([]model.UserAchievement, 0, rows)
	progress = append(progress, model.UserAchievement{UserId: 7, AchievementId: 8, Progress: 1})
	for index := 1; index < rows; index++ {
		progress = append(progress, model.UserAchievement{UserId: int64(1000 + index), AchievementId: 8, Progress: index})
	}
	if err := db.Create(&progress).Error; err != nil {
		t.Fatalf("seed progress: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(7)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetAchievementDetailLogic(ctx, &svc.ServiceContext{AchievementModel: model.NewAchievementModel(requestDB), UserAchievementModel: model.NewUserAchievementModel(requestDB)}).GetAchievementDetail(&types.GetAchievementDetailReq{AchievementId: 8})
	if err != nil || !resp.Success {
		t.Fatalf("get achievement detail: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
