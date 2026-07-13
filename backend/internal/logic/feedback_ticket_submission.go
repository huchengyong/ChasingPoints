package logic

import (
	"context"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

const (
	feedbackTicketMaxContentRunes = 2000
	feedbackTicketMaxContactRunes = 128
)

type FeedbackTicketSubmission struct {
	UserId   int64
	Source   string
	Category string
	Content  string
	Contact  string
}

func CreateFeedbackTicket(ctx context.Context, svcCtx *svc.ServiceContext, input FeedbackTicketSubmission) (*types.CreateFeedbackTicketResp, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return &types.CreateFeedbackTicketResp{Success: false, Message: "请输入投诉举报或反馈内容"}, nil
	}
	if len([]rune(content)) > feedbackTicketMaxContentRunes {
		return &types.CreateFeedbackTicketResp{Success: false, Message: "内容不能超过2000字"}, nil
	}

	contact := strings.TrimSpace(input.Contact)
	if len([]rune(contact)) > feedbackTicketMaxContactRunes {
		return &types.CreateFeedbackTicketResp{Success: false, Message: "联系方式不能超过128字"}, nil
	}

	var userID *int64
	if input.UserId > 0 {
		value := input.UserId
		userID = &value
	}

	ticket := &model.FeedbackTicket{
		UserId:   userID,
		Source:   model.NormalizeFeedbackTicketSource(input.Source),
		Category: model.NormalizeFeedbackTicketCategory(input.Category),
		Content:  content,
		Contact:  contact,
		Status:   model.FeedbackTicketStatusPending,
	}
	if err := svcCtx.FeedbackTicketModel.Create(ticket); err != nil {
		return nil, err
	}

	_ = ctx
	return &types.CreateFeedbackTicketResp{
		Success:  true,
		Message:  "提交成功，我们会尽快处理",
		TicketId: ticket.Id,
	}, nil
}
