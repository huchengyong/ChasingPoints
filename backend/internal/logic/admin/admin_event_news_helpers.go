package admin

import (
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const adminEventNewsTimeLayout = "2006-01-02 15:04:05"

func parseAdminEventNewsTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation(adminEventNewsTimeLayout, trimmed, time.Local)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func buildAdminEventNewsInfo(item model.EventNews) types.EventNewsInfo {
	info := types.EventNewsInfo{
		Id:         item.Id,
		Title:      item.Title,
		GameType:   item.GameType,
		SourceType: item.SourceType,
		SourceName: item.SourceName,
		SourceUrl:  item.SourceUrl,
		CoverImage: item.CoverImage,
		Summary:    item.Summary,
		Content:    item.Content,
		Country:    item.Country,
		City:       item.City,
		Venue:      item.Venue,
		Status:     item.Status,
		StageText:  item.StageText,
		ResultText: item.ResultText,
		Featured:   item.Featured,
		Published:  item.Published,
	}
	if item.StartTime != nil {
		info.StartTime = item.StartTime.Format(adminEventNewsTimeLayout)
	}
	if item.EndTime != nil {
		info.EndTime = item.EndTime.Format(adminEventNewsTimeLayout)
	}
	if item.SortTime != nil {
		info.SortTime = item.SortTime.Format(adminEventNewsTimeLayout)
	}
	if item.PublishedAt != nil {
		info.PublishedAt = item.PublishedAt.Format(adminEventNewsTimeLayout)
	}
	info.CreatedAt = item.CreatedAt.Format(adminEventNewsTimeLayout)
	info.UpdatedAt = item.UpdatedAt.Format(adminEventNewsTimeLayout)
	return info
}

func applyAdminEventNewsPatch(item *model.EventNews, req *types.AdminEventNewsUpdateReq) error {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return err
	}
	sortTime, err := parseAdminEventNewsTime(req.SortTime)
	if err != nil {
		return err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return err
	}

	if strings.TrimSpace(req.Title) != "" {
		item.Title = strings.TrimSpace(req.Title)
	}
	item.GameType = req.GameType
	if strings.TrimSpace(req.SourceType) != "" {
		item.SourceType = strings.TrimSpace(req.SourceType)
	}
	if strings.TrimSpace(req.SourceName) != "" {
		item.SourceName = strings.TrimSpace(req.SourceName)
	}
	if strings.TrimSpace(req.SourceUrl) != "" {
		item.SourceUrl = strings.TrimSpace(req.SourceUrl)
	}
	if strings.TrimSpace(req.CoverImage) != "" {
		item.CoverImage = strings.TrimSpace(req.CoverImage)
	}
	if strings.TrimSpace(req.Summary) != "" {
		item.Summary = strings.TrimSpace(req.Summary)
	}
	if strings.TrimSpace(req.Content) != "" {
		item.Content = req.Content
	}
	if strings.TrimSpace(req.Country) != "" {
		item.Country = strings.TrimSpace(req.Country)
	}
	if strings.TrimSpace(req.City) != "" {
		item.City = strings.TrimSpace(req.City)
	}
	if strings.TrimSpace(req.Venue) != "" {
		item.Venue = strings.TrimSpace(req.Venue)
	}
	item.Status = req.Status
	if startTime != nil {
		item.StartTime = startTime
	}
	if sortTime != nil {
		item.SortTime = sortTime
	}
	if endTime != nil {
		item.EndTime = endTime
	}
	if strings.TrimSpace(req.StageText) != "" {
		item.StageText = strings.TrimSpace(req.StageText)
	}
	if strings.TrimSpace(req.ResultText) != "" {
		item.ResultText = strings.TrimSpace(req.ResultText)
	}
	item.Featured = req.Featured
	return nil
}
