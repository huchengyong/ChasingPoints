package admin

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminFeedbackTicketTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFeedbackSchema(db); err != nil {
		t.Fatalf("prepare feedback schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                  db,
		FeedbackTicketModel: model.NewFeedbackTicketModel(db),
	}
}

func seedAdminFeedbackTicket(t *testing.T, svcCtx *svc.ServiceContext, ticket *model.FeedbackTicket) {
	t.Helper()
	if err := svcCtx.FeedbackTicketModel.Create(ticket); err != nil {
		t.Fatalf("seed feedback ticket: %v", err)
	}
}

func TestAdminGetFeedbackTicketListFiltersPendingComplaints(t *testing.T) {
	svcCtx := newAdminFeedbackTicketTestSvc(t)
	seedAdminFeedbackTicket(t, svcCtx, &model.FeedbackTicket{
		Source:   model.FeedbackTicketSourceApp,
		Category: model.FeedbackTicketCategoryComplaint,
		Content:  "App 投诉",
		Status:   model.FeedbackTicketStatusPending,
	})
	seedAdminFeedbackTicket(t, svcCtx, &model.FeedbackTicket{
		Source:   model.FeedbackTicketSourceWebsite,
		Category: model.FeedbackTicketCategoryReport,
		Content:  "官网举报",
		Status:   model.FeedbackTicketStatusResolved,
	})

	logic := NewAdminGetFeedbackTicketListLogic(adminCtx(9101), svcCtx)
	resp, err := logic.AdminGetFeedbackTicketList(&types.AdminFeedbackTicketListReq{
		Page:     1,
		PageSize: 20,
		Status:   model.FeedbackTicketStatusPending,
		Category: model.FeedbackTicketCategoryComplaint,
	})
	if err != nil {
		t.Fatalf("admin get feedback ticket list: %v", err)
	}
	if !resp.Success || resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected one pending complaint, got %#v", resp)
	}
	if resp.List[0].CategoryText != "投诉" || resp.List[0].StatusText != "待处理" {
		t.Fatalf("expected display labels, got %#v", resp.List[0])
	}
}

func TestAdminProcessFeedbackTicketStoresHandlerAndResult(t *testing.T) {
	svcCtx := newAdminFeedbackTicketTestSvc(t)
	ticket := &model.FeedbackTicket{
		Source:   model.FeedbackTicketSourceWebsite,
		Category: model.FeedbackTicketCategoryReport,
		Content:  "举报冒充官方账号",
		Status:   model.FeedbackTicketStatusPending,
	}
	seedAdminFeedbackTicket(t, svcCtx, ticket)

	logic := NewAdminProcessFeedbackTicketLogic(adminCtx(9102), svcCtx)
	resp, err := logic.AdminProcessFeedbackTicket(&types.AdminFeedbackTicketProcessReq{
		TicketId:      ticket.Id,
		Status:        model.FeedbackTicketStatusResolved,
		ProcessResult: "已封禁冒充账号",
	})
	if err != nil {
		t.Fatalf("admin process feedback ticket: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected process success, got %#v", resp)
	}

	stored, err := svcCtx.FeedbackTicketModel.FindById(ticket.Id)
	if err != nil {
		t.Fatalf("find processed feedback ticket: %v", err)
	}
	if stored == nil || stored.HandlerId == nil || *stored.HandlerId != 9102 {
		t.Fatalf("expected handler id 9102, got %#v", stored)
	}
	if stored.ProcessResult != "已封禁冒充账号" || stored.ProcessedAt == nil {
		t.Fatalf("expected process result and time, got %#v", stored)
	}
}
