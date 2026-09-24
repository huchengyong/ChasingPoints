package model

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSeasonChallengeSnapshotModelUpsertBatchIsIdempotent(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&SeasonChallengeSnapshot{}); err != nil {
		t.Fatalf("auto migrate snapshots: %v", err)
	}

	model := NewSeasonChallengeSnapshotModel(db)
	archivedAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	snapshot := SeasonChallengeSnapshot{
		SeasonId:      3,
		UserId:        8,
		GameType:      2,
		ChallengeKey:  "season_wins_10",
		ChallengeName: "状态正盛",
		Threshold:     10,
		Progress:      7,
		ArchivedAt:    archivedAt,
	}
	if err := model.UpsertBatchWithTx(nil, []SeasonChallengeSnapshot{snapshot}); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	snapshot.Progress = 10
	snapshot.Completed = 1
	if err := model.UpsertBatchWithTx(nil, []SeasonChallengeSnapshot{snapshot}); err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}

	list, err := model.FindByUserSeasonAndGameType(8, 3, 2)
	if err != nil {
		t.Fatalf("find snapshots: %v", err)
	}
	if len(list) != 1 || list[0].Progress != 10 || list[0].Completed != 1 {
		t.Fatalf("expected one completed snapshot, got %+v", list)
	}
}

func TestSeasonSettlementModelCreateFindAndSave(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&SeasonSettlement{}); err != nil {
		t.Fatalf("auto migrate settlements: %v", err)
	}

	model := NewSeasonSettlementModel(db)
	startedAt := time.Now()
	settlement := &SeasonSettlement{
		SeasonId:  3,
		Status:    SeasonSettlementStatusRunning,
		Attempts:  1,
		StartedAt: &startedAt,
	}
	if err := model.CreateWithTx(nil, settlement); err != nil {
		t.Fatalf("create settlement: %v", err)
	}

	stored, err := model.FindBySeasonId(3)
	if err != nil || stored == nil {
		t.Fatalf("find settlement: stored=%+v err=%v", stored, err)
	}
	completedAt := time.Now()
	stored.Status = SeasonSettlementStatusCompleted
	stored.CompletedAt = &completedAt
	if err := model.SaveWithTx(nil, stored); err != nil {
		t.Fatalf("save settlement: %v", err)
	}

	stored, err = model.FindBySeasonId(3)
	if err != nil || stored == nil || stored.Status != SeasonSettlementStatusCompleted {
		t.Fatalf("expected completed settlement: stored=%+v err=%v", stored, err)
	}
}

func TestUserTitleModelFindRecentPermanentExcludesAchievementTitles(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&UserTitle{}); err != nil {
		t.Fatalf("auto migrate titles: %v", err)
	}

	oldAt := time.Now().Add(-time.Hour)
	newAt := time.Now()
	titles := []UserTitle{
		{UserId: 9, TitleKey: "achievement", TitleName: "成就称号", SourceType: "achievement", SourceRefId: 1, GrantedAt: &newAt},
		{UserId: 9, TitleKey: "season", TitleName: "赛季称号", SourceType: "season", SourceRefId: 2, GrantedAt: &oldAt},
		{UserId: 9, TitleKey: "tournament", TitleName: "赛事称号", SourceType: "tournament", SourceRefId: 3, GrantedAt: &newAt},
	}
	if err := db.Create(&titles).Error; err != nil {
		t.Fatalf("seed titles: %v", err)
	}

	list, err := NewUserTitleModel(db).FindRecentPermanentByUserId(9, 2)
	if err != nil {
		t.Fatalf("find recent permanent titles: %v", err)
	}
	if len(list) != 2 || list[0].SourceType != "tournament" || list[1].SourceType != "season" {
		t.Fatalf("unexpected permanent titles: %+v", list)
	}
}

func TestUserTitleModelEquipTitleSwitchesExclusiveTitle(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&UserTitle{}); err != nil {
		t.Fatalf("auto migrate titles: %v", err)
	}

	equippedAt := time.Now().Add(-time.Hour)
	titles := []UserTitle{
		{UserId: 9, TitleKey: "old", TitleName: "旧称号", SourceType: "achievement", SourceRefId: 1, Equipped: 1, EquippedAt: &equippedAt},
		{UserId: 9, TitleKey: "new", TitleName: "新称号", SourceType: "season", SourceRefId: 2},
	}
	if err := db.Create(&titles).Error; err != nil {
		t.Fatalf("seed titles: %v", err)
	}

	if err := NewUserTitleModel(db).EquipTitle(9, titles[1].Id, true); err != nil {
		t.Fatalf("equip new title: %v", err)
	}

	var stored []UserTitle
	if err := db.Where("user_id = ?", 9).Order("id ASC").Find(&stored).Error; err != nil {
		t.Fatalf("reload titles: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 titles, got %d", len(stored))
	}
	if stored[0].Equipped != 0 || stored[0].EquippedAt != nil {
		t.Fatalf("expected old title unequipped, got %+v", stored[0])
	}
	if stored[1].Equipped != 1 || stored[1].EquippedAt == nil {
		t.Fatalf("expected new title equipped, got %+v", stored[1])
	}
}

func TestUserTitleModelEquipTitleAllowsNoCurrentTitle(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&UserTitle{}); err != nil {
		t.Fatalf("auto migrate titles: %v", err)
	}

	equippedAt := time.Now()
	title := UserTitle{UserId: 9, TitleKey: "current", TitleName: "当前称号", SourceType: "achievement", SourceRefId: 1, Equipped: 1, EquippedAt: &equippedAt}
	if err := db.Create(&title).Error; err != nil {
		t.Fatalf("seed title: %v", err)
	}

	model := NewUserTitleModel(db)
	if err := model.EquipTitle(9, title.Id, false); err != nil {
		t.Fatalf("unequip title: %v", err)
	}

	stored, err := model.FindEquippedByUserId(9)
	if err != nil {
		t.Fatalf("find equipped title: %v", err)
	}
	if stored != nil {
		t.Fatalf("expected no current title, got %+v", stored)
	}
}

func TestUserTitleModelEquipTitleRollsBackInvalidOrForeignSelection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		foreignID bool
	}{
		{name: "invalid title"},
		{name: "foreign title", foreignID: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newAchievementClosedLoopTestDB(t)
			if err := db.AutoMigrate(&UserTitle{}); err != nil {
				t.Fatalf("auto migrate titles: %v", err)
			}

			equippedAt := time.Now().Add(-time.Hour)
			titles := []UserTitle{
				{UserId: 9, TitleKey: "current", TitleName: "当前称号", SourceType: "achievement", SourceRefId: 1, Equipped: 1, EquippedAt: &equippedAt},
				{UserId: 10, TitleKey: "foreign", TitleName: "他人称号", SourceType: "season", SourceRefId: 2},
			}
			if err := db.Create(&titles).Error; err != nil {
				t.Fatalf("seed titles: %v", err)
			}

			titleID := int64(999999)
			if tc.foreignID {
				titleID = titles[1].Id
			}
			err := NewUserTitleModel(db).EquipTitle(9, titleID, true)
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("expected record not found, got %v", err)
			}

			var current UserTitle
			if err := db.First(&current, titles[0].Id).Error; err != nil {
				t.Fatalf("reload current title: %v", err)
			}
			if current.Equipped != 1 || current.EquippedAt == nil {
				t.Fatalf("expected original title preserved after rollback, got %+v", current)
			}
		})
	}
}

func TestNotificationModelCreateIfAbsentUsesDedupeKey(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&Notification{}); err != nil {
		t.Fatalf("auto migrate notifications: %v", err)
	}

	model := NewNotificationModel(db)
	dedupeKey := "season_rollover:3:4"
	created, err := model.CreateIfAbsent(&Notification{UserId: 9, Type: "season_rollover", DedupeKey: &dedupeKey, Title: "S4 已开启"})
	if err != nil || !created {
		t.Fatalf("create notification: created=%v err=%v", created, err)
	}
	created, err = model.CreateIfAbsent(&Notification{UserId: 9, Type: "season_rollover", DedupeKey: &dedupeKey, Title: "重复"})
	if err != nil {
		t.Fatalf("create duplicate notification: %v", err)
	}
	if created {
		t.Fatal("expected duplicate notification to be ignored")
	}

	stored, err := model.FindLatestUnreadByType(9, "season_rollover")
	if err != nil || stored == nil || stored.Title != "S4 已开启" {
		t.Fatalf("unexpected stored notification: stored=%+v err=%v", stored, err)
	}
}

func TestAchievementProgressEventModelSumsSeasonMetricsByScope(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&AchievementProgressEvent{}); err != nil {
		t.Fatalf("auto migrate progress events: %v", err)
	}

	model := NewAchievementProgressEventModel(db)
	startAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)
	events := []*AchievementProgressEvent{
		NewAchievementProgressEvent(10, "match", 1, 3, "matches_total", 1, startAt.Add(time.Hour)),
		NewAchievementProgressEvent(10, "match", 2, 3, "matches_total", 1, startAt.Add(2*time.Hour)),
		NewAchievementProgressEvent(10, "match", 2, 3, "wins_total", 1, startAt.Add(2*time.Hour)),
		NewAchievementProgressEvent(10, "match", 3, 2, "matches_total", 5, startAt.Add(3*time.Hour)),
		NewAchievementProgressEvent(10, "match", 4, 3, "matches_total", 9, endAt.Add(time.Hour)),
	}
	for _, event := range events {
		if _, err := model.CreateIfAbsent(event); err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	totals, err := model.SumMetricsBetween(10, 3, []string{"matches_total", "wins_total"}, startAt, endAt)
	if err != nil {
		t.Fatalf("sum metrics: %v", err)
	}
	if totals["matches_total"] != 2 || totals["wins_total"] != 1 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestMatchModelMarkAchievementSynced(t *testing.T) {
	db := newAchievementClosedLoopTestDB(t)
	if err := db.AutoMigrate(&Match{}); err != nil {
		t.Fatalf("auto migrate matches: %v", err)
	}
	match := Match{UserId: 1, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Visibility: MatchVisibilityPublic, Status: 2, MatchTime: time.Now()}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}

	syncedAt := time.Now()
	if err := NewMatchModel(db).MarkAchievementSynced(match.Id, syncedAt); err != nil {
		t.Fatalf("mark achievement synced: %v", err)
	}
	var stored Match
	if err := db.First(&stored, match.Id).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	if stored.AchievementSyncedAt == nil {
		t.Fatal("expected achievement synced timestamp")
	}
}
