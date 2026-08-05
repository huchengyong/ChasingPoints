package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"chasing_points/internal/config"
	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	dryRun     = flag.Bool("dry-run", false, "only print the career achievement rebuild summary")
)

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	service := achievementx.NewCareerAchievementRebuildService(svc.NewServiceContext(c))
	var (
		summary *achievementx.CareerAchievementRebuildSummary
		err     error
	)
	if *dryRun {
		summary, err = service.DryRun(context.Background())
	} else {
		summary, err = service.Rebuild(context.Background())
	}
	if err != nil {
		log.Fatalf("生涯成就重建失败: %v", err)
	}

	fmt.Printf(
		"career achievement rebuild completed: dry_run=%t matches=%d tournaments=%d users=%d events=%d events_created=%d achievements_unlocked=%d titles_granted=%d skipped_no_player=%d skipped_ambiguous_special=%d\n",
		*dryRun,
		summary.MatchesTotal,
		summary.TournamentsTotal,
		summary.UsersTotal,
		summary.EventsTotal,
		summary.EventsCreated,
		summary.AchievementsUnlocked,
		summary.TitlesGranted,
		summary.SkippedNoPlayer,
		summary.SkippedAmbiguousSpecial,
	)
}
