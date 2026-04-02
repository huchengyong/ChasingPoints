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

func parseAdminEventNewsTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(adminEventNewsTimeLayout, trimmed)
	if err != nil {
		return nil, err
	}
	utcTime := parsed.UTC()
	return &utcTime, nil
}

func buildAdminEventNewsInfo(item model.EventNews, tournament *model.Tournament, matches []model.TournamentMatch) types.EventNewsInfo {
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
		Featured:   item.Featured,
		Published:  item.Published,
		MatchCount: len(matches),
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

	if tournament != nil {
		info.TournamentId = tournament.Id
		info.TournamentName = tournament.Name
		if info.GameType == 0 {
			info.GameType = tournament.GameType
		}
		if info.City == "" {
			info.City = tournament.City
		}
		if info.Venue == "" {
			info.Venue = tournament.VenueName
		}
		if info.StartTime == "" && tournament.StartTime != nil {
			info.StartTime = tournament.StartTime.Format(adminEventNewsTimeLayout)
		}
		if info.EndTime == "" && tournament.EndTime != nil {
			info.EndTime = tournament.EndTime.Format(adminEventNewsTimeLayout)
		}
	}
	if info.TournamentName == "" {
		info.TournamentName = item.Title
	}

	summaryMatch := pickAdminSummaryMatch(matches)
	if summaryMatch != nil {
		info.CurrentRoundText = summaryMatch.RoundName
		info.LatestResultText = formatAdminMatchSummary(*summaryMatch)
	}

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
	item.StartTime = startTime
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}
	item.SortTime = sortTime
	item.EndTime = endTime
	item.Featured = req.Featured
	return nil
}

func buildAdminTournament(req *types.AdminEventNewsCreateReq) (*model.Tournament, error) {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return nil, err
	}

	return &model.Tournament{
		CreatorId:      0,
		Name:           fallbackTournamentName(strings.TrimSpace(req.TournamentName), strings.TrimSpace(req.Title)),
		Description:    strings.TrimSpace(req.Description),
		GameType:       req.GameType,
		Format:         1,
		MaxPlayers:     128,
		CurrentPlayers: 0,
		Status:         req.Status,
		City:           strings.TrimSpace(req.City),
		VenueName:      strings.TrimSpace(req.Venue),
		StartTime:      startTime,
		EndTime:        endTime,
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

func validateAdminEventNewsTimeRange(startTime, endTime *time.Time) string {
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return "结束时间不能早于开始时间"
	}
	return ""
}

func applyAdminTournamentUpdate(item *model.Tournament, req *types.AdminEventNewsUpdateReq) error {
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
	item.GameType = req.GameType
	item.Status = req.Status
	item.City = strings.TrimSpace(req.City)
	item.VenueName = strings.TrimSpace(req.Venue)
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

func validateAdminEventNewsReq(title string, gameType, status int, startTime, sortTime *time.Time) string {
	if title == "" {
		return "请输入标题"
	}
	if gameType <= 0 {
		return "请选择球种"
	}
	if status < model.EventNewsStatusUpcoming || status > model.EventNewsStatusCanceled {
		return "请选择正确的状态"
	}
	if startTime == nil && sortTime == nil {
		return "开始时间和排序时间至少填写一个"
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
		info.StartTime = item.StartTime.Format(adminEventNewsTimeLayout)
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
