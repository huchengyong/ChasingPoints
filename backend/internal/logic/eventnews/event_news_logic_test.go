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
		DB:                 db,
		EventNewsModel:     model.NewEventNewsModel(db),
		EventNewsStageModel: model.NewEventNewsStageModel(db),
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

func createStage(t *testing.T, svcCtx *svc.ServiceContext, stage *model.EventNewsStage) *model.EventNewsStage {
	t.Helper()

	if err := svcCtx.EventNewsStageModel.Create(stage); err != nil {
		t.Fatalf("create stage: %v", err)
	}
	return stage
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

func TestGetEventNewsListFiltersPublishedEventParentsAndExposesStageSummary(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	snooker := createEvent(t, svcCtx, &model.EventNews{
		Title:      "2026斯诺克世锦赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		City:       "谢菲尔德",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)),
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    snooker.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		Status:     model.EventNewsStatusFinished,
		ResultText: "资格赛收官",
		SortTime:   mustTimePtr(time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)),
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    snooker.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     model.EventNewsStatusLive,
		ResultText: "赵心童晋级16强",
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 30, 0, 0, time.UTC)),
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
	if resp.List[0].CurrentStageText != "32强" {
		t.Fatalf("expected current stage summary from stages, got %#v", resp.List[0])
	}
	if resp.List[0].LatestResultText != "赵心童晋级16强" {
		t.Fatalf("expected latest result summary from stages, got %#v", resp.List[0])
	}
	if resp.List[0].StageCount != 2 {
		t.Fatalf("expected two stages, got %#v", resp.List[0])
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

func TestGetEventNewsDetailReturnsGroupedStagesForPublishedEvent(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	event := createEvent(t, svcCtx, &model.EventNews{
		Title:      "2026斯诺克世锦赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 50, 0, 0, time.UTC)),
		Content:    "赛事详情正文",
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "16强",
		StageOrder: 30,
		Status:     model.EventNewsStatusUpcoming,
		ResultText: "待更新",
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		Status:     model.EventNewsStatusFinished,
		ResultText: "资格赛结束",
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     model.EventNewsStatusLive,
		ResultText: "32强进行中",
	})

	unpublished := createEvent(t, svcCtx, &model.EventNews{
		Title:      "未发布赛事",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		Published:  false,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 12, 10, 0, 0, time.UTC)),
	})

	logic := NewGetEventNewsDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventId: event.Id})
	if err != nil {
		t.Fatalf("get published detail: %v", err)
	}
	if !resp.Success || resp.Event == nil {
		t.Fatalf("expected published detail success, got %#v", resp)
	}
	if resp.EventNews == nil || resp.EventNews.Id != event.Id {
		t.Fatalf("expected legacy event_news payload mirror, got %#v", resp)
	}
	if resp.Event.Title != event.Title || resp.Event.Content != "赛事详情正文" {
		t.Fatalf("unexpected event detail payload: %#v", resp.Event)
	}
	if len(resp.Stages) != 3 {
		t.Fatalf("expected three ordered stages, got %#v", resp.Stages)
	}
	if resp.Stages[0].StageName != "资格赛" || resp.Stages[1].StageName != "32强" || resp.Stages[2].StageName != "16强" {
		t.Fatalf("expected stages sorted by stage_order, got %#v", resp.Stages)
	}

	resp, err = logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventId: unpublished.Id})
	if err != nil {
		t.Fatalf("get unpublished detail: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected unpublished detail to be rejected, got %#v", resp)
	}
}

func TestGetEventNewsDetailAcceptsLegacyEventNewsIDParam(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	event := createEvent(t, svcCtx, &model.EventNews{
		Title:      "兼容详情赛事",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Published:  true,
	})

	logic := NewGetEventNewsDetailLogic(context.Background(), svcCtx)
	resp, err := logic.GetEventNewsDetail(&types.GetEventNewsDetailReq{EventNewsId: event.Id})
	if err != nil {
		t.Fatalf("get legacy detail: %v", err)
	}
	if !resp.Success || resp.Event == nil || resp.Event.Id != event.Id {
		t.Fatalf("expected legacy event_news_id to resolve detail, got %#v", resp)
	}
	if resp.EventNews == nil || resp.EventNews.Id != event.Id {
		t.Fatalf("expected event_news mirror field, got %#v", resp)
	}
}

func TestGetFeaturedEventNewsPrefersFeaturedEventAndIncludesStagePreview(t *testing.T) {
	svcCtx := newEventNewsTestSvc(t)
	defer withEventNewsNow(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))()

	featured := createEvent(t, svcCtx, &model.EventNews{
		Title:      "焦点赛事",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusUpcoming,
		Featured:   true,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 13, 0, 0, 0, time.UTC)),
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    featured.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		Status:     model.EventNewsStatusUpcoming,
		ResultText: "明日开打",
	})

	other := createEvent(t, svcCtx, &model.EventNews{
		Title:      "普通进行中赛事",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusLive,
		Published:  true,
		SortTime:   mustTimePtr(time.Date(2026, 3, 23, 11, 55, 0, 0, time.UTC)),
	})
	createStage(t, svcCtx, &model.EventNewsStage{
		EventId:    other.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     model.EventNewsStatusLive,
		ResultText: "32强激战中",
	})

	logic := NewGetFeaturedEventNewsLogic(context.Background(), svcCtx)
	resp, err := logic.GetFeaturedEventNews()
	if err != nil {
		t.Fatalf("get featured: %v", err)
	}
	if !resp.Success || resp.Event == nil {
		t.Fatalf("expected featured success, got %#v", resp)
	}
	if resp.EventNews == nil || resp.EventNews.Id != featured.Id {
		t.Fatalf("expected featured legacy event_news mirror, got %#v", resp)
	}
	if resp.Event.Id != featured.Id {
		t.Fatalf("expected featured event to win, got %#v", resp.Event)
	}
	if resp.Event.CurrentStageText != "资格赛" || resp.Event.LatestResultText != "明日开打" {
		t.Fatalf("expected stage preview on featured event, got %#v", resp.Event)
	}
}
