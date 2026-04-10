package admin

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildAdminFeedbackTicketInfo(ticket model.FeedbackTicket) types.AdminFeedbackTicketInfo {
	info := types.AdminFeedbackTicketInfo{
		Id:            ticket.Id,
		Source:        ticket.Source,
		SourceText:    feedbackTicketSourceText(ticket.Source),
		Category:      ticket.Category,
		CategoryText:  feedbackTicketCategoryText(ticket.Category),
		Content:       ticket.Content,
		Contact:       ticket.Contact,
		Status:        ticket.Status,
		StatusText:    feedbackTicketStatusText(ticket.Status),
		ProcessResult: ticket.ProcessResult,
		CreatedAt:     ticket.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if ticket.UserId != nil {
		info.UserId = *ticket.UserId
	}
	if ticket.HandlerId != nil {
		info.HandlerId = *ticket.HandlerId
	}
	if ticket.ProcessedAt != nil {
		info.ProcessedAt = ticket.ProcessedAt.Format("2006-01-02 15:04:05")
	}
	return info
}

func feedbackTicketSourceText(source string) string {
	switch source {
	case model.FeedbackTicketSourceWebsite:
		return "官网"
	default:
		return "App"
	}
}

func feedbackTicketCategoryText(category string) string {
	switch category {
	case model.FeedbackTicketCategoryComplaint:
		return "投诉"
	case model.FeedbackTicketCategoryReport:
		return "举报"
	default:
		return "意见反馈"
	}
}

func feedbackTicketStatusText(status int) string {
	switch status {
	case model.FeedbackTicketStatusProcessing:
		return "处理中"
	case model.FeedbackTicketStatusResolved:
		return "已办结"
	case model.FeedbackTicketStatusClosed:
		return "已关闭"
	default:
		return "待处理"
	}
}
