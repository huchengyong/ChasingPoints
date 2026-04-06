package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"chasing_points/internal/config"
	"chasing_points/internal/logic/wstsync"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

const (
	defaultConfigFile = "etc/chasing_points-api.yaml"
	wstSeasonsURL     = "https://seasons.snooker.web.gc.wstservices.co.uk"
	wstTournamentsURL = "https://tournaments.snooker.web.gc.wstservices.co.uk"
	wstMatchesURL     = "https://matches.snooker.web.gc.wstservices.co.uk"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	configFile, syncArgs, err := extractConfigArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("解析命令参数失败: %v", err)
	}

	params, err := wstsync.ParseSyncParams(syncArgs)
	if err != nil {
		log.Fatalf("解析同步参数失败: %v", err)
	}

	var c config.Config
	conf.MustLoad(configFile, &c, conf.UseEnv())

	svcCtx := svc.NewServiceContext(c)
	client := wstsync.NewClient(wstSeasonsURL, wstTournamentsURL, wstMatchesURL)
	service := wstsync.NewService(svcCtx, client)

	summary, err := service.Sync(context.Background(), params)
	if err != nil {
		log.Fatalf("WST 历史同步失败: %v", err)
	}

	fmt.Printf(
		"wst sync completed: mode=%s dry_run=%t publish=%t seasons_fetched=%d candidate_seasons=%d tournaments_fetched=%d tournaments_selected=%d matches_scanned=%d matches_selected=%d players_prepared=%d tournaments_prepared=%d matches_prepared=%d event_news_projected=%d\n",
		summary.Mode,
		summary.DryRun,
		summary.Publish,
		summary.SeasonsFetched,
		summary.CandidateSeasons,
		summary.TournamentsFetched,
		summary.TournamentsSelected,
		summary.MatchesScanned,
		summary.MatchesSelected,
		summary.PlayersPrepared,
		summary.TournamentsPrepared,
		summary.MatchesPrepared,
		summary.EventNewsProjected,
	)
}

func extractConfigArgs(args []string) (string, []string, error) {
	configFile := defaultConfigFile
	remaining := make([]string, 0, len(args))

	for index := 0; index < len(args); index++ {
		current := args[index]
		switch {
		case current == "-f":
			if index+1 >= len(args) {
				return "", nil, fmt.Errorf("-f requires a value")
			}
			configFile = strings.TrimSpace(args[index+1])
			index++
		case strings.HasPrefix(current, "-f="):
			configFile = strings.TrimSpace(strings.TrimPrefix(current, "-f="))
		default:
			remaining = append(remaining, current)
		}
	}

	if configFile == "" {
		configFile = defaultConfigFile
	}
	return configFile, remaining, nil
}
