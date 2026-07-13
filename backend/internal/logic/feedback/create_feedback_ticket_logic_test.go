package feedback

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFeedbackLogicTestSvc(t *testing.T) *svc.ServiceContext {
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

func TestCreateFeedbackTicketAllowsAnonymousWebsiteSubmission(t *testing.T) {
	svcCtx := newFeedbackLogicTestSvc(t)
	logic := NewCreateFeedbackTicketLogic(context.Background(), svcCtx)

	resp, err := logic.CreateFeedbackTicket(&types.CreateFeedbackTicketReq{
		Source:   model.FeedbackTicketSourceWebsite,
		Category: model.FeedbackTicketCategoryComplaint,
		Content:  "官网联系页提交的投诉举报内容",
		Contact:  "complaint@example.com",
	})
	if err != nil {
		t.Fatalf("create public feedback ticket: %v", err)
	}
	if !resp.Success || resp.TicketId == 0 {
		t.Fatalf("expected public feedback success, got %#v", resp)
	}

	stored, err := svcCtx.FeedbackTicketModel.FindById(resp.TicketId)
	if err != nil {
		t.Fatalf("find public feedback ticket: %v", err)
	}
	if stored == nil || stored.UserId != nil {
		t.Fatalf("expected anonymous website ticket, got %#v", stored)
	}
}

func TestCreateFeedbackTicketRejectsEmptyContent(t *testing.T) {
	svcCtx := newFeedbackLogicTestSvc(t)
	logic := NewCreateFeedbackTicketLogic(context.Background(), svcCtx)

	resp, err := logic.CreateFeedbackTicket(&types.CreateFeedbackTicketReq{
		Source:   model.FeedbackTicketSourceWebsite,
		Category: model.FeedbackTicketCategoryReport,
		Content:  "   ",
	})
	if err != nil {
		t.Fatalf("empty feedback returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected empty content to fail, got %#v", resp)
	}
}
