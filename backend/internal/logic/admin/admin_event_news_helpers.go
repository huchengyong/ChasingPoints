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

func applyAdminEventNewsUpdate(item *model.EventNews, req *types.AdminEventNewsUpdateReq) error {
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

	item.Title = strings.TrimSpace(req.Title)
	item.GameType = req.GameType
	item.SourceType = strings.TrimSpace(req.SourceType)
	item.SourceName = strings.TrimSpace(req.SourceName)
	item.SourceUrl = strings.TrimSpace(req.SourceUrl)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.Summary = strings.TrimSpace(req.Summary)
	item.Content = req.Content
	item.Country = strings.TrimSpace(req.Country)
	item.City = strings.TrimSpace(req.City)
	item.Venue = strings.TrimSpace(req.Venue)
	item.Status = req.Status
	item.StartTime = startTime
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}
	item.SortTime = sortTime
	item.EndTime = endTime
	item.StageText = strings.TrimSpace(req.StageText)
	item.ResultText = strings.TrimSpace(req.ResultText)
	item.Featured = req.Featured
	return nil
}
