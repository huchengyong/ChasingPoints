package wstsync

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type DataClient interface {
	FetchSeasons(ctx context.Context) ([]SeasonResource, error)
	FetchTournamentsBySeason(ctx context.Context, season int) ([]TournamentResource, error)
	FetchMatchesPage(ctx context.Context, pageNumber, pageSize int) (MatchListResponse, error)
	FetchPageCoverImage(ctx context.Context, pageURL string) (string, error)
}

type SyncSummary struct {
	Mode                SyncMode
	DryRun              bool
	Publish             bool
	SeasonsFetched      int
	CandidateSeasons    int
	TournamentsFetched  int
	TournamentsSelected int
	MatchesScanned      int
	MatchesSelected     int
	PlayersPrepared     int
	TournamentsPrepared int
	MatchesPrepared     int
	EventNewsProjected  int
}

type Service struct {
	svcCtx *svc.ServiceContext
	client DataClient
	now    func() time.Time
}

func NewService(svcCtx *svc.ServiceContext, client DataClient) *Service {
	return &Service{
		svcCtx: svcCtx,
		client: client,
		now:    time.Now,
	}
}

func (s *Service) Sync(ctx context.Context, params SyncParams) (*SyncSummary, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, fmt.Errorf("wst sync service is not initialized")
	}
	if s.client == nil {
		return nil, fmt.Errorf("wst sync client is not initialized")
	}

	now := s.now().UTC()
	summary := &SyncSummary{
		Mode:    params.Mode,
		DryRun:  params.DryRun,
		Publish: params.Publish,
	}

	seasonIDs, seasonsFetched, err := s.resolveSeasonIDs(ctx, params)
	if err != nil {
		return nil, err
	}
	summary.SeasonsFetched = seasonsFetched
	summary.CandidateSeasons = len(seasonIDs)

	tournaments, err := s.fetchCandidateTournaments(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}
	summary.TournamentsFetched = len(tournaments)

	selected := selectTournamentsForParams(tournaments, params)
	summary.TournamentsSelected = len(selected)
	if len(selected) == 0 {
		return summary, nil
	}

	targetTournamentIDs := make(map[string]struct{}, len(selected))
	for _, item := range selected {
		targetTournamentIDs[item.ID] = struct{}{}
	}

	matches, scannedCount, err := s.scanMatchesForTournaments(ctx, targetTournamentIDs)
	if err != nil {
		return nil, err
	}
	summary.MatchesScanned = scannedCount
	summary.MatchesSelected = len(matches)

	playerRecords := buildPlayerUpsertRecords(matches)
	matchRecords := buildMatchUpsertRecords(matches)
	coverImages := s.resolveTournamentCoverImages(ctx, selected)
	tournamentRecords := buildTournamentUpsertRecords(selected, matchRecords, coverImages, params.GameType, now)
	summary.PlayersPrepared = len(playerRecords)
	summary.TournamentsPrepared = len(tournamentRecords)
	summary.MatchesPrepared = len(matchRecords)
	summary.EventNewsProjected = len(tournamentRecords)

	if params.DryRun {
		return summary, nil
	}

	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := UpsertPlayers(tx, now, playerRecords); err != nil {
			return err
		}
		if err := UpsertTournaments(tx, now, tournamentRecords); err != nil {
			return err
		}
		if err := UpsertMatches(tx, now, matchRecords); err != nil {
			return err
		}
		if err := deleteMissingOfficialMatches(tx, tournamentRecords, matchRecords); err != nil {
			return err
		}
		if err := s.projectEventNews(tx, tournamentRecords, params.Publish); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *Service) resolveSeasonIDs(ctx context.Context, params SyncParams) ([]int, int, error) {
	if params.Mode == SyncModeSeason {
		return []int{params.Season}, 0, nil
	}

	seasons, err := s.client.FetchSeasons(ctx)
	if err != nil {
		return nil, 0, err
	}

	window := syncWindowFromParams(params)
	ids := resolveSeasonIDsForWindow(seasons, window)
	if len(ids) == 0 {
		ids = extractAllSeasonIDs(seasons)
	}
	if len(ids) == 0 {
		return nil, len(seasons), fmt.Errorf("no WST seasons resolved for sync window")
	}
	return ids, len(seasons), nil
}

func (s *Service) fetchCandidateTournaments(ctx context.Context, seasonIDs []int) ([]TournamentResource, error) {
	seen := make(map[string]TournamentResource)
	for _, seasonID := range seasonIDs {
		items, err := s.client.FetchTournamentsBySeason(ctx, seasonID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			seen[item.ID] = item
		}
	}

	result := make([]TournamentResource, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := mustDateValue(result[i].Attributes.StartDate)
		right := mustDateValue(result[j].Attributes.StartDate)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *Service) scanMatchesForTournaments(ctx context.Context, tournamentIDs map[string]struct{}) ([]MatchResource, int, error) {
	if len(tournamentIDs) == 0 {
		return []MatchResource{}, 0, nil
	}

	pageNumber := 1
	scanned := 0
	seen := make(map[string]struct{})
	filtered := make([]MatchResource, 0)
	for {
		page, err := s.client.FetchMatchesPage(ctx, pageNumber, defaultMatchesPageSize)
		if err != nil {
			return nil, scanned, err
		}
		if len(page.Data) == 0 {
			break
		}

		scanned += len(page.Data)
		for _, item := range FilterMatchesByTournamentIDs(page.Data, tournamentIDs) {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			filtered = append(filtered, item)
		}
		pageNumber++
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		leftRound := resolveRoundOrder(filtered[i].Attributes.Round)
		rightRound := resolveRoundOrder(filtered[j].Attributes.Round)
		if filtered[i].Attributes.TournamentID != filtered[j].Attributes.TournamentID {
			return filtered[i].Attributes.TournamentID < filtered[j].Attributes.TournamentID
		}
		if leftRound != rightRound {
			return leftRound < rightRound
		}
		leftTime := mustDateTimeValue(filtered[i].Attributes.StartDateTime)
		rightTime := mustDateTimeValue(filtered[j].Attributes.StartDateTime)
		if !leftTime.Equal(rightTime) {
			return leftTime.Before(rightTime)
		}
		if filtered[i].Attributes.FixtureNumber != filtered[j].Attributes.FixtureNumber {
			return filtered[i].Attributes.FixtureNumber < filtered[j].Attributes.FixtureNumber
		}
		return filtered[i].ID < filtered[j].ID
	})

	return filtered, scanned, nil
}

func (s *Service) projectEventNews(db *gorm.DB, records []TournamentUpsertRecord, publish bool) error {
	projector := NewEventNewsProjector(model.NewEventNewsModel(db))

	sourceTournamentIDs := make([]string, 0, len(records))
	for _, item := range records {
		sourceTournamentID := strings.TrimSpace(item.SourceTournamentId)
		if sourceTournamentID == "" {
			continue
		}
		sourceTournamentIDs = append(sourceTournamentIDs, sourceTournamentID)
	}
	if len(sourceTournamentIDs) == 0 {
		return nil
	}

	tournamentMap, err := model.NewTournamentModel(db).FindBySourceTournamentIds(wstSourceType, sourceTournamentIDs)
	if err != nil {
		return err
	}

	tournaments := make([]model.Tournament, 0, len(tournamentMap))
	for _, sourceTournamentID := range sourceTournamentIDs {
		tournament, ok := tournamentMap[sourceTournamentID]
		if ok {
			tournaments = append(tournaments, tournament)
		}
	}
	if len(tournaments) == 0 {
		return nil
	}

	tournamentIDs := make([]int64, 0, len(tournaments))
	for _, item := range tournaments {
		tournamentIDs = append(tournamentIDs, item.Id)
	}

	matchMap, err := model.NewTournamentMatchModel(db).FindByTournamentIds(tournamentIDs)
	if err != nil {
		return err
	}

	for _, tournament := range tournaments {
		if _, err := projector.ProjectTournamentEventNews(tournament, matchMap[tournament.Id], publish); err != nil {
			return err
		}
	}
	return nil
}

func syncWindowFromParams(params SyncParams) DateWindow {
	if params.From == nil || params.To == nil {
		return DateWindow{}
	}
	return DateWindow{
		From: params.From.UTC(),
		To:   params.To.UTC(),
	}
}

func resolveSeasonIDsForWindow(items []SeasonResource, window DateWindow) []int {
	seen := make(map[int]struct{})
	ids := make([]int, 0, len(items))
	for _, item := range items {
		seasonID, err := strconv.Atoi(strings.TrimSpace(item.ID))
		if err != nil {
			continue
		}
		seasonWindow, ok := parseSeasonWindow(item.Attributes.Name)
		if !ok || !seasonWindow.Overlaps(window.From, window.To) {
			continue
		}
		if _, exists := seen[seasonID]; exists {
			continue
		}
		seen[seasonID] = struct{}{}
		ids = append(ids, seasonID)
	}
	sort.Ints(ids)
	return ids
}

func extractAllSeasonIDs(items []SeasonResource) []int {
	seen := make(map[int]struct{})
	ids := make([]int, 0, len(items))
	for _, item := range items {
		seasonID, err := strconv.Atoi(strings.TrimSpace(item.ID))
		if err != nil {
			continue
		}
		if _, ok := seen[seasonID]; ok {
			continue
		}
		seen[seasonID] = struct{}{}
		ids = append(ids, seasonID)
	}
	sort.Ints(ids)
	return ids
}

func parseSeasonWindow(name string) (DateWindow, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return DateWindow{}, false
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return DateWindow{}, false
	}

	startYear, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return DateWindow{}, false
	}
	endPart := strings.TrimSpace(parts[1])
	if len(endPart) == 2 {
		suffix, err := strconv.Atoi(endPart)
		if err != nil {
			return DateWindow{}, false
		}
		endYear := (startYear/100)*100 + suffix
		if endYear < startYear {
			endYear += 100
		}
		return DateWindow{
			From: time.Date(startYear, time.January, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(endYear, time.December, 31, 0, 0, 0, 0, time.UTC),
		}, true
	}

	endYear, err := strconv.Atoi(endPart)
	if err != nil {
		return DateWindow{}, false
	}
	return DateWindow{
		From: time.Date(startYear, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(endYear, time.December, 31, 0, 0, 0, 0, time.UTC),
	}, true
}

func selectTournamentsForParams(items []TournamentResource, params SyncParams) []TournamentResource {
	selected := make([]TournamentResource, 0, len(items))
	switch params.Mode {
	case SyncModeSeason:
		for _, item := range items {
			if !params.IncludeQualifiers && IsQualifierTournament(item.Attributes.Name) {
				continue
			}
			selected = append(selected, item)
		}
	default:
		selected = SelectTournamentsForWindow(items, syncWindowFromParams(params), params.IncludeQualifiers)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		left := mustDateValue(selected[i].Attributes.StartDate)
		right := mustDateValue(selected[j].Attributes.StartDate)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return selected[i].ID < selected[j].ID
	})
	return selected
}

func buildPlayerUpsertRecords(matches []MatchResource) []PlayerUpsertRecord {
	seen := make(map[string]PlayerUpsertRecord)
	for _, match := range matches {
		for _, player := range []WstPlayer{match.Attributes.HomePlayer, match.Attributes.AwayPlayer} {
			sourcePlayerID := strings.TrimSpace(player.PlayerID)
			if sourcePlayerID == "" {
				continue
			}
			seen[sourcePlayerID] = PlayerUpsertRecord{
				SourceType:     wstSourceType,
				SourcePlayerId: sourcePlayerID,
				FirstName:      strings.TrimSpace(player.FirstName),
				LastName:       strings.TrimSpace(player.Surname),
				DisplayName:    buildPlayerDisplayName(strings.TrimSpace(player.FirstName), strings.TrimSpace(player.Surname)),
				Avatar:         resolvePlayerAvatarURL(player.Media),
				CountryCode:    strings.TrimSpace(player.CountryCode),
				FlagEmoji:      buildFlagEmoji(player.CountryCode),
			}
		}
	}

	result := make([]PlayerUpsertRecord, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SourcePlayerId < result[j].SourcePlayerId
	})
	return result
}

func buildTournamentUpsertRecords(
	tournaments []TournamentResource,
	matches []MatchUpsertRecord,
	coverImages map[string]string,
	gameType int,
	now time.Time,
) []TournamentUpsertRecord {
	matchesByTournament := make(map[string][]MatchUpsertRecord)
	for _, item := range matches {
		matchesByTournament[item.SourceTournamentId] = append(matchesByTournament[item.SourceTournamentId], item)
	}

	result := make([]TournamentUpsertRecord, 0, len(tournaments))
	for _, item := range tournaments {
		startDate, _ := parseDateOnly(item.Attributes.StartDate)
		endDate, _ := parseDateOnly(item.Attributes.EndDate)
		coverImage := strings.TrimSpace(coverImages[item.ID])
		if coverImage == "" {
			coverImage = defaultTournamentCoverImage
		}
		result = append(result, TournamentUpsertRecord{
			SourceType:         wstSourceType,
			SourceTournamentId: strings.TrimSpace(item.ID),
			SourceSeasonId:     strings.TrimSpace(item.Attributes.Season.ID),
			Name:               strings.TrimSpace(item.Attributes.Name),
			CoverImage:         coverImage,
			GameType:           gameType,
			Status:             resolveTournamentStatus(item, matchesByTournament[item.ID], now),
			Country:            strings.TrimSpace(item.Attributes.Country),
			City:               strings.TrimSpace(item.Attributes.City),
			VenueName:          "",
			StartDate:          startDate,
			EndDate:            endDate,
			InformationPage:    strings.TrimSpace(item.Attributes.InformationPage),
			TicketingLink:      strings.TrimSpace(item.Attributes.TicketingLink),
		})
	}
	return result
}

func (s *Service) resolveTournamentCoverImages(ctx context.Context, tournaments []TournamentResource) map[string]string {
	result := make(map[string]string, len(tournaments))
	if s == nil || s.client == nil {
		return result
	}

	for _, item := range tournaments {
		if sourceTournamentID := strings.TrimSpace(item.ID); sourceTournamentID != "" {
			result[sourceTournamentID] = s.resolveTournamentCoverImage(ctx, item)
		}
	}
	return result
}

func (s *Service) resolveTournamentCoverImage(ctx context.Context, item TournamentResource) string {
	pageCandidates := []string{
		normalizeWSTPageURL(item.Attributes.InformationPage),
		normalizeWSTPageURL(item.Attributes.TicketingLink),
	}

	fallbackCover := ""
	for _, pageURL := range pageCandidates {
		if pageURL == "" {
			continue
		}

		coverImage, err := s.client.FetchPageCoverImage(ctx, pageURL)
		if err != nil {
			continue
		}

		coverImage = strings.TrimSpace(coverImage)
		if coverImage == "" {
			continue
		}
		if fallbackCover == "" {
			fallbackCover = coverImage
		}
		if coverImage != defaultTournamentCoverImage {
			return coverImage
		}
	}

	return fallbackCover
}

func normalizeWSTPageURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.IsAbs() {
		return parsed.String()
	}

	base, err := url.Parse(wstSiteBaseURL)
	if err != nil {
		return trimmed
	}
	ref, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	return base.ResolveReference(ref).String()
}

func buildMatchUpsertRecords(matches []MatchResource) []MatchUpsertRecord {
	perRoundCounter := make(map[string]int)
	result := make([]MatchUpsertRecord, 0, len(matches))
	for _, item := range matches {
		startTime, _ := parseWSTDateTime(item.Attributes.StartDateTime)
		roundOrder := resolveRoundOrder(item.Attributes.Round)
		groupKey := item.Attributes.TournamentID + "|" + strings.TrimSpace(item.Attributes.Round)
		matchOrder := item.Attributes.FixtureNumber
		if matchOrder <= 0 {
			perRoundCounter[groupKey]++
			matchOrder = perRoundCounter[groupKey]
		}

		result = append(result, MatchUpsertRecord{
			SourceType:         wstSourceType,
			SourceMatchId:      strings.TrimSpace(item.ID),
			SourceTournamentId: strings.TrimSpace(item.Attributes.TournamentID),
			RoundName:          strings.TrimSpace(item.Attributes.Round),
			RoundOrder:         roundOrder,
			MatchOrder:         matchOrder,
			StartTime:          startTime,
			BestOf:             item.Attributes.NumberOfFrames,
			HomePlayerSourceId: strings.TrimSpace(item.Attributes.HomePlayerID),
			HomePlayerName:     firstNonEmptyValue(buildPlayerDisplayName(item.Attributes.HomePlayer.FirstName, item.Attributes.HomePlayer.Surname), item.Attributes.HomePlayerID),
			AwayPlayerSourceId: strings.TrimSpace(item.Attributes.AwayPlayerID),
			AwayPlayerName:     firstNonEmptyValue(buildPlayerDisplayName(item.Attributes.AwayPlayer.FirstName, item.Attributes.AwayPlayer.Surname), item.Attributes.AwayPlayerID),
			HomeScore:          item.Attributes.HomePlayerScore,
			AwayScore:          item.Attributes.AwayPlayerScore,
			SourceStatus:       strings.TrimSpace(item.Attributes.Status),
			PlayersAllocated:   item.Attributes.PlayersAllocated,
		})
	}
	return result
}

func resolveTournamentStatus(item TournamentResource, matches []MatchUpsertRecord, now time.Time) int {
	var (
		hasLive      bool
		hasUpcoming  bool
		hasFinished  bool
		hasCanceled  bool
		matchCount   = len(matches)
		startDate, _ = parseDateOnly(item.Attributes.StartDate)
		endDate, _   = parseDateOnly(item.Attributes.EndDate)
	)

	for _, match := range matches {
		switch resolveOfficialMatchStatus(match, now, isOfficialMatchPlaceholder(match)) {
		case model.EventNewsStatusLive:
			hasLive = true
		case model.EventNewsStatusUpcoming:
			hasUpcoming = true
		case model.EventNewsStatusFinished:
			hasFinished = true
		case model.EventNewsStatusCanceled:
			hasCanceled = true
		}
	}

	if hasTournamentEnded(endDate, now) {
		if hasCanceled && !hasFinished && !hasLive {
			return model.EventNewsStatusCanceled
		}
		return model.EventNewsStatusFinished
	}

	if hasLive {
		return model.EventNewsStatusLive
	}
	if hasFinished && hasUpcoming {
		if hasTournamentEnded(endDate, now) {
			return model.EventNewsStatusFinished
		}
		return model.EventNewsStatusLive
	}
	if hasUpcoming {
		return model.EventNewsStatusUpcoming
	}
	if hasFinished {
		return model.EventNewsStatusFinished
	}
	if matchCount > 0 && hasCanceled {
		return model.EventNewsStatusCanceled
	}

	switch {
	case startDate != nil && now.Before(startDate.UTC()):
		return model.EventNewsStatusUpcoming
	case hasTournamentEnded(endDate, now):
		return model.EventNewsStatusFinished
	case startDate != nil:
		return model.EventNewsStatusLive
	default:
		return model.EventNewsStatusUpcoming
	}
}

func deleteMissingOfficialMatches(db *gorm.DB, tournaments []TournamentUpsertRecord, matches []MatchUpsertRecord) error {
	sourceTournamentIDs := make([]string, 0, len(tournaments))
	for _, item := range tournaments {
		sourceTournamentID := strings.TrimSpace(item.SourceTournamentId)
		if sourceTournamentID == "" {
			continue
		}
		sourceTournamentIDs = append(sourceTournamentIDs, sourceTournamentID)
	}
	if len(sourceTournamentIDs) == 0 {
		return nil
	}

	tournamentMap, err := model.NewTournamentModel(db).FindBySourceTournamentIds(wstSourceType, sourceTournamentIDs)
	if err != nil {
		return err
	}

	tournamentIDs := make([]int64, 0, len(tournamentMap))
	for _, sourceTournamentID := range sourceTournamentIDs {
		tournament, ok := tournamentMap[sourceTournamentID]
		if ok {
			tournamentIDs = append(tournamentIDs, tournament.Id)
		}
	}
	if len(tournamentIDs) == 0 {
		return nil
	}

	currentMatchSourceIDs := collectMatchSourceIDs(matches)
	query := db.Where("source_type = ? AND tournament_id IN ?", wstSourceType, tournamentIDs)
	if len(currentMatchSourceIDs) > 0 {
		query = query.Where("source_match_id NOT IN ?", currentMatchSourceIDs)
	}
	return query.Delete(&model.TournamentMatch{}).Error
}

func hasTournamentEnded(endDate *time.Time, now time.Time) bool {
	if endDate == nil {
		return false
	}
	return now.After(endDate.UTC().Add(24 * time.Hour))
}

func resolveRoundOrder(roundName string) int {
	normalized := normalizeOfficialText(roundName)
	switch normalized {
	case "final":
		return 70
	case "semi finals", "semi final", "semi-finals", "semi-final":
		return 60
	case "quarter finals", "quarter final", "quarter-finals", "quarter-final", "last 8":
		return 50
	case "last 16":
		return 40
	case "last 32":
		return 30
	case "last 64":
		return 20
	case "last 128":
		return 10
	}

	if strings.HasPrefix(normalized, "round ") {
		numberText := strings.TrimSpace(strings.TrimPrefix(normalized, "round "))
		if value, err := strconv.Atoi(numberText); err == nil {
			return value * 10
		}
	}
	return 0
}

func parseWSTDateTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, trimmed, time.UTC)
		if err == nil {
			normalized := parsed.UTC()
			return &normalized, nil
		}
	}
	return nil, fmt.Errorf("invalid WST datetime: %s", value)
}

func mustDateValue(value string) time.Time {
	parsed, err := parseDateOnly(value)
	if err != nil || parsed == nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func mustDateTimeValue(value string) time.Time {
	parsed, err := parseWSTDateTime(value)
	if err != nil || parsed == nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func buildPlayerDisplayName(firstName, lastName string) string {
	return strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
}

func resolvePlayerAvatarURL(media WstPlayerMedia) string {
	for _, candidate := range []string{media.AppProfile, media.Profile} {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			return trimmed
		}
		return "https://images.gc.wstservices.co.uk/fit-in/400x600/" + strings.TrimLeft(trimmed, "/")
	}
	return ""
}

func buildFlagEmoji(countryCode string) string {
	normalized := strings.ToLower(strings.TrimSpace(countryCode))
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "gb-") {
		return "🏴"
	}

	base := normalized
	if index := strings.Index(base, "-"); index > 0 {
		base = base[:index]
	}
	if len(base) != 2 {
		return ""
	}

	runes := []rune(strings.ToUpper(base))
	return string([]rune{regionalIndicator(runes[0]), regionalIndicator(runes[1])})
}

func regionalIndicator(letter rune) rune {
	return 0x1F1A5 + letter
}

func firstNonEmptyValue(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}
