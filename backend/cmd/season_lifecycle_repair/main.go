package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/logic"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile     = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	dryRun         = flag.Bool("dry-run", false, "only inspect the continuous season lifecycle repair plan")
	batchSize      = flag.Int("batch-size", 100, "maximum deterministic season windows per batch")
	afterStartDate = flag.String("after-start-date", "", "exclusive YYYY-MM-DD window cursor")
	checkpointFile = flag.String("checkpoint", "", "checkpoint file updated after each repaired window")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	service := logic.NewSeasonLifecycleRepairService(svc.NewServiceContext(c))
	cursor := strings.TrimSpace(*afterStartDate)
	if cursor == "" && strings.TrimSpace(*checkpointFile) != "" {
		stored, err := readRepairCheckpoint(*checkpointFile)
		if err != nil {
			log.Fatalf("读取连续赛季修复 checkpoint 失败: %v", err)
		}
		cursor = stored
	}
	for {
		options := logic.SeasonLifecycleRepairOptions{DryRun: *dryRun, BatchSize: *batchSize, AfterStartDate: cursor}
		if !*dryRun && strings.TrimSpace(*checkpointFile) != "" {
			options.Checkpoint = func(next string) error { return writeRepairCheckpoint(*checkpointFile, next) }
		}
		summary, err := service.RunBatchAt(context.Background(), time.Now(), options)
		if summary != nil {
			fmt.Print(formatRepairSummary(*dryRun, summary))
		}
		if err != nil {
			log.Fatalf("连续赛季修复失败: %v", err)
		}
		if summary == nil {
			log.Fatal("连续赛季修复未返回汇总")
		}
		if len(summary.Conflicts) > 0 {
			log.Fatalf("连续赛季修复已停止，请先处理排期冲突")
		}
		if !summary.HasMore {
			break
		}
		if summary.NextCursor == "" || summary.NextCursor == cursor {
			log.Fatal("连续赛季修复 cursor 未推进")
		}
		cursor = summary.NextCursor
	}
}

func readRepairCheckpoint(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

func writeRepairCheckpoint(path, cursor string) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(strings.TrimSpace(cursor)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func formatRepairSummary(dryRun bool, summary *logic.SeasonLifecycleRepairSummary) string {
	if summary == nil {
		return ""
	}
	var output strings.Builder
	fmt.Fprintf(
		&output,
		"season lifecycle repair: dry_run=%t state=%s problem=%q anchor_date=%s initial_number=%d cycle_months=%d timezone=%s cursor=%s next_cursor=%s has_more=%t\n",
		dryRun,
		summary.State,
		summary.Problem,
		summary.Policy.AnchorDate,
		summary.Policy.InitialNumber,
		summary.Policy.CycleMonths,
		summary.Policy.Timezone,
		summary.Cursor,
		summary.NextCursor,
		summary.HasMore,
	)
	for _, window := range summary.Windows {
		fmt.Fprintf(
			&output,
			"window: name=%s start_date=%s end_date=%s season_id=%d status=%d exists=%t created=%t settlement=%s\n",
			window.Name,
			window.StartDate,
			window.EndDate,
			window.SeasonId,
			window.Status,
			window.Exists,
			window.Created,
			window.Settlement,
		)
	}
	for _, conflict := range summary.Conflicts {
		fmt.Fprintf(&output, "conflict: %s\n", conflict)
	}
	for _, failure := range summary.Failures {
		fmt.Fprintf(&output, "failure: %s\n", failure)
	}
	fmt.Fprintf(
		&output,
		"season lifecycle repair completed: dry_run=%t state=%s windows_planned=%d windows_created=%d seasons_to_settle=%d seasons_settled=%d events_covered=%d users_covered=%d estimated_challenge_snapshots=%d estimated_season_records=%d estimated_season_titles=%d challenge_snapshots_written=%d season_records_written=%d season_titles_granted=%d cursor=%s next_cursor=%s has_more=%t conflicts=%d failures=%d\n",
		dryRun,
		summary.State,
		summary.WindowsPlanned,
		summary.WindowsCreated,
		summary.SeasonsToSettle,
		summary.SeasonsSettled,
		summary.EventsCovered,
		summary.UsersCovered,
		summary.EstimatedChallengeSnapshots,
		summary.EstimatedSeasonRecords,
		summary.EstimatedSeasonTitles,
		summary.ChallengeSnapshotsWritten,
		summary.SeasonRecordsWritten,
		summary.SeasonTitlesGranted,
		summary.Cursor,
		summary.NextCursor,
		summary.HasMore,
		len(summary.Conflicts),
		len(summary.Failures),
	)
	return output.String()
}
