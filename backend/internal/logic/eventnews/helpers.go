package eventnews

import (
	"sort"
	"strconv"
	"strings"
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

func mapEventNewsInfo(item model.EventNews, tournament *model.Tournament, matches []model.TournamentMatch) types.EventNewsInfo {
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
		Featured:   item.Featured,
		Published:  item.Published,
		MatchCount: len(matches),
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

	if tournament != nil {
		resp.TournamentId = tournament.Id
		resp.TournamentName = tournament.Name
		if resp.GameType == 0 {
			resp.GameType = tournament.GameType
		}
		if resp.City == "" {
			resp.City = tournament.City
		}
		if resp.Venue == "" {
			resp.Venue = tournament.VenueName
		}
		if resp.StartTime == "" && tournament.StartTime != nil {
			resp.StartTime = tournament.StartTime.Format(eventNewsTimeLayout)
		}
		if resp.EndTime == "" && tournament.EndTime != nil {
			resp.EndTime = tournament.EndTime.Format(eventNewsTimeLayout)
		}
	}
	if resp.TournamentName == "" {
		resp.TournamentName = item.Title
	}

	summaryMatch := pickSummaryMatch(matches)
	if summaryMatch != nil {
		resp.CurrentRoundText = summaryMatch.RoundName
		resp.LatestResultText = formatMatchSummary(*summaryMatch)
	}

	return resp
}

func mapEventNewsMatchInfo(eventID int64, item model.TournamentMatch) types.EventNewsMatchInfo {
	resp := types.EventNewsMatchInfo{
		Id:             item.Id,
		EventId:        eventID,
		TournamentId:   item.TournamentId,
		SourceType:     item.SourceType,
		SourceMatchId:  item.SourceMatchId,
		RoundName:      item.RoundName,
		RoundOrder:     item.RoundOrder,
		MatchOrder:     item.MatchOrder,
		Status:         item.Status,
		BestOf:         item.BestOf,
		HomePlayerId:   firstNonZeroInt64(item.HomePlayerId, item.Player1Id),
		HomePlayerName: firstNonEmpty(item.HomePlayerName),
		AwayPlayerId:   firstNonZeroInt64(item.AwayPlayerId, item.Player2Id),
		AwayPlayerName: firstNonEmpty(item.AwayPlayerName),
		HomeScore:      item.HomeScore,
		AwayScore:      item.AwayScore,
		WinnerSide:     normalizeWinnerSide(item),
		IsPlaceholder:  item.IsPlaceholder,
		CreatedAt:      item.CreatedAt.Format(eventNewsTimeLayout),
		UpdatedAt:      item.UpdatedAt.Format(eventNewsTimeLayout),
	}
	if item.StartTime != nil {
		resp.StartTime = item.StartTime.Format(eventNewsTimeLayout)
	}
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

func mapTournamentInfo(item *model.Tournament) *types.TournamentInfo {
	if item == nil {
		return nil
	}

	info := &types.TournamentInfo{
		Id:             item.Id,
		CreatorId:      item.CreatorId,
		Name:           item.Name,
		Description:    item.Description,
		GameType:       item.GameType,
		Format:         item.Format,
		MaxPlayers:     item.MaxPlayers,
		CurrentPlayers: item.CurrentPlayers,
		Status:         item.Status,
		City:           item.City,
		VenueName:      item.VenueName,
		CreatedAt:      item.CreatedAt.Format(eventNewsTimeLayout),
	}
	if item.StartTime != nil {
		info.StartTime = item.StartTime.Format(eventNewsTimeLayout)
	}
	if item.EndTime != nil {
		info.EndTime = item.EndTime.Format(eventNewsTimeLayout)
	}
	return info
}

func pickSummaryMatch(matches []model.TournamentMatch) *model.TournamentMatch {
	if len(matches) == 0 {
		return nil
	}

	candidates := filterMatchesByStatus(matches, model.EventNewsStatusLive)
	if len(candidates) == 0 {
		candidates = filterMatchesByStatus(matches, model.EventNewsStatusUpcoming)
	}
	if len(candidates) == 0 {
		candidates = filterMatchesByStatus(matches, model.EventNewsStatusFinished)
	}
	if len(candidates) == 0 {
		candidates = append(candidates, matches...)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Status == model.EventNewsStatusUpcoming || candidates[j].Status == model.EventNewsStatusUpcoming {
			return matchSortValue(candidates[i]) < matchSortValue(candidates[j])
		}
		return matchSortValue(candidates[i]) > matchSortValue(candidates[j])
	})

	chosen := candidates[0]
	return &chosen
}

func filterMatchesByStatus(matches []model.TournamentMatch, status int) []model.TournamentMatch {
	filtered := make([]model.TournamentMatch, 0, len(matches))
	for _, match := range matches {
		if match.Status == status {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func matchSortValue(item model.TournamentMatch) int64 {
	return int64(item.RoundOrder)*1_000_000 + int64(item.MatchOrder)*1_000 + int64(item.Id)
}

func formatMatchSummary(item model.TournamentMatch) string {
	home := displayPlayerName(item.HomePlayerName, item.Player1Id)
	away := displayPlayerName(item.AwayPlayerName, item.Player2Id)
	scoreText := "-"
	if item.Status == model.EventNewsStatusLive || item.Status == model.EventNewsStatusFinished {
		scoreText = strings.TrimSpace(
			strings.Join([]string{intToString(item.HomeScore), "-", intToString(item.AwayScore)}, " "),
		)
	}
	if home == "" && away == "" {
		return scoreText
	}
	return strings.TrimSpace(home + " " + scoreText + " " + away)
}

func displayPlayerName(name string, fallbackID int64) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if fallbackID > 0 {
		return "选手#" + intToString(int(fallbackID))
	}
	return "待定"
}

func normalizeWinnerSide(item model.TournamentMatch) int {
	if item.WinnerSide != 0 {
		return item.WinnerSide
	}
	switch {
	case item.WinnerId > 0 && item.WinnerId == firstNonZeroInt64(item.HomePlayerId, item.Player1Id):
		return 1
	case item.WinnerId > 0 && item.WinnerId == firstNonZeroInt64(item.AwayPlayerId, item.Player2Id):
		return 2
	default:
		return 0
	}
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
