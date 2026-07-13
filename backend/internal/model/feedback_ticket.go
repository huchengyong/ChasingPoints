package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	FeedbackTicketSourceApp     = "app"
	FeedbackTicketSourceWebsite = "website"
)

const (
	FeedbackTicketCategoryFeedback  = "feedback"
	FeedbackTicketCategoryComplaint = "complaint"
	FeedbackTicketCategoryReport    = "report"
)

const (
	FeedbackTicketStatusPending    = 1
	FeedbackTicketStatusProcessing = 2
	FeedbackTicketStatusResolved   = 3
	FeedbackTicketStatusClosed     = 4
)

type FeedbackTicket struct {
	Id            int64      `gorm:"primarykey" json:"id"`
	UserId        *int64     `gorm:"index" json:"user_id"`
	Source        string     `gorm:"size:32;not null;default:'app';index" json:"source"`
	Category      string     `gorm:"size:32;not null;default:'feedback';index" json:"category"`
	Content       string     `gorm:"type:text;not null" json:"content"`
	Contact       string     `gorm:"size:128;not null;default:''" json:"contact"`
	Status        int        `gorm:"not null;default:1;index" json:"status"`
	HandlerId     *int64     `json:"handler_id"`
	ProcessResult string     `gorm:"type:text;not null;default:''" json:"process_result"`
	ProcessedAt   *time.Time `json:"processed_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FeedbackTicket) TableName() string {
	return "feedback_tickets"
}

type FeedbackTicketAdminFilter struct {
	Page     int
	PageSize int
	Status   int
	Category string
	Source   string
}

type FeedbackTicketModel struct {
	db *gorm.DB
}

func NewFeedbackTicketModel(db *gorm.DB) *FeedbackTicketModel {
	return &FeedbackTicketModel{db: db}
}

func NormalizeFeedbackTicketSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case FeedbackTicketSourceWebsite:
		return FeedbackTicketSourceWebsite
	default:
		return FeedbackTicketSourceApp
	}
}

func NormalizeFeedbackTicketCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case FeedbackTicketCategoryComplaint:
		return FeedbackTicketCategoryComplaint
	case FeedbackTicketCategoryReport:
		return FeedbackTicketCategoryReport
	default:
		return FeedbackTicketCategoryFeedback
	}
}

func IsFeedbackTicketProcessStatus(status int) bool {
	return status == FeedbackTicketStatusProcessing ||
		status == FeedbackTicketStatusResolved ||
		status == FeedbackTicketStatusClosed
}

func (m *FeedbackTicketModel) Create(ticket *FeedbackTicket) error {
	return m.db.Create(ticket).Error
}

func (m *FeedbackTicketModel) FindById(id int64) (*FeedbackTicket, error) {
	var ticket FeedbackTicket
	err := m.db.First(&ticket, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &ticket, err
}

func (m *FeedbackTicketModel) FindListForAdmin(filter FeedbackTicketAdminFilter) ([]FeedbackTicket, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&FeedbackTicket{})
	if filter.Status > 0 {
		query = query.Where("status = ?", filter.Status)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query = query.Where("category = ?", NormalizeFeedbackTicketCategory(category))
	}
	if source := strings.TrimSpace(filter.Source); source != "" {
		query = query.Where("source = ?", NormalizeFeedbackTicketSource(source))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []FeedbackTicket
	err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *FeedbackTicketModel) Process(ticketId int64, status int, processResult string, handlerId int64, processedAt time.Time) error {
	updates := map[string]any{
		"status":         status,
		"handler_id":     handlerId,
		"process_result": strings.TrimSpace(processResult),
		"processed_at":   processedAt,
	}

	result := m.db.Model(&FeedbackTicket{}).
		Where("id = ?", ticketId).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
