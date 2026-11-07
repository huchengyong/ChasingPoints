package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/logic/wstsync"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

const (
	defaultConfigFile = "etc/chasing_points-api.yaml"
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
	client := wstsync.NewClient(wstsync.DefaultSeasonsURL, wstsync.DefaultTournamentsURL, wstsync.DefaultMatchesURL)
	service := wstsync.NewService(svcCtx, client)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	summary, err := service.Sync(ctx, params)
	if err != nil {
		log.Printf("WST 同步失败: %v", err)
		if summary != nil && params.Backfill {
			printBackfillSummary(summary)
			os.Exit(summary.ExitCode)
		}
		log.Fatalf("WST 同步失败: %v", err)
	}

	if params.Backfill {
		printBackfillSummary(summary)
		os.Exit(summary.ExitCode)
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

func printBackfillSummary(s *wstsync.SyncSummary) {
	from := ""
	to := ""
	if s.From != nil {
		from = s.From.Format("2006-01-02")
	}
	if s.To != nil {
		to = s.To.Format("2006-01-02")
	}

	fmt.Printf("\n========== WST Backfill Summary ==========\n")
	fmt.Printf("status:       %s\n", s.Status)
	fmt.Printf("mode:         %s\n", s.Mode)
	fmt.Printf("dry_run:      %t\n", s.DryRun)
	fmt.Printf("publish:      %t\n", s.Publish)
	fmt.Printf("range:        %s ~ %s\n", from, to)
	if s.CoverageStart != nil && s.CoverageEnd != nil {
		fmt.Printf("coverage:     %s ~ %s\n", s.CoverageStart.Format("2006-01-02"), s.CoverageEnd.Format("2006-01-02"))
	}
	fmt.Printf("elapsed:      %s\n", s.Elapsed.Round(time.Second))
	fmt.Printf("commit_state: %s\n", s.CommitState)
	fmt.Printf("---\n")
	fmt.Printf("seasons_fetched:      %d\n", s.SeasonsFetched)
	fmt.Printf("candidate_seasons:    %d\n", s.CandidateSeasons)
	fmt.Printf("tournaments_fetched:  %d\n", s.TournamentsFetched)
	fmt.Printf("tournaments_selected: %d\n", s.TournamentsSelected)
	fmt.Printf("---\n")
	fmt.Printf("match_pages:          %d\n", s.MatchPages)
	fmt.Printf("matches_scanned:      %d\n", s.MatchesScanned)
	fmt.Printf("matches_selected:     %d\n", s.MatchesSelected)
	fmt.Printf("matches_skipped:      %d\n", s.MatchesSkipped)
	fmt.Printf("matches_retained:     %d\n", s.MatchesRetained)
	fmt.Printf("---\n")
	fmt.Printf("players_prepared:     %d\n", s.PlayersPrepared)
	fmt.Printf("tournaments_prepared: %d\n", s.TournamentsPrepared)
	fmt.Printf("matches_prepared:     %d\n", s.MatchesPrepared)
	fmt.Printf("event_news_projected: %d\n", s.EventNewsProjected)
	fmt.Printf("publish_affected:     %d\n", s.PublishAffected)
	fmt.Printf("players_committed:    %s\n", committedCount(s.PlayersCommitted, s.CommittedCountsKnown))
	fmt.Printf("tournaments_committed: %s\n", committedCount(s.TournamentsCommitted, s.CommittedCountsKnown))
	fmt.Printf("matches_committed:    %s\n", committedCount(s.MatchesCommitted, s.CommittedCountsKnown))
	fmt.Printf("event_news_committed: %s\n", committedCount(s.EventNewsCommitted, s.CommittedCountsKnown))
	for _, item := range s.Years {
		fmt.Printf("year %d: tournaments=%d matches=%d\n", item.Year, item.Tournaments, item.Matches)
	}
	if len(s.Warnings) > 0 {
		fmt.Printf("---\n")
		fmt.Printf("warnings: %d\n", len(s.Warnings))
		for _, w := range s.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	fmt.Printf("==========================================\n\n")
}

func committedCount(value int, known bool) string {
	if !known {
		return "unknown"
	}
	return fmt.Sprintf("%d", value)
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
