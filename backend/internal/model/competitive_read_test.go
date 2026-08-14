package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCompetitiveReadModelParticipantProjectionIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&MatchParticipantResult{},
		&UserCompetitiveStats{},
		&UserOpponentStats{},
		&UserOpponentStrengthBucket{},
		&CompetitiveReadModelRebuildCheckpoint{},
	); err != nil {
		t.Fatalf("prepare competitive read schema: %v", err)
	}

	model := NewCompetitiveReadModel(db)
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	created, err := model.InsertParticipantIfAbsentWithTx(nil, &MatchParticipantResult{
		MatchId:         10,
		UserId:          1,
		OpponentUserId:  2,
		OpponentNameKey: "user:2",
		OpponentName:    "对手",
		GameType:        3,
		MatchMode:       MatchModeRanked,
		Result:          1,
		CompletedAt:     completedAt,
	})
	if err != nil || !created {
		t.Fatalf("insert first projection: created=%t err=%v", created, err)
	}
	created, err = model.InsertParticipantIfAbsentWithTx(nil, &MatchParticipantResult{
		MatchId: 10, UserId: 1, GameType: 3, MatchMode: MatchModeRanked, CompletedAt: completedAt,
	})
	if err != nil || created {
		t.Fatalf("repeat projection must be ignored: created=%t err=%v", created, err)
	}

	found, err := model.FindParticipantByMatchAndUser(10, 1)
	if err != nil || found == nil || found.OpponentUserId != 2 {
		t.Fatalf("find inserted projection: result=%+v err=%v", found, err)
	}

	if err := model.UpsertCheckpoint(&CompetitiveReadModelRebuildCheckpoint{JobName: "competitive", CursorMatchId: 10}); err != nil {
		t.Fatalf("create checkpoint: %v", err)
	}
	if err := model.UpsertCheckpoint(&CompetitiveReadModelRebuildCheckpoint{JobName: "competitive", CursorMatchId: 20, Paused: true}); err != nil {
		t.Fatalf("update checkpoint: %v", err)
	}
	checkpoint, err := model.FindCheckpoint("competitive")
	if err != nil || checkpoint == nil || checkpoint.CursorMatchId != 20 || !checkpoint.Paused {
		t.Fatalf("checkpoint upsert result=%+v err=%v", checkpoint, err)
	}
}
