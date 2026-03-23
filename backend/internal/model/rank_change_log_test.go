package model

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestRankChangeLogTableName(t *testing.T) {
	var log RankChangeLog

	if got := log.TableName(); got != "rank_change_logs" {
		t.Fatalf("expected table name rank_change_logs, got %s", got)
	}
}

func TestCreateRankChangeLogsAllowsEmptySlice(t *testing.T) {
	model := NewRankingModel(nil)

	if err := model.CreateRankChangeLogs(nil, nil); err != nil {
		t.Fatalf("expected nil error for empty logs, got %v", err)
	}
}

func TestUpdateRankingSnapshotRejectsNilRanking(t *testing.T) {
	model := NewRankingModel(nil)

	if err := model.UpdateRankingSnapshot(nil, nil); err == nil {
		t.Fatal("expected error for nil ranking")
	}
}

func TestLeaderboardOrderClauseUsesStableTieBreaker(t *testing.T) {
	got := leaderboardOrderClause("ur")
	want := "ur.rank_score DESC, ur.total_wins DESC, ur.updated_at ASC, ur.user_id ASC"

	if got != want {
		t.Fatalf("expected order clause %q, got %q", want, got)
	}
}

func TestHigherRankingConditionUsesSameTieBreakerOrder(t *testing.T) {
	ranking := &UserRanking{
		GameType:  3,
		RankScore: 500,
		TotalWins: 12,
		UserId:    99,
	}

	condition, args := higherRankingCondition(ranking)
	if condition == "" {
		t.Fatal("expected non-empty condition")
	}
	if len(args) != 11 {
		t.Fatalf("expected 11 args, got %d", len(args))
	}
	if args[0] != ranking.GameType {
		t.Fatalf("expected first arg to be game type %d, got %#v", ranking.GameType, args[0])
	}
	if args[1] != ranking.RankScore || args[2] != ranking.RankScore {
		t.Fatalf("expected rank score args to follow game type, got %#v", args[:3])
	}
	if !strings.Contains(condition, "game_type = ? AND") {
		t.Fatalf("expected condition to be filtered by game_type first, got %q", condition)
	}
}

func TestNormalizeRankingGameTypeDefaultsToChineseEight(t *testing.T) {
	if got := normalizeRankingGameType(0); got != defaultRankingGameType {
		t.Fatalf("expected default ranking game type %d, got %d", defaultRankingGameType, got)
	}
}

func TestSeasonRecordUsesMigratedUniqueIndexName(t *testing.T) {
	field, ok := reflect.TypeOf(SeasonRecord{}).FieldByName("GameType")
	if !ok {
		t.Fatal("expected GameType field on SeasonRecord")
	}
	gormTag := field.Tag.Get("gorm")
	if !strings.Contains(gormTag, "uniqueIndex:uk_season_user_game") {
		t.Fatalf("expected season record gorm tag to use uk_season_user_game, got %q", gormTag)
	}
}

func TestLeaderboardEligibilityConditionFiltersUsersWithoutMatches(t *testing.T) {
	got := leaderboardEligibilityCondition("ur")
	want := "(ur.total_wins + ur.total_losses) > 0"

	if got != want {
		t.Fatalf("expected leaderboard eligibility condition %q, got %q", want, got)
	}
}

func TestBuildLegacyRankChangeLogRowsOmitsGameTypeColumn(t *testing.T) {
	rows := buildLegacyRankChangeLogRows([]RankChangeLog{{
		UserId:           1,
		MatchId:          2,
		ChangeType:       rankChangeTypeMatchResult,
		GameType:         4,
		Result:           "win",
		BaseScore:        20,
		AchievementScore: 5,
		FinalChange:      25,
		BeforeScore:      100,
		AfterScore:       125,
		BeforeLevel:      1,
		AfterLevel:       2,
		OperatorUserId:   99,
		Remark:           "legacy insert",
		EffectiveAt:      time.Unix(100, 0),
		CreatedAt:        time.Unix(200, 0),
	}})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if _, ok := rows[0]["game_type"]; ok {
		t.Fatal("expected legacy row payload to omit game_type")
	}
	if got := rows[0]["user_id"]; got != int64(1) {
		t.Fatalf("expected user_id 1, got %#v", got)
	}
}

func TestApplyRankChangeLogGameTypeFilterCanSkipGameTypeConstraint(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry run gorm db: %v", err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return applyRankChangeLogGameTypeFilter(
			tx.Model(&RankChangeLog{}).Where("user_id = ?", 1),
			false,
			4,
		).Find(&[]RankChangeLog{})
	})
	if strings.Contains(sql, "game_type") {
		t.Fatalf("expected sql without game_type filter, got %q", sql)
	}
}

func TestApplyRankChangeLogGameTypeFilterAddsGameTypeConstraintWhenEnabled(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry run gorm db: %v", err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return applyRankChangeLogGameTypeFilter(
			tx.Model(&RankChangeLog{}).Where("user_id = ?", 1),
			true,
			4,
		).Find(&[]RankChangeLog{})
	})
	if !strings.Contains(sql, "game_type") {
		t.Fatalf("expected sql with game_type filter, got %q", sql)
	}
}
