package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchRefereeHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&Match{}); err != nil {
		t.Fatalf("prepare referee history schema: %v", err)
	}

	return db
}

func TestListByRefereeUserIdReturnsOnlyCompletedAndCancelled(t *testing.T) {
	db := newMatchRefereeHistoryTestDB(t)
	matchModel := NewMatchModel(db)

	refereeID := int64(100)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	opponentID := int64(200)

	// Status 1 (ongoing) — should NOT be returned
	if err := matchModel.Create(&Match{
		Id:              1,
		UserId:          10,
		OpponentId:      &opponentID,
		OpponentName:    "对手A",
		GameType:        3,
		Status:          1,
		RefereeUserId:   &refereeID,
		RefereeJoinedAt: &now,
		MyScore:         1,
		OpponentScore:   0,
		MatchTime:       now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("create match 1: %v", err)
	}

	// Status 2 (completed) — SHOULD be returned
	endTime2 := now.Add(30 * time.Minute)
	win := 1
	if err := matchModel.Create(&Match{
		Id:               2,
		UserId:           10,
		OpponentId:       &opponentID,
		OpponentName:     "对手B",
		GameType:         3,
		Status:           2,
		Result:           &win,
		RefereeUserId:    &refereeID,
		RefereeJoinedAt:  &now,
		EndTime:          &endTime2,
		MyScore:          10,
		OpponentScore:    5,
		MatchTime:        now.Add(-2 * time.Hour),
		CompletionSource: CompletionSourceReferee,
	}); err != nil {
		t.Fatalf("create match 2: %v", err)
	}

	// Status 3 (cancelled) — SHOULD be returned
	endTime3 := now.Add(15 * time.Minute)
	if err := matchModel.Create(&Match{
		Id:               3,
		UserId:           10,
		OpponentId:       &opponentID,
		OpponentName:     "对手C",
		GameType:         1,
		Status:           3,
		RefereeUserId:    &refereeID,
		RefereeJoinedAt:  &now,
		EndTime:          &endTime3,
		MyScore:          3,
		OpponentScore:    0,
		MatchTime:        now.Add(-3 * time.Hour),
		CompletionSource: CompletionSourceUnknown,
	}); err != nil {
		t.Fatalf("create match 3: %v", err)
	}

	// Status 2 (completed) but different referee — should NOT be returned
	otherReferee := int64(999)
	endTime4 := now.Add(45 * time.Minute)
	if err := matchModel.Create(&Match{
		Id:               4,
		UserId:           10,
		OpponentId:       &opponentID,
		OpponentName:     "对手D",
		GameType:         2,
		Status:           2,
		Result:           &win,
		RefereeUserId:    &otherReferee,
		RefereeJoinedAt:  &now,
		EndTime:          &endTime4,
		MyScore:          7,
		OpponentScore:    3,
		MatchTime:        now.Add(-4 * time.Hour),
		CompletionSource: CompletionSourceReferee,
	}); err != nil {
		t.Fatalf("create match 4: %v", err)
	}

	matches, total, err := matchModel.ListByRefereeUserId(refereeID, 0, 20)
	if err != nil {
		t.Fatalf("ListByRefereeUserId: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 matches for referee %d, got total=%d", refereeID, total)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches in list, got %d", len(matches))
	}

	// Results should be ordered by end_time DESC (newest first)
	// match 2 endTime = now+30min, match 3 endTime = now+15min
	if matches[0].Id != 2 {
		t.Fatalf("expected match 2 (latest end_time) first, got match %d", matches[0].Id)
	}
	if matches[1].Id != 3 {
		t.Fatalf("expected match 3 second, got match %d", matches[1].Id)
	}
}

func TestListByRefereeUserIdPagination(t *testing.T) {
	db := newMatchRefereeHistoryTestDB(t)
	matchModel := NewMatchModel(db)

	refereeID := int64(200)
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	endTime := now.Add(30 * time.Minute)

	for i := int64(1); i <= 5; i++ {
		opponentID := int64(2000 + i)
		if err := matchModel.Create(&Match{
			Id:               i,
			UserId:           10,
			OpponentId:       &opponentID,
			OpponentName:     "对手",
			GameType:         3,
			Status:           2,
			RefereeUserId:    &refereeID,
			RefereeJoinedAt:  &now,
			EndTime:          &endTime,
			MyScore:          5,
			OpponentScore:    3,
			MatchTime:        now.Add(-time.Duration(i) * time.Hour),
			CompletionSource: CompletionSourceReferee,
		}); err != nil {
			t.Fatalf("create match %d: %v", i, err)
		}
	}

	// Page 1: offset=0, limit=2
	matches, total, err := matchModel.ListByRefereeUserId(refereeID, 0, 2)
	if err != nil {
		t.Fatalf("ListByRefereeUserId page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total=5, got %d", total)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches in page 1, got %d", len(matches))
	}

	// Page 3: offset=4, limit=2 — should return only 1 item
	matches, total, err = matchModel.ListByRefereeUserId(refereeID, 4, 2)
	if err != nil {
		t.Fatalf("ListByRefereeUserId page 3: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total still 5, got %d", total)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match in last page, got %d", len(matches))
	}

	// Empty offset beyond results
	matches, total, err = matchModel.ListByRefereeUserId(refereeID, 10, 2)
	if err != nil {
		t.Fatalf("ListByRefereeUserId far offset: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total still 5, got %d", total)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestListByRefereeUserIdNoMatches(t *testing.T) {
	db := newMatchRefereeHistoryTestDB(t)
	matchModel := NewMatchModel(db)

	matches, total, err := matchModel.ListByRefereeUserId(404, 0, 20)
	if err != nil {
		t.Fatalf("ListByRefereeUserId empty: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total=0, got %d", total)
	}
	if len(matches) != 0 {
		t.Fatalf("expected empty list, got %d items", len(matches))
	}
}
