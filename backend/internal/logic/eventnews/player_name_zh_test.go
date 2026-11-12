package eventnews

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/gorm"
)

// 只给赵心童配中文名，对手保持英文，覆盖"有中文名用中文名、没有回退英文"
func seedPlayerNameZhEvent(t *testing.T, svcCtx *svc.ServiceContext, title string, sortTime time.Time) *model.EventNews {
	t.Helper()

	tournament, event := createPublishedTournamentEvent(t, svcCtx, title, model.EventNewsStatusLive,
		time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), sortTime)
	zhao := createPlayer(t, svcCtx, &model.Player{SourceType: "official", SourcePlayerId: title + "-zhao", DisplayName: "Zhao Xintong", NameZh: "赵心童"})
	unNooh := createPlayer(t, svcCtx, &model.Player{SourceType: "official", SourcePlayerId: title + "-un-nooh", DisplayName: "Thepchaiya Un-Nooh"})
	createMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   tournament.Id,
		SourceType:     "official",
		RoundName:      "Group Semi Finals",
		Status:         model.EventNewsStatusLive,
		HomePlayerId:   zhao.Id,
		HomePlayerName: "Zhao Xintong",
		AwayPlayerId:   unNooh.Id,
		AwayPlayerName: "Thepchaiya Un-Nooh",
		HomeScore:      3,
		AwayScore:      1,
	})
	return event
}

func TestGetEventNewsViewExposesPlayerChineseNamesAlongsideEnglish(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))()
	event := seedPlayerNameZhEvent(t, svcCtx, "Champion of Champions 2026", time.Date(2026, 11, 12, 11, 0, 0, 0, time.UTC))

	resp, err := NewGetEventNewsViewLogic(context.Background(), svcCtx).GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil || !resp.Success || len(resp.Matches) != 1 {
		t.Fatalf("expected one match, got %#v err=%v", resp, err)
	}
	match := resp.Matches[0]
	if match.HomePlayerName != "Zhao Xintong" || match.HomePlayerNameZh != "赵心童" {
		t.Fatalf("expected English name kept with Chinese name alongside, got %#v", match)
	}
	if match.AwayPlayerName != "Thepchaiya Un-Nooh" || match.AwayPlayerNameZh != "" {
		t.Fatalf("expected player without Chinese name to have empty name_zh, got %#v", match)
	}
	if resp.EventNews.LatestResultText != "赵心童 3 - 1 Thepchaiya Un-Nooh" {
		t.Fatalf("expected summary to prefer Chinese name, got %q", resp.EventNews.LatestResultText)
	}
}

// listPlayerQueries 在独立的内存库（以子测试名区分）里造 events 个赛讯，返回列表接口查询 players 表的次数。
func listPlayerQueries(t *testing.T, events int) (int, *types.GetEventNewsListResp) {
	t.Helper()
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))()
	for i := 0; i < events; i++ {
		seedPlayerNameZhEvent(t, svcCtx, "Event "+string(rune('A'+i)), time.Date(2026, 11, 12, 11, i, 0, 0, time.UTC))
	}

	playerQueries := 0
	if err := svcCtx.DB.Callback().Query().After("gorm:query").Register("count_player_queries", func(db *gorm.DB) {
		if db.Statement.Table == "players" {
			playerQueries++
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	resp, err := NewGetEventNewsListLogic(context.Background(), svcCtx).GetEventNewsList(&types.GetEventNewsListReq{
		Page: 1, PageSize: 20, GameType: 1, Status: -1,
	})
	if err != nil || !resp.Success {
		t.Fatalf("get list: %#v err=%v", resp, err)
	}
	return playerQueries, resp
}

func TestGetEventNewsListSummaryPrefersChineseNamesWithOneBatchedPlayerQuery(t *testing.T) {
	counts := map[int]int{}
	for _, events := range []int{1, 5} {
		t.Run(string(rune('0'+events)), func(t *testing.T) {
			queries, resp := listPlayerQueries(t, events)
			if len(resp.List) != events {
				t.Fatalf("expected %d events, got %d", events, len(resp.List))
			}
			for _, item := range resp.List {
				if item.LatestResultText != "赵心童 3 - 1 Thepchaiya Un-Nooh" {
					t.Fatalf("expected Chinese name in list summary, got %q", item.LatestResultText)
				}
			}
			counts[events] = queries
		})
	}
	if counts[1] != 1 || counts[5] != 1 {
		t.Fatalf("expected one batched player query regardless of page size, got %v", counts)
	}
}
