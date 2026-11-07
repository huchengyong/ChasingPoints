package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAchievementClosedLoopTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	return db
}

func TestAchievementClosedLoopTableNames(t *testing.T) {
	testCases := []struct {
		name string
		got  string
		want string
	}{
		{name: "achievement", got: (Achievement{}).TableName(), want: "achievements"},
		{name: "user achievement", got: (UserAchievement{}).TableName(), want: "user_achievements"},
		{name: "user title", got: (UserTitle{}).TableName(), want: "user_titles"},
		{name: "match achievement", got: (MatchAchievement{}).TableName(), want: "match_achievements"},
		{name: "achievement progress event", got: (AchievementProgressEvent{}).TableName(), want: "achievement_progress_events"},
		{name: "season challenge snapshot", got: (SeasonChallengeSnapshot{}).TableName(), want: "season_challenge_snapshots"},
		{name: "season settlement", got: (SeasonSettlement{}).TableName(), want: "season_settlements"},
	}

	for _, tc := range testCases {
		if tc.got != tc.want {
			t.Fatalf("%s table name: expected %s, got %s", tc.name, tc.want, tc.got)
		}
	}
}

func TestAchievementClosedLoopSchemaIncludesNewFields(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&Achievement{}, &UserAchievement{}, &UserTitle{}, &MatchAchievement{}, &AchievementProgressEvent{}, &Match{}, &Notification{}, &SeasonChallengeSnapshot{}, &SeasonSettlement{}); err != nil {
		t.Fatalf("auto migrate achievement closed loop schema: %v", err)
	}

	testCases := []struct {
		name   string
		model  any
		column string
	}{
		{name: "achievement game type", model: &Achievement{}, column: "game_type"},
		{name: "achievement metric key", model: &Achievement{}, column: "metric_key"},
		{name: "achievement progress mode", model: &Achievement{}, column: "progress_mode"},
		{name: "achievement reward title key", model: &Achievement{}, column: "reward_title_key"},
		{name: "achievement reward title name", model: &Achievement{}, column: "reward_title_name"},
		{name: "achievement sort", model: &Achievement{}, column: "sort"},
		{name: "achievement status", model: &Achievement{}, column: "status"},
		{name: "achievement updated at", model: &Achievement{}, column: "updated_at"},
		{name: "user achievement reward granted", model: &UserAchievement{}, column: "reward_granted"},
		{name: "user achievement reward granted at", model: &UserAchievement{}, column: "reward_granted_at"},
		{name: "user achievement updated at", model: &UserAchievement{}, column: "updated_at"},
		{name: "user achievement unlock source type", model: &UserAchievement{}, column: "unlocked_source_type"},
		{name: "user achievement unlock source id", model: &UserAchievement{}, column: "unlocked_source_id"},
		{name: "user title title key", model: &UserTitle{}, column: "title_key"},
		{name: "user title source type", model: &UserTitle{}, column: "source_type"},
		{name: "user title source ref id", model: &UserTitle{}, column: "source_ref_id"},
		{name: "user title source ref name", model: &UserTitle{}, column: "source_ref_name"},
		{name: "user title granted by achievement id", model: &UserTitle{}, column: "granted_by_achievement_id"},
		{name: "user title equipped at", model: &UserTitle{}, column: "equipped_at"},
		{name: "user title granted at", model: &UserTitle{}, column: "granted_at"},
		{name: "match achievement actor", model: &MatchAchievement{}, column: "actor"},
		{name: "progress event source type", model: &AchievementProgressEvent{}, column: "source_type"},
		{name: "progress event source id", model: &AchievementProgressEvent{}, column: "source_id"},
		{name: "progress event game type", model: &AchievementProgressEvent{}, column: "game_type"},
		{name: "progress event metric key", model: &AchievementProgressEvent{}, column: "metric_key"},
		{name: "progress event metric value", model: &AchievementProgressEvent{}, column: "metric_value"},
		{name: "progress event occurred at", model: &AchievementProgressEvent{}, column: "occurred_at"},
		{name: "match achievement synced at", model: &Match{}, column: "achievement_synced_at"},
		{name: "notification dedupe key", model: &Notification{}, column: "dedupe_key"},
		{name: "season challenge key", model: &SeasonChallengeSnapshot{}, column: "challenge_key"},
		{name: "season challenge archived at", model: &SeasonChallengeSnapshot{}, column: "archived_at"},
		{name: "season settlement status", model: &SeasonSettlement{}, column: "status"},
		{name: "season settlement next season", model: &SeasonSettlement{}, column: "next_season_id"},
	}

	for _, tc := range testCases {
		if !db.Migrator().HasColumn(tc.model, tc.column) {
			t.Fatalf("%s: expected column %s", tc.name, tc.column)
		}
	}
}

func TestMatchModelSaveAchievementWithTxKeepsActorBucketsSeparate(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&MatchAchievement{}); err != nil {
		t.Fatalf("auto migrate match achievements: %v", err)
	}

	model := NewMatchModel(db)
	if err := model.SaveAchievementWithTx(nil, 88, "snooker_break_50", 1, 1); err != nil {
		t.Fatalf("save first achievement: %v", err)
	}
	if err := model.SaveAchievementWithTx(nil, 88, "snooker_break_50", 2, 1); err != nil {
		t.Fatalf("save second achievement: %v", err)
	}
	if err := model.SaveAchievementWithTx(nil, 88, "snooker_break_50", 3, 2); err != nil {
		t.Fatalf("save opponent achievement: %v", err)
	}

	var actorOne MatchAchievement
	if err := db.Where("match_id = ? AND achievement_type = ? AND actor = ?", 88, "snooker_break_50", 1).First(&actorOne).Error; err != nil {
		t.Fatalf("find actor 1 achievement: %v", err)
	}
	if actorOne.Count != 3 {
		t.Fatalf("expected actor 1 count 3, got %d", actorOne.Count)
	}

	var actorTwo MatchAchievement
	if err := db.Where("match_id = ? AND achievement_type = ? AND actor = ?", 88, "snooker_break_50", 2).First(&actorTwo).Error; err != nil {
		t.Fatalf("find actor 2 achievement: %v", err)
	}
	if actorTwo.Count != 3 {
		t.Fatalf("expected actor 2 count 3, got %d", actorTwo.Count)
	}

	var total int64
	if err := db.Model(&MatchAchievement{}).Where("match_id = ? AND achievement_type = ?", 88, "snooker_break_50").Count(&total).Error; err != nil {
		t.Fatalf("count achievements: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 actor buckets, got %d", total)
	}
}

func TestAchievementProgressEventModelCreateIfAbsentIsIdempotent(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&AchievementProgressEvent{}); err != nil {
		t.Fatalf("auto migrate progress events: %v", err)
	}

	model := NewAchievementProgressEventModel(db)
	event := NewAchievementProgressEvent(1001, "match", 3003, 1, "snooker_clearance", 1)

	created, err := model.CreateIfAbsent(event)
	if err != nil {
		t.Fatalf("create first event: %v", err)
	}
	if !created {
		t.Fatal("expected first event insert to create row")
	}

	duplicate := NewAchievementProgressEvent(1001, "match", 3003, 2, "snooker_clearance", 99)
	created, err = model.CreateIfAbsent(duplicate)
	if err != nil {
		t.Fatalf("create duplicate event: %v", err)
	}
	if created {
		t.Fatal("expected duplicate event to be ignored")
	}

	stored, err := model.FindBySourceMetric(1001, "match", 3003, "snooker_clearance")
	if err != nil {
		t.Fatalf("find stored event: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored event")
	}
	if stored.MetricValue != 1 {
		t.Fatalf("expected original metric value 1, got %d", stored.MetricValue)
	}
	if stored.GameType != 1 {
		t.Fatalf("expected original game type 1, got %d", stored.GameType)
	}

	var total int64
	if err := db.Model(&AchievementProgressEvent{}).Count(&total).Error; err != nil {
		t.Fatalf("count progress events: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 progress event row, got %d", total)
	}
}
