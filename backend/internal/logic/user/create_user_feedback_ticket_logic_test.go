package user

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

func newUserFeedbackLogicTestSvc(t *testing.T) *svc.ServiceContext {
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

func TestCreateUserFeedbackTicketStoresCurrentUser(t *testing.T) {
	svcCtx := newUserFeedbackLogicTestSvc(t)
	ctx := context.WithValue(context.Background(), "user_id", int64(12345))
	logic := NewCreateUserFeedbackTicketLogic(ctx, svcCtx)

	resp, err := logic.CreateUserFeedbackTicket(&types.CreateFeedbackTicketReq{
		Source:   model.FeedbackTicketSourceApp,
		Category: model.FeedbackTicketCategoryReport,
		Content:  "App 内举报一条违规动态",
		Contact:  "13800000000",
	})
	if err != nil {
		t.Fatalf("create user feedback ticket: %v", err)
	}
	if !resp.Success || resp.TicketId == 0 {
		t.Fatalf("expected user feedback success, got %#v", resp)
	}

	stored, err := svcCtx.FeedbackTicketModel.FindById(resp.TicketId)
	if err != nil {
		t.Fatalf("find user feedback ticket: %v", err)
	}
	if stored == nil || stored.UserId == nil || *stored.UserId != 12345 {
		t.Fatalf("expected ticket user id 12345, got %#v", stored)
	}
}
