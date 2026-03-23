package admin

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

func newEventNewsAdminTestSvc(t *testing.T) *svc.ServiceContext {
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

func adminTestCtx(adminID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", adminID)
}

func adminTestTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.Local)
}

func adminTestTimeString(t time.Time) string {
	return t.Format(adminEventNewsTimeLayout)
}

func createSeedEventNews(t *testing.T, svcCtx *svc.ServiceContext, item *model.EventNews) *model.EventNews {
	t.Helper()
	if err := svcCtx.EventNewsModel.Create(item); err != nil {
		t.Fatalf("seed event news: %v", err)
	}
	return item
}

func TestAdminCreateEventNewsCreatesDraft(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	logic := NewAdminCreateEventNewsLogic(adminTestCtx(-100), svcCtx)
	start := adminTestTime(2026, 3, 23, 10, 0, 0)

	resp, err := logic.AdminCreateEventNews(&types.AdminEventNewsCreateReq{
		Title:      "斯诺克公开赛",
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
		StageText:  "资格赛",
		ResultText: "待更新",
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
		t.Fatalf("find created event news: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected one created item, got total=%d list=%#v", total, list)
	}
	if list[0].Published {
		t.Fatalf("expected draft event, got published item: %#v", list[0])
	}
	if list[0].Title != "斯诺克公开赛" {
		t.Fatalf("unexpected created item: %#v", list[0])
	}
}

func TestAdminUpdatePublishAndDeleteEventNews(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	baseStart := adminTestTime(2026, 3, 24, 11, 0, 0)
	seed := createSeedEventNews(t, svcCtx, &model.EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "北京",
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  &baseStart,
		SortTime:   &baseStart,
		Published:  false,
	})

	updateLogic := NewAdminUpdateEventNewsLogic(adminTestCtx(-100), svcCtx)
	updatedStart := adminTestTime(2026, 3, 24, 12, 0, 0)
	updatedSort := adminTestTime(2026, 3, 24, 12, 5, 0)
	resp, err := updateLogic.AdminUpdateEventNews(&types.AdminEventNewsUpdateReq{
		EventNewsId: seed.Id,
		Title:       "中式八球超级赛",
		GameType:    3,
		SourceType:  "official",
		SourceName:  "CBSA",
		SourceUrl:   "https://example.com/eight-ball",
		CoverImage:  "https://example.com/eight.jpg",
		Summary:     "更新后的摘要",
		Content:     "更新后的正文",
		Country:     "中国",
		City:        "上海",
		Venue:       "球房A",
		StartTime:   adminTestTimeString(updatedStart),
		SortTime:    adminTestTimeString(updatedSort),
		Status:      model.EventNewsStatusLive,
		StageText:   "正赛",
		ResultText:  "进行中",
		Featured:    true,
	})
	if err != nil {
		t.Fatalf("update event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected update success, got %#v", resp)
	}

	refreshed, err := svcCtx.EventNewsModel.FindById(seed.Id)
	if err != nil {
		t.Fatalf("find updated event news: %v", err)
	}
	if refreshed == nil || refreshed.Title != "中式八球超级赛" || refreshed.Featured != true || refreshed.Status != model.EventNewsStatusLive {
		t.Fatalf("unexpected updated record: %#v", refreshed)
	}
	if refreshed.City != "上海" || refreshed.SourceName != "CBSA" {
		t.Fatalf("expected updated structured fields, got %#v", refreshed)
	}

	publishLogic := NewAdminPublishEventNewsLogic(adminTestCtx(-100), svcCtx)
	resp, err = publishLogic.AdminPublishEventNews(&types.AdminEventNewsPublishReq{
		EventNewsId: seed.Id,
		Published:   true,
	})
	if err != nil {
		t.Fatalf("publish event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected publish success, got %#v", resp)
	}

	published, err := svcCtx.EventNewsModel.FindById(seed.Id)
	if err != nil {
		t.Fatalf("find published event news: %v", err)
	}
	if published == nil || !published.Published || published.PublishedAt == nil {
		t.Fatalf("expected published record with published_at, got %#v", published)
	}

	resp, err = publishLogic.AdminPublishEventNews(&types.AdminEventNewsPublishReq{
		EventNewsId: seed.Id,
		Published:   false,
	})
	if err != nil {
		t.Fatalf("unpublish event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected unpublish success, got %#v", resp)
	}

	unpublished, err := svcCtx.EventNewsModel.FindById(seed.Id)
	if err != nil {
		t.Fatalf("find unpublished event news: %v", err)
	}
	if unpublished == nil || unpublished.Published || unpublished.PublishedAt != nil {
		t.Fatalf("expected unpublished record with cleared published_at, got %#v", unpublished)
	}

	deleteLogic := NewAdminDeleteEventNewsLogic(adminTestCtx(-100), svcCtx)
	resp, err = deleteLogic.AdminDeleteEventNews(&types.AdminEventNewsIdReq{EventNewsId: seed.Id})
	if err != nil {
		t.Fatalf("delete event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected delete success, got %#v", resp)
	}

	deleted, err := svcCtx.EventNewsModel.FindById(seed.Id)
	if err != nil {
		t.Fatalf("find deleted event news: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected soft deleted record to be hidden, got %#v", deleted)
	}
}

func TestAdminUpdateEventNewsClearsFieldsAndFollowsStartTime(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	originalStart := adminTestTime(2026, 3, 24, 11, 0, 0)
	originalEnd := adminTestTime(2026, 3, 24, 18, 0, 0)
	seed := createSeedEventNews(t, svcCtx, &model.EventNews{
		Title:      "赛事情报原始标题",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		SourceUrl:  "https://example.com/original",
		CoverImage: "https://example.com/original.jpg",
		Summary:    "原始摘要",
		Content:    "原始正文",
		Country:    "英国",
		City:       "伦敦",
		Venue:      "Venue A",
		StartTime:  &originalStart,
		EndTime:    &originalEnd,
		Status:     model.EventNewsStatusUpcoming,
		StageText:  "原始阶段",
		ResultText: "原始赛果",
		Featured:   false,
		SortTime:   &originalStart,
		Published:  false,
	})

	updateLogic := NewAdminUpdateEventNewsLogic(adminTestCtx(-100), svcCtx)
	nextStart := adminTestTime(2026, 3, 25, 10, 30, 0)
	resp, err := updateLogic.AdminUpdateEventNews(&types.AdminEventNewsUpdateReq{
		EventNewsId: seed.Id,
		Title:       "赛事情报原始标题",
		GameType:    1,
		SourceType:  "",
		SourceName:  "",
		SourceUrl:   "",
		CoverImage:  "",
		Summary:     "",
		Content:     "",
		Country:     "",
		City:        "",
		Venue:       "",
		StartTime:   adminTestTimeString(nextStart),
		EndTime:     "",
		Status:      model.EventNewsStatusLive,
		StageText:   "",
		ResultText:  "",
		Featured:    true,
	})
	if err != nil {
		t.Fatalf("clear update event news: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected clear update success, got %#v", resp)
	}

	refreshed, err := svcCtx.EventNewsModel.FindById(seed.Id)
	if err != nil {
		t.Fatalf("find cleared event news: %v", err)
	}
	if refreshed == nil {
		t.Fatal("expected refreshed event news")
	}
	if refreshed.SourceUrl != "" || refreshed.Summary != "" || refreshed.Content != "" || refreshed.StageText != "" || refreshed.ResultText != "" {
		t.Fatalf("expected cleared string fields, got %#v", refreshed)
	}
	if refreshed.EndTime != nil {
		t.Fatalf("expected end_time cleared, got %#v", refreshed.EndTime)
	}
	if refreshed.StartTime == nil || refreshed.SortTime == nil {
		t.Fatalf("expected start_time and sort_time set, got %#v", refreshed)
	}
	if !refreshed.StartTime.Equal(*refreshed.SortTime) {
		t.Fatalf("expected sort_time to follow start_time, got start=%v sort=%v", refreshed.StartTime, refreshed.SortTime)
	}
	if !refreshed.Featured || refreshed.Status != model.EventNewsStatusLive {
		t.Fatalf("expected other fields updated too, got %#v", refreshed)
	}
}

func TestAdminEventNewsListFiltersAndRejectsNonAdmin(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	firstStart := adminTestTime(2026, 3, 25, 9, 0, 0)
	secondStart := adminTestTime(2026, 3, 26, 9, 0, 0)
	createSeedEventNews(t, svcCtx, &model.EventNews{
		Title:      "斯诺克焦点赛",
		GameType:   1,
		SourceType: "official",
		SourceName: "WST",
		Status:     model.EventNewsStatusLive,
		Featured:   true,
		Published:  true,
		StartTime:  &firstStart,
		SortTime:   &firstStart,
	})
	createSeedEventNews(t, svcCtx, &model.EventNews{
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
		t.Fatalf("expected one filtered item, got %#v", resp)
	}
	if resp.List[0].Title != "斯诺克焦点赛" || !resp.List[0].Published {
		t.Fatalf("unexpected filtered item: %#v", resp.List[0])
	}

	emptyResp, err := listLogic.AdminGetEventNewsList(&types.AdminEventNewsListReq{
		Page:      1,
		PageSize:  20,
		GameType:  4,
		Status:    model.EventNewsStatusCanceled,
		Published: 1,
	})
	if err != nil {
		t.Fatalf("get empty admin list: %v", err)
	}
	if !emptyResp.Success || emptyResp.Code != 0 {
		t.Fatalf("expected empty list success, got %#v", emptyResp)
	}
	if emptyResp.List == nil || len(emptyResp.List) != 0 {
		t.Fatalf("expected empty slice, got %#v", emptyResp.List)
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
