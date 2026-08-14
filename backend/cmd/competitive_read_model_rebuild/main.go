package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/logic"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	dryRun     = flag.Bool("dry-run", false, "only inspect bounded rebuild batches")
	batchSize  = flag.Int("batch-size", 100, "matches per batch, maximum 1000")
	maxBatches = flag.Int("max-batches", 0, "maximum batches in this run; 0 means until caught up")
	rateLimit  = flag.Duration("rate-limit", 0, "delay between batches, for example 200ms")
	pause      = flag.Bool("pause", false, "persist a paused checkpoint and exit")
	resume     = flag.Bool("resume", false, "clear a paused checkpoint before rebuilding")
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	rebuild := logic.NewCompetitiveReadModelRebuildService(svc.NewServiceContext(c))
	if *pause {
		if err := rebuild.Pause(); err != nil {
			log.Fatalf("暂停竞技读模型回建失败: %v", err)
		}
		fmt.Println("competitive read model rebuild paused")
		return
	}
	if *resume {
		if err := rebuild.Resume(); err != nil {
			log.Fatalf("恢复竞技读模型回建失败: %v", err)
		}
	}

	summary, err := rebuild.Rebuild(context.Background(), logic.CompetitiveReadModelRebuildOptions{
		BatchSize:  *batchSize,
		MaxBatches: *maxBatches,
		RateLimit:  normalizeRateLimit(*rateLimit),
		DryRun:     *dryRun,
	})
	if err != nil {
		log.Fatalf("竞技读模型回建失败: %v", err)
	}
	fmt.Printf("competitive read model rebuild: dry_run=%t paused=%t batches=%d matches=%d participant_rows=%d competitive_users=%d cursor=%d\n",
		summary.DryRun,
		summary.Paused,
		summary.Batches,
		summary.MatchesScanned,
		summary.ParticipantRows,
		summary.CompetitiveUsers,
		summary.LastCursorMatchID,
	)
}

func normalizeRateLimit(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}
