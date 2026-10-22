package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRankingAnalysisTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&RankChangeLog{}); err != nil {
		t.Fatalf("prepare rank change log schema: %v", err)
	}
	return db
}

func TestListMatchRankChangesForAnalysisIsBoundedFilteredAndNewestFirst(t *testing.T) {
	db := newRankingAnalysisTestDB(t)
	base := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	var logs []RankChangeLog
	for i := 0; i < MaxAnalysisRankChangeRows+50; i++ {
		logs = append(logs, RankChangeLog{UserId: 1, MatchId: int64(i + 1), ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: i + 1, Remark: "r", EffectiveAt: base.Add(time.Duration(i) * time.Minute)})
	}
	// 批量行的默认字段必须一致（SQLite 不支持多行 INSERT 里混用 DEFAULT）。
	extra := func(userID, matchID int64, gameType int, at time.Time) RankChangeLog {
		return RankChangeLog{UserId: userID, MatchId: matchID, ChangeType: RankChangeTypeMatchResult, GameType: gameType, Result: "win", AfterScore: 1, Remark: "r", EffectiveAt: at}
	}
	logs = append(logs,
		extra(1, 9001, 1, base.Add(24*time.Hour)),  // 其他球种
		extra(2, 9002, 3, base.Add(24*time.Hour)),  // 别人的记录
		extra(1, 9003, 3, base.Add(-48*time.Hour)), // since 之前
	)
	if err := db.CreateInBatches(&logs, 200).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}
	// 规则切换记录的 match_id 为 0，与上面的批量行默认值不同，单独插入。
	if err := db.Create(&RankChangeLog{UserId: 1, MatchId: 0, ChangeType: RankChangeTypeRatingReset, GameType: 3, Result: "reset", AfterScore: 1, Remark: "r", EffectiveAt: base.Add(24 * time.Hour)}).Error; err != nil {
		t.Fatalf("seed reset log: %v", err)
	}
	model := NewRankingModel(db)

	got, err := model.ListMatchRankChangesForAnalysis(1, 3, base, 0)
	if err != nil {
		t.Fatalf("list for analysis: %v", err)
	}
	if len(got) != MaxAnalysisRankChangeRows {
		t.Fatalf("read must be capped at %d rows, got %d", MaxAnalysisRankChangeRows, len(got))
	}
	if got[0].MatchId != int64(MaxAnalysisRankChangeRows+50) || got[0].AfterScore != MaxAnalysisRankChangeRows+50 || got[0].Remark != "r" || got[0].Result != "win" {
		t.Fatalf("newest row must come first with the analysis columns: %+v", got[0])
	}
	for _, row := range got {
		if row.UserId != 0 || row.ChangeType != "" {
			t.Fatalf("only the analysis columns may be selected, got %+v", row)
		}
	}

	small, err := model.ListMatchRankChangesForAnalysis(1, 3, base, 10)
	if err != nil || len(small) != 10 {
		t.Fatalf("explicit limit must be honoured: len=%d err=%v", len(small), err)
	}
	other, err := model.ListMatchRankChangesForAnalysis(1, 1, base, 0)
	if err != nil || len(other) != 1 || other[0].MatchId != 9001 {
		t.Fatalf("game type filter: %+v err=%v", other, err)
	}
}

func TestListOpponentBeforeScoresReturnsTheOtherSideOfEachMatch(t *testing.T) {
	db := newRankingAnalysisTestDB(t)
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	if err := db.Create(&[]RankChangeLog{
		{UserId: 1, MatchId: 10, ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 1500, EffectiveAt: at},
		{UserId: 2, MatchId: 10, ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 1700, EffectiveAt: at},
		{UserId: 1, MatchId: 11, ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 1510, EffectiveAt: at}, // 对手没有记录
		{UserId: 1, MatchId: 12, ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 1520, EffectiveAt: at},
		{UserId: 3, MatchId: 12, ChangeType: RankChangeTypeMatchResult, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 1300, EffectiveAt: at},
		{UserId: 4, MatchId: 12, ChangeType: RankChangeTypeRatingReset, GameType: 3, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 999, EffectiveAt: at}, // 不是对局结算
		{UserId: 5, MatchId: 13, ChangeType: RankChangeTypeMatchResult, GameType: 1, Result: "win", AfterScore: 1, Remark: "r", BeforeScore: 888, EffectiveAt: at}, // 其他球种
	}).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	got, err := NewRankingModel(db).ListOpponentBeforeScores([]int64{10, 11, 12, 13}, 1, 3)
	if err != nil {
		t.Fatalf("list opponents: %v", err)
	}
	byMatch := map[int64]OpponentRatingRef{}
	for _, ref := range got {
		byMatch[ref.MatchId] = ref
	}
	if len(got) != 2 || byMatch[10].UserId != 2 || byMatch[10].BeforeScore != 1700 || byMatch[12].UserId != 3 || byMatch[12].BeforeScore != 1300 {
		t.Fatalf("unexpected opponent refs: %+v", got)
	}

	empty, err := NewRankingModel(db).ListOpponentBeforeScores(nil, 1, 3)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty input must not query: %v err=%v", empty, err)
	}
}
