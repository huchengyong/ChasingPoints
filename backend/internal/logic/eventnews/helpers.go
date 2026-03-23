package eventnews

import (
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const eventNewsTimeLayout = "2006-01-02 15:04:05"

var eventNewsNow = time.Now

func normalizeEventNewsPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func mapEventNewsInfo(item model.EventNews) types.EventNewsInfo {
	resp := types.EventNewsInfo{
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
		resp.StartTime = item.StartTime.Format(eventNewsTimeLayout)
	}
	if item.EndTime != nil {
		resp.EndTime = item.EndTime.Format(eventNewsTimeLayout)
	}
	if item.SortTime != nil {
		resp.SortTime = item.SortTime.Format(eventNewsTimeLayout)
	}
	if item.PublishedAt != nil {
		resp.PublishedAt = item.PublishedAt.Format(eventNewsTimeLayout)
	}
	resp.CreatedAt = item.CreatedAt.Format(eventNewsTimeLayout)
	resp.UpdatedAt = item.UpdatedAt.Format(eventNewsTimeLayout)
	return resp
}

func eventNewsEffectiveTime(item model.EventNews) time.Time {
	switch {
	case item.SortTime != nil && !item.SortTime.IsZero():
		return item.SortTime.UTC()
	case item.StartTime != nil && !item.StartTime.IsZero():
		return item.StartTime.UTC()
	default:
		return item.CreatedAt.UTC()
	}
}

func eventNewsStatusPriority(status int) int {
	switch status {
	case model.EventNewsStatusLive:
		return 0
	case model.EventNewsStatusUpcoming:
		return 1
	case model.EventNewsStatusFinished:
		return 2
	case model.EventNewsStatusCanceled:
		return 3
	default:
		return 4
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func eventNewsLess(now time.Time, left, right model.EventNews) bool {
	leftPriority := eventNewsStatusPriority(left.Status)
	rightPriority := eventNewsStatusPriority(right.Status)
	if leftPriority != rightPriority {
		return leftPriority < rightPriority
	}

	leftDelta := absDuration(now.Sub(eventNewsEffectiveTime(left)))
	rightDelta := absDuration(now.Sub(eventNewsEffectiveTime(right)))
	if leftDelta != rightDelta {
		return leftDelta < rightDelta
	}

	leftTime := eventNewsEffectiveTime(left)
	rightTime := eventNewsEffectiveTime(right)
	if !leftTime.Equal(rightTime) {
		return leftTime.After(rightTime)
	}

	return left.Id > right.Id
}

func sortEventNewsItems(items []model.EventNews, now time.Time) {
	sort.SliceStable(items, func(i, j int) bool {
		return eventNewsLess(now, items[i], items[j])
	})
}

func pickBestFeaturedEventNews(items []model.EventNews, now time.Time) *model.EventNews {
	if len(items) == 0 {
		return nil
	}

	candidates := make([]model.EventNews, 0, len(items))
	for _, item := range items {
		if item.Featured {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		candidates = append(candidates, items...)
	}

	sortEventNewsItems(candidates, now)
	best := candidates[0]
	return &best
}

func paginateEventNewsItems(items []model.EventNews, page, pageSize int) []model.EventNews {
	page, pageSize = normalizeEventNewsPage(page, pageSize)
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []model.EventNews{}
	}

	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	if start == end {
		return []model.EventNews{}
	}

	pageItems := make([]model.EventNews, 0, end-start)
	pageItems = append(pageItems, items[start:end]...)
	return pageItems
}
