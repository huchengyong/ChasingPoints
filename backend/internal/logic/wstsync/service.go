package wstsync

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/logic/eventnews"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DataClient interface {
	FetchSeasons(ctx context.Context) ([]SeasonResource, error)
	FetchTournamentsBySeason(ctx context.Context, season int) ([]TournamentResource, error)
	FetchMatchesPage(ctx context.Context, pageNumber, pageSize int) (MatchListResponse, error)
	FetchPageCoverImage(ctx context.Context, pageURL string) (string, error)
}

type SyncSummary struct {
	Mode                 SyncMode
	DryRun               bool
	Publish              bool
	SeasonsFetched       int
	CandidateSeasons     int
	TournamentsFetched   int
	TournamentsSelected  int
	MatchesScanned       int
	MatchPages           int
	MatchesSelected      int
	PlayersPrepared      int
	TournamentsPrepared  int
	MatchesPrepared      int
	MatchesSkipped       int
	MatchesRetained      int
	EventNewsProjected   int
	PublishAffected      int
	PlayersCommitted     int
	TournamentsCommitted int
	MatchesCommitted     int
	EventNewsCommitted   int
	CommittedCountsKnown bool
	Status               string
	ExitCode             int
	From                 *time.Time
	To                   *time.Time
	Elapsed              time.Duration
	CommitState          string
	Warnings             []string
	Years                []BackfillYearSummary
	CoverageStart        *time.Time
	CoverageEnd          *time.Time
}

type Service struct {
	svcCtx      *svc.ServiceContext
	client      DataClient
	imageMirror WSTImageMirror
	now         func() time.Time
}

func NewService(svcCtx *svc.ServiceContext, client DataClient) *Service {
	service := &Service{
		svcCtx: svcCtx,
		client: client,
		now:    time.Now,
	}
	if svcCtx != nil && svcCtx.QiniuUploadService != nil {
		service.imageMirror = svcCtx.QiniuUploadService
	}
	return service
}

func (s *Service) Sync(ctx context.Context, params SyncParams) (*SyncSummary, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, fmt.Errorf("wst sync service is not initialized")
	}
	if s.client == nil {
		return nil, fmt.Errorf("wst sync client is not initialized")
	}

	startTime := time.Now()
	now := s.now().UTC()
	summary := &SyncSummary{
		Mode:                 params.Mode,
		DryRun:               params.DryRun,
		Publish:              params.Publish,
		From:                 params.From,
		To:                   params.To,
		CommitState:          "not_started",
		CommittedCountsKnown: true,
	}
	defer func() { summary.Elapsed = time.Since(startTime) }()

	seasonIDs, seasonsFetched, seasonWarnings, err := s.resolveSeasonIDs(ctx, params)
	if err != nil {
		summary.Status = BackfillStatusFailed
		summary.ExitCode = ExitCodeFailed
		return summary, err
	}
	summary.SeasonsFetched = seasonsFetched
	summary.CandidateSeasons = len(seasonIDs)
	summary.Warnings = append(summary.Warnings, seasonWarnings...)

	tournaments, err := s.fetchCandidateTournaments(ctx, seasonIDs)
	if err != nil {
		summary.Status = BackfillStatusFailed
		summary.ExitCode = ExitCodeFailed
		return summary, err
	}
	summary.TournamentsFetched = len(tournaments)
	if params.Backfill {
		s.addTournamentDateWarnings(summary, tournaments)
	}

	selected := selectTournamentsForParams(tournaments, params)
	summary.TournamentsSelected = len(selected)
	setBackfillCoverage(summary, selected)
	if len(selected) == 0 {
		summary.Status = BackfillStatusNeedsReview
		summary.ExitCode = ExitCodeNeedsReview
		return summary, nil
	}

	targetTournamentIDs := make(map[string]struct{}, len(selected))
	for _, item := range selected {
		targetTournamentIDs[item.ID] = struct{}{}
	}

	matches, scannedCount, matchPages, err := s.scanMatchesForTournaments(ctx, targetTournamentIDs)
	if err != nil {
		summary.MatchesScanned = scannedCount
		summary.MatchPages = matchPages
		summary.Status = BackfillStatusFailed
		summary.ExitCode = ExitCodeFailed
		return summary, err
	}
	summary.MatchesScanned = scannedCount
	summary.MatchPages = matchPages
	summary.MatchesSelected = len(matches)
	if params.Backfill {
		summary.Years = buildBackfillYearSummaries(params, selected, matches)
		for _, item := range summary.Years {
			if item.Tournaments == 0 {
				summary.Warnings = append(summary.Warnings, fmt.Sprintf("year %d has no selected tournaments", item.Year))
			}
		}
	}

	playerRecords := buildPlayerUpsertRecords(matches)
	matchRecords := buildMatchUpsertRecords(matches)

	// Backfill only writes records with confirmed scores and an explicit status.
	if params.Backfill {
		confirmed := make([]MatchUpsertRecord, 0, len(matchRecords))
		for _, rec := range matchRecords {
			rec.PreserveSourceStatus = true
			if rec.ScoresConfirmed && isKnownOfficialMatchStatus(rec.SourceStatus) && (rec.PlayersAllocated || rec.PlayersAllocatedPresent) {
				confirmed = append(confirmed, rec)
			} else {
				summary.MatchesSkipped++
				summary.Warnings = append(summary.Warnings, fmt.Sprintf("skipped match %s (missing scores, status, or player allocation)", rec.SourceMatchId))
			}
		}
		matchRecords = confirmed
		summary.MatchesPrepared = len(matchRecords)
	} else {
		summary.MatchesPrepared = len(matchRecords)
	}
	if params.Backfill {
		s.addNoMatchWarnings(summary, selected, matches)
	}

	coverImages := s.resolveTournamentCoverImages(ctx, selected)
	tournamentRecords := buildTournamentUpsertRecords(selected, matchRecords, coverImages, params.GameType, now)
	summary.PlayersPrepared = len(playerRecords)
	summary.TournamentsPrepared = len(tournamentRecords)
	if !params.Backfill {
		summary.MatchesPrepared = len(matchRecords)
	}
	summary.EventNewsProjected = len(tournamentRecords)

	// Backfill pre-check: detect soft-deleted records and duplicate IDs.
	if params.Backfill {
		if err := s.preCheckBackfill(ctx, tournamentRecords, playerRecords, matchRecords); err != nil {
			summary.Status = BackfillStatusFailed
			summary.ExitCode = ExitCodeFailed
			summary.Warnings = append(summary.Warnings, err.Error())
			return summary, nil
		}
		if err := s.loadBackfillImpact(ctx, tournamentRecords, matchRecords, summary); err != nil {
			summary.Status = BackfillStatusFailed
			summary.ExitCode = ExitCodeFailed
			return summary, err
		}
	}
	if params.DryRun {
		summary.Status = s.finalizeStatus(summary)
		summary.ExitCode = exitCodeForStatus(summary.Status)
		return summary, nil
	}

	if err := s.mirrorPreparedImages(ctx, playerRecords, tournamentRecords); err != nil {
		summary.Status = BackfillStatusFailed
		summary.ExitCode = ExitCodeFailed
		return summary, err
	}

	transactionBodyComplete := false
	if err := s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := UpsertPlayers(tx, now, playerRecords); err != nil {
			return err
		}
		if err := UpsertTournaments(tx, now, tournamentRecords); err != nil {
			return err
		}
		if err := UpsertMatches(tx, now, matchRecords); err != nil {
			return err
		}
		// In backfill mode, skip deletion of missing matches.
		if !params.Backfill {
			if err := deleteMissingOfficialMatches(tx, tournamentRecords, matchRecords); err != nil {
				return err
			}
		}
		if err := s.projectEventNews(tx, tournamentRecords, params.Publish); err != nil {
			return err
		}
		transactionBodyComplete = true
		return nil
	}); err != nil {
		summary.Status = BackfillStatusFailed
		summary.ExitCode = ExitCodeFailed
		if transactionBodyComplete {
			summary.CommitState = "unknown"
			summary.CommittedCountsKnown = false
		} else {
			summary.CommitState = "rolled_back"
		}
		return summary, err
	}
	summary.CommitState = "committed"
	summary.PlayersCommitted = len(playerRecords)
	summary.TournamentsCommitted = len(tournamentRecords)
	summary.MatchesCommitted = len(matchRecords)
	summary.EventNewsCommitted = len(tournamentRecords)

	if err := eventnews.BumpEventNewsCacheVersion(ctx, s.svcCtx); err != nil {
		logx.WithContext(ctx).Errorf("WST同步后更新赛讯缓存版本失败: err=%v", err)
		summary.Status = BackfillStatusNeedsReview
		summary.ExitCode = ExitCodeNeedsReview
		summary.Warnings = append(summary.Warnings, "cache version bump failed after commit")
		return summary, nil
	}

	summary.Status = s.finalizeStatus(summary)
	summary.ExitCode = exitCodeForStatus(summary.Status)
	return summary, nil
}

func buildBackfillYearSummaries(params SyncParams, tournaments []TournamentResource, matches []MatchResource) []BackfillYearSummary {
	if params.From == nil || params.To == nil {
		return nil
	}
	years := make(map[int]*BackfillYearSummary)
	for year := params.From.Year(); year <= params.To.Year(); year++ {
		years[year] = &BackfillYearSummary{Year: year}
	}
	tournamentYears := make(map[string]int, len(tournaments))
	for _, item := range tournaments {
		start, err := parseDateOnly(item.Attributes.StartDate)
		if err != nil || start == nil {
			continue
		}
		year := start.Year()
		tournamentYears[item.ID] = year
		if item := years[year]; item != nil {
			item.Tournaments++
		}
	}
	for _, item := range matches {
		if year := tournamentYears[item.Attributes.TournamentID]; years[year] != nil {
			years[year].Matches++
		}
	}
	result := make([]BackfillYearSummary, 0, len(years))
	for year := params.From.Year(); year <= params.To.Year(); year++ {
		result = append(result, *years[year])
	}
	return result
}

func setBackfillCoverage(summary *SyncSummary, tournaments []TournamentResource) {
	if summary.Mode != SyncModeBackfill {
		return
	}
	for _, item := range tournaments {
		start, end, err := tournamentDateRange(item)
		if err != nil {
			continue
		}
		if summary.CoverageStart == nil || start.Before(*summary.CoverageStart) {
			value := start
			summary.CoverageStart = &value
		}
		if summary.CoverageEnd == nil || end.After(*summary.CoverageEnd) {
			value := end
			summary.CoverageEnd = &value
		}
	}
}

func (s *Service) loadBackfillImpact(ctx context.Context, tournaments []TournamentUpsertRecord, matches []MatchUpsertRecord, summary *SyncSummary) error {
	sourceIDs := make([]string, 0, len(tournaments))
	for _, item := range tournaments {
		sourceIDs = append(sourceIDs, item.SourceTournamentId)
	}
	existing, err := model.NewTournamentModel(s.svcCtx.DB.WithContext(ctx)).FindBySourceTournamentIds(wstSourceType, sourceIDs)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(existing))
	for _, item := range existing {
		ids = append(ids, item.Id)
	}
	if len(ids) == 0 {
		return nil
	}
	current, err := model.NewTournamentMatchModel(s.svcCtx.DB.WithContext(ctx)).FindByTournamentIds(ids)
	if err != nil {
		return err
	}
	prepared := make(map[string]struct{}, len(matches))
	for _, item := range matches {
		prepared[item.SourceMatchId] = struct{}{}
	}
	for _, list := range current {
		for _, item := range list {
			if item.SourceType == wstSourceType {
				if _, ok := prepared[item.SourceMatchId]; !ok {
					summary.MatchesRetained++
				}
			}
		}
	}
	var publishAffected int64
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.EventNews{}).
		Where("tournament_id IN ? AND source_type = ? AND published <> ?", ids, wstSourceType, summary.Publish).
		Count(&publishAffected).Error; err != nil {
		return err
	}
	summary.PublishAffected = int(publishAffected)
	if summary.MatchesRetained > 0 {
		summary.Warnings = append(summary.Warnings, fmt.Sprintf("retained %d existing official matches", summary.MatchesRetained))
	}
	return nil
}

// preCheckBackfill checks for soft-deleted records and duplicate active source IDs.
// It returns an error if any conflicts are found.
func (s *Service) preCheckBackfill(ctx context.Context, tournaments []TournamentUpsertRecord, players []PlayerUpsertRecord, matches []MatchUpsertRecord) error {
	db := s.svcCtx.DB.WithContext(ctx)
	tournamentSourceIDs := make(map[string]struct{}, len(tournaments))
	for _, record := range tournaments {
		sourceID := strings.TrimSpace(record.SourceTournamentId)
		if sourceID == "" {
			return fmt.Errorf("pre-check tournament has empty source id")
		}
		tournamentSourceIDs[sourceID] = struct{}{}
	}
	playerSourceIDs := make(map[string]struct{}, len(players))
	for _, record := range players {
		sourceID := strings.TrimSpace(record.SourcePlayerId)
		if sourceID == "" {
			return fmt.Errorf("pre-check player has empty source id")
		}
		playerSourceIDs[sourceID] = struct{}{}
	}
	for _, record := range matches {
		if strings.TrimSpace(record.SourceMatchId) == "" {
			return fmt.Errorf("pre-check match has empty source id")
		}
		if _, ok := tournamentSourceIDs[strings.TrimSpace(record.SourceTournamentId)]; !ok {
			return fmt.Errorf("pre-check match %s references unknown tournament %s", record.SourceMatchId, record.SourceTournamentId)
		}
		if record.PlayersAllocated {
			for _, sourceID := range []string{record.HomePlayerSourceId, record.AwayPlayerSourceId} {
				if _, ok := playerSourceIDs[strings.TrimSpace(sourceID)]; !ok {
					return fmt.Errorf("pre-check match %s references unknown player %s", record.SourceMatchId, sourceID)
				}
			}
		}
	}

	// Check players for soft-deleted records.
	playerIDs := make([]string, 0, len(players))
	for _, r := range players {
		playerIDs = append(playerIDs, r.SourcePlayerId)
	}
	if len(playerIDs) > 0 {
		var softDeleted []model.Player
		if err := db.Unscoped().Where("source_type = ? AND source_player_id IN ? AND deleted_at IS NOT NULL", wstSourceType, playerIDs).Find(&softDeleted).Error; err != nil {
			return fmt.Errorf("pre-check player query: %w", err)
		}
		for _, p := range softDeleted {
			return fmt.Errorf("soft-deleted player found: source_player_id=%s, please resolve before retrying", p.SourcePlayerId)
		}
	}

	// Check matches for soft-deleted records.
	matchIDs := make([]string, 0, len(matches))
	for _, r := range matches {
		matchIDs = append(matchIDs, r.SourceMatchId)
	}
	if len(matchIDs) > 0 {
		var softDeleted []model.TournamentMatch
		if err := db.Unscoped().Where("source_type = ? AND source_match_id IN ? AND deleted_at IS NOT NULL", wstSourceType, matchIDs).Find(&softDeleted).Error; err != nil {
			return fmt.Errorf("pre-check match query: %w", err)
		}
		for _, m := range softDeleted {
			return fmt.Errorf("soft-deleted match found: source_match_id=%s, please resolve before retrying", m.SourceMatchId)
		}
	}

	// Event news is keyed by the local tournament ID. A soft-deleted row would be
	// invisible to the projector and cause it to create a replacement row.
	tournamentIDs := make([]string, 0, len(tournamentSourceIDs))
	for sourceID := range tournamentSourceIDs {
		tournamentIDs = append(tournamentIDs, sourceID)
	}
	if len(tournamentIDs) > 0 {
		existing, err := model.NewTournamentModel(db).FindBySourceTournamentIds(wstSourceType, tournamentIDs)
		if err != nil {
			return fmt.Errorf("pre-check tournament query: %w", err)
		}
		localIDs := make([]int64, 0, len(existing))
		for _, tournament := range existing {
			localIDs = append(localIDs, tournament.Id)
		}
		if len(localIDs) > 0 {
			var softDeleted []model.EventNews
			if err := db.Unscoped().Where("source_type = ? AND tournament_id IN ? AND deleted_at IS NOT NULL", wstSourceType, localIDs).Find(&softDeleted).Error; err != nil {
				return fmt.Errorf("pre-check event news query: %w", err)
			}
			if len(softDeleted) > 0 {
				return fmt.Errorf("soft-deleted event news found: tournament_id=%d, please resolve before retrying", softDeleted[0].TournamentId)
			}
		}
	}

	return nil
}

func (s *Service) addNoMatchWarnings(summary *SyncSummary, tournaments []TournamentResource, matches []MatchResource) {
	matched := make(map[string]int, len(matches))
	for _, item := range matches {
		matched[item.Attributes.TournamentID]++
	}
	for _, item := range tournaments {
		actual := matched[item.ID]
		if actual == 0 {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("tournament %s has no official matches", item.ID))
		}
		if expected := item.Attributes.DatedMatchCount; expected != nil && actual != *expected {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("tournament %s returned %d matches, official metadata reports %d dated matches", item.ID, actual, *expected))
		}
		if total, dated := item.Attributes.MatchCount, item.Attributes.DatedMatchCount; total != nil && dated != nil && *total != *dated {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("tournament %s reports %d total matches but only %d with start dates", item.ID, *total, *dated))
		}
	}
}

func (s *Service) addTournamentDateWarnings(summary *SyncSummary, tournaments []TournamentResource) {
	for _, item := range tournaments {
		if _, _, err := tournamentDateRange(item); err != nil {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("skipped tournament %s (%v)", item.ID, err))
		}
	}
}

// finalizeStatus determines the final status based on summary contents.
func (s *Service) finalizeStatus(summary *SyncSummary) string {
	if len(summary.Warnings) > 0 {
		return BackfillStatusNeedsReview
	}
	return BackfillStatusCompleted
}

func exitCodeForStatus(status string) int {
	switch status {
	case BackfillStatusCompleted:
		return ExitCodeCompleted
	case BackfillStatusNeedsReview:
		return ExitCodeNeedsReview
	default:
		return ExitCodeFailed
	}
}

func (s *Service) resolveSeasonIDs(ctx context.Context, params SyncParams) ([]int, int, []string, error) {
	if params.Mode == SyncModeSeason {
		return []int{params.Season}, 0, nil, nil
	}

	seasons, err := s.client.FetchSeasons(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	window := syncWindowFromParams(params)
	ids, unknown := resolveSeasonIDsForWindow(seasons, window)
	if len(ids) == 0 {
		ids = extractAllSeasonIDs(seasons)
	}
	if len(ids) == 0 {
		return nil, len(seasons), nil, fmt.Errorf("no WST seasons resolved for sync window")
	}
	warnings := make([]string, 0, len(unknown))
	for _, item := range unknown {
		warnings = append(warnings, fmt.Sprintf("season %s has unrecognized name %q and was included for tournament date filtering", item.ID, item.Attributes.Name))
	}
	return ids, len(seasons), warnings, nil
}

func (s *Service) fetchCandidateTournaments(ctx context.Context, seasonIDs []int) ([]TournamentResource, error) {
	seen := make(map[string]TournamentResource)
	for _, seasonID := range seasonIDs {
		items, err := s.client.FetchTournamentsBySeason(ctx, seasonID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if strings.TrimSpace(item.ID) == "" {
				return nil, fmt.Errorf("wst tournament in season %d has an empty id", seasonID)
			}
			if current, ok := seen[item.ID]; ok && !reflect.DeepEqual(current, item) {
				return nil, fmt.Errorf("wst tournament duplicate id %s has conflicting data across seasons", item.ID)
			}
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

func (s *Service) scanMatchesForTournaments(ctx context.Context, tournamentIDs map[string]struct{}) ([]MatchResource, int, int, error) {
	if len(tournamentIDs) == 0 {
		return []MatchResource{}, 0, 0, nil
	}

	pageNumber := 1
	scanned := 0
	seenAll := make(map[string]MatchResource)
	seenFiltered := make(map[string]struct{})
	filtered := make([]MatchResource, 0)
	var expectedTotal *int
	dataPages := 0
	for {
		if pageNumber > maxPaginationPages {
			return nil, scanned, dataPages, fmt.Errorf("match pagination exceeded page limit %d", maxPaginationPages)
		}

		page, err := s.client.FetchMatchesPage(ctx, pageNumber, defaultMatchesPageSize)
		if err != nil {
			return nil, scanned, dataPages, err
		}
		if err := validatePaginationMeta("matches", pageNumber, len(page.Data), page.Meta, &expectedTotal); err != nil {
			return nil, scanned, dataPages, err
		}
		if len(page.Data) == 0 {
			if page.Links != nil && page.Links.Next != nil {
				return nil, scanned, dataPages, fmt.Errorf("wst matches page %d is empty but links.next is present", pageNumber)
			}
			if expectedTotal != nil && len(seenAll) != *expectedTotal {
				return nil, scanned, dataPages, fmt.Errorf("match pagination ended with %d unique records, expected %d", len(seenAll), *expectedTotal)
			}
			break
		}
		dataPages = pageNumber

		// Progress detection: track all raw IDs on this page before filtering.
		pageNewIDs := 0
		for _, item := range page.Data {
			if strings.TrimSpace(item.ID) == "" {
				return nil, scanned, dataPages, fmt.Errorf("wst matches page %d contains an empty id", pageNumber)
			}
			if current, ok := seenAll[item.ID]; ok {
				if !reflect.DeepEqual(current, item) {
					return nil, scanned, dataPages, fmt.Errorf("wst match duplicate id %s has conflicting data", item.ID)
				}
				continue
			}
			seenAll[item.ID] = item
			pageNewIDs++
		}
		if pageNewIDs == 0 {
			return nil, scanned, dataPages, fmt.Errorf("match pagination stalled at page %d (no new IDs)", pageNumber)
		}
		if expectedTotal != nil && len(seenAll) > *expectedTotal {
			return nil, scanned, dataPages, fmt.Errorf("match pagination returned %d unique records, expected %d", len(seenAll), *expectedTotal)
		}

		scanned += len(page.Data)
		for _, item := range FilterMatchesByTournamentIDs(page.Data, tournamentIDs) {
			if _, ok := seenFiltered[item.ID]; ok {
				continue
			}
			seenFiltered[item.ID] = struct{}{}
			filtered = append(filtered, item)
		}
		if expectedTotal != nil && len(seenAll) == *expectedTotal {
			if page.Links != nil && page.Links.Next != nil {
				return nil, scanned, dataPages, fmt.Errorf("wst matches reached expected total %d at page %d but links.next is present", *expectedTotal, pageNumber)
			}
			break
		}
		if page.Links != nil && page.Links.Next == nil {
			if expectedTotal != nil {
				return nil, scanned, dataPages, fmt.Errorf("wst matches pagination ended at page %d before expected total", pageNumber)
			}
			break
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

	return filtered, scanned, dataPages, nil
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

func resolveSeasonIDsForWindow(items []SeasonResource, window DateWindow) ([]int, []SeasonResource) {
	seen := make(map[int]struct{})
	ids := make([]int, 0, len(items))
	unknown := make([]SeasonResource, 0)
	for _, item := range items {
		seasonID, err := strconv.Atoi(strings.TrimSpace(item.ID))
		if err != nil {
			continue
		}
		seasonWindow, ok := parseSeasonWindow(item.Attributes.Name)
		if ok && !seasonWindow.Overlaps(window.From, window.To) {
			continue
		}
		if !ok {
			unknown = append(unknown, item)
		}
		if _, exists := seen[seasonID]; exists {
			continue
		}
		seen[seasonID] = struct{}{}
		ids = append(ids, seasonID)
	}
	sort.Ints(ids)
	return ids, unknown
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

	for _, pageURL := range pageCandidates {
		if pageURL == "" {
			continue
		}

		coverImage, err := s.client.FetchPageCoverImage(ctx, pageURL)
		if err != nil {
			continue
		}

		coverImage = strings.TrimSpace(coverImage)
		if coverImage == "" || coverImage == defaultTournamentCoverImage {
			continue
		}
		return coverImage
	}

	return ""
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

		homeScore, awayScore, scoresConfirmed := 0, 0, false
		if item.Attributes.HomePlayerScore != nil && item.Attributes.AwayPlayerScore != nil {
			homeScore = *item.Attributes.HomePlayerScore
			awayScore = *item.Attributes.AwayPlayerScore
			scoresConfirmed = true
		}

		result = append(result, MatchUpsertRecord{
			SourceType:              wstSourceType,
			SourceMatchId:           strings.TrimSpace(item.ID),
			SourceTournamentId:      strings.TrimSpace(item.Attributes.TournamentID),
			RoundName:               strings.TrimSpace(item.Attributes.Round),
			RoundOrder:              roundOrder,
			MatchOrder:              matchOrder,
			StartTime:               startTime,
			BestOf:                  item.Attributes.NumberOfFrames,
			HomePlayerSourceId:      strings.TrimSpace(item.Attributes.HomePlayerID),
			HomePlayerName:          firstNonEmptyValue(buildPlayerDisplayName(item.Attributes.HomePlayer.FirstName, item.Attributes.HomePlayer.Surname), item.Attributes.HomePlayerID),
			AwayPlayerSourceId:      strings.TrimSpace(item.Attributes.AwayPlayerID),
			AwayPlayerName:          firstNonEmptyValue(buildPlayerDisplayName(item.Attributes.AwayPlayer.FirstName, item.Attributes.AwayPlayer.Surname), item.Attributes.AwayPlayerID),
			HomeScore:               homeScore,
			AwayScore:               awayScore,
			ScoresConfirmed:         scoresConfirmed,
			SourceStatus:            strings.TrimSpace(item.Attributes.Status),
			PlayersAllocated:        item.Attributes.PlayersAllocated,
			PlayersAllocatedPresent: item.Attributes.PlayersAllocatedPresent,
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
		subCode := strings.TrimPrefix(normalized, "gb-")
		switch subCode {
		case "eng", "sct", "wls":
			return buildSubdivisionFlag("gb" + subCode)
		default:
			return ""
		}
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

func buildSubdivisionFlag(subdivision string) string {
	var b strings.Builder
	b.WriteRune(0x1F3F4) // black flag base
	for _, ch := range subdivision {
		if ch >= 'a' && ch <= 'z' {
			b.WriteRune(0xE0000 + ch) // tag letter
		}
	}
	b.WriteRune(0xE007F) // cancel tag
	return b.String()
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
