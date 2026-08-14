package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCareerAchievementRebuildBatchWritesAreIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&AchievementProgressEvent{}, &UserAchievement{}); err != nil {
		t.Fatalf("prepare rebuild batch schema: %v", err)
	}

	occurredAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	events := []AchievementProgressEvent{
		*NewAchievementProgressEvent(1, "match", 10, 3, "matches_total", 1, occurredAt),
		*NewAchievementProgressEvent(2, "match", 10, 3, "matches_total", 1, occurredAt),
	}
	eventModel := NewAchievementProgressEventModel(db)
	created, err := eventModel.CreateCareerRebuildBatch(events)
	if err != nil || created != 2 {
		t.Fatalf("create event batch: created=%d err=%v", created, err)
	}
	created, err = eventModel.CreateCareerRebuildBatch(events)
	if err != nil || created != 0 {
		t.Fatalf("retry event batch must be idempotent: created=%d err=%v", created, err)
	}

	snapshots := []UserAchievement{
		{UserId: 1, AchievementId: 1, Progress: 1},
		{UserId: 2, AchievementId: 1, Progress: 1},
	}
	if err := NewUserAchievementModel(db).UpsertCareerRebuildSnapshots(snapshots); err != nil {
		t.Fatalf("upsert snapshot batch: %v", err)
	}
	var snapshotCount int64
	if err := db.Model(&UserAchievement{}).Count(&snapshotCount).Error; err != nil || snapshotCount != 2 {
		t.Fatalf("unexpected snapshot batch count: count=%d err=%v", snapshotCount, err)
	}
}

func TestUpsertCareerRebuildSnapshotKeepsConcurrentProgressMonotonic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&UserAchievement{}); err != nil {
		t.Fatalf("prepare user achievement schema: %v", err)
	}

	item := UserAchievement{UserId: 1, AchievementId: 2, Progress: 49}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("seed stale achievement: %v", err)
	}
	staleSnapshot := item

	liveUnlock := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	if err := db.Model(&UserAchievement{}).
		Where("user_id = ? AND achievement_id = ?", 1, 2).
		Updates(map[string]any{
			"progress":             50,
			"unlocked":             1,
			"unlocked_at":          &liveUnlock,
			"unlocked_source_type": "match",
			"unlocked_source_id":   100,
			"reward_granted":       1,
			"reward_granted_at":    &liveUnlock,
		}).Error; err != nil {
		t.Fatalf("simulate concurrent live unlock: %v", err)
	}

	achievementModel := NewUserAchievementModel(db)
	if err := achievementModel.UpsertCareerRebuildSnapshot(&staleSnapshot); err != nil {
		t.Fatalf("upsert stale rebuild snapshot: %v", err)
	}

	var current UserAchievement
	if err := db.Where("user_id = ? AND achievement_id = ?", 1, 2).First(&current).Error; err != nil {
		t.Fatalf("find current achievement: %v", err)
	}
	if current.Progress != 50 || current.Unlocked != 1 || current.RewardGranted != 1 || current.UnlockedAt == nil || !current.UnlockedAt.Equal(liveUnlock) || current.UnlockedSourceType != "match" || current.UnlockedSourceId != 100 {
		t.Fatalf("stale rebuild snapshot regressed live progress: %+v", current)
	}

	historicalUnlock := liveUnlock.Add(-24 * time.Hour)
	historicalSnapshot := UserAchievement{
		UserId:             1,
		AchievementId:      2,
		Progress:           40,
		Unlocked:           1,
		UnlockedAt:         &historicalUnlock,
		UnlockedSourceType: "tournament_finish",
		UnlockedSourceId:   80,
		RewardGranted:      1,
		RewardGrantedAt:    &historicalUnlock,
	}
	if err := achievementModel.UpsertCareerRebuildSnapshot(&historicalSnapshot); err != nil {
		t.Fatalf("upsert earlier historical snapshot: %v", err)
	}
	if err := db.Where("user_id = ? AND achievement_id = ?", 1, 2).First(&current).Error; err != nil {
		t.Fatalf("reload current achievement: %v", err)
	}
	if current.Progress != 50 || current.UnlockedAt == nil || !current.UnlockedAt.Equal(historicalUnlock) || current.UnlockedSourceType != "tournament_finish" || current.UnlockedSourceId != 80 || current.RewardGrantedAt == nil || !current.RewardGrantedAt.Equal(historicalUnlock) {
		t.Fatalf("historical merge should preserve high progress and earlier source: %+v", current)
	}
}

func TestCareerAchievementRebuildSourceQueriesReturnOnlyAuthoritativeHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&Match{}, &MatchRound{}, &MatchAction{}, &MatchAchievement{}, &Tournament{}, &TournamentParticipant{}); err != nil {
		t.Fatalf("prepare rebuild source schema: %v", err)
	}

	opponentID := int64(2)
	win := 1
	draw := 3
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	matches := []Match{
		{Id: 1, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &win, MatchTime: base},
		{Id: 2, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModePractice, Status: 2, Result: &win, MatchTime: base.Add(time.Hour)},
		{Id: 3, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &draw, MatchTime: base.Add(2 * time.Hour)},
		{Id: 4, UserId: 1, OpponentName: "线下对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &win, MatchTime: base.Add(3 * time.Hour)},
		{Id: 5, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &win, MatchTime: base.Add(4 * time.Hour)},
		{Id: 6, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &win, MatchTime: base.Add(5 * time.Hour)},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	winner := 1
	rounds := []MatchRound{
		{MatchId: 1, RoundNo: 1, Winner: &winner, WinType: "break_clear"},
		{MatchId: 2, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 3, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 4, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 6, RoundNo: 1, Winner: &winner, WinType: "normal"},
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed rounds: %v", err)
	}
	if err := db.Model(&MatchRound{}).Where("match_id = ?", 6).Update("win_type", nil).Error; err != nil {
		t.Fatalf("seed nullable historical win type: %v", err)
	}
	if err := db.Create(&[]MatchAchievement{
		{MatchId: 1, Actor: 1, AchievementType: "break_and_run", Count: 1},
		{MatchId: 2, Actor: 1, AchievementType: "golden_break", Count: 1},
	}).Error; err != nil {
		t.Fatalf("seed match achievements: %v", err)
	}

	matchModel := NewMatchModel(db)
	validMatches, err := matchModel.ListCompletedForAchievementRebuild()
	if err != nil {
		t.Fatalf("list valid matches: %v", err)
	}
	if len(validMatches) != 2 || validMatches[0].Id != 1 || validMatches[1].Id != 6 {
		t.Fatalf("expected authoritative matches 1 and nullable-win-type match 6, got %+v", validMatches)
	}
	seasonMatches, err := matchModel.ListCompletedForSeasonRecords()
	if err != nil || len(seasonMatches) != 2 || seasonMatches[0].Id != 1 || seasonMatches[1].Id != 6 {
		t.Fatalf("season records must use the same authoritative matches: matches=%+v err=%v", seasonMatches, err)
	}
	bulkRounds, err := matchModel.ListCompletedRoundsByMatchIDs([]int64{1, 6})
	if err != nil || len(bulkRounds) != 2 || bulkRounds[0].WinType != "break_clear" || bulkRounds[1].MatchId != 6 {
		t.Fatalf("unexpected bulk rounds: rounds=%+v err=%v", bulkRounds, err)
	}
	bulkAchievements, err := matchModel.ListAchievementsByMatchIDs([]int64{1})
	if err != nil || len(bulkAchievements) != 1 || bulkAchievements[0].MatchId != 1 {
		t.Fatalf("unexpected bulk achievements: achievements=%+v err=%v", bulkAchievements, err)
	}

	tournaments := []Tournament{
		{Id: 20, CreatorId: 1, Name: "进行中赛事", GameType: 3, Status: 1, CreatedAt: base},
		{Id: 10, CreatorId: 1, Name: "已结束赛事", GameType: 2, Status: 2, CreatedAt: base.Add(-time.Hour)},
	}
	if err := db.Create(&tournaments).Error; err != nil {
		t.Fatalf("seed tournaments: %v", err)
	}
	participants := []TournamentParticipant{
		{TournamentId: 20, UserId: 2, CreatedAt: base.Add(2 * time.Hour)},
		{TournamentId: 10, UserId: 1, FinalRank: 1, Status: 3, CreatedAt: base.Add(time.Hour)},
	}
	if err := db.Create(&participants).Error; err != nil {
		t.Fatalf("seed participants: %v", err)
	}

	tournamentList, err := NewTournamentModel(db).ListForAchievementRebuild()
	if err != nil || len(tournamentList) != 2 || tournamentList[0].Id != 10 || tournamentList[1].Id != 20 {
		t.Fatalf("unexpected rebuild tournaments: tournaments=%+v err=%v", tournamentList, err)
	}
	participantList, err := NewTournamentParticipantModel(db).ListForAchievementRebuild()
	if err != nil || len(participantList) != 2 || participantList[0].TournamentId != 10 || participantList[1].TournamentId != 20 {
		t.Fatalf("unexpected rebuild participants: participants=%+v err=%v", participantList, err)
	}
}
