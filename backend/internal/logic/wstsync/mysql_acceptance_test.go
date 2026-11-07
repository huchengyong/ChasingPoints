package wstsync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/model"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormMySQL "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWSTBackfillMySQLAcceptance(t *testing.T) {
	if os.Getenv("RUN_WST_MYSQL_ACCEPTANCE_TESTS") != "1" {
		t.Skip("set RUN_WST_MYSQL_ACCEPTANCE_TESTS=1 to run WST MySQL acceptance")
	}

	db, rawDB := newWSTMySQLAcceptanceDB(t)

	t.Run("official generated unique indexes", func(t *testing.T) {
		assertWSTGeneratedUniqueIndexes(t, rawDB)
	})
	t.Run("soft-deleted IDs fail dry-run and import pre-checks", func(t *testing.T) {
		assertWSTSoftDeletePreChecks(t, db, rawDB)
	})
	t.Run("repeated import keeps all business IDs", func(t *testing.T) {
		client := mysqlAcceptanceClient("idempotent-tournament", "idempotent-match", "idempotent-home", "idempotent-away", "Idempotent Tournament")
		service := newWSTSyncServiceForTest(t, db, client)
		service.now = func() time.Time { return time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC) }

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err != nil || summary.Status != BackfillStatusCompleted || summary.CommitState != "committed" {
			t.Fatalf("first backfill failed: summary=%+v err=%v", summary, err)
		}
		first := mysqlAcceptanceSnapshotForPrefix(t, rawDB, "idempotent-")
		if len(first.PlayerIDs) != 2 || len(first.TournamentIDs) != 1 || len(first.MatchIDs) != 1 || len(first.EventNewsIDs) != 1 {
			t.Fatalf("unexpected four-table snapshot after first import: %+v", first)
		}

		summary, err = service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err != nil || summary.Status != BackfillStatusCompleted || summary.CommitState != "committed" {
			t.Fatalf("second backfill failed: summary=%+v err=%v", summary, err)
		}
		second := mysqlAcceptanceSnapshotForPrefix(t, rawDB, "idempotent-")
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("repeated import changed business IDs: first=%+v second=%+v", first, second)
		}
	})
	t.Run("mid-transaction failure rolls back all facts", func(t *testing.T) {
		mustMySQLExec(t, rawDB, "CREATE UNIQUE INDEX `uk_wst_acceptance_event_title` ON `event_news_events` (`title`)")
		mustMySQLExec(t, rawDB, "INSERT INTO `event_news_events` (`title`, `game_type`, `source_type`) VALUES (?, ?, ?)", "Rollback Tournament", 1, "manual")

		service := newWSTSyncServiceForTest(t, db, mysqlAcceptanceClient("rollback-tournament", "rollback-match", "rollback-home", "rollback-away", "Rollback Tournament"))
		service.now = func() time.Time { return time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC) }
		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err == nil || summary == nil || summary.Status != BackfillStatusFailed || summary.CommitState != "rolled_back" {
			t.Fatalf("expected confirmed rollback: summary=%+v err=%v", summary, err)
		}
		var mysqlErr *mysqlDriver.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 || !strings.Contains(mysqlErr.Message, "uk_wst_acceptance_event_title") {
			t.Fatalf("expected failure at event-news write, got %v", err)
		}
		if got := mysqlAcceptanceSnapshotForPrefix(t, rawDB, "rollback-"); len(got.PlayerIDs)+len(got.TournamentIDs)+len(got.MatchIDs)+len(got.EventNewsIDs) != 0 {
			t.Fatalf("transaction failure left partial WST facts: %+v", got)
		}
	})
}

func newWSTMySQLAcceptanceDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()

	config := mysqlDriver.NewConfig()
	config.User = os.Getenv("MYSQL_USER")
	config.Passwd = os.Getenv("MYSQL_PASSWORD")
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(os.Getenv("MYSQL_HOST"), os.Getenv("MYSQL_PORT"))
	config.ParseTime = true
	config.Loc = time.UTC
	if config.User == "" || os.Getenv("MYSQL_HOST") == "" || os.Getenv("MYSQL_PORT") == "" {
		t.Fatal("MYSQL_HOST, MYSQL_PORT and MYSQL_USER are required for WST MySQL acceptance")
	}

	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	if err := admin.Ping(); err != nil {
		t.Fatalf("ping MySQL: %v", err)
	}

	databaseName := fmt.Sprintf("chasing_points_wst_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + databaseName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create temporary WST database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`"); err != nil {
			t.Errorf("drop temporary WST database: %v", err)
		}
	})

	config.DBName = databaseName
	dsn := config.FormatDSN()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate WST MySQL acceptance test")
	}
	migrationsDir := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../migrations"))
	command := exec.Command("goose", "-dir", migrationsDir, "mysql", dsn, "up")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("apply migrations to temporary WST database: %v: %s", err, redactMySQLAcceptanceOutput(string(output), dsn, config.Passwd))
	}

	db, err := gorm.Open(gormMySQL.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open temporary WST Gorm database: %v", err)
	}
	rawDB, err := db.DB()
	if err != nil {
		t.Fatalf("get temporary WST SQL database: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	return db, rawDB
}

func redactMySQLAcceptanceOutput(output, dsn, password string) string {
	output = strings.ReplaceAll(output, dsn, "[redacted DSN]")
	if password != "" {
		output = strings.ReplaceAll(output, password, "[redacted password]")
	}
	return output
}

func assertWSTGeneratedUniqueIndexes(t *testing.T, db *sql.DB) {
	t.Helper()

	parent := mustMySQLInsertID(t, db, "INSERT INTO `tournaments` (`creator_id`, `name`, `game_type`) VALUES (0, 'WST index parent', 1)")
	tests := []struct {
		name, table, generatedColumn, index string
		insert                              func(sourceType, sourceID string) error
	}{
		{
			name: "players", table: "players", generatedColumn: "official_source_player_id", index: "uk_players_official_source_player_id",
			insert: func(sourceType, sourceID string) error {
				_, err := db.Exec("INSERT INTO `players` (`source_type`, `source_player_id`, `display_name`) VALUES (?, ?, 'WST acceptance')", sourceType, sourceID)
				return err
			},
		},
		{
			name: "tournaments", table: "tournaments", generatedColumn: "official_source_tournament_id", index: "uk_tournaments_official_source_tournament_id",
			insert: func(sourceType, sourceID string) error {
				_, err := db.Exec("INSERT INTO `tournaments` (`creator_id`, `name`, `game_type`, `source_type`, `source_tournament_id`) VALUES (0, 'WST acceptance', 1, ?, ?)", sourceType, sourceID)
				return err
			},
		},
		{
			name: "matches", table: "tournament_matches", generatedColumn: "official_source_match_id", index: "uk_tournament_matches_official_source_match_id",
			insert: func(sourceType, sourceID string) error {
				_, err := db.Exec("INSERT INTO `tournament_matches` (`tournament_id`, `source_type`, `source_match_id`) VALUES (?, ?, ?)", parent, sourceType, sourceID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			if err := db.QueryRow(`
				SELECT COUNT(*)
				FROM information_schema.COLUMNS c
				JOIN information_schema.STATISTICS s
				  ON s.TABLE_SCHEMA = c.TABLE_SCHEMA
				 AND s.TABLE_NAME = c.TABLE_NAME
				 AND s.COLUMN_NAME = c.COLUMN_NAME
				WHERE c.TABLE_SCHEMA = DATABASE() AND c.TABLE_NAME = ? AND c.COLUMN_NAME = ?
				  AND c.EXTRA LIKE '%GENERATED%' AND c.GENERATION_EXPRESSION <> ''
				  AND s.INDEX_NAME = ? AND s.NON_UNIQUE = 0`, tt.table, tt.generatedColumn, tt.index).Scan(&count); err != nil || count != 1 {
				t.Fatalf("missing generated unique index %s: count=%d err=%v", tt.index, count, err)
			}

			sourceID := "wst-unique-" + tt.name
			if err := tt.insert("official", sourceID); err != nil {
				t.Fatalf("insert first official %s row: %v", tt.name, err)
			}
			expectMySQLDuplicate(t, tt.insert("official", sourceID))
			for i := 0; i < 2; i++ {
				if err := tt.insert("manual", sourceID); err != nil {
					t.Fatalf("manual source ID was incorrectly constrained for %s: %v", tt.name, err)
				}
				if err := tt.insert("official", ""); err != nil {
					t.Fatalf("empty official source ID was incorrectly constrained for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func assertWSTSoftDeletePreChecks(t *testing.T, db *gorm.DB, rawDB *sql.DB) {
	t.Helper()

	player := &model.Player{SourceType: wstSourceType, SourcePlayerId: "mysql-soft-player", DisplayName: "Deleted Player"}
	if err := db.Create(player).Error; err != nil {
		t.Fatalf("seed soft-deleted player: %v", err)
	}
	if err := db.Delete(player).Error; err != nil {
		t.Fatalf("soft-delete player: %v", err)
	}
	_, duplicatePlayerErr := rawDB.Exec("INSERT INTO `players` (`source_type`, `source_player_id`) VALUES (?, ?)", wstSourceType, player.SourcePlayerId)
	expectMySQLDuplicate(t, duplicatePlayerErr)
	assertWSTPreCheckModes(t, db, rawDB, mysqlAcceptanceClient("soft-player-tournament", "soft-player-match", player.SourcePlayerId, "soft-player-away", "Soft Player Tournament"), player.SourcePlayerId)

	tournament := &model.Tournament{Name: "Soft Match Parent", GameType: 1, SourceType: wstSourceType, SourceTournamentId: "soft-match-tournament", SourceSeasonId: "2025"}
	if err := db.Create(tournament).Error; err != nil {
		t.Fatalf("seed soft-match tournament: %v", err)
	}
	match := &model.TournamentMatch{TournamentId: tournament.Id, SourceType: wstSourceType, SourceMatchId: "mysql-soft-match"}
	if err := db.Create(match).Error; err != nil {
		t.Fatalf("seed soft-deleted match: %v", err)
	}
	if err := db.Delete(match).Error; err != nil {
		t.Fatalf("soft-delete match: %v", err)
	}
	_, duplicateMatchErr := rawDB.Exec("INSERT INTO `tournament_matches` (`tournament_id`, `source_type`, `source_match_id`) VALUES (?, ?, ?)", tournament.Id, wstSourceType, match.SourceMatchId)
	expectMySQLDuplicate(t, duplicateMatchErr)
	assertWSTPreCheckModes(t, db, rawDB, mysqlAcceptanceClient(tournament.SourceTournamentId, match.SourceMatchId, "soft-match-home", "soft-match-away", tournament.Name), match.SourceMatchId)

	eventTournament := &model.Tournament{Name: "Soft Event Parent", GameType: 1, SourceType: wstSourceType, SourceTournamentId: "soft-event-tournament", SourceSeasonId: "2025"}
	if err := db.Create(eventTournament).Error; err != nil {
		t.Fatalf("seed soft-event tournament: %v", err)
	}
	event := &model.EventNews{Title: "Soft Event", TournamentId: eventTournament.Id, GameType: 1, SourceType: wstSourceType}
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("seed soft-deleted event news: %v", err)
	}
	if err := db.Delete(event).Error; err != nil {
		t.Fatalf("soft-delete event news: %v", err)
	}
	assertWSTPreCheckModes(t, db, rawDB, mysqlAcceptanceClient(eventTournament.SourceTournamentId, "soft-event-match", "soft-event-home", "soft-event-away", eventTournament.Name), fmt.Sprintf("tournament_id=%d", eventTournament.Id))
}

func assertWSTPreCheckModes(t *testing.T, db *gorm.DB, rawDB *sql.DB, client *fakeDataClient, conflictID string) {
	t.Helper()

	for _, dryRun := range []bool{true, false} {
		before := readMySQLAcceptanceCounts(t, rawDB)
		summary, err := newWSTSyncServiceForTest(t, db, client).Sync(context.Background(), mysqlAcceptanceParams(dryRun))
		if err != nil || summary == nil || summary.Status != BackfillStatusFailed || summary.ExitCode != ExitCodeFailed {
			t.Fatalf("dry_run=%t did not fail pre-check: summary=%+v err=%v", dryRun, summary, err)
		}
		if !strings.Contains(strings.Join(summary.Warnings, "\n"), conflictID) {
			t.Fatalf("dry_run=%t did not identify conflict %q: %+v", dryRun, conflictID, summary.Warnings)
		}
		if after := readMySQLAcceptanceCounts(t, rawDB); after != before {
			t.Fatalf("dry_run=%t pre-check wrote data: before=%+v after=%+v", dryRun, before, after)
		}
	}
}

func mysqlAcceptanceClient(tournamentID, matchID, homeID, awayID, name string) *fakeDataClient {
	return &fakeDataClient{
		seasons: []SeasonResource{{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}}},
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {{ID: tournamentID, Attributes: TournamentAttributes{Name: name, StartDate: "2025-06-01", EndDate: "2025-06-07", Season: TournamentSeason{ID: "2025"}}}},
		},
		matchesPages: []MatchListResponse{{Data: []MatchResource{{
			ID: matchID,
			Attributes: MatchAttributes{
				TournamentID: tournamentID, HomePlayerID: homeID, AwayPlayerID: awayID,
				HomePlayerScore: intPtr(5), AwayPlayerScore: intPtr(3), StartDateTime: "2025-06-01 12:00:00",
				Round: "Round 1", Status: "Completed", NumberOfFrames: 9, FixtureNumber: 1,
				PlayersAllocated: true, PlayersAllocatedPresent: true,
				HomePlayer: WstPlayer{PlayerID: homeID, FirstName: "Home"}, AwayPlayer: WstPlayer{PlayerID: awayID, FirstName: "Away"},
			},
		}}}},
	}
}

func mysqlAcceptanceParams(dryRun bool) SyncParams {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	return SyncParams{Mode: SyncModeBackfill, Backfill: true, DryRun: dryRun, Publish: true, IncludeQualifiers: true, GameType: 1, From: &from, To: &to}
}

type mysqlAcceptanceTableCounts struct {
	Players, Tournaments, Matches, EventNews int64
}

func readMySQLAcceptanceCounts(t *testing.T, db *sql.DB) mysqlAcceptanceTableCounts {
	t.Helper()
	var counts mysqlAcceptanceTableCounts
	for _, item := range []struct {
		table string
		value *int64
	}{{"players", &counts.Players}, {"tournaments", &counts.Tournaments}, {"tournament_matches", &counts.Matches}, {"event_news_events", &counts.EventNews}} {
		if err := db.QueryRow("SELECT COUNT(*) FROM `" + item.table + "`").Scan(item.value); err != nil {
			t.Fatalf("count %s: %v", item.table, err)
		}
	}
	return counts
}

type mysqlAcceptanceSnapshot struct {
	PlayerIDs, TournamentIDs, MatchIDs, EventNewsIDs []int64
}

func mysqlAcceptanceSnapshotForPrefix(t *testing.T, db *sql.DB, sourcePrefix string) mysqlAcceptanceSnapshot {
	t.Helper()
	pattern := sourcePrefix + "%"
	return mysqlAcceptanceSnapshot{
		PlayerIDs:     mysqlAcceptanceIDs(t, db, "SELECT id FROM players WHERE source_type = ? AND source_player_id LIKE ? ORDER BY id", wstSourceType, pattern),
		TournamentIDs: mysqlAcceptanceIDs(t, db, "SELECT id FROM tournaments WHERE source_type = ? AND source_tournament_id LIKE ? ORDER BY id", wstSourceType, pattern),
		MatchIDs:      mysqlAcceptanceIDs(t, db, "SELECT id FROM tournament_matches WHERE source_type = ? AND source_match_id LIKE ? ORDER BY id", wstSourceType, pattern),
		EventNewsIDs:  mysqlAcceptanceIDs(t, db, "SELECT e.id FROM event_news_events e JOIN tournaments t ON t.id = e.tournament_id WHERE e.source_type = ? AND t.source_tournament_id LIKE ? ORDER BY e.id", wstSourceType, pattern),
	}
}

func mysqlAcceptanceIDs(t *testing.T, db *sql.DB, query string, args ...any) []int64 {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("query WST business IDs: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan WST business ID: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate WST business IDs: %v", err)
	}
	return ids
}

func mustMySQLExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("execute MySQL acceptance statement: %v", err)
	}
}

func mustMySQLInsertID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("insert MySQL acceptance row: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read MySQL acceptance insert ID: %v", err)
	}
	return id
}

func expectMySQLDuplicate(t *testing.T, err error) {
	t.Helper()
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
		t.Fatalf("expected MySQL duplicate-key error 1062, got %v", err)
	}
}
