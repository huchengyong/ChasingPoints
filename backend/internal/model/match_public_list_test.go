package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchPublicListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&User{}, &Friend{}, &Match{}, &MatchRound{}); err != nil {
		t.Fatalf("prepare public match list schema: %v", err)
	}

	return db
}

func seedPublicListUser(t *testing.T, db *gorm.DB, id int64, nickname string) {
	t.Helper()

	if err := db.Create(&User{Id: id, Nickname: nickname}).Error; err != nil {
		t.Fatalf("seed user %d: %v", id, err)
	}
}

func seedPublicListMatch(t *testing.T, db *gorm.DB, match Match) {
	t.Helper()

	if match.MatchTime.IsZero() {
		match.MatchTime = time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	}
	if match.Visibility == "" {
		match.Visibility = MatchVisibilityPublic
	}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("seed match %d: %v", match.Id, err)
	}
}

func TestMatchModelListPublicMatchesFiltersHallByStatusAndGameType(t *testing.T) {
	db := newMatchPublicListTestDB(t)
	matchModel := NewMatchModel(db)

	seedPublicListUser(t, db, 1, "甲")
	seedPublicListUser(t, db, 2, "乙")
	seedPublicListUser(t, db, 3, "丙")

	opponent2 := int64(2)
	opponent3 := int64(3)
	seedPublicListMatch(t, db, Match{Id: 101, UserId: 1, OpponentId: &opponent2, OpponentName: "乙", GameType: 2, Status: 1, MyScore: 9, OpponentScore: 7})
	seedPublicListMatch(t, db, Match{Id: 102, UserId: 1, OpponentId: &opponent3, OpponentName: "丙", GameType: 3, Status: 1, MyScore: 5, OpponentScore: 4})
	seedPublicListMatch(t, db, Match{Id: 103, UserId: 2, OpponentId: &opponent3, OpponentName: "丙", GameType: 2, Status: 2, MyScore: 11, OpponentScore: 8})

	rows, total, err := matchModel.ListPublicMatches(PublicMatchListOptions{
		Scope:    "hall",
		Status:   1,
		GameType: 2,
		Offset:   0,
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("list hall matches: %v", err)
	}

	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected one hall match, got total=%d len=%d", total, len(rows))
	}
	if rows[0].Id != 101 || rows[0].Player1Name != "甲" || rows[0].Player2Name != "乙" {
		t.Fatalf("unexpected hall row: %#v", rows[0])
	}
}

func TestMatchModelListPublicMatchesFriendsScopeIncludesFriendParticipantMatches(t *testing.T) {
	db := newMatchPublicListTestDB(t)
	matchModel := NewMatchModel(db)

	seedPublicListUser(t, db, 1, "我")
	seedPublicListUser(t, db, 2, "好友")
	seedPublicListUser(t, db, 3, "好友对手")
	seedPublicListUser(t, db, 4, "陌生人")

	if err := db.Create(&Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	friendOpponent := int64(3)
	strangerOpponent := int64(3)
	seedPublicListMatch(t, db, Match{Id: 201, UserId: 2, OpponentId: &friendOpponent, OpponentName: "好友对手", GameType: 2, Status: 1, MyScore: 3, OpponentScore: 2})
	seedPublicListMatch(t, db, Match{Id: 202, UserId: 4, OpponentId: &strangerOpponent, OpponentName: "好友对手", GameType: 2, Status: 2, MyScore: 8, OpponentScore: 6})

	rows, total, err := matchModel.ListPublicMatches(PublicMatchListOptions{
		Scope:        "friends",
		ViewerUserId: 1,
		GameType:     2,
		Offset:       0,
		Limit:        20,
	})
	if err != nil {
		t.Fatalf("list friend matches: %v", err)
	}

	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected one friend match, got total=%d len=%d", total, len(rows))
	}
	if rows[0].Id != 201 || rows[0].Status != 1 {
		t.Fatalf("unexpected friend row: %#v", rows[0])
	}
}
