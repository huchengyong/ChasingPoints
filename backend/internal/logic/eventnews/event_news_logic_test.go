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
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusLive,
		City:       "谢菲尔德",
		VenueName:  "Crucible",
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
}

func TestGetEventNewsListFiltersByEventStatus(t *testing.T) {
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
		SortTime:   mustTimePtr(time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)),
	})

	tournament := &model.Tournament{
		CreatorId:   1,
		Name:        "Sportsbet.io Tour Championship 2026",
		GameType:    1,
		Status:      1,
		City:        "Manchester",
		VenueName:   "Manchester Central",
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

	if err := svcCtx.TournamentMatchModel.Create(&model.TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "Quarter Finals",
		RoundOrder:     10,
		MatchOrder:     1,
		StartTime:      mustTimePtr(time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)),
		Status:         1,
		BestOf:         19,
		HomePlayerName: "Neil Robertson",
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
}

func TestGetFeaturedEventNewsPrefersFeaturedEventAndIncludesMatchPreview(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	featuredTournament := createTournament(t, svcCtx, &model.Tournament{
		Name:       "焦点赛事",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusUpcoming,
	})
	featured := createEvent(t, svcCtx, &model.EventNews{
		Title:        "焦点赛事",
		TournamentId: featuredTournament.Id,
		GameType:     1,
		SourceType:   "official",
		SourceName:   "WST",
		Status:       model.EventNewsStatusUpcoming,
		Featured:     true,
		Published:    true,
		SortTime:     mustTimePtr(time.Date(2026, 3, 23, 13, 0, 0, 0, time.UTC)),
	})
	createMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   featuredTournament.Id,
		RoundName:      "资格赛",
		RoundOrder:     10,
		MatchOrder:     1,
		Status:         model.EventNewsStatusUpcoming,
		HomePlayerName: "选手A",
		AwayPlayerName: "选手B",
	})

	otherTournament := createTournament(t, svcCtx, &model.Tournament{
		Name:       "普通进行中赛事",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusLive,
	})
	createEvent(t, svcCtx, &model.EventNews{
		Title:        "普通进行中赛事",
		TournamentId: otherTournament.Id,
		GameType:     1,
		SourceType:   "manual",
		SourceName:   "Admin",
		Status:       model.EventNewsStatusLive,
		Published:    true,
		SortTime:     mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
	})
	createMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   otherTournament.Id,
		RoundName:      "32强",
		RoundOrder:     20,
		MatchOrder:     1,
		Status:         model.EventNewsStatusLive,
		HomePlayerName: "甲",
		AwayPlayerName: "乙",
		HomeScore:      4,
		AwayScore:      3,
	})

	logic := NewGetFeaturedEventNewsLogic(context.Background(), svcCtx)
	resp, err := logic.GetFeaturedEventNews()
	if err != nil {
		t.Fatalf("get featured: %v", err)
	}
	if !resp.Success || resp.Event == nil {
		t.Fatalf("expected featured success, got %#v", resp)
	}
	if resp.Event.Id != featured.Id {
		t.Fatalf("expected featured event to win, got %#v", resp.Event)
	}
	if resp.Event.CurrentRoundText != "资格赛" || resp.Event.LatestResultText != "选手A - 选手B" {
		t.Fatalf("expected match preview on featured event, got %#v", resp.Event)
	}
}
