package eventnews

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

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

	createEvent(t, svcCtx, &model.EventNews{
		Title:      "进行中赛事",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
	})
	createEvent(t, svcCtx, &model.EventNews{
		Title:      "即将开始赛事",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)),
	})

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
