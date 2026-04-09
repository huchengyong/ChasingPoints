package eventnews

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEventNewsTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                   db,
		EventNewsModel:       model.NewEventNewsModel(db),
		TournamentModel:      model.NewTournamentModel(db),
		TournamentMatchModel: model.NewTournamentMatchModel(db),
		PlayerModel:          model.NewPlayerModel(db),
	}
}

func attachEventNewsTestRedis(t *testing.T, svcCtx *svc.ServiceContext) *miniredis.Miniredis {
	t.Helper()

	mr := miniredis.RunT(t)
	svcCtx.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		if svcCtx.Redis != nil {
			_ = svcCtx.Redis.Close()
		}
		mr.Close()
	})
	return mr
}

func mustTimePtr(t time.Time) *time.Time {
	v := t
	return &v
}

func createEvent(t *testing.T, svcCtx *svc.ServiceContext, event *model.EventNews) *model.EventNews {
	t.Helper()

	if err := svcCtx.EventNewsModel.Create(event); err != nil {
		t.Fatalf("create event: %v", err)
	}
	return event
}

func createTournament(t *testing.T, svcCtx *svc.ServiceContext, tournament *model.Tournament) *model.Tournament {
	t.Helper()

	if err := svcCtx.TournamentModel.Create(tournament); err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	return tournament
}

func createMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.TournamentMatch) *model.TournamentMatch {
	t.Helper()

	if err := svcCtx.TournamentMatchModel.Create(match); err != nil {
		t.Fatalf("create match: %v", err)
	}
	return match
}

func createPlayer(t *testing.T, svcCtx *svc.ServiceContext, player *model.Player) *model.Player {
	t.Helper()

	if err := svcCtx.PlayerModel.Create(player); err != nil {
		t.Fatalf("create player: %v", err)
	}
	return player
}

func createPublishedTournamentEvent(t *testing.T, svcCtx *svc.ServiceContext, title string, status int, tournamentStart, tournamentEnd time.Time, sortTime time.Time) (*model.Tournament, *model.EventNews) {
	t.Helper()

	tournament := createTournament(t, svcCtx, &model.Tournament{
		Name:      title,
		GameType:  1,
		Status:    model.EventNewsStatusLive,
		Country:   "英国",
		City:      "曼彻斯特",
		VenueName: "Manchester Central",
		StartDate: mustTimePtr(tournamentStart),
		EndDate:   mustTimePtr(tournamentEnd),
		StartTime: mustTimePtr(tournamentStart),
		EndTime:   mustTimePtr(tournamentEnd),
	})

	event := createEvent(t, svcCtx, &model.EventNews{
		Title:        title,
		TournamentId: tournament.Id,
		GameType:     1,
		SourceType:   "official",
		SourceName:   "WST",
		City:         "曼彻斯特",
		Status:       status,
		Published:    true,
		SortTime:     mustTimePtr(sortTime),
	})

	return tournament, event
}

func withEventNewsNow(now time.Time) func() {
	old := eventNewsNow
	eventNewsNow = func() time.Time {
		return now
	}
	return func() {
		eventNewsNow = old
	}
}

func TestGetEventNewsListFiltersPublishedEventParentsAndExposesMatchSummary(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	tournament := createTournament(t, svcCtx, &model.Tournament{
		Name:       "2026斯诺克世锦赛",
		CoverImage: "https://example.com/tournament-cover.png",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusLive,
		Country:    "英国",
		City:       "谢菲尔德",
		VenueName:  "Crucible",
		StartDate:  mustTimePtr(time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)),
		EndDate:    mustTimePtr(time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)),
		StartTime:  mustTimePtr(time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)),
	})

	snooker := createEvent(t, svcCtx, &model.EventNews{
		Title:        "2026斯诺克世锦赛",
		TournamentId: tournament.Id,
		GameType:     1,
		SourceType:   "official",
		SourceName:   "WST",
		City:         "谢菲尔德",
		Status:       model.EventNewsStatusLive,
		Published:    true,
		SortTime:     mustTimePtr(time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)),
	})
	createMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "32强",
		RoundOrder:     20,
		MatchOrder:     1,
		Status:         model.EventNewsStatusLive,
		HomePlayerName: "赵心童",
		AwayPlayerName: "马克",
		HomeScore:      6,
		AwayScore:      2,
	})

	createEvent(t, svcCtx, &model.EventNews{
		Title:      "未发布斯诺克赛",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusLive,
		Published:  false,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 40, 0, 0, time.UTC)),
	})
	createEvent(t, svcCtx, &model.EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 30, 0, 0, time.UTC)),
	})

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		GameType: 1,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one published snooker event, got %#v", resp)
	}
	if resp.List[0].Title != snooker.Title {
		t.Fatalf("expected snooker event, got %#v", resp.List[0])
	}
	if resp.List[0].CurrentRoundText != "32强" {
		t.Fatalf("expected current round summary from matches, got %#v", resp.List[0])
	}
	if resp.List[0].LatestResultText != "赵心童 6 - 2 马克" {
		t.Fatalf("expected latest result summary from matches, got %#v", resp.List[0])
	}
	if resp.List[0].MatchCount != 1 {
		t.Fatalf("expected one match, got %#v", resp.List[0])
	}
	if resp.List[0].CoverImage != "https://example.com/tournament-cover.png" {
		t.Fatalf("expected tournament cover fallback, got %#v", resp.List[0])
	}
	if resp.List[0].StartDate != "2026-03-23" || resp.List[0].EndDate != "2026-03-30" {
		t.Fatalf("expected date range from tournament, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListFiltersByEventStatusAndStatusPriority(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	_, _ = createPublishedTournamentEvent(t, svcCtx, "进行中赛事", model.EventNewsStatusLive, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC))
	_, _ = createPublishedTournamentEvent(t, svcCtx, "即将开始赛事", model.EventNewsStatusUpcoming, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	fullResp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get full list: %v", err)
	}
	if len(fullResp.List) != 2 || fullResp.List[0].Title != "进行中赛事" {
		t.Fatalf("expected live event to rank before upcoming, got %#v", fullResp.List)
	}

	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   model.EventNewsStatusLive,
	})
	if err != nil {
		t.Fatalf("get list by event status: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one live event, got %#v", resp)
	}
	if resp.List[0].Title != "进行中赛事" || resp.List[0].Status != model.EventNewsStatusLive {
		t.Fatalf("expected only live event rows, got %#v", resp.List)
	}
}

func TestGetEventNewsListAppliesDefaultCurrentYear(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, _ = createPublishedTournamentEvent(t, svcCtx, "2025公开赛", model.EventNewsStatusLive, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 8, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC))
	_, currentYearEvent := createPublishedTournamentEvent(t, svcCtx, "2026公开赛", model.EventNewsStatusLive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one current-year event, got %#v", resp)
	}
	if resp.List[0].Id != currentYearEvent.Id || resp.List[0].Title != currentYearEvent.Title {
		t.Fatalf("expected current-year event only, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListFiltersByYear(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, previousYearEvent := createPublishedTournamentEvent(t, svcCtx, "2025公开赛", model.EventNewsStatusLive, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 8, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 1, 11, 0, 0, 0, time.UTC))
	_, _ = createPublishedTournamentEvent(t, svcCtx, "2026公开赛", model.EventNewsStatusLive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Year:     2025,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one 2025 event, got %#v", resp)
	}
	if resp.List[0].Id != previousYearEvent.Id || resp.List[0].Title != previousYearEvent.Title {
		t.Fatalf("expected 2025 event only, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListPrefersFromToOverYear(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, windowEvent := createPublishedTournamentEvent(t, svcCtx, "2025巡回赛", model.EventNewsStatusLive, time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 0, 0, 0, 0, time.UTC), time.Date(2025, 10, 1, 11, 0, 0, 0, time.UTC))
	_, _ = createPublishedTournamentEvent(t, svcCtx, "2026巡回赛", model.EventNewsStatusLive, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Year:     2026,
		From:     "2025-01-01",
		To:       "2025-12-31",
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one 2025 event, got %#v", resp)
	}
	if resp.List[0].Id != windowEvent.Id || resp.List[0].Title != windowEvent.Title {
		t.Fatalf("expected from/to to take precedence over year, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListIncludesCrossYearOverlap(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, overlapEvent := createPublishedTournamentEvent(t, svcCtx, "跨年赛事", model.EventNewsStatusLive, time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 30, 11, 0, 0, 0, time.UTC))
	_, _ = createPublishedTournamentEvent(t, svcCtx, "2025早期赛事", model.EventNewsStatusLive, time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 10, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Year:     2026,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one overlapping event, got %#v", resp)
	}
	if resp.List[0].Id != overlapEvent.Id || resp.List[0].Title != overlapEvent.Title {
		t.Fatalf("expected cross-year overlap event, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListRejectsInvalidDateParams(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	createPublishedTournamentEvent(t, svcCtx, "2026公开赛", model.EventNewsStatusLive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)

	cases := []struct {
		name string
		req  *types.GetEventNewsListReq
	}{
		{
			name: "only from",
			req: &types.GetEventNewsListReq{
				Page:     1,
				PageSize: 20,
				From:     "2026-01-01",
				Status:   -1,
			},
		},
		{
			name: "only to",
			req: &types.GetEventNewsListReq{
				Page:     1,
				PageSize: 20,
				To:       "2026-12-31",
				Status:   -1,
			},
		},
		{
			name: "to before from",
			req: &types.GetEventNewsListReq{
				Page:     1,
				PageSize: 20,
				From:     "2026-12-31",
				To:       "2026-01-01",
				Status:   -1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := logic.GetEventNewsList(tc.req)
			if err != nil {
				t.Fatalf("get list: %v", err)
			}
			if resp.Success {
				t.Fatalf("expected failure response, got %#v", resp)
			}
			if resp.Total != 0 || len(resp.List) != 0 {
				t.Fatalf("expected empty result, got %#v", resp)
			}
		})
	}
}

func TestGetEventNewsListUsesRedisCacheUntilVersionBumps(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	attachEventNewsTestRedis(t, svcCtx)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, firstEvent := createPublishedTournamentEvent(t, svcCtx, "2026公开赛A", model.EventNewsStatusLive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC))

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	req := &types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	}

	firstResp, err := logic.GetEventNewsList(req)
	if err != nil {
		t.Fatalf("get first list: %v", err)
	}
	if firstResp.Total != 1 || len(firstResp.List) != 1 || firstResp.List[0].Id != firstEvent.Id {
		t.Fatalf("expected first event in initial response, got %#v", firstResp)
	}

	_, secondEvent := createPublishedTournamentEvent(t, svcCtx, "2026公开赛B", model.EventNewsStatusUpcoming, time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC))
	cachedResp, err := logic.GetEventNewsList(req)
	if err != nil {
		t.Fatalf("get cached list: %v", err)
	}
	if cachedResp.Total != 1 || len(cachedResp.List) != 1 {
		t.Fatalf("expected cached response to stay at one event, got %#v", cachedResp)
	}
	if cachedResp.List[0].Id != firstEvent.Id {
		t.Fatalf("expected cached response to preserve first event, got %#v", cachedResp.List[0])
	}

	if err := BumpEventNewsCacheVersion(context.Background(), svcCtx); err != nil {
		t.Fatalf("bump cache version: %v", err)
	}
	refreshedResp, err := logic.GetEventNewsList(req)
	if err != nil {
		t.Fatalf("get refreshed list: %v", err)
	}
	if refreshedResp.Total != 2 || len(refreshedResp.List) != 2 {
		t.Fatalf("expected refreshed response to include both events, got %#v", refreshedResp)
	}
	if refreshedResp.List[0].Id != firstEvent.Id && refreshedResp.List[0].Id != secondEvent.Id {
		t.Fatalf("unexpected refreshed payload: %#v", refreshedResp.List)
	}
}

func TestGetEventNewsListFallsBackWhenRedisUnavailable(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	mr := attachEventNewsTestRedis(t, svcCtx)
	restoreNow := withEventNewsNow(time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC))
	defer restoreNow()

	_, expectedEvent := createPublishedTournamentEvent(t, svcCtx, "2026公开赛", model.EventNewsStatusLive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC))
	mr.Close()

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
	})
	if err != nil {
		t.Fatalf("get list with unavailable redis: %v", err)
	}
	if !resp.Success || resp.Total != 1 || len(resp.List) != 1 || resp.List[0].Id != expectedEvent.Id {
		t.Fatalf("expected db fallback response, got %#v", resp)
	}
}

func TestGetEventNewsViewReturnsEventTournamentAndMatches(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC))()

	event := createEvent(t, svcCtx, &model.EventNews{
		Title:      "Sportsbet.io Tour Championship 2026",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Published:  true,
		Status:     model.EventNewsStatusLive,
		StartDate:  mustTimePtr(time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)),
		EndDate:    mustTimePtr(time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)),
		SortTime:   mustTimePtr(time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)),
	})

	tournament := &model.Tournament{
		CreatorId:   1,
		Name:        "Sportsbet.io Tour Championship 2026",
		CoverImage:  "https://example.com/tour-cover.png",
		GameType:    1,
		Status:      1,
		Country:     "英国",
		City:        "Manchester",
		VenueName:   "Manchester Central",
		StartDate:   mustTimePtr(time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)),
		EndDate:     mustTimePtr(time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)),
		StartTime:   mustTimePtr(time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)),
		EndTime:     mustTimePtr(time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)),
		Description: "Tour Championship detail",
	}
	if err := svcCtx.TournamentModel.Create(tournament); err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	if err := svcCtx.EventNewsModel.UpdateTournamentBinding(event.Id, tournament.Id); err != nil {
		t.Fatalf("bind tournament: %v", err)
	}

	homePlayer := createPlayer(t, svcCtx, &model.Player{
		SourceType:  "official",
		FirstName:   "Neil",
		LastName:    "Robertson",
		DisplayName: "Neil Robertson",
		Avatar:      "https://example.com/neil.png",
		FlagEmoji:   "🇦🇺",
	})
	awayPlayer := createPlayer(t, svcCtx, &model.Player{
		SourceType:  "official",
		FirstName:   "Barry",
		LastName:    "Hawkins",
		DisplayName: "Barry Hawkins",
		Avatar:      "https://example.com/barry.png",
		FlagEmoji:   "🏴",
	})

	if err := svcCtx.TournamentMatchModel.Create(&model.TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "Quarter Finals",
		RoundOrder:     10,
		MatchOrder:     1,
		StartTime:      mustTimePtr(time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)),
		Status:         1,
		BestOf:         19,
		HomePlayerId:   homePlayer.Id,
		HomePlayerName: "Neil Robertson",
		AwayPlayerId:   awayPlayer.Id,
		AwayPlayerName: "Barry Hawkins",
		HomeScore:      5,
		AwayScore:      3,
		WinnerSide:     0,
	}); err != nil {
		t.Fatalf("create tournament match: %v", err)
	}

	logic := NewGetEventNewsViewLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get event news view: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.EventNews == nil || resp.EventNews.Id != event.Id {
		t.Fatalf("expected event news payload, got %#v", resp)
	}
	if resp.Tournament == nil || resp.Tournament.Id != tournament.Id {
		t.Fatalf("expected bound tournament payload, got %#v", resp)
	}
	if len(resp.Matches) != 1 {
		t.Fatalf("expected one tournament match, got %#v", resp.Matches)
	}
	if resp.Matches[0].RoundName != "Quarter Finals" || resp.Matches[0].HomePlayerName != "Neil Robertson" || resp.Matches[0].AwayPlayerName != "Barry Hawkins" {
		t.Fatalf("unexpected match payload: %#v", resp.Matches[0])
	}
	if resp.EventNews.StartDate != "2026-03-30" || resp.EventNews.EndDate != "2026-04-05" {
		t.Fatalf("expected event date range, got %#v", resp.EventNews)
	}
	if resp.Matches[0].StartTime != "2026-04-02T20:00:00+08:00" {
		t.Fatalf("expected shanghai time output, got %#v", resp.Matches[0])
	}
	if resp.Matches[0].HomePlayerAvatar != "https://example.com/neil.png" || resp.Matches[0].AwayPlayerAvatar != "https://example.com/barry.png" {
		t.Fatalf("expected player avatars, got %#v", resp.Matches[0])
	}
	if resp.Matches[0].HomePlayerFirstName != "Neil" || resp.Matches[0].HomePlayerLastName != "Robertson" || resp.Matches[0].HomePlayerFlagEmoji != "🇦🇺" {
		t.Fatalf("expected home player name parts and flag, got %#v", resp.Matches[0])
	}
	if resp.Matches[0].AwayPlayerFirstName != "Barry" || resp.Matches[0].AwayPlayerLastName != "Hawkins" || resp.Matches[0].AwayPlayerFlagEmoji != "🏴" {
		t.Fatalf("expected away player name parts and flag, got %#v", resp.Matches[0])
	}
}

func TestGetEventNewsViewUsesRedisCacheUntilVersionBumps(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	attachEventNewsTestRedis(t, svcCtx)

	event := createEvent(t, svcCtx, &model.EventNews{
		Title:      "Sportsbet.io Tour Championship 2026",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Published:  true,
		Status:     model.EventNewsStatusLive,
	})

	logic := NewGetEventNewsViewLogic(context.Background(), svcCtx)
	firstResp, err := logic.GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get first view: %v", err)
	}
	if !firstResp.Success || firstResp.EventNews == nil || firstResp.EventNews.Title != "Sportsbet.io Tour Championship 2026" {
		t.Fatalf("unexpected first view response: %#v", firstResp)
	}

	event.Title = "已更新的赛事标题"
	if err := svcCtx.EventNewsModel.Update(event); err != nil {
		t.Fatalf("update event title: %v", err)
	}

	cachedResp, err := logic.GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get cached view: %v", err)
	}
	if cachedResp.EventNews == nil || cachedResp.EventNews.Title != "Sportsbet.io Tour Championship 2026" {
		t.Fatalf("expected cached title before version bump, got %#v", cachedResp)
	}

	if err := BumpEventNewsCacheVersion(context.Background(), svcCtx); err != nil {
		t.Fatalf("bump cache version: %v", err)
	}
	refreshedResp, err := logic.GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get refreshed view: %v", err)
	}
	if refreshedResp.EventNews == nil || refreshedResp.EventNews.Title != "已更新的赛事标题" {
		t.Fatalf("expected refreshed title after version bump, got %#v", refreshedResp)
	}
}

func TestGetEventNewsViewFormatsOfficialMatchStartTimeWithoutExtraTimezoneShift(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)

	event := createEvent(t, svcCtx, &model.EventNews{
		Title:      "Halo World Championship 2026 Qualifiers",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Published:  true,
		Status:     model.EventNewsStatusLive,
	})

	tournament := &model.Tournament{
		CreatorId:          0,
		Name:               "Halo World Championship 2026 Qualifiers",
		GameType:           1,
		Status:             1,
		SourceType:         "official",
		SourceTournamentId: "wst-qualifiers",
	}
	if err := svcCtx.TournamentModel.Create(tournament); err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	if err := svcCtx.EventNewsModel.UpdateTournamentBinding(event.Id, tournament.Id); err != nil {
		t.Fatalf("bind tournament: %v", err)
	}

	matchStart := time.Date(2026, 4, 8, 17, 0, 0, 0, shanghaiLocation)
	if err := svcCtx.TournamentMatchModel.Create(&model.TournamentMatch{
		TournamentId:   tournament.Id,
		SourceType:     "official",
		SourceMatchId:  "match-grace-hugill",
		RoundName:      "Round 1",
		RoundOrder:     10,
		MatchOrder:     25,
		StartTime:      &matchStart,
		Status:         1,
		BestOf:         19,
		HomePlayerName: "David Grace",
		AwayPlayerName: "Ashley Hugill",
		HomeScore:      0,
		AwayScore:      1,
	}); err != nil {
		t.Fatalf("create tournament match: %v", err)
	}

	logic := NewGetEventNewsViewLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsView(&types.GetEventNewsViewReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get event news view: %v", err)
	}
	if len(resp.Matches) != 1 {
		t.Fatalf("expected one match, got %#v", resp.Matches)
	}
	if resp.Matches[0].StartTime != "2026-04-08T17:00:00+08:00" {
		t.Fatalf("expected official match shanghai wall clock output, got %#v", resp.Matches[0].StartTime)
	}
}
