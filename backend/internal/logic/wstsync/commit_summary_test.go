package wstsync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"chasing_points/internal/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackfillCommittedCounts(t *testing.T) {
	t.Run("dry run reports known zero", func(t *testing.T) {
		client := mysqlAcceptanceClient("dry-run-tournament", "dry-run-match", "dry-run-home", "dry-run-away", "Dry Run Tournament")
		client.matchesPages[0].Data[0].Attributes.HomePlayerScore = nil
		service := newWSTSyncServiceForTest(t, newWSTSyncServiceTestDB(t), client)

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(true))
		if err != nil || summary.Status != BackfillStatusNeedsReview || summary.ExitCode != ExitCodeNeedsReview {
			t.Fatalf("expected dry-run review result: summary=%+v err=%v", summary, err)
		}
		assertCommittedCounts(t, summary, true, 0, 0, 0, 0)
	})

	t.Run("pre-transaction failure reports known zero", func(t *testing.T) {
		db := newWSTSyncServiceTestDB(t)
		player := &model.Player{SourceType: wstSourceType, SourcePlayerId: "precheck-home", DisplayName: "Deleted"}
		if err := db.Create(player).Error; err != nil {
			t.Fatalf("seed player: %v", err)
		}
		if err := db.Delete(player).Error; err != nil {
			t.Fatalf("soft-delete player: %v", err)
		}
		service := newWSTSyncServiceForTest(t, db, mysqlAcceptanceClient("precheck-tournament", "precheck-match", "precheck-home", "precheck-away", "Precheck Tournament"))

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err != nil || summary.Status != BackfillStatusFailed {
			t.Fatalf("expected pre-check failure: summary=%+v err=%v", summary, err)
		}
		assertCommittedCounts(t, summary, true, 0, 0, 0, 0)
	})

	t.Run("confirmed rollback reports known zero", func(t *testing.T) {
		db := newWSTSyncServiceTestDB(t)
		if err := db.Callback().Create().Before("gorm:create").Register("wst:fail_event_news_for_commit_summary", func(tx *gorm.DB) {
			if tx.Statement.Table == "event_news_events" {
				tx.AddError(errEventNewsWrite)
			}
		}); err != nil {
			t.Fatalf("register event-news failure: %v", err)
		}
		service := newWSTSyncServiceForTest(t, db, mysqlAcceptanceClient("rollback-summary-tournament", "rollback-summary-match", "rollback-summary-home", "rollback-summary-away", "Rollback Tournament"))

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if !errors.Is(err, errEventNewsWrite) || summary.CommitState != "rolled_back" {
			t.Fatalf("expected confirmed rollback: summary=%+v err=%v", summary, err)
		}
		assertCommittedCounts(t, summary, true, 0, 0, 0, 0)
	})

	t.Run("successful commit reports actual counts", func(t *testing.T) {
		service := newWSTSyncServiceForTest(t, newWSTSyncServiceTestDB(t), mysqlAcceptanceClient("success-tournament", "success-match", "success-home", "success-away", "Success Tournament"))

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err != nil || summary.CommitState != "committed" {
			t.Fatalf("expected successful commit: summary=%+v err=%v", summary, err)
		}
		assertCommittedCounts(t, summary, true, 2, 1, 1, 1)
	})

	t.Run("cache failure preserves actual counts", func(t *testing.T) {
		service := newWSTSyncServiceForTest(t, newWSTSyncServiceTestDB(t), mysqlAcceptanceClient("cache-tournament", "cache-match", "cache-home", "cache-away", "Cache Tournament"))
		redisClient := redis.NewClient(&redis.Options{Addr: "unused"})
		if err := redisClient.Close(); err != nil {
			t.Fatalf("close test redis client: %v", err)
		}
		service.svcCtx.Redis = redisClient

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if err != nil || summary.Status != BackfillStatusNeedsReview || summary.CommitState != "committed" {
			t.Fatalf("expected committed data with cache warning: summary=%+v err=%v", summary, err)
		}
		assertCommittedCounts(t, summary, true, 2, 1, 1, 1)
	})

	t.Run("unknown commit reports unknown counts", func(t *testing.T) {
		baseDB := newWSTSyncServiceTestDB(t)
		rawDB, err := baseDB.DB()
		if err != nil {
			t.Fatalf("get sqlite connection: %v", err)
		}
		unknownDB, err := gorm.Open(sqlite.New(sqlite.Config{Conn: &unknownCommitPool{DB: rawDB}}), &gorm.Config{})
		if err != nil {
			t.Fatalf("open commit-error database: %v", err)
		}
		service := newWSTSyncServiceForTest(t, unknownDB, mysqlAcceptanceClient("unknown-tournament", "unknown-match", "unknown-home", "unknown-away", "Unknown Tournament"))

		summary, err := service.Sync(context.Background(), mysqlAcceptanceParams(false))
		if !errors.Is(err, errUnknownCommit) || summary.CommitState != "unknown" {
			t.Fatalf("expected unknown commit result: summary=%+v err=%v", summary, err)
		}
		if summary.CommittedCountsKnown {
			t.Fatalf("expected committed counts to be unknown: %+v", summary)
		}
	})
}

func assertCommittedCounts(t *testing.T, summary *SyncSummary, known bool, players, tournaments, matches, eventNews int) {
	t.Helper()
	if summary == nil {
		t.Fatal("expected sync summary")
	}
	if summary.CommittedCountsKnown != known || summary.PlayersCommitted != players || summary.TournamentsCommitted != tournaments || summary.MatchesCommitted != matches || summary.EventNewsCommitted != eventNews {
		t.Fatalf("unexpected committed counts: %+v", summary)
	}
}

var errUnknownCommit = errors.New("commit result unknown")
var errEventNewsWrite = errors.New("event news write failed")

type unknownCommitPool struct {
	*sql.DB
}

func (p *unknownCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &unknownCommitTx{Tx: tx}, nil
}

type unknownCommitTx struct {
	*sql.Tx
}

func (tx *unknownCommitTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	return errUnknownCommit
}
