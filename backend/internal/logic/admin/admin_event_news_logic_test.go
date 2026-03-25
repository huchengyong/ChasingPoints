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
		DB:                  db,
		EventNewsModel:      model.NewEventNewsModel(db),
		EventNewsStageModel: model.NewEventNewsStageModel(db),
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

func createSeedStage(t *testing.T, svcCtx *svc.ServiceContext, item *model.EventNewsStage) *model.EventNewsStage {
	t.Helper()
	if err := svcCtx.EventNewsStageModel.Create(item); err != nil {
		t.Fatalf("seed stage: %v", err)
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

func TestAdminCreateUpdateAndDeleteEventNewsStageAndCascadeOnEventDelete(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	start := adminTestTime(2026, 3, 24, 11, 0, 0)
	event := createSeedEvent(t, svcCtx, &model.EventNews{
		Title:      "中式八球公开赛",
		GameType:   3,
		SourceType: "manual",
		SourceName: "Admin",
		City:       "北京",
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  &start,
		SortTime:   &start,
		Published:  false,
	})

	createStageLogic := NewAdminCreateEventNewsStageLogic(adminTestCtx(-100), svcCtx)
	resp, err := createStageLogic.AdminCreateEventNewsStage(&types.AdminEventNewsStageCreateReq{
		EventId:    event.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		StartTime:  adminTestTimeString(start),
		Status:     model.EventNewsStatusUpcoming,
		ResultText: "待开赛",
	})
	if err != nil {
		t.Fatalf("create event stage: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected stage create success, got %#v", resp)
	}

	stages, err := svcCtx.EventNewsStageModel.FindByEventId(event.Id)
	if err != nil {
		t.Fatalf("find stages after create: %v", err)
	}
	if len(stages) != 1 || stages[0].StageName != "资格赛" {
		t.Fatalf("expected one created stage, got %#v", stages)
	}

	updateStageLogic := NewAdminUpdateEventNewsStageLogic(adminTestCtx(-100), svcCtx)
	resp, err = updateStageLogic.AdminUpdateEventNewsStage(&types.AdminEventNewsStageUpdateReq{
		StageId:    stages[0].Id,
		EventId:    event.Id,
		StageName:  "32强",
		StageOrder: 20,
		StartTime:  adminTestTimeString(start),
		Status:     model.EventNewsStatusLive,
		ResultText: "32强进行中",
	})
	if err != nil {
		t.Fatalf("update event stage: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected stage update success, got %#v", resp)
	}

	refreshedStage, err := svcCtx.EventNewsStageModel.FindById(stages[0].Id)
	if err != nil {
		t.Fatalf("find updated stage: %v", err)
	}
	if refreshedStage == nil || refreshedStage.StageName != "32强" || refreshedStage.Status != model.EventNewsStatusLive {
		t.Fatalf("unexpected updated stage: %#v", refreshedStage)
	}

	deleteStageLogic := NewAdminDeleteEventNewsStageLogic(adminTestCtx(-100), svcCtx)
	resp, err = deleteStageLogic.AdminDeleteEventNewsStage(&types.AdminEventNewsStageIdReq{StageId: stages[0].Id})
	if err != nil {
		t.Fatalf("delete event stage: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected stage delete success, got %#v", resp)
	}

	emptyStages, err := svcCtx.EventNewsStageModel.FindByEventId(event.Id)
	if err != nil {
		t.Fatalf("find stages after delete: %v", err)
	}
	if len(emptyStages) != 0 {
		t.Fatalf("expected stages cleared after delete, got %#v", emptyStages)
	}

	createSeedStage(t, svcCtx, &model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "16强",
		StageOrder: 30,
		Status:     model.EventNewsStatusUpcoming,
		ResultText: "待更新",
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

	deletedStages, err := svcCtx.EventNewsStageModel.FindByEventId(event.Id)
	if err != nil {
		t.Fatalf("find stages after event delete: %v", err)
	}
	if len(deletedStages) != 0 {
		t.Fatalf("expected stage cascade delete, got %#v", deletedStages)
	}
}

func TestAdminEventNewsStageRejectsInvalidTimeRange(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	start := adminTestTime(2026, 3, 24, 11, 0, 0)
	end := adminTestTime(2026, 3, 24, 10, 0, 0)
	event := createSeedEvent(t, svcCtx, &model.EventNews{
		Title:      "时间校验赛事",
		GameType:   1,
		SourceType: "manual",
		SourceName: "Admin",
		Status:     model.EventNewsStatusUpcoming,
		StartTime:  &start,
		SortTime:   &start,
	})

	createStageLogic := NewAdminCreateEventNewsStageLogic(adminTestCtx(-100), svcCtx)
	createResp, err := createStageLogic.AdminCreateEventNewsStage(&types.AdminEventNewsStageCreateReq{
		EventId:    event.Id,
		StageName:  "资格赛",
		StageOrder: 10,
		StartTime:  adminTestTimeString(start),
		EndTime:    adminTestTimeString(end),
		Status:     model.EventNewsStatusUpcoming,
	})
	if err != nil {
		t.Fatalf("create invalid range stage: %v", err)
	}
	if createResp.Success || createResp.Code != 400 || createResp.Message != "结束时间不能早于开始时间" {
		t.Fatalf("expected invalid time range to be rejected on create, got %#v", createResp)
	}

	stage := createSeedStage(t, svcCtx, &model.EventNewsStage{
		EventId:    event.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     model.EventNewsStatusUpcoming,
	})

	updateStageLogic := NewAdminUpdateEventNewsStageLogic(adminTestCtx(-100), svcCtx)
	updateResp, err := updateStageLogic.AdminUpdateEventNewsStage(&types.AdminEventNewsStageUpdateReq{
		StageId:    stage.Id,
		EventId:    event.Id,
		StageName:  "32强",
		StageOrder: 20,
		StartTime:  adminTestTimeString(start),
		EndTime:    adminTestTimeString(end),
		Status:     model.EventNewsStatusLive,
	})
	if err != nil {
		t.Fatalf("update invalid range stage: %v", err)
	}
	if updateResp.Success || updateResp.Code != 400 || updateResp.Message != "结束时间不能早于开始时间" {
		t.Fatalf("expected invalid time range to be rejected on update, got %#v", updateResp)
	}
}

func TestAdminEventNewsListFiltersAndRejectsNonAdmin(t *testing.T) {
	svcCtx := newEventNewsAdminTestSvc(t)
	firstStart := adminTestTime(2026, 3, 25, 9, 0, 0)
	secondStart := adminTestTime(2026, 3, 26, 9, 0, 0)
	first := createSeedEvent(t, svcCtx, &model.EventNews{
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
	createSeedStage(t, svcCtx, &model.EventNewsStage{
		EventId:    first.Id,
		StageName:  "32强",
		StageOrder: 20,
		Status:     model.EventNewsStatusLive,
		ResultText: "32强进行中",
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
	if resp.List[0].CurrentStageText != "32强" || resp.List[0].LatestResultText != "32强进行中" || resp.List[0].StageCount != 1 {
		t.Fatalf("expected stage summary fields in admin list, got %#v", resp.List[0])
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
