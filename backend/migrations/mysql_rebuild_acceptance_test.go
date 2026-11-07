package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLCompetitiveReadRebuildCheckpointAndAudit(t *testing.T) {
	if os.Getenv("RUN_MYSQL_REBUILD_TESTS") != "1" {
		t.Skip("set RUN_MYSQL_REBUILD_TESTS=1 to run representative MySQL rebuild acceptance")
	}
	admin, err := sql.Open("mysql", mysqlTestDSN(t, ""))
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	defer admin.Close()
	databaseName := fmt.Sprintf("chasing_points_rebuild_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + databaseName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create rebuild database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`") }()
	dsn := mysqlTestDSN(t, databaseName)
	command := exec.Command("goose", "-dir", ".", "mysql", dsn, "up")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("apply rebuild migrations: %v: %s", err, output)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open rebuild Gorm database: %v", err)
	}
	seasonRecords := model.NewSeasonRecordModel(db)
	if err := seasonRecords.ApplyCompetitiveMatchWithTx(nil, model.SeasonRecordMatchDelta{
		SeasonId:       1,
		UserId:         1,
		GameType:       3,
		StartRankScore: 100,
		EndRankScore:   110,
		Won:            true,
	}); err != nil {
		t.Fatalf("MySQL JSON rewards must accept a new incremental season record: %v", err)
	}
	var rewardsValid sql.NullBool
	if err := db.Raw("SELECT JSON_VALID(rewards) FROM season_records WHERE season_id = ? AND user_id = ? AND game_type = ?", 1, 1, 3).Scan(&rewardsValid).Error; err != nil || rewardsValid.Valid {
		t.Fatalf("incremental season rewards must remain SQL NULL: valid=%+v err=%v", rewardsValid, err)
	}
	if err := db.Where("season_id = ? AND user_id = ? AND game_type = ?", 1, 1, 3).Delete(&model.SeasonRecord{}).Error; err != nil {
		t.Fatalf("remove JSON compatibility probe: %v", err)
	}

	seedCompetitiveRebuildData(t, db, 500)
	svcCtx := competitiveRebuildServiceContext(db)
	rebuild := logicx.NewCompetitiveReadModelRebuildService(svcCtx)

	dryRun, err := rebuild.DryRun(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 73})
	if err != nil || dryRun.MatchesScanned != 500 || dryRun.BackfillBatches != 7 || dryRun.Batches != 0 || dryRun.LastCursorMatchID != 500 {
		t.Fatalf("unexpected rebuild dry run: summary=%+v err=%v", dryRun, err)
	}
	if err := rebuild.Pause(); err != nil {
		t.Fatalf("pause rebuild: %v", err)
	}
	paused, err := rebuild.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 73})
	if err != nil || !paused.Paused || paused.MatchesScanned != 0 {
		t.Fatalf("paused rebuild must not scan: summary=%+v err=%v", paused, err)
	}
	if err := rebuild.Resume(); err != nil {
		t.Fatalf("resume rebuild: %v", err)
	}
	partial, err := rebuild.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 73, MaxBatches: 2})
	if err != nil || partial.MatchesScanned != 146 || partial.LastCursorMatchID != 146 {
		t.Fatalf("unexpected partial rebuild: summary=%+v err=%v", partial, err)
	}
	restartedCtx := competitiveRebuildServiceContext(db)
	restarted := logicx.NewCompetitiveReadModelRebuildService(restartedCtx)
	remaining, err := restarted.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 73})
	if err != nil || remaining.MatchesScanned != 854 || remaining.LastCursorMatchID != 500 {
		t.Fatalf("checkpoint resume failed: summary=%+v err=%v", remaining, err)
	}
	retry, err := restarted.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 73})
	if err != nil || retry.MatchesScanned != 0 || retry.LastCursorMatchID != 500 {
		t.Fatalf("completed rebuild retry must be a no-op: summary=%+v err=%v", retry, err)
	}
	var missingCompletedAt int64
	if err := db.Model(&model.Match{}).Where("status = ? AND completed_at IS NULL", 2).Count(&missingCompletedAt).Error; err != nil || missingCompletedAt != 0 {
		t.Fatalf("rebuild must backfill completed_at: missing=%d err=%v", missingCompletedAt, err)
	}
	var participantRows int64
	if err := db.Model(&model.MatchParticipantResult{}).Count(&participantRows).Error; err != nil || participantRows != 1000 {
		t.Fatalf("unexpected participant projection count: count=%d err=%v", participantRows, err)
	}
	audit, err := logicx.NewCompetitiveReadAuditService(competitiveRebuildServiceContext(db)).Audit(context.Background(), 500)
	if err != nil || audit.MatchesSampled != 500 || len(audit.Differences) != 0 {
		t.Fatalf("rebuild audit must converge without differences: summary=%+v err=%v", audit, err)
	}
	if output, downErr := exec.Command("goose", "-dir", ".", "mysql", dsn, "down-to", "20260811101000").CombinedOutput(); downErr != nil {
		t.Fatalf("rollback hot-read migrations: %v: %s", downErr, output)
	}
	if !mysqlIndexExists(t, db, "venues", "idx_status_geo_status") || mysqlIndexExists(t, db, "venues", "idx_venues_nearby_candidates") {
		t.Fatal("hot-read migration down must restore the prior venue index and remove its replacement")
	}
	if output, upErr := exec.Command("goose", "-dir", ".", "mysql", dsn, "up").CombinedOutput(); upErr != nil {
		t.Fatalf("reapply hot-read migrations: %v: %s", upErr, output)
	}
	if mysqlIndexExists(t, db, "venues", "idx_status_geo_status") || !mysqlIndexExists(t, db, "venues", "idx_venues_nearby_candidates") {
		t.Fatal("hot-read migration up must restore the optimized venue index transition")
	}

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	missingUserID := int64(999)
	otherUserID := int64(1)
	win := 1
	failedAt := base.Add(1000 * time.Minute)
	if err := db.Create(&model.Match{Id: 501, UserId: missingUserID, OpponentId: &otherUserID, OpponentName: "user-1", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: failedAt.Add(-time.Minute), EndTime: &failedAt, CompletedAt: &failedAt}).Error; err != nil {
		t.Fatalf("seed failing rebuild match: %v", err)
	}
	if _, rebuildErr := restarted.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 10}); rebuildErr == nil {
		t.Fatal("rebuild must stop before advancing a failed match checkpoint")
	}
	checkpoint, err := restartedCtx.CompetitiveReadModel.FindCheckpoint("competitive-read-model-v1")
	if err != nil || checkpoint == nil || checkpoint.CursorMatchId != 500 {
		t.Fatalf("failed MySQL projection advanced checkpoint: checkpoint=%+v err=%v", checkpoint, err)
	}
	if err := db.Create(&model.User{Id: missingUserID, Nickname: "repaired"}).Error; err != nil {
		t.Fatalf("repair missing user: %v", err)
	}
	if _, rebuildErr := restarted.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 10}); rebuildErr != nil {
		t.Fatalf("retry failed MySQL projection: %v", rebuildErr)
	}
	if projection, findErr := restartedCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(501, missingUserID); findErr != nil || projection == nil {
		t.Fatalf("retry must project previously failed MySQL match: projection=%+v err=%v", projection, findErr)
	}

	firstUserID, secondUserID := int64(777), int64(778)
	if err := db.Create(&[]model.User{{Id: firstUserID, Nickname: "chronology-a"}, {Id: secondUserID, Nickname: "chronology-b"}}).Error; err != nil {
		t.Fatalf("seed chronology users: %v", err)
	}
	loss := 2
	earlyAt := base.Add(1500 * time.Minute)
	lateAt := base.Add(2000 * time.Minute)
	if err := db.Create(&[]model.Match{
		{Id: 502, UserId: firstUserID, OpponentId: &secondUserID, OpponentName: "chronology-b", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: lateAt.Add(-time.Minute), EndTime: &lateAt, CompletedAt: &lateAt},
		{Id: 503, UserId: firstUserID, OpponentId: &secondUserID, OpponentName: "chronology-b", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &loss, MatchTime: earlyAt.Add(-time.Minute), EndTime: &earlyAt, CompletedAt: &earlyAt},
	}).Error; err != nil {
		t.Fatalf("seed reverse completed_at matches: %v", err)
	}
	if _, rebuildErr := restarted.Rebuild(context.Background(), logicx.CompetitiveReadModelRebuildOptions{BatchSize: 10}); rebuildErr != nil {
		t.Fatalf("rebuild reverse completed_at matches: %v", rebuildErr)
	}
	stats, err := restartedCtx.CompetitiveReadModel.FindStats(firstUserID, 0)
	if err != nil || stats == nil || stats.CurrentWinStreak != 1 || stats.MaxWinStreak != 1 || stats.LastMatchId != 502 {
		t.Fatalf("MySQL rebuild must replay completed_at chronology: stats=%+v err=%v", stats, err)
	}
	catchUpAudit, err := logicx.NewCompetitiveReadAuditService(restartedCtx).Audit(context.Background(), 1000)
	if err != nil || len(catchUpAudit.Differences) != 0 {
		t.Fatalf("catch-up audit after failure/reverse replay: summary=%+v err=%v", catchUpAudit, err)
	}
}

func mysqlIndexExists(t *testing.T, db *gorm.DB, table, index string) bool {
	t.Helper()
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?", table, index).Scan(&count).Error; err != nil {
		t.Fatalf("inspect index %s.%s: %v", table, index, err)
	}
	return count > 0
}

func competitiveRebuildServiceContext(db *gorm.DB) *svc.ServiceContext {
	return &svc.ServiceContext{
		DB: db, UserModel: model.NewUserModel(db), MatchModel: model.NewMatchModel(db),
		RankingModel: model.NewRankingModel(db), CompetitiveReadModel: model.NewCompetitiveReadModel(db),
	}
}

func seedCompetitiveRebuildData(t *testing.T, db *gorm.DB, matchCount int) {
	t.Helper()
	users := make([]model.User, 0, 100)
	for userID := 1; userID <= 100; userID++ {
		users = append(users, model.User{Id: int64(userID), Nickname: fmt.Sprintf("user-%d", userID)})
	}
	if err := db.CreateInBatches(&users, 100).Error; err != nil {
		t.Fatalf("seed rebuild users: %v", err)
	}
	matches := make([]model.Match, 0, matchCount)
	logs := make([]model.RankChangeLog, 0, matchCount*2)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for index := 1; index <= matchCount; index++ {
		player1 := int64((index-1)%100 + 1)
		player2 := int64(index%100 + 1)
		result := 1
		if index%2 == 0 {
			result = 2
		}
		endedAt := base.Add(time.Duration(index) * time.Minute)
		matches = append(matches, model.Match{
			Id: int64(index), UserId: player1, OpponentId: &player2, OpponentName: fmt.Sprintf("user-%d", player2),
			GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
			Status: 2, Result: &result, MyScore: 3, OpponentScore: 1, MatchTime: endedAt.Add(-time.Minute), EndTime: &endedAt,
		})
		player1After, player2After := 1010, 990
		if result == 2 {
			player1After, player2After = 990, 1010
		}
		logs = append(logs,
			model.RankChangeLog{UserId: player1, MatchId: int64(index), ChangeType: "match_result", GameType: 3, BeforeScore: 1000, AfterScore: player1After, BeforeLevel: 1, AfterLevel: 1, EffectiveAt: endedAt},
			model.RankChangeLog{UserId: player2, MatchId: int64(index), ChangeType: "match_result", GameType: 3, BeforeScore: 1000, AfterScore: player2After, BeforeLevel: 1, AfterLevel: 1, EffectiveAt: endedAt},
		)
	}
	if err := db.CreateInBatches(&matches, 100).Error; err != nil {
		t.Fatalf("seed rebuild matches: %v", err)
	}
	if err := db.CreateInBatches(&logs, 200).Error; err != nil {
		t.Fatalf("seed rebuild rank logs: %v", err)
	}
}
