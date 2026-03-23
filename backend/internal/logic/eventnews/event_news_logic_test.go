package eventnews

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
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

	return &svc.ServiceContext{
		DB:             db,
		EventNewsModel: model.NewEventNewsModel(db),
	}
}

func mustTimePtr(t time.Time) *time.Time {
	v := t
	return &v
}

func createEventNews(t *testing.T, svcCtx *svc.ServiceContext, news *model.EventNews) *model.EventNews {
	t.Helper()

	if err := svcCtx.EventNewsModel.Create(news); err != nil {
		t.Fatalf("create event news: %v", err)
	}
	return news
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

func TestGetEventNewsListFiltersByGameTypeAndPublishedState(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "斯诺克公开赛A",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		City:       "上海",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "斯诺克公开赛B",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "上海",
		Status:     model.EventNewsStatusLive,
		Published:  false,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 30, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "北京",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 45, 0, 0, time.UTC)),
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
	if resp.Total != 1 {
		t.Fatalf("expected one published snooker event, got %d", resp.Total)
	}
	if resp.List == nil || len(resp.List) != 1 {
		t.Fatalf("expected one list item, got %#v", resp.List)
	}
	if resp.List[0].Title != "斯诺克公开赛A" {
		t.Fatalf("expected published snooker item, got %#v", resp.List[0])
	}
	if resp.List[0].Published != true {
		t.Fatalf("expected public item to stay published, got %#v", resp.List[0])
	}
}

func TestGetEventNewsListFiltersByStatusAndSortsByConsumerPriority(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "进行中-更近",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "进行中-更远",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 10, 30, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "即将开始",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "已结束",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusFinished,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 40, 0, 0, time.UTC)),
	})

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   model.EventNewsStatusLive,
	})
	if err != nil {
		t.Fatalf("get list by status: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 2 {
		t.Fatalf("expected two live events, got %d", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected two list items, got %#v", resp.List)
	}
	if resp.List[0].Title != "进行中-更近" || resp.List[1].Title != "进行中-更远" {
		t.Fatalf("expected live events ordered by proximity, got %#v", resp.List)
	}
	for _, item := range resp.List {
		if item.Status != model.EventNewsStatusLive {
			t.Fatalf("expected only live items, got %#v", item)
		}
	}
}

func TestGetEventNewsListShowsAllStatusesWhenStatusIsMinusOne(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "进行中",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "即将开始",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)),
	})
	createEventNews(t, svcCtx, &model.EventNews{
		Title:      "已结束",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusFinished,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 40, 0, 0, time.UTC)),
	})

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		Status:   -1,
		GameType: 1,
	})
	if err != nil {
		t.Fatalf("get list with sentinel status: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 3 {
		t.Fatalf("expected three items, got %d", resp.Total)
	}
	if len(resp.List) != 3 {
		t.Fatalf("expected three list items, got %#v", resp.List)
	}
	if resp.List[0].Title != "进行中" || resp.List[1].Title != "即将开始" || resp.List[2].Title != "已结束" {
		t.Fatalf("expected consumer-friendly ordering, got %#v", resp.List)
	}
}

func TestGetEventNewsListReturnsEmptySliceWhenNoMatches(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	logic := NewGetEventNewsListLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsList(&types.GetEventNewsListReq{
		Page:     1,
		PageSize: 20,
		GameType: 1,
	})
	if err != nil {
		t.Fatalf("get empty list: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
	if resp.List == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %#v", resp.List)
	}
}

func TestGetEventNewsDetailOnlyReturnsPublishedContent(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	published := createEventNews(t, svcCtx, &model.EventNews{
		Title:      "已发布详情",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 50, 0, 0, time.UTC)),
		Content:    "detail content",
	})
	unpublished := createEventNews(t, svcCtx, &model.EventNews{
		Title:      "未发布详情",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  false,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 10, 0, 0, time.UTC)),
	})

	logic := NewGetEventNewsDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventNewsId: published.Id})
	if err != nil {
		t.Fatalf("get published detail: %v", err)
	}
	if !resp.Success || resp.EventNews == nil {
		t.Fatalf("expected published detail success, got %#v", resp)
	}
	if resp.EventNews.Title != "已发布详情" || resp.EventNews.Content != "detail content" {
		t.Fatalf("unexpected detail payload: %#v", resp.EventNews)
	}

	resp, err = logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventNewsId: unpublished.Id})
	if err != nil {
		t.Fatalf("get unpublished detail: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected unpublished detail to be rejected, got %#v", resp)
	}

	resp, err = logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventNewsId: 999999})
	if err != nil {
		t.Fatalf("get missing detail: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected missing detail to be rejected, got %#v", resp)
	}
}

func TestGetFeaturedEventNewsPrefersFeaturedThenFallsBackToBestActiveEvent(t *testing.T) {
	t.Run("featured item wins", func(t *testing.T) {
		svcCtx := newEventNewsTestSvc(t)
		defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

		featured := createEventNews(t, svcCtx, &model.EventNews{
			Title:      "焦点赛事",
			GameType:   1,
			SourceType: "official",
			SourceName: "WST",
			Status:     model.EventNewsStatusUpcoming,
			Featured:   true,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 13, 0, 0, 0, time.UTC)),
		})
		createEventNews(t, svcCtx, &model.EventNews{
			Title:      "普通进行中",
			GameType:   1,
			SourceType: "manual",
			SourceName: "Admin",
			Status:     model.EventNewsStatusLive,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
		})

		logic := NewGetFeaturedEventNewsLogic(context.Background(), svcCtx)
		resp, err := logic.GetFeaturedEventNews()
		if err != nil {
			t.Fatalf("get featured: %v", err)
		}
		if !resp.Success || resp.EventNews == nil {
			t.Fatalf("expected featured success, got %#v", resp)
		}
		if resp.EventNews.Id != featured.Id {
			t.Fatalf("expected featured item to win, got %#v", resp.EventNews)
		}
	})

	t.Run("fallback prefers live over upcoming and finished", func(t *testing.T) {
		svcCtx := newEventNewsTestSvc(t)
		defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

		bestLive := createEventNews(t, svcCtx, &model.EventNews{
			Title:      "更近的进行中",
			GameType:   1,
			SourceType: "official",
			SourceName: "WST",
			Status:     model.EventNewsStatusLive,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 56, 0, 0, time.UTC)),
		})
		createEventNews(t, svcCtx, &model.EventNews{
			Title:      "更远的进行中",
			GameType:   1,
			SourceType: "official",
			SourceName: "WST",
			Status:     model.EventNewsStatusLive,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 20, 0, 0, time.UTC)),
		})
		createEventNews(t, svcCtx, &model.EventNews{
			Title:      "即将开始",
			GameType:   3,
			SourceType: "manual",
			SourceName: "Admin",
			Status:     model.EventNewsStatusUpcoming,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 20, 0, 0, time.UTC)),
		})
		createEventNews(t, svcCtx, &model.EventNews{
			Title:      "刚结束",
			GameType:   2,
			SourceType: "manual",
			SourceName: "Admin",
			Status:     model.EventNewsStatusFinished,
			Published:  true,
			SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 58, 0, 0, time.UTC)),
		})

		logic := NewGetFeaturedEventNewsLogic(context.Background(), svcCtx)
		resp, err := logic.GetFeaturedEventNews()
		if err != nil {
			t.Fatalf("get fallback featured: %v", err)
		}
		if !resp.Success || resp.EventNews == nil {
			t.Fatalf("expected fallback success, got %#v", resp)
		}
		if resp.EventNews.Id != bestLive.Id {
			t.Fatalf("expected nearest live event, got %#v", resp.EventNews)
		}
	})
}
