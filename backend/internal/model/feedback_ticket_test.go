package model

import (
	"testing"
	"time"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFeedbackTicketTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFeedbackSchema(db); err != nil {
		t.Fatalf("prepare feedback schema: %v", err)
	}

	return db
}

func TestFeedbackTicketModelCreatesAndFindsTicket(t *testing.T) {
	db := newFeedbackTicketTestDB(t)
	ticketModel := NewFeedbackTicketModel(db)

	userID := int64(101)
	ticket := &FeedbackTicket{
		UserId:   &userID,
		Source:   FeedbackTicketSourceApp,
		Category: FeedbackTicketCategoryComplaint,
		Content:  "有人冒用我的昵称发布违规内容",
		Contact:  "13800000000",
		Status:   FeedbackTicketStatusPending,
	}
	if err := ticketModel.Create(ticket); err != nil {
		t.Fatalf("create feedback ticket: %v", err)
	}
	if ticket.Id == 0 {
		t.Fatal("expected ticket id to be populated")
	}

	stored, err := ticketModel.FindById(ticket.Id)
	if err != nil {
		t.Fatalf("find feedback ticket: %v", err)
	}
	if stored == nil || stored.UserId == nil || *stored.UserId != userID {
		t.Fatalf("expected stored app ticket with user id, got %#v", stored)
	}
	if stored.Category != FeedbackTicketCategoryComplaint || stored.Status != FeedbackTicketStatusPending {
		t.Fatalf("unexpected stored category/status: %#v", stored)
	}
}

func TestFeedbackTicketModelFindListForAdminFilters(t *testing.T) {
	db := newFeedbackTicketTestDB(t)
	ticketModel := NewFeedbackTicketModel(db)

	seed := []FeedbackTicket{
		{Source: FeedbackTicketSourceApp, Category: FeedbackTicketCategoryComplaint, Content: "投诉内容", Status: FeedbackTicketStatusPending},
		{Source: FeedbackTicketSourceWebsite, Category: FeedbackTicketCategoryReport, Content: "举报内容", Status: FeedbackTicketStatusResolved},
		{Source: FeedbackTicketSourceWebsite, Category: FeedbackTicketCategoryComplaint, Content: "官网投诉", Status: FeedbackTicketStatusPending},
	}
	for i := range seed {
		if err := ticketModel.Create(&seed[i]); err != nil {
			t.Fatalf("seed feedback ticket %d: %v", i, err)
		}
	}

	list, total, err := ticketModel.FindListForAdmin(FeedbackTicketAdminFilter{
		Page:     1,
		PageSize: 20,
		Status:   FeedbackTicketStatusPending,
		Category: FeedbackTicketCategoryComplaint,
	})
	if err != nil {
		t.Fatalf("find admin feedback list: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected two pending complaints, total=%d list=%#v", total, list)
	}
	if list[0].Id <= list[1].Id {
		t.Fatalf("expected newest ticket first, got ids %d then %d", list[0].Id, list[1].Id)
	}
}

func TestFeedbackTicketModelProcessStoresHandlerResultAndTime(t *testing.T) {
	db := newFeedbackTicketTestDB(t)
	ticketModel := NewFeedbackTicketModel(db)

	ticket := &FeedbackTicket{
		Source:   FeedbackTicketSourceWebsite,
		Category: FeedbackTicketCategoryReport,
		Content:  "有人发布违规宣传",
		Contact:  "report@example.com",
		Status:   FeedbackTicketStatusPending,
	}
	if err := ticketModel.Create(ticket); err != nil {
		t.Fatalf("create feedback ticket: %v", err)
	}

	processedAt := time.Date(2026, 4, 10, 11, 0, 0, 0, time.UTC)
	if err := ticketModel.Process(ticket.Id, FeedbackTicketStatusResolved, "已核查并处理违规内容", 9001, processedAt); err != nil {
		t.Fatalf("process feedback ticket: %v", err)
	}

	stored, err := ticketModel.FindById(ticket.Id)
	if err != nil {
		t.Fatalf("find processed feedback ticket: %v", err)
	}
	if stored == nil || stored.Status != FeedbackTicketStatusResolved {
		t.Fatalf("expected resolved ticket, got %#v", stored)
	}
	if stored.HandlerId == nil || *stored.HandlerId != 9001 {
		t.Fatalf("expected handler id 9001, got %#v", stored)
	}
	if stored.ProcessResult != "已核查并处理违规内容" {
		t.Fatalf("expected process result to be saved, got %q", stored.ProcessResult)
	}
	if stored.ProcessedAt == nil || !stored.ProcessedAt.Equal(processedAt) {
		t.Fatalf("expected processed_at %v, got %#v", processedAt, stored.ProcessedAt)
	}
}
