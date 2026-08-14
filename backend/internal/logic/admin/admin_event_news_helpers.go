package admin

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const adminEventNewsTimeLayout = "2006-01-02 15:04:05"
const adminEventNewsDateLayout = "2006-01-02"

var adminShanghaiLocation = time.FixedZone("UTC+8", 8*60*60)

func parseAdminEventNewsTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation(adminEventNewsTimeLayout, trimmed, adminShanghaiLocation)
	if err != nil {
		return nil, err
	}
	utcTime := parsed.UTC()
	return &utcTime, nil
}

func parseAdminEventNewsDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.ParseInLocation(adminEventNewsDateLayout, trimmed, adminShanghaiLocation)
	if err != nil {
		return nil, err
	}
	dateOnly := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	return &dateOnly, nil
}

func buildAdminEventNewsInfo(item model.EventNews, tournament *model.Tournament, matches []model.TournamentMatch) types.EventNewsInfo {
	info := types.EventNewsInfo{
		Id:          item.Id,
		Title:       item.Title,
		GameType:    item.GameType,
		SourceType:  item.SourceType,
		SourceName:  item.SourceName,
		SourceUrl:   item.SourceUrl,
		CoverImage:  strings.TrimSpace(item.CoverImage),
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
		info.StartDate = item.StartDate.Format(adminEventNewsDateLayout)
	}
	if item.EndDate != nil {
		info.EndDate = item.EndDate.Format(adminEventNewsDateLayout)
	}
	if item.StartTime != nil {
		info.StartTime = formatAdminDisplayTime(item.SourceType, item.StartTime)
	}
	if item.EndTime != nil {
		info.EndTime = formatAdminDisplayTime(item.SourceType, item.EndTime)
	}
	if item.SortTime != nil {
		info.SortTime = reinterpretAdminUTC(*item.SortTime).In(adminShanghaiLocation).Format(adminEventNewsTimeLayout)
	}
	if item.PublishedAt != nil {
		info.PublishedAt = item.PublishedAt.Format(adminEventNewsTimeLayout)
	}
	info.CreatedAt = item.CreatedAt.Format(adminEventNewsTimeLayout)
	info.UpdatedAt = item.UpdatedAt.Format(adminEventNewsTimeLayout)

	if tournament != nil {
		info.TournamentId = tournament.Id
		info.TournamentName = tournament.Name
		info.Description = strings.TrimSpace(tournament.Description)
		if info.CoverImage == "" {
			info.CoverImage = strings.TrimSpace(tournament.CoverImage)
		}
		if info.GameType == 0 {
			info.GameType = tournament.GameType
		}
		if info.Country == "" {
			info.Country = tournament.Country
		}
		if info.City == "" {
			info.City = tournament.City
		}
		if info.Venue == "" {
			info.Venue = tournament.VenueName
		}
		if info.StartDate == "" && tournament.StartDate != nil {
			info.StartDate = tournament.StartDate.Format(adminEventNewsDateLayout)
		}
		if info.EndDate == "" && tournament.EndDate != nil {
			info.EndDate = tournament.EndDate.Format(adminEventNewsDateLayout)
		}
		if info.StartTime == "" && tournament.StartTime != nil {
			info.StartTime = formatAdminDisplayTime(tournament.SourceType, tournament.StartTime)
		}
		if info.EndTime == "" && tournament.EndTime != nil {
			info.EndTime = formatAdminDisplayTime(tournament.SourceType, tournament.EndTime)
		}
	}
	if info.TournamentName == "" {
		info.TournamentName = item.Title
	}
	if info.EndDate == "" {
		info.EndDate = info.StartDate
	}

	summaryMatch := pickAdminSummaryMatch(matches)
	if summaryMatch != nil {
		info.CurrentRoundText = summaryMatch.RoundName
		info.LatestResultText = formatAdminMatchSummary(*summaryMatch)
	}

	return info
}

func applyAdminEventNewsUpdate(item *model.EventNews, req *types.AdminEventNewsUpdateReq) error {
	startDate, err := parseAdminEventNewsDate(req.StartDate)
	if err != nil {
		return err
	}
	endDate, err := parseAdminEventNewsDate(req.EndDate)
	if err != nil {
		return err
	}
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
	if req.TournamentId > 0 {
		item.TournamentId = req.TournamentId
	}
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
	item.StartDate = startDate
	item.EndDate = firstNonNilTime(endDate, startDate)
	item.StartTime = startTime
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}
	item.SortTime = sortTime
	item.EndTime = endTime
	return nil
}

func buildAdminTournament(req *types.AdminEventNewsCreateReq) (*model.Tournament, error) {
	startDate, err := parseAdminEventNewsDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseAdminEventNewsDate(req.EndDate)
	if err != nil {
		return nil, err
	}
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return nil, err
	}

	return &model.Tournament{
		CreatorId:       0,
		Name:            fallbackTournamentName(strings.TrimSpace(req.TournamentName), strings.TrimSpace(req.Title)),
		Description:     strings.TrimSpace(req.Description),
		CoverImage:      strings.TrimSpace(req.CoverImage),
		GameType:        req.GameType,
		Format:          1,
		MaxPlayers:      128,
		CurrentPlayers:  0,
		Status:          req.Status,
		Country:         strings.TrimSpace(req.Country),
		City:            strings.TrimSpace(req.City),
		VenueName:       strings.TrimSpace(req.Venue),
		SourceType:      strings.TrimSpace(req.SourceType),
		InformationPage: strings.TrimSpace(req.SourceUrl),
		StartDate:       startDate,
		EndDate:         firstNonNilTime(endDate, startDate),
		StartTime:       startTime,
		EndTime:         endTime,
	}, nil
}

func buildAdminEventNewsMatchFromCreate(req *types.AdminEventNewsMatchCreateReq, tournamentId int64) (*model.TournamentMatch, error) {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return nil, err
	}

	match := &model.TournamentMatch{
		TournamentId:   tournamentId,
		SourceType:     strings.TrimSpace(req.SourceType),
		SourceMatchId:  strings.TrimSpace(req.SourceMatchId),
		RoundName:      strings.TrimSpace(req.RoundName),
		RoundOrder:     req.RoundOrder,
		RoundNumber:    req.RoundOrder,
		MatchOrder:     req.MatchOrder,
		StartTime:      startTime,
		Status:         req.Status,
		BestOf:         req.BestOf,
		HomePlayerId:   req.HomePlayerId,
		HomePlayerName: strings.TrimSpace(req.HomePlayerName),
		AwayPlayerId:   req.AwayPlayerId,
		AwayPlayerName: strings.TrimSpace(req.AwayPlayerName),
		HomeScore:      req.HomeScore,
		AwayScore:      req.AwayScore,
		WinnerSide:     req.WinnerSide,
		IsPlaceholder:  req.IsPlaceholder,
		Player1Id:      req.HomePlayerId,
		Player2Id:      req.AwayPlayerId,
	}
	match.WinnerId = deriveWinnerID(match)
	return match, nil
}

func validateAdminEventNewsTimeRange(startDate, endDate, startTime, endTime *time.Time) string {
	if startDate == nil {
		return "请填写开始日期"
	}
	if endDate == nil {
		return "请填写结束日期"
	}
	if endDate.Before(*startDate) {
		return "结束日期不能早于开始日期"
	}
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return "结束时间不能早于开始时间"
	}
	return ""
}

func applyAdminTournamentUpdate(item *model.Tournament, req *types.AdminEventNewsUpdateReq) error {
	startDate, err := parseAdminEventNewsDate(req.StartDate)
	if err != nil {
		return err
	}
	endDate, err := parseAdminEventNewsDate(req.EndDate)
	if err != nil {
		return err
	}
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return err
	}

	item.Name = fallbackTournamentName(strings.TrimSpace(req.TournamentName), strings.TrimSpace(req.Title))
	item.Description = strings.TrimSpace(req.Description)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.GameType = req.GameType
	item.Status = req.Status
	item.Country = strings.TrimSpace(req.Country)
	item.City = strings.TrimSpace(req.City)
	item.VenueName = strings.TrimSpace(req.Venue)
	item.SourceType = strings.TrimSpace(req.SourceType)
	item.InformationPage = strings.TrimSpace(req.SourceUrl)
	item.StartDate = startDate
	item.EndDate = firstNonNilTime(endDate, startDate)
	item.StartTime = startTime
	item.EndTime = endTime
	return nil
}

func applyAdminEventNewsMatchUpdate(item *model.TournamentMatch, req *types.AdminEventNewsMatchUpdateReq) error {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return err
	}

	item.SourceType = strings.TrimSpace(req.SourceType)
	item.SourceMatchId = strings.TrimSpace(req.SourceMatchId)
	item.RoundName = strings.TrimSpace(req.RoundName)
	item.RoundOrder = req.RoundOrder
	item.RoundNumber = req.RoundOrder
	item.MatchOrder = req.MatchOrder
	item.StartTime = startTime
	item.Status = req.Status
	item.BestOf = req.BestOf
	item.HomePlayerId = req.HomePlayerId
	item.HomePlayerName = strings.TrimSpace(req.HomePlayerName)
	item.AwayPlayerId = req.AwayPlayerId
	item.AwayPlayerName = strings.TrimSpace(req.AwayPlayerName)
	item.HomeScore = req.HomeScore
	item.AwayScore = req.AwayScore
	item.WinnerSide = req.WinnerSide
	item.IsPlaceholder = req.IsPlaceholder
	item.Player1Id = req.HomePlayerId
	item.Player2Id = req.AwayPlayerId
	item.WinnerId = deriveWinnerID(item)
	return nil
}

func validateAdminEventNewsReq(title string, gameType, status int, startDate, endDate *time.Time) string {
	if title == "" {
		return "请输入标题"
	}
	if gameType <= 0 {
		return "请选择球种"
	}
	if status < model.EventNewsStatusUpcoming || status > model.EventNewsStatusCanceled {
		return "请选择正确的状态"
	}
	if startDate == nil {
		return "请填写开始日期"
	}
	if endDate == nil {
		return "请填写结束日期"
	}
	return ""
}

func validateAdminEventNewsMatchReq(roundName string, roundOrder, matchOrder, status int) string {
	if roundName == "" {
		return "请输入轮次名称"
	}
	if roundOrder <= 0 {
		return "请输入正确的轮次排序"
	}
	if matchOrder <= 0 {
		return "请输入正确的比赛排序"
	}
	if status < model.EventNewsStatusUpcoming || status > model.EventNewsStatusCanceled {
		return "请选择正确的比赛状态"
	}
	return ""
}

func pickAdminSummaryMatch(matches []model.TournamentMatch) *model.TournamentMatch {
	if len(matches) == 0 {
		return nil
	}

	candidates := filterAdminMatchesByStatus(matches, model.EventNewsStatusLive)
	if len(candidates) == 0 {
		candidates = filterAdminMatchesByStatus(matches, model.EventNewsStatusUpcoming)
	}
	if len(candidates) == 0 {
		candidates = filterAdminMatchesByStatus(matches, model.EventNewsStatusFinished)
	}
	if len(candidates) == 0 {
		candidates = append(candidates, matches...)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Status == model.EventNewsStatusUpcoming || candidates[j].Status == model.EventNewsStatusUpcoming {
			return adminMatchSortValue(candidates[i]) < adminMatchSortValue(candidates[j])
		}
		return adminMatchSortValue(candidates[i]) > adminMatchSortValue(candidates[j])
	})

	chosen := candidates[0]
	return &chosen
}

func filterAdminMatchesByStatus(matches []model.TournamentMatch, status int) []model.TournamentMatch {
	filtered := make([]model.TournamentMatch, 0, len(matches))
	for _, match := range matches {
		if match.Status == status {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func buildAdminEventNewsMatchInfo(eventID int64, item model.TournamentMatch) types.EventNewsMatchInfo {
	info := types.EventNewsMatchInfo{
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
		HomePlayerId:   item.HomePlayerId,
		HomePlayerName: item.HomePlayerName,
		AwayPlayerId:   item.AwayPlayerId,
		AwayPlayerName: item.AwayPlayerName,
		HomeScore:      item.HomeScore,
		AwayScore:      item.AwayScore,
		WinnerSide:     item.WinnerSide,
		IsPlaceholder:  item.IsPlaceholder,
		CreatedAt:      item.CreatedAt.Format(adminEventNewsTimeLayout),
		UpdatedAt:      item.UpdatedAt.Format(adminEventNewsTimeLayout),
	}
	if item.StartTime != nil {
		info.StartTime = formatAdminDisplayTime(item.SourceType, item.StartTime)
	}
	return info
}

func formatAdminMatchSummary(item model.TournamentMatch) string {
	return strings.TrimSpace(fallbackPlayerName(item.HomePlayerName) + " " + adminScoreText(item) + " " + fallbackPlayerName(item.AwayPlayerName))
}

func adminScoreText(item model.TournamentMatch) string {
	if item.Status == model.EventNewsStatusLive || item.Status == model.EventNewsStatusFinished {
		return strings.TrimSpace(intToAdminString(item.HomeScore) + " - " + intToAdminString(item.AwayScore))
	}
	return "-"
}

func fallbackPlayerName(name string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return "待定"
}

func adminMatchSortValue(item model.TournamentMatch) int64 {
	return int64(item.RoundOrder)*1_000_000 + int64(item.MatchOrder)*1_000 + int64(item.Id)
}

func intToAdminString(value int) string {
	return strconv.Itoa(value)
}

func deriveWinnerID(match *model.TournamentMatch) int64 {
	switch match.WinnerSide {
	case 1:
		return match.HomePlayerId
	case 2:
		return match.AwayPlayerId
	default:
		return 0
	}
}

func fallbackTournamentName(tournamentName, title string) string {
	if tournamentName != "" {
		return tournamentName
	}
	return title
}

func reinterpretAdminUTC(value time.Time) time.Time {
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

func normalizeAdminStoredShanghaiClock(value time.Time) time.Time {
	return time.Date(
		value.Year(),
		value.Month(),
		value.Day(),
		value.Hour(),
		value.Minute(),
		value.Second(),
		value.Nanosecond(),
		adminShanghaiLocation,
	)
}

func formatAdminDisplayTime(sourceType string, value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(sourceType), "official") {
		return normalizeAdminStoredShanghaiClock(*value).In(adminShanghaiLocation).Format(adminEventNewsTimeLayout)
	}
	return reinterpretAdminUTC(*value).In(adminShanghaiLocation).Format(adminEventNewsTimeLayout)
}

func firstNonNilTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil && !value.IsZero() {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
