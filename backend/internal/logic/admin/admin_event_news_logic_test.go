package admin

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

func newEventNewsAdminTestSvc(t *testing.T) *svc.ServiceContext {
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

func adminTestCtx(adminID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", adminID)
}

func adminTestTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.Local)
}

func adminTestTimeString(t time.Time) string {
	return t.Format(adminEventNewsTimeLayout)
}

func createSeedEvent(t *testing.T, svcCtx *svc.ServiceContext, item *model.EventNews) *model.EventNews {
	t.Helper()
	if err := svcCtx.EventNewsModel.Create(item); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	return item
}

func createSeedTournament(t *testing.T, svcCtx *svc.ServiceContext, item *model.Tournament) *model.Tournament {
	t.Helper()
	if err := svcCtx.TournamentModel.Create(item); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	return item
}

func createSeedMatch(t *testing.T, svcCtx *svc.ServiceContext, item *model.TournamentMatch) *model.TournamentMatch {
	t.Helper()
	if err := svcCtx.TournamentMatchModel.Create(item); err != nil {
		t.Fatalf("seed match: %v", err)
	}
	return item
}

func TestAdminCreateUpdatePublishAndDeleteEventNews(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	logic := NewAdminCreateEventNewsLogic(adminTestCtx(-100), svcCtx)
	start := adminTestTime(2026, 3, 23, 10, 0, 0)

	resp, err := logic.AdminCreateEventNews(&types.AdminEventNewsCreateReq{
		Title:      "斯诺克世界锦标赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		SourceUrl:  "https://example.com/snooker",
		CoverImage: "https://example.com/cover.jpg",
		Summary:    "今日焦点赛事",
		Content:    "详细内容",
		Country:    "英国",
		City:       "谢菲尔德",
		Venue:      "Crucible",
		StartTime:  adminTestTimeString(start),
		Status:     model.EventNewsStatusUpcoming,
		Featured:   false,
	})
	if err != nil {
		t.Fatalf("create event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected create success, got %#v", resp)
	}

	list, total, err := svcCtx.EventNewsModel.FindList(1, 10, 1, model.EventNewsStatusUpcoming, "", -1)
	if err != nil {
		t.Fatalf("find created event: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected one created event, got total=%d list=%#v", total, list)
	}
	if list[0].TournamentId <= 0 {
		t.Fatalf("expected event to bind tournament, got %#v", list[0])
	}
	if list[0].Published {
		t.Fatalf("expected draft event, got published item: %#v", list[0])
	}

	updateLogic := NewAdminUpdateEventNewsLogic(adminTestCtx(-100), svcCtx)
	updatedStart := adminTestTime(2026, 3, 24, 12, 0, 0)
	updatedSort := adminTestTime(2026, 3, 24, 12, 5, 0)
	resp, err = updateLogic.AdminUpdateEventNews(&types.AdminEventNewsUpdateReq{
		EventId:    list[0].Id,
		Title:      "斯诺克世锦赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		SourceUrl:  "https://example.com/world",
		CoverImage: "https://example.com/world.jpg",
		Summary:    "更新后的摘要",
		Content:    "更新后的正文",
		Country:    "英国",
		City:       "谢菲尔德",
		Venue:      "Crucible Theatre",
		StartTime:  adminTestTimeString(updatedStart),
		SortTime:   adminTestTimeString(updatedSort),
		Status:     model.EventNewsStatusLive,
		Featured:   true,
	})
	if err != nil {
		t.Fatalf("update event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected update success, got %#v", resp)
	}

	refreshed, err := svcCtx.EventNewsModel.FindById(list[0].Id)
	if err != nil {
		t.Fatalf("find updated event: %v", err)
	}
	if refreshed == nil || refreshed.Title != "斯诺克世锦赛" || !refreshed.Featured || refreshed.Status != model.EventNewsStatusLive {
		t.Fatalf("unexpected updated event: %#v", refreshed)
	}
	if refreshed.TournamentId <= 0 {
		t.Fatalf("expected updated event to keep tournament binding, got %#v", refreshed)
	}

	publishLogic := NewAdminPublishEventNewsLogic(adminTestCtx(-100), svcCtx)
	resp, err = publishLogic.AdminPublishEventNews(&types.AdminEventNewsPublishReq{
		EventId:   list[0].Id,
		Published: true,
	})
	if err != nil {
		t.Fatalf("publish event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected publish success, got %#v", resp)
	}

	published, err := svcCtx.EventNewsModel.FindById(list[0].Id)
	if err != nil {
		t.Fatalf("find published event: %v", err)
	}
	if published == nil || !published.Published || published.PublishedAt == nil {
		t.Fatalf("expected published event with published_at, got %#v", published)
	}

	resp, err = publishLogic.AdminPublishEventNews(&types.AdminEventNewsPublishReq{
		EventId:   list[0].Id,
		Published: false,
	})
	if err != nil {
		t.Fatalf("unpublish event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected unpublish success, got %#v", resp)
	}

	unpublished, err := svcCtx.EventNewsModel.FindById(list[0].Id)
	if err != nil {
		t.Fatalf("find unpublished event: %v", err)
	}
	if unpublished == nil || unpublished.Published || unpublished.PublishedAt != nil {
		t.Fatalf("expected unpublished event with cleared published_at, got %#v", unpublished)
	}
}

func TestAdminCreateUpdateDeleteEventNewsMatchAndCascadeOnEventDelete(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	start := adminTestTime(2026, 3, 24, 11, 0, 0)
	tournament := createSeedTournament(t, svcCtx, &model.Tournament{
		Name:       "中式八球公开赛实体",
		GameType:   3,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  &start,
	})
	event := createSeedEvent(t, svcCtx, &model.EventNews{
		Title:        "中式八球公开赛",
		TournamentId: tournament.Id,
		GameType:     3,
		SourceType:   "manual",
		SourceName:   "Admin",
		City:         "北京",
		Status:       model.EventNewsStatusUpcoming,
		StartTime:    &start,
		SortTime:     &start,
		Published:    false,
	})

	createMatchLogic := NewAdminCreateEventNewsMatchLogic(adminTestCtx(-100), svcCtx)
	resp, err := createMatchLogic.AdminCreateEventNewsMatch(&types.AdminEventNewsMatchCreateReq{
		EventId:        event.Id,
		RoundName:      "资格赛",
		RoundOrder:     10,
		MatchOrder:     1,
		StartTime:      adminTestTimeString(start),
		Status:         model.EventNewsStatusUpcoming,
		HomePlayerName: "选手甲",
		AwayPlayerName: "选手乙",
	})
	if err != nil {
		t.Fatalf("create event match: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected match create success, got %#v", resp)
	}

	matches, err := svcCtx.TournamentMatchModel.FindByTournament(event.TournamentId)
	if err != nil {
		t.Fatalf("find matches after create: %v", err)
	}
	if len(matches) != 1 || matches[0].RoundName != "资格赛" {
		t.Fatalf("expected one created match, got %#v", matches)
	}

	updateMatchLogic := NewAdminUpdateEventNewsMatchLogic(adminTestCtx(-100), svcCtx)
	resp, err = updateMatchLogic.AdminUpdateEventNewsMatch(&types.AdminEventNewsMatchUpdateReq{
		MatchId:        matches[0].Id,
		EventId:        event.Id,
		RoundName:      "32强",
		RoundOrder:     20,
		MatchOrder:     1,
		StartTime:      adminTestTimeString(start),
		Status:         model.EventNewsStatusLive,
		HomePlayerName: "选手甲",
		AwayPlayerName: "选手乙",
		HomeScore:      5,
		AwayScore:      3,
	})
	if err != nil {
		t.Fatalf("update event match: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected match update success, got %#v", resp)
	}

	refreshedMatch, err := svcCtx.TournamentMatchModel.FindById(matches[0].Id)
	if err != nil {
		t.Fatalf("find updated match: %v", err)
	}
	if refreshedMatch == nil || refreshedMatch.RoundName != "32强" || refreshedMatch.Status != model.EventNewsStatusLive {
		t.Fatalf("unexpected updated match: %#v", refreshedMatch)
	}

	deleteMatchLogic := NewAdminDeleteEventNewsMatchLogic(adminTestCtx(-100), svcCtx)
	resp, err = deleteMatchLogic.AdminDeleteEventNewsMatch(&types.AdminEventNewsMatchIdReq{MatchId: matches[0].Id})
	if err != nil {
		t.Fatalf("delete event match: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected match delete success, got %#v", resp)
	}

	emptyMatches, err := svcCtx.TournamentMatchModel.FindByTournament(event.TournamentId)
	if err != nil {
		t.Fatalf("find matches after delete: %v", err)
	}
	if len(emptyMatches) != 0 {
		t.Fatalf("expected matches cleared after delete, got %#v", emptyMatches)
	}

	createSeedMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   event.TournamentId,
		RoundName:      "16强",
		RoundOrder:     30,
		MatchOrder:     1,
		Status:         model.EventNewsStatusUpcoming,
		HomePlayerName: "选手A",
		AwayPlayerName: "选手B",
	})
	deleteEventLogic := NewAdminDeleteEventNewsLogic(adminTestCtx(-100), svcCtx)
	resp, err = deleteEventLogic.AdminDeleteEventNews(&types.AdminEventNewsIdReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("delete event: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected event delete success, got %#v", resp)
	}

	deletedEvent, err := svcCtx.EventNewsModel.FindById(event.Id)
	if err != nil {
		t.Fatalf("find deleted event: %v", err)
	}
	if deletedEvent != nil {
		t.Fatalf("expected event hidden after delete, got %#v", deletedEvent)
	}

	deletedMatches, err := svcCtx.TournamentMatchModel.FindByTournament(event.TournamentId)
	if err != nil {
		t.Fatalf("find matches after event delete: %v", err)
	}
	if len(deletedMatches) != 0 {
		t.Fatalf("expected match cascade delete, got %#v", deletedMatches)
	}
}

func TestAdminEventNewsListFiltersAndRejectsNonAdmin(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	firstStart := adminTestTime(2026, 3, 25, 9, 0, 0)
	secondStart := adminTestTime(2026, 3, 26, 9, 0, 0)
	tournament := createSeedTournament(t, svcCtx, &model.Tournament{
		Name:       "斯诺克焦点赛实体",
		GameType:   1,
		Format:     1,
		MaxPlayers: 16,
		Status:     model.EventNewsStatusLive,
		StartTime:  &firstStart,
	})
	createSeedEvent(t, svcCtx, &model.EventNews{
		Title:        "斯诺克焦点赛",
		TournamentId: tournament.Id,
		GameType:     1,
		SourceType:   "official",
		SourceName:   "WST",
		Status:       model.EventNewsStatusLive,
		Featured:     true,
		Published:    true,
		StartTime:    &firstStart,
		SortTime:     &firstStart,
	})
	createSeedMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId:   tournament.Id,
		RoundName:      "32强",
		RoundOrder:     20,
		MatchOrder:     1,
		Status:         model.EventNewsStatusLive,
		HomePlayerName: "A",
		AwayPlayerName: "B",
		HomeScore:      3,
		AwayScore:      2,
	})
	createSeedEvent(t, svcCtx, &model.EventNews{
		Title:      "中式九球预告",
		GameType:   2,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Featured:   false,
		Published:  false,
		StartTime:  &secondStart,
		SortTime:   &secondStart,
	})

	listLogic := NewAdminGetEventNewsListLogic(adminTestCtx(-100), svcCtx)
	resp, err := listLogic.AdminGetEventNewsList(&types.AdminEventNewsListReq{
		Page:      1,
		PageSize:  20,
		GameType:  1,
		Status:    model.EventNewsStatusLive,
		Published: 1,
	})
	if err != nil {
		t.Fatalf("get admin event news list: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected list success, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one filtered event, got %#v", resp)
	}
	if resp.List[0].Title != "斯诺克焦点赛" || !resp.List[0].Published {
		t.Fatalf("unexpected filtered event: %#v", resp.List[0])
	}
	if resp.List[0].CurrentRoundText != "32强" || resp.List[0].LatestResultText != "A 3 - 2 B" || resp.List[0].MatchCount != 1 {
		t.Fatalf("expected match summary fields in admin list, got %#v", resp.List[0])
	}

	rejectedLogic := NewAdminCreateEventNewsLogic(adminTestCtx(100), svcCtx)
	rejectedResp, err := rejectedLogic.AdminCreateEventNews(&types.AdminEventNewsCreateReq{
		Title:      "未授权赛事",
		GameType:   1,
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  adminTestTimeString(firstStart),
		SourceType: "manual",
	})
	if err != nil {
		t.Fatalf("non-admin create returned error: %v", err)
	}
	if rejectedResp.Success || rejectedResp.Code != 401 {
		t.Fatalf("expected non-admin to be rejected, got %#v", rejectedResp)
	}
}
