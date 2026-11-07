package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChallengeActiveReadHasFixedUpperBound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}, &User{}, &Match{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	now := time.Now().UTC()
	items := make([]Challenge, challengeReadLimit+1)
	for index := range items {
		items[index] = Challenge{Id: int64(index + 1), FromUserId: 2, ToUserId: 1, Status: ChallengeStatusAccepted, ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(time.Duration(index) * time.Second)}
	}
	if err := db.CreateInBatches(items, 100).Error; err != nil {
		t.Fatalf("seed challenges: %v", err)
	}
	rows, err := NewChallengeModel(db).ListActiveByUserWithRows(1, now)
	if err != nil || len(rows) != challengeReadLimit {
		t.Fatalf("active challenge read must be bounded: count=%d err=%v", len(rows), err)
	}
}

func TestExpiredChallengeLosesQualificationWithoutWorker(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}, &User{}, &Match{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	challengeModel := NewChallengeModel(db)
	if err := db.Create(&Challenge{Id: 1, FromUserId: 2, ToUserId: 1, Status: ChallengeStatusPending, ExpiresAt: time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatalf("seed expired challenge: %v", err)
	}
	if err := db.Create(&Challenge{Id: 2, FromUserId: 3, ToUserId: 1, Status: ChallengeStatusPending, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatalf("seed valid challenge: %v", err)
	}
	now := time.Now()
	if count, err := challengeModel.CountReceivedPending(1, now); err != nil || count != 1 {
		t.Fatalf("expired pending must not count: count=%d err=%v", count, err)
	}
	if rows, err := challengeModel.ListReceivedPendingWithRows(1, now); err != nil || len(rows) != 1 || rows[0].Id != 2 {
		t.Fatalf("expired pending must not be listed: rows=%+v err=%v", rows, err)
	}
	if summary, err := challengeModel.FindCurrentSummaryByUser(2, now); err != nil || summary != nil {
		t.Fatalf("expired own pending must not block re-send: summary=%+v err=%v", summary, err)
	}
	expired, err := challengeModel.ListExpiredUnstarted(10, now)
	if err != nil || len(expired) != 1 || expired[0].Id != 1 {
		t.Fatalf("worker must only see expired unstarted: list=%+v err=%v", expired, err)
	}
	if err := db.Create(&Challenge{Id: 3, FromUserId: 2, ToUserId: 1, Status: ChallengeStatusStarted, ExpiresAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed started challenge: %v", err)
	}
	if again, err := challengeModel.ListExpiredUnstarted(10, now); err != nil || len(again) != 1 {
		t.Fatalf("started challenge must never expire: list=%+v err=%v", again, err)
	}
	if ok, err := challengeModel.MarkExpiredIfUnstarted(1, now); err != nil || !ok {
		t.Fatalf("conditional expiry must succeed: ok=%v err=%v", ok, err)
	}
	var stored Challenge
	if err := db.First(&stored, 1).Error; err != nil || stored.Status != ChallengeStatusExpired {
		t.Fatalf("expired challenge must be marked: challenge=%+v err=%v", stored, err)
	}
}

func TestChallengeStatusTransitionsAreConditional(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	challengeModel := NewChallengeModel(db)
	if err := db.Create(&Challenge{Id: 1, FromUserId: 1, ToUserId: 2, Status: ChallengeStatusPending, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatalf("seed challenge: %v", err)
	}
	if ok, err := challengeModel.UpdateStatusWithTx(nil, 1, []int{ChallengeStatusAccepted}, ChallengeStatusCancelled, nil); err != nil || ok {
		t.Fatalf("wrong from-status must not transition: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.UpdateStatusWithTx(nil, 1, []int{ChallengeStatusPending}, ChallengeStatusAccepted, nil); err != nil || !ok {
		t.Fatalf("expected transition: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.SetWaitingWithTx(nil, 1, 2, time.Now()); err != nil || !ok {
		t.Fatalf("expected waiting set: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.SetWaitingWithTx(nil, 1, 1, time.Now()); err != nil || ok {
		t.Fatalf("second waiter must be rejected: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.ClearWaitingWithTx(nil, 1, 1); err != nil || ok {
		t.Fatalf("clearing another user's wait must fail: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.ClearWaitingWithTx(nil, 1, 2); err != nil || !ok {
		t.Fatalf("expected wait cleared: ok=%v err=%v", ok, err)
	}
	if ok, err := challengeModel.MarkCompletedByMatchIdWithTx(nil, 1); err != nil || ok {
		t.Fatalf("accepted challenge must not be marked completed: ok=%v err=%v", ok, err)
	}
}

func TestFindMainByIdResolvesMergedAlias(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Challenge{}); err != nil {
		t.Fatalf("prepare challenge schema: %v", err)
	}
	main := &Challenge{Id: 10, FromUserId: 1, ToUserId: 2, Status: ChallengeStatusAccepted, ExpiresAt: time.Now().Add(time.Hour)}
	alias := &Challenge{Id: 11, FromUserId: 2, ToUserId: 1, Status: ChallengeStatusMerged, ExpiresAt: time.Now().Add(time.Hour)}
	alias.MergedIntoId = &main.Id
	if err := db.Create(main).Error; err != nil {
		t.Fatalf("seed main: %v", err)
	}
	if err := db.Create(alias).Error; err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	resolved, err := NewChallengeModel(db).FindMainById(11)
	if err != nil || resolved == nil || resolved.Id != 10 {
		t.Fatalf("alias must resolve to main: resolved=%+v err=%v", resolved, err)
	}
}
