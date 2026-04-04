package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"chasing_points/internal/config"
	"chasing_points/internal/logic"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile   = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	dryRun       = flag.Bool("dry-run", false, "only print the backfill summary")
	tournamentID = flag.Int64("tournament-id", 0, "only backfill the specified tournament")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	svcCtx := svc.NewServiceContext(c)
	service := logic.NewTournamentMatchPlayerBackfillService(svcCtx)

	var (
		summary *logic.TournamentMatchPlayerBackfillSummary
		err     error
	)

	if *dryRun {
		summary, err = service.DryRun(context.Background(), *tournamentID)
	} else {
		summary, err = service.Rebuild(context.Background(), *tournamentID)
	}
	if err != nil {
		log.Fatalf("赛事球员回填失败: %v", err)
	}

	fmt.Printf(
		"tournament match player backfill completed: dry_run=%t tournament_id=%d scanned=%d home=%d away=%d player1=%d player2=%d winner=%d placeholders=%d\n",
		*dryRun,
		*tournamentID,
		summary.MatchesScanned,
		summary.HomeLinked,
		summary.AwayLinked,
		summary.Player1Linked,
		summary.Player2Linked,
		summary.WinnerLinked,
		summary.PlaceholderSkipped,
	)
}
