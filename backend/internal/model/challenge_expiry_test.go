package model

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPendingChallengeReadHasFixedUpperBound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	now := time.Now().UTC()
	items := make([]Challenge, pendingChallengeReadLimit+1)
	for index := range items {
		items[index] = Challenge{Id: int64(index + 1), FromUserId: 2, ToUserId: 1, Status: 1, ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(time.Duration(index) * time.Second)}
	}
	if err := db.CreateInBatches(items, 100).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}
	list, err := NewChallengeModel(db).GetPendingByUserId(1)
	if err != nil || len(list) != pendingChallengeReadLimit {
		t.Fatalf("pending challenge read must be bounded: count=%d err=%v", len(list), err)
	}
}

func TestExpiredChallengeCannotBeAcceptedOrRejectedWithoutGlobalSweep(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	if err := db.Create(&Challenge{Id: 1, FromUserId: 2, ToUserId: 1, Status: 0, ExpiresAt: time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatalf("seed expired challenge: %v", err)
	}
	challengeModel := NewChallengeModel(db)
	if err := challengeModel.Accept(1, 1, 0); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired challenge must not be accepted: %v", err)
	}
	if err := challengeModel.Reject(1, 1); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired challenge must not be rejected: %v", err)
	}
	var stored Challenge
	if err := db.First(&stored, 1).Error; err != nil || stored.Status != 0 {
		t.Fatalf("accept/reject must not mutate expired challenge outside worker: challenge=%+v err=%v", stored, err)
	}
}
