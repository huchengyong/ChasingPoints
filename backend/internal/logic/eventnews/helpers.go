package eventnews

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const eventNewsTimeLayout = "2006-01-02 15:04:05"
const eventNewsDateLayout = "2006-01-02"
const wstImageHost = "images.gc.wstservices.co.uk"

var eventNewsNow = time.Now
var shanghaiLocation = time.FixedZone("UTC+8", 8*60*60)

type eventNewsDateWindow struct {
	From string
	To   string
}

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

func buildEventNewsDateWindow(req *types.GetEventNewsListReq, now time.Time) (eventNewsDateWindow, error) {
	if req == nil {
		return eventNewsDateWindow{}, errors.New("请求参数错误")
	}

	fromText := strings.TrimSpace(req.From)
	toText := strings.TrimSpace(req.To)
	hasFrom := fromText != ""
	hasTo := toText != ""
	if hasFrom != hasTo {
		return eventNewsDateWindow{}, errors.New("请求参数错误")
	}

	if hasFrom {
		fromDate, err := parseEventNewsDate(fromText)
		if err != nil {
			return eventNewsDateWindow{}, err
		}
		toDate, err := parseEventNewsDate(toText)
		if err != nil {
			return eventNewsDateWindow{}, err
		}
		if toDate.Before(fromDate) {
			return eventNewsDateWindow{}, errors.New("请求参数错误")
		}

		return eventNewsDateWindow{
			From: fromDate.Format(eventNewsDateLayout),
			To:   toDate.Format(eventNewsDateLayout),
		}, nil
	}

	year := req.Year
	if year <= 0 {
		year = now.In(shanghaiLocation).Year()
	}
	if year <= 0 {
		return eventNewsDateWindow{}, errors.New("请求参数错误")
	}

	return eventNewsDateWindow{
		From: fmt.Sprintf("%04d-01-01", year),
		To:   fmt.Sprintf("%04d-12-31", year),
	}, nil
}

func parseEventNewsDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("请求参数错误")
	}

	parsed, err := time.Parse(eventNewsDateLayout, trimmed)
	if err != nil {
		return time.Time{}, errors.New("请求参数错误")
	}
	return parsed, nil
}

func filterEventNewsItemsByDateWindow(items []model.EventNews, tournamentMap map[int64]*model.Tournament, window eventNewsDateWindow) []model.EventNews {
	if window.From == "" && window.To == "" {
		return append([]model.EventNews{}, items...)
	}

	filtered := make([]model.EventNews, 0, len(items))
	for _, item := range items {
		tournament := tournamentMap[item.TournamentId]
		startDate, endDate, ok := resolveEventNewsDateRange(item, tournament)
		if !ok {
			continue
		}
		if eventNewsDateWindowOverlaps(window, startDate, endDate) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func resolveEventNewsDateRange(item model.EventNews, tournament *model.Tournament) (string, string, bool) {
	startDate := firstNonEmptyDateString(tournamentStartDate(tournament), item.StartDate)
	endDate := firstNonEmptyDateString(tournamentEndDate(tournament), item.EndDate)
	if startDate == "" && endDate == "" {
		return "", "", false
	}
	if startDate == "" {
		startDate = endDate
	}
	if endDate == "" {
		endDate = startDate
	}
	if startDate == "" || endDate == "" {
		return "", "", false
	}
	if endDate < startDate {
		return "", "", false
	}
	return startDate, endDate, true
}

func tournamentStartDate(item *model.Tournament) string {
	if item == nil || item.StartDate == nil {
		return ""
	}
	return item.StartDate.Format(eventNewsDateLayout)
}

func tournamentEndDate(item *model.Tournament) string {
	if item == nil || item.EndDate == nil {
		return ""
	}
	return item.EndDate.Format(eventNewsDateLayout)
}

func firstNonEmptyDateString(left string, right *time.Time) string {
	if strings.TrimSpace(left) != "" {
		return strings.TrimSpace(left)
	}
	if right == nil {
		return ""
	}
	return right.Format(eventNewsDateLayout)
}

func eventNewsDateWindowOverlaps(window eventNewsDateWindow, startDate, endDate string) bool {
	if window.From == "" || window.To == "" || startDate == "" || endDate == "" {
		return false
	}
	return !(endDate < window.From || startDate > window.To)
}

func mapEventNewsInfo(item model.EventNews, tournament *model.Tournament, matches []model.TournamentMatch) types.EventNewsInfo {
	resp := types.EventNewsInfo{
		Id:          item.Id,
		Title:       item.Title,
		GameType:    item.GameType,
		SourceType:  item.SourceType,
		SourceName:  item.SourceName,
		SourceUrl:   item.SourceUrl,
		CoverImage:  sanitizePublicImageURL(item.CoverImage),
		Summary:     item.Summary,
		Content:     item.Content,
		Description: "",
		Country:     item.Country,
		City:        item.City,
		Venue:       item.Venue,
		Status:      item.Status,
		Published:   item.Published,
		MatchCount:  len(matches),
	}
	if item.StartDate != nil {
		resp.StartDate = item.StartDate.Format(eventNewsDateLayout)
	}
	if item.EndDate != nil {
		resp.EndDate = item.EndDate.Format(eventNewsDateLayout)
	}
	if item.StartTime != nil {
		resp.StartTime = formatDisplayTimeBySource(item.SourceType, item.StartTime)
	}
	if item.EndTime != nil {
		resp.EndTime = formatDisplayTimeBySource(item.SourceType, item.EndTime)
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
		resp.Description = strings.TrimSpace(tournament.Description)
		if resp.CoverImage == "" {
			resp.CoverImage = sanitizePublicImageURL(tournament.CoverImage)
		}
		if resp.GameType == 0 {
			resp.GameType = tournament.GameType
		}
		if resp.Country == "" {
			resp.Country = tournament.Country
		}
		if resp.City == "" {
			resp.City = tournament.City
		}
		if resp.Venue == "" {
			resp.Venue = tournament.VenueName
		}
		if resp.StartDate == "" && tournament.StartDate != nil {
			resp.StartDate = tournament.StartDate.Format(eventNewsDateLayout)
		}
		if resp.EndDate == "" && tournament.EndDate != nil {
			resp.EndDate = tournament.EndDate.Format(eventNewsDateLayout)
		}
		if resp.StartTime == "" && tournament.StartTime != nil {
			resp.StartTime = formatDisplayTimeBySource(tournament.SourceType, tournament.StartTime)
		}
		if resp.EndTime == "" && tournament.EndTime != nil {
			resp.EndTime = formatDisplayTimeBySource(tournament.SourceType, tournament.EndTime)
		}
	}
	if resp.TournamentName == "" {
		resp.TournamentName = item.Title
	}
	if resp.EndDate == "" {
		resp.EndDate = resp.StartDate
	}

	summaryMatch := pickSummaryMatch(matches)
	if summaryMatch != nil {
		resp.CurrentRoundText = summaryMatch.RoundName
		resp.LatestResultText = formatMatchSummary(*summaryMatch)
	}

	return resp
}

func mapEventNewsMatchInfo(eventID int64, item model.TournamentMatch, players map[int64]model.Player) types.EventNewsMatchInfo {
	homePlayerID := firstNonZeroInt64(item.HomePlayerId, item.Player1Id)
	awayPlayerID := firstNonZeroInt64(item.AwayPlayerId, item.Player2Id)
	homePlayer := players[homePlayerID]
	awayPlayer := players[awayPlayerID]
	homePlayerName := firstNonEmpty(item.HomePlayerName, buildPlayerDisplayName(homePlayer))
	awayPlayerName := firstNonEmpty(item.AwayPlayerName, buildPlayerDisplayName(awayPlayer))
	homeFirstName, homeLastName := resolvePlayerNameParts(homePlayer, homePlayerName)
	awayFirstName, awayLastName := resolvePlayerNameParts(awayPlayer, awayPlayerName)
	resp := types.EventNewsMatchInfo{
		Id:                  item.Id,
		EventId:             eventID,
		TournamentId:        item.TournamentId,
		SourceType:          item.SourceType,
		SourceMatchId:       item.SourceMatchId,
		RoundName:           item.RoundName,
		RoundOrder:          item.RoundOrder,
		MatchOrder:          item.MatchOrder,
		Status:              item.Status,
		BestOf:              item.BestOf,
		HomePlayerId:        homePlayerID,
		HomePlayerName:      homePlayerName,
		HomePlayerFirstName: homeFirstName,
		HomePlayerLastName:  homeLastName,
		HomePlayerFlagEmoji: strings.TrimSpace(homePlayer.FlagEmoji),
		HomePlayerAvatar:    resolvePlayerAvatar(players, homePlayerID),
		AwayPlayerId:        awayPlayerID,
		AwayPlayerName:      awayPlayerName,
		AwayPlayerFirstName: awayFirstName,
		AwayPlayerLastName:  awayLastName,
		AwayPlayerFlagEmoji: strings.TrimSpace(awayPlayer.FlagEmoji),
		AwayPlayerAvatar:    resolvePlayerAvatar(players, awayPlayerID),
		HomeScore:           item.HomeScore,
		AwayScore:           item.AwayScore,
		WinnerSide:          normalizeWinnerSide(item),
		IsPlaceholder:       item.IsPlaceholder,
		CreatedAt:           item.CreatedAt.Format(eventNewsTimeLayout),
		UpdatedAt:           item.UpdatedAt.Format(eventNewsTimeLayout),
	}
	if item.StartTime != nil {
		resp.StartTime = formatDisplayTimeBySource(item.SourceType, item.StartTime)
	}
	return resp
}

func eventNewsEffectiveTime(item model.EventNews) time.Time {
	switch {
	case item.SortTime != nil && !item.SortTime.IsZero():
		return item.SortTime.UTC()
	case item.StartTime != nil && !item.StartTime.IsZero():
		return normalizeDisplayTimeBySource(item.SourceType, *item.StartTime)
	case item.StartDate != nil && !item.StartDate.IsZero():
		return item.StartDate.UTC()
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
		CoverImage:     sanitizePublicImageURL(item.CoverImage),
		GameType:       item.GameType,
		Format:         item.Format,
		MaxPlayers:     item.MaxPlayers,
		CurrentPlayers: item.CurrentPlayers,
		Status:         item.Status,
		Country:        item.Country,
		City:           item.City,
		VenueName:      item.VenueName,
		CreatedAt:      item.CreatedAt.Format(eventNewsTimeLayout),
	}
	if item.StartDate != nil {
		info.StartDate = item.StartDate.Format(eventNewsDateLayout)
	}
	if item.EndDate != nil {
		info.EndDate = item.EndDate.Format(eventNewsDateLayout)
	}
	if item.StartTime != nil {
		info.StartTime = formatDisplayTimeBySource(item.SourceType, item.StartTime)
	}
	if item.EndTime != nil {
		info.EndTime = formatDisplayTimeBySource(item.SourceType, item.EndTime)
	}
	if info.EndDate == "" {
		info.EndDate = info.StartDate
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

func collectMatchPlayerIDs(matches []model.TournamentMatch) []int64 {
	seen := make(map[int64]struct{})
	ids := make([]int64, 0, len(matches)*2)
	for _, match := range matches {
		for _, id := range []int64{firstNonZeroInt64(match.HomePlayerId, match.Player1Id), firstNonZeroInt64(match.AwayPlayerId, match.Player2Id)} {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func resolvePlayerAvatar(players map[int64]model.Player, playerID int64) string {
	if playerID <= 0 {
		return ""
	}
	player, ok := players[playerID]
	if !ok {
		return ""
	}
	return sanitizePublicImageURL(player.Avatar)
}

func sanitizePublicImageURL(rawURL string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && strings.EqualFold(parsed.Hostname(), wstImageHost) {
		return ""
	}
	return value
}

func buildPlayerDisplayName(player model.Player) string {
	return firstNonEmpty(
		player.DisplayName,
		strings.TrimSpace(strings.TrimSpace(player.FirstName)+" "+strings.TrimSpace(player.LastName)),
	)
}

func resolvePlayerNameParts(player model.Player, fallbackName string) (string, string) {
	firstName := strings.TrimSpace(player.FirstName)
	lastName := strings.TrimSpace(player.LastName)
	if firstName != "" || lastName != "" {
		return firstName, lastName
	}
	return splitPlayerName(fallbackName)
}

func splitPlayerName(name string) (string, string) {
	text := strings.TrimSpace(name)
	if text == "" || text == "待定" {
		return "", text
	}

	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return "", text
	}

	firstName := strings.Join(parts[:len(parts)-1], " ")
	lastName := parts[len(parts)-1]
	return firstName, lastName
}

func reinterpretStoredUTC(value time.Time) time.Time {
	return time.Date(
		value.Year(),
		value.Month(),
		value.Day(),
		value.Hour(),
		value.Minute(),
		value.Second(),
		value.Nanosecond(),
		time.UTC,
	)
}

func formatUTCDisplayTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return reinterpretStoredUTC(*value).In(shanghaiLocation).Format(time.RFC3339)
}

func normalizeStoredShanghaiClock(value time.Time) time.Time {
	return time.Date(
		value.Year(),
		value.Month(),
		value.Day(),
		value.Hour(),
		value.Minute(),
		value.Second(),
		value.Nanosecond(),
		shanghaiLocation,
	)
}

func normalizeDisplayTimeBySource(sourceType string, value time.Time) time.Time {
	if strings.EqualFold(strings.TrimSpace(sourceType), "official") {
		return normalizeStoredShanghaiClock(value)
	}
	return reinterpretStoredUTC(value)
}

func formatDisplayTimeBySource(sourceType string, value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return normalizeDisplayTimeBySource(sourceType, *value).In(shanghaiLocation).Format(time.RFC3339)
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
