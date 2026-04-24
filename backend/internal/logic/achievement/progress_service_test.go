package achievement

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAchievementProgressTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(
		&model.Achievement{},
		&model.UserAchievement{},
		&model.UserTitle{},
		&model.AchievementProgressEvent{},
	); err != nil {
		t.Fatalf("auto migrate achievement progress schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                            db,
		AchievementModel:              model.NewAchievementModel(db),
		UserAchievementModel:          model.NewUserAchievementModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
	}
}

func TestProgressServiceAggregatesSumUnlocksAndGrantsTitleOnce(t *testing.T) {
	svcCtx := newAchievementProgressTestSvc(t)
	service := NewAchievementProgressService(svcCtx)
	seedAchievementProgressDef(t, svcCtx, &model.Achievement{
		Id:              1,
		Key:             "wins_10",
		Name:            "十胜球手",
		Category:        "wins",
		GameType:        1,
		MetricKey:       MetricWinsTotal,
		ProgressMode:    ProgressModeSum,
		Threshold:       10,
		RewardTitleKey:  "title_wins_10",
		RewardTitleName: "胜场新星",
		Status:          1,
	})

	mustAppendProgressEvent(t, service, AchievementProgressEventInput{
		UserId: 1001, SourceType: SourceTypeMatch, SourceId: 3001, GameType: 1, MetricKey: MetricWinsTotal, MetricValue: 4,
	}, true)
	mustAppendProgressEvent(t, service, AchievementProgressEventInput{
		UserId: 1001, SourceType: SourceTypeMatch, SourceId: 3002, GameType: 1, MetricKey: MetricWinsTotal, MetricValue: 6,
	}, true)
	if err := service.RefreshUserAchievements(1001); err != nil {
		t.Fatalf("refresh user achievements: %v", err)
	}
	if err := service.RefreshUserAchievements(1001); err != nil {
		t.Fatalf("refresh user achievements again: %v", err)
	}

	var ua model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", 1001, 1).First(&ua).Error; err != nil {
		t.Fatalf("find user achievement: %v", err)
	}
	if ua.Progress != 10 {
		t.Fatalf("expected progress 10, got %d", ua.Progress)
	}
	if ua.Unlocked != 1 || ua.UnlockedAt == nil {
		t.Fatalf("expected unlocked achievement with unlocked_at, got unlocked=%d unlocked_at=%v", ua.Unlocked, ua.UnlockedAt)
	}
	if ua.RewardGranted != 1 || ua.RewardGrantedAt == nil {
		t.Fatalf("expected granted reward with timestamp, got granted=%d granted_at=%v", ua.RewardGranted, ua.RewardGrantedAt)
	}

	var titles []model.UserTitle
	if err := svcCtx.DB.Where("user_id = ?", 1001).Find(&titles).Error; err != nil {
		t.Fatalf("find user titles: %v", err)
	}
	if len(titles) != 1 {
		t.Fatalf("expected one reward title, got %d", len(titles))
	}
	title := titles[0]
	if title.TitleKey != "title_wins_10" || title.TitleName != "胜场新星" {
		t.Fatalf("unexpected title: key=%s name=%s", title.TitleKey, title.TitleName)
	}
	if title.SourceType != SourceTypeAchievement || title.SourceRefId != 1 || title.SourceRefName != "十胜球手" {
		t.Fatalf("unexpected title source: type=%s ref_id=%d ref_name=%s", title.SourceType, title.SourceRefId, title.SourceRefName)
	}
	if title.GrantedByAchievementId == nil || *title.GrantedByAchievementId != 1 {
		t.Fatalf("expected granted_by_achievement_id 1, got %v", title.GrantedByAchievementId)
	}
}

func TestProgressServiceAggregatesMaxMetrics(t *testing.T) {
	svcCtx := newAchievementProgressTestSvc(t)
	service := NewAchievementProgressService(svcCtx)
	seedAchievementProgressDef(t, svcCtx, &model.Achievement{
		Id:           2,
		Key:          "streak_5",
		Name:         "五连胜",
		Category:     "streak",
		GameType:     1,
		MetricKey:    MetricMaxWinStreak,
		ProgressMode: ProgressModeMax,
		Threshold:    5,
		Status:       1,
	})

	mustAppendProgressEvent(t, service, AchievementProgressEventInput{
		UserId: 1002, SourceType: SourceTypeMatch, SourceId: 4001, GameType: 1, MetricKey: MetricMaxWinStreak, MetricValue: 3,
	}, true)
	mustAppendProgressEvent(t, service, AchievementProgressEventInput{
		UserId: 1002, SourceType: SourceTypeMatch, SourceId: 4002, GameType: 1, MetricKey: MetricMaxWinStreak, MetricValue: 7,
	}, true)
	if err := service.RefreshUserAchievements(1002); err != nil {
		t.Fatalf("refresh user achievements: %v", err)
	}

	var ua model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", 1002, 2).First(&ua).Error; err != nil {
		t.Fatalf("find user achievement: %v", err)
	}
	if ua.Progress != 7 {
		t.Fatalf("expected max progress 7, got %d", ua.Progress)
	}
	if ua.Unlocked != 1 {
		t.Fatalf("expected max achievement unlocked, got %d", ua.Unlocked)
	}
}

func TestProgressServiceIgnoresDuplicateEvents(t *testing.T) {
	svcCtx := newAchievementProgressTestSvc(t)
	service := NewAchievementProgressService(svcCtx)
	seedAchievementProgressDef(t, svcCtx, &model.Achievement{
		Id:           3,
		Key:          "match_10",
		Name:         "十场对局",
		Category:     "match",
		GameType:     1,
		MetricKey:    MetricMatchesTotal,
		ProgressMode: ProgressModeSum,
		Threshold:    10,
		Status:       1,
	})

	event := AchievementProgressEventInput{
		UserId: 1003, SourceType: SourceTypeMatch, SourceId: 5001, GameType: 1, MetricKey: MetricMatchesTotal, MetricValue: 5,
	}
	mustAppendProgressEvent(t, service, event, true)
	event.MetricValue = 99
	mustAppendProgressEvent(t, service, event, false)
	if err := service.RefreshUserAchievements(1003); err != nil {
		t.Fatalf("refresh user achievements: %v", err)
	}

	var ua model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", 1003, 3).First(&ua).Error; err != nil {
		t.Fatalf("find user achievement: %v", err)
	}
	if ua.Progress != 5 {
		t.Fatalf("expected duplicate event to keep progress 5, got %d", ua.Progress)
	}
	if ua.Unlocked != 0 || ua.UnlockedAt != nil {
		t.Fatalf("expected locked achievement, got unlocked=%d unlocked_at=%v", ua.Unlocked, ua.UnlockedAt)
	}

	var total int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).Count(&total).Error; err != nil {
		t.Fatalf("count progress events: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected one stored event, got %d", total)
	}
}

func seedAchievementProgressDef(t *testing.T, svcCtx *svc.ServiceContext, achievement *model.Achievement) {
	t.Helper()

	if err := svcCtx.DB.Create(achievement).Error; err != nil {
		t.Fatalf("seed achievement %s: %v", achievement.Key, err)
	}
}

func mustAppendProgressEvent(t *testing.T, service *AchievementProgressService, event AchievementProgressEventInput, wantCreated bool) {
	t.Helper()

	created, err := service.AppendEvent(event)
	if err != nil {
		t.Fatalf("append progress event: %v", err)
	}
	if created != wantCreated {
		t.Fatalf("expected created=%v, got %v", wantCreated, created)
	}
}
