package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"billiard_master/internal/config"
	"billiard_master/internal/logic"
	"billiard_master/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/billiard_master-api.yaml", "the config file")
	dryRun     = flag.Bool("dry-run", false, "only print the rebuild summary")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	svcCtx := svc.NewServiceContext(c)
	rebuildService := logic.NewRankRebuildService(svcCtx)

	var (
		summary *logic.RankRebuildSummary
		err     error
	)

	if *dryRun {
		summary, err = rebuildService.DryRun(context.Background())
	} else {
		summary, err = rebuildService.Rebuild(context.Background())
	}
	if err != nil {
		log.Fatalf("段位回放失败: %v", err)
	}

	fmt.Printf(
		"rank rebuild completed: dry_run=%t matches=%d users=%d game_types=%v rank_logs=%d season_records=%d skipped_draws=%d skipped_no_player=%d\n",
		*dryRun,
		summary.MatchesTotal,
		summary.UsersTotal,
		summary.GameTypes,
		summary.RankLogsTotal,
		summary.SeasonRecords,
		summary.SkippedDraws,
		summary.SkippedNoPlayer,
	)
}
