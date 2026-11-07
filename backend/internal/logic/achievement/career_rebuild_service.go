package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

const careerAchievementRebuildBatchSize = 500

type CareerAchievementRebuildSummary struct {
	MatchesTotal            int
	TournamentsTotal        int
	UsersTotal              int
	EventsTotal             int
	EventsCreated           int
	AchievementsUnlocked    int
	TitlesGranted           int
	SkippedNoPlayer         int
	SkippedAmbiguousSpecial int
}

type CareerAchievementRebuildService struct {
	svcCtx       *svc.ServiceContext
	titleService *TitleGrantService
}

type careerRebuildEventKey struct {
	UserId     int64
	SourceType string
	SourceId   int64
	MetricKey  string
}

type careerRebuildPlan struct {
	events                  map[careerRebuildEventKey]model.AchievementProgressEvent
	matchesTotal            int
	tournamentsTotal        int
	skippedNoPlayer         int
	skippedAmbiguousSpecial int
}

type careerRebuildSnapshotPlan struct {
	snapshot   model.UserAchievement
	definition model.Achievement
	newUnlock  bool
	needsTitle bool
	grantAt    time.Time
}

type careerStreakState struct {
	current int
	max     int
}

type careerUserGameKey struct {
	userId   int64
	gameType int
}

type careerSpecialFactKey struct {
	roundNo int
	actor   int
	metric  string
}

func NewCareerAchievementRebuildService(svcCtx *svc.ServiceContext) *CareerAchievementRebuildService {
	return &CareerAchievementRebuildService{
		svcCtx:       svcCtx,
		titleService: NewTitleGrantService(svcCtx),
	}
}

func (s *CareerAchievementRebuildService) DryRun(ctx context.Context) (*CareerAchievementRebuildSummary, error) {
	return s.run(ctx, true)
}

func (s *CareerAchievementRebuildService) Rebuild(ctx context.Context) (*CareerAchievementRebuildSummary, error) {
	return s.run(ctx, false)
}

func (s *CareerAchievementRebuildService) run(ctx context.Context, dryRun bool) (*CareerAchievementRebuildSummary, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	plan, err := s.buildEventPlan(ctx)
	if err != nil {
		return nil, err
	}
	existingEvents, err := s.svcCtx.AchievementProgressEventModel.ListAllForCareerRebuild()
	if err != nil {
		return nil, err
	}

	existingKeys := make(map[careerRebuildEventKey]struct{}, len(existingEvents))
	userSet := make(map[int64]struct{})
	for _, event := range existingEvents {
		existingKeys[careerEventKey(event)] = struct{}{}
		if event.UserId > 0 {
			userSet[event.UserId] = struct{}{}
		}
	}
	missingEvents := make([]model.AchievementProgressEvent, 0)
	for key, event := range plan.events {
		if event.UserId > 0 {
			userSet[event.UserId] = struct{}{}
		}
		if _, exists := existingKeys[key]; exists {
			continue
		}
		missingEvents = append(missingEvents, event)
	}
	sortCareerRebuildEvents(missingEvents)

	effectiveEvents := make([]model.AchievementProgressEvent, 0, len(existingEvents)+len(missingEvents))
	effectiveEvents = append(effectiveEvents, existingEvents...)
	effectiveEvents = append(effectiveEvents, missingEvents...)
	sortCareerRebuildEvents(effectiveEvents)
	userIds := sortedCareerRebuildUserIDs(userSet)

	definitions, err := s.svcCtx.AchievementModel.FindActive()
	if err != nil {
		return nil, err
	}
	existingAchievements, err := s.svcCtx.UserAchievementModel.FindByUserIds(userIds)
	if err != nil {
		return nil, err
	}
	existingTitles, err := s.svcCtx.UserTitleModel.FindAchievementTitlesByUserIds(userIds)
	if err != nil {
		return nil, err
	}
	snapshotPlans := buildCareerRebuildSnapshotPlans(userIds, definitions, effectiveEvents, existingAchievements, existingTitles)

	summary := &CareerAchievementRebuildSummary{
		MatchesTotal:            plan.matchesTotal,
		TournamentsTotal:        plan.tournamentsTotal,
		UsersTotal:              len(userIds),
		EventsTotal:             len(plan.events),
		EventsCreated:           len(missingEvents),
		SkippedNoPlayer:         plan.skippedNoPlayer,
		SkippedAmbiguousSpecial: plan.skippedAmbiguousSpecial,
	}
	for _, snapshotPlan := range snapshotPlans {
		if snapshotPlan.newUnlock {
			summary.AchievementsUnlocked++
		}
		if snapshotPlan.needsTitle {
			summary.TitlesGranted++
		}
	}
	if dryRun {
		return summary, nil
	}

	summary.EventsCreated = 0
	for start := 0; start < len(missingEvents); start += careerAchievementRebuildBatchSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		end := start + careerAchievementRebuildBatchSize
		if end > len(missingEvents) {
			end = len(missingEvents)
		}
		created, createErr := s.svcCtx.AchievementProgressEventModel.CreateCareerRebuildBatch(missingEvents[start:end])
		if createErr != nil {
			return nil, createErr
		}
		summary.EventsCreated += int(created)
	}

	summary.AchievementsUnlocked = 0
	summary.TitlesGranted = 0
	for start := 0; start < len(snapshotPlans); start += careerAchievementRebuildBatchSize {
		end := start + careerAchievementRebuildBatchSize
		if end > len(snapshotPlans) {
			end = len(snapshotPlans)
		}
		snapshots := make([]model.UserAchievement, 0, end-start)
		for i := start; i < end; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			item := &snapshotPlans[i]
			if item.needsTitle {
				created, grantErr := s.titleService.GrantAchievementTitleAt(item.snapshot.UserId, item.definition, item.grantAt)
				if grantErr != nil {
					return nil, grantErr
				}
				if created {
					summary.TitlesGranted++
				}
			}
			snapshots = append(snapshots, item.snapshot)
			if item.newUnlock {
				summary.AchievementsUnlocked++
			}
		}
		if err := s.svcCtx.UserAchievementModel.UpsertCareerRebuildSnapshots(snapshots); err != nil {
			return nil, err
		}
	}
	return summary, nil
}

func (s *CareerAchievementRebuildService) validate() error {
	if s == nil || s.svcCtx == nil || s.svcCtx.MatchModel == nil || s.svcCtx.TournamentModel == nil ||
		s.svcCtx.TournamentParticipantModel == nil || s.svcCtx.AchievementModel == nil ||
		s.svcCtx.UserAchievementModel == nil || s.svcCtx.UserTitleModel == nil ||
		s.svcCtx.AchievementProgressEventModel == nil || s.svcCtx.DB == nil {
		return fmt.Errorf("career achievement rebuild service is unavailable")
	}
	return nil
}

func (s *CareerAchievementRebuildService) buildEventPlan(ctx context.Context) (*careerRebuildPlan, error) {
	plan := &careerRebuildPlan{events: make(map[careerRebuildEventKey]model.AchievementProgressEvent)}
	matches, err := s.svcCtx.MatchModel.ListCompletedForAchievementRebuild()
	if err != nil {
		return nil, err
	}
	plan.matchesTotal = len(matches)
	matchIds := make([]int64, 0, len(matches))
	for _, match := range matches {
		matchIds = append(matchIds, match.Id)
	}
	rounds, err := s.svcCtx.MatchModel.ListCompletedRoundsByMatchIDs(matchIds)
	if err != nil {
		return nil, err
	}
	actions, err := s.svcCtx.MatchModel.ListActiveActionsByMatchIDs(matchIds)
	if err != nil {
		return nil, err
	}
	storedAchievements, err := s.svcCtx.MatchModel.ListAchievementsByMatchIDs(matchIds)
	if err != nil {
		return nil, err
	}
	roundsByMatch := groupCareerRoundsByMatch(rounds)
	actionsByMatch := groupCareerActionsByMatch(actions)
	storedByMatch := groupCareerStoredAchievementsByMatch(storedAchievements)
	streaks := make(map[careerUserGameKey]*careerStreakState)

	for i := range matches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		match := &matches[i]
		if match.OpponentId == nil || *match.OpponentId <= 0 || *match.OpponentId == match.UserId {
			plan.skippedNoPlayer++
			continue
		}
		occurredAt := careerMatchOccurredAt(match)
		player1 := match.UserId
		player2 := *match.OpponentId
		plan.addEvent(player1, SourceTypeMatch, match.Id, match.GameType, MetricMatchesTotal, 1, occurredAt)
		plan.addEvent(player2, SourceTypeMatch, match.Id, match.GameType, MetricMatchesTotal, 1, occurredAt)

		player1Won := match.Result != nil && *match.Result == 1
		winner := player2
		if player1Won {
			winner = player1
		}
		plan.addEvent(winner, SourceTypeMatch, match.Id, match.GameType, MetricWinsTotal, 1, occurredAt)
		for _, entry := range []struct {
			userId int64
			won    bool
		}{{player1, player1Won}, {player2, !player1Won}} {
			key := careerUserGameKey{userId: entry.userId, gameType: match.GameType}
			state := streaks[key]
			if state == nil {
				state = &careerStreakState{}
				streaks[key] = state
			}
			if entry.won {
				state.current++
				if state.current > state.max {
					state.max = state.current
				}
			} else {
				state.current = 0
			}
			plan.addEvent(entry.userId, SourceTypeMatch, match.Id, match.GameType, MetricMaxWinStreak, state.max, occurredAt)
		}

		specialByActor, skipped := resolveCareerSpecialMetrics(match, roundsByMatch[match.Id], actionsByMatch[match.Id], storedByMatch[match.Id])
		plan.skippedAmbiguousSpecial += skipped
		userByActor := map[int]int64{1: player1, 2: player2}
		for actor, metrics := range specialByActor {
			userId := userByActor[actor]
			for metricKey, value := range metrics {
				plan.addEvent(userId, SourceTypeMatch, match.Id, match.GameType, metricKey, value, occurredAt)
			}
		}
	}

	tournaments, err := s.svcCtx.TournamentModel.ListForAchievementRebuild()
	if err != nil {
		return nil, err
	}
	participants, err := s.svcCtx.TournamentParticipantModel.ListForAchievementRebuild()
	if err != nil {
		return nil, err
	}
	plan.tournamentsTotal = len(tournaments)
	tournamentById := make(map[int64]model.Tournament, len(tournaments))
	for _, tournament := range tournaments {
		tournamentById[tournament.Id] = tournament
	}
	for _, participant := range participants {
		tournament, ok := tournamentById[participant.TournamentId]
		if !ok || participant.UserId <= 0 {
			continue
		}
		plan.addEvent(participant.UserId, SourceTypeTournamentJoin, tournament.Id, tournament.GameType, MetricTournamentJoinTotal, 1, careerTournamentJoinOccurredAt(participant, tournament))
		if tournament.Status != 2 || participant.FinalRank <= 0 {
			continue
		}
		finishedAt := careerTournamentFinishOccurredAt(tournament)
		plan.addEvent(participant.UserId, SourceTypeTournamentFinish, tournament.Id, tournament.GameType, MetricTournamentFinishTotal, 1, finishedAt)
		if participant.FinalRank == 1 {
			plan.addEvent(participant.UserId, SourceTypeTournamentFinish, tournament.Id, tournament.GameType, MetricTournamentChampionTotal, 1, finishedAt)
		}
	}
	return plan, nil
}

func (p *careerRebuildPlan) addEvent(userId int64, sourceType string, sourceId int64, gameType int, metricKey string, metricValue int, occurredAt time.Time) {
	if userId <= 0 || sourceId <= 0 || metricKey == "" || metricValue <= 0 {
		return
	}
	event := *model.NewAchievementProgressEvent(userId, sourceType, sourceId, gameType, metricKey, metricValue, occurredAt)
	key := careerEventKey(event)
	if existing, ok := p.events[key]; ok && existing.MetricValue >= event.MetricValue {
		return
	}
	p.events[key] = event
}

func careerEventKey(event model.AchievementProgressEvent) careerRebuildEventKey {
	return careerRebuildEventKey{UserId: event.UserId, SourceType: event.SourceType, SourceId: event.SourceId, MetricKey: event.MetricKey}
}

func sortCareerRebuildEvents(events []model.AchievementProgressEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		leftAt := careerEventOccurredAt(events[i])
		rightAt := careerEventOccurredAt(events[j])
		if !leftAt.Equal(rightAt) {
			return leftAt.Before(rightAt)
		}
		if events[i].UserId != events[j].UserId {
			return events[i].UserId < events[j].UserId
		}
		if events[i].SourceType != events[j].SourceType {
			return events[i].SourceType < events[j].SourceType
		}
		if events[i].SourceId != events[j].SourceId {
			return events[i].SourceId < events[j].SourceId
		}
		return events[i].MetricKey < events[j].MetricKey
	})
}

func buildCareerRebuildSnapshotPlans(
	userIds []int64,
	definitions []model.Achievement,
	events []model.AchievementProgressEvent,
	existingAchievements []model.UserAchievement,
	existingTitles []model.UserTitle,
) []careerRebuildSnapshotPlan {
	eventsByUserMetric := make(map[int64]map[string][]model.AchievementProgressEvent)
	for _, event := range events {
		if eventsByUserMetric[event.UserId] == nil {
			eventsByUserMetric[event.UserId] = make(map[string][]model.AchievementProgressEvent)
		}
		eventsByUserMetric[event.UserId][event.MetricKey] = append(eventsByUserMetric[event.UserId][event.MetricKey], event)
	}
	existingByUserAchievement := make(map[[2]int64]model.UserAchievement, len(existingAchievements))
	for _, item := range existingAchievements {
		existingByUserAchievement[[2]int64{item.UserId, item.AchievementId}] = item
	}
	titleByUserAchievement := make(map[[2]int64]model.UserTitle, len(existingTitles))
	for _, title := range existingTitles {
		titleByUserAchievement[[2]int64{title.UserId, title.SourceRefId}] = title
	}

	plans := make([]careerRebuildSnapshotPlan, 0)
	for _, userId := range userIds {
		for _, definition := range definitions {
			metricEvents := eventsByUserMetric[userId][definition.MetricKey]
			progress, crossing := aggregateCareerRebuildProgress(metricEvents, definition)
			existing, hasExisting := existingByUserAchievement[[2]int64{userId, definition.Id}]
			if progress == 0 && !hasExisting {
				continue
			}

			snapshot := existing
			snapshot.UserId = userId
			snapshot.AchievementId = definition.Id
			if progress > snapshot.Progress {
				snapshot.Progress = progress
			}
			newUnlock := false
			if crossing != nil {
				if snapshot.Unlocked == 0 {
					snapshot.Unlocked = 1
					newUnlock = true
				}
				if snapshot.UnlockedAt == nil || careerEventOccurredAt(*crossing).Before(*snapshot.UnlockedAt) {
					unlockedAt := careerEventOccurredAt(*crossing)
					snapshot.UnlockedAt = &unlockedAt
					snapshot.UnlockedSourceType = crossing.SourceType
					snapshot.UnlockedSourceId = crossing.SourceId
				}
			}

			plan := careerRebuildSnapshotPlan{snapshot: snapshot, definition: definition, newUnlock: newUnlock}
			if snapshot.Unlocked == 1 && definition.RewardTitleKey != "" && definition.RewardTitleName != "" {
				title, hasTitle := titleByUserAchievement[[2]int64{userId, definition.Id}]
				plan.needsTitle = !hasTitle
				plan.grantAt = careerRebuildGrantTime(snapshot, title, hasTitle)
				snapshot.RewardGranted = 1
				if !plan.grantAt.IsZero() && (snapshot.RewardGrantedAt == nil || plan.grantAt.Before(*snapshot.RewardGrantedAt)) {
					grantedAt := plan.grantAt
					snapshot.RewardGrantedAt = &grantedAt
				}
				plan.snapshot = snapshot
			}
			plans = append(plans, plan)
		}
	}
	return plans
}

func aggregateCareerRebuildProgress(events []model.AchievementProgressEvent, definition model.Achievement) (int, *model.AchievementProgressEvent) {
	progress := 0
	var crossing *model.AchievementProgressEvent
	for i := range events {
		event := &events[i]
		if definition.GameType > 0 && event.GameType != definition.GameType {
			continue
		}
		if definition.ProgressMode == ProgressModeMax {
			if event.MetricValue > progress {
				progress = event.MetricValue
			}
		} else {
			progress += event.MetricValue
		}
		if crossing == nil && progress >= definition.Threshold {
			copy := *event
			crossing = &copy
		}
	}
	return progress, crossing
}

func careerRebuildGrantTime(snapshot model.UserAchievement, title model.UserTitle, hasTitle bool) time.Time {
	if snapshot.UnlockedAt != nil {
		return *snapshot.UnlockedAt
	}
	if hasTitle {
		if title.GrantedAt != nil {
			return *title.GrantedAt
		}
		return title.CreatedAt
	}
	return snapshot.CreatedAt
}

func resolveCareerSpecialMetrics(match *model.Match, rounds []model.MatchRound, actions []model.MatchAction, stored []model.MatchAchievement) (map[int]map[string]int, int) {
	facts := make(map[careerSpecialFactKey]struct{})
	for _, round := range rounds {
		if round.Winner == nil || (*round.Winner != 1 && *round.Winner != 2) {
			continue
		}
		metric := careerMetricFromAchievementType(round.WinType)
		if metric != "" {
			facts[careerSpecialFactKey{roundNo: round.RoundNo, actor: *round.Winner, metric: metric}] = struct{}{}
		}
	}
	for _, action := range actions {
		if action.ActionType != "win" || (action.Actor != 1 && action.Actor != 2) || action.ExtraData == nil {
			continue
		}
		var payload struct {
			WinType string `json:"win_type"`
		}
		if json.Unmarshal([]byte(*action.ExtraData), &payload) != nil {
			continue
		}
		metric := careerMetricFromAchievementType(payload.WinType)
		if metric != "" {
			facts[careerSpecialFactKey{roundNo: action.RoundNo, actor: action.Actor, metric: metric}] = struct{}{}
		}
	}
	derived := make(map[int]map[string]int)
	for fact := range facts {
		if derived[fact.actor] == nil {
			derived[fact.actor] = make(map[string]int)
		}
		derived[fact.actor][fact.metric]++
	}
	if match != nil && match.AchievementSyncedAt != nil {
		trusted := make(map[int]map[string]int)
		skipped := 0
		for _, record := range stored {
			metric := careerMetricFromAchievementType(record.AchievementType)
			if metric == "" || record.Count <= 0 {
				continue
			}
			if record.Actor != 1 && record.Actor != 2 {
				skipped++
				continue
			}
			if trusted[record.Actor] == nil {
				trusted[record.Actor] = make(map[string]int)
			}
			trusted[record.Actor][metric] += record.Count
		}
		if len(trusted) > 0 {
			return trusted, skipped
		}
		return derived, skipped
	}

	skipped := 0
	for _, record := range stored {
		metric := careerMetricFromAchievementType(record.AchievementType)
		if metric == "" || record.Count <= 0 {
			continue
		}
		if record.Actor == 2 {
			if derived[2] == nil {
				derived[2] = make(map[string]int)
			}
			if record.Count > derived[2][metric] {
				derived[2][metric] = record.Count
			}
			continue
		}
		if record.Actor != 1 || derived[1][metric] == 0 {
			skipped++
		}
	}
	return derived, skipped
}

func careerMetricFromAchievementType(achievementType string) string {
	switch achievementType {
	case "break_clear", "break_and_run":
		return MetricBreakClearTotal
	case "continue_clear", "run_out":
		return MetricContinueClearTotal
	case "golden_break", "small_gold":
		return MetricGoldenBreakTotal
	case "nine_on_break", "big_gold":
		return MetricNineOnBreakTotal
	case "break_50":
		return MetricBreak50Total
	case "break_100":
		return MetricBreak100Total
	case "break_147":
		return MetricBreak147Total
	default:
		return ""
	}
}

func groupCareerRoundsByMatch(list []model.MatchRound) map[int64][]model.MatchRound {
	result := make(map[int64][]model.MatchRound)
	for _, item := range list {
		result[item.MatchId] = append(result[item.MatchId], item)
	}
	return result
}

func groupCareerActionsByMatch(list []model.MatchAction) map[int64][]model.MatchAction {
	result := make(map[int64][]model.MatchAction)
	for _, item := range list {
		result[item.MatchId] = append(result[item.MatchId], item)
	}
	return result
}

func groupCareerStoredAchievementsByMatch(list []model.MatchAchievement) map[int64][]model.MatchAchievement {
	result := make(map[int64][]model.MatchAchievement)
	for _, item := range list {
		result[item.MatchId] = append(result[item.MatchId], item)
	}
	return result
}

func careerMatchOccurredAt(match *model.Match) time.Time {
	if match == nil {
		return time.Time{}
	}
	if match.EndTime != nil && !match.EndTime.IsZero() {
		return *match.EndTime
	}
	if !match.MatchTime.IsZero() {
		return match.MatchTime
	}
	return match.CreatedAt
}

func careerTournamentJoinOccurredAt(participant model.TournamentParticipant, tournament model.Tournament) time.Time {
	if !participant.CreatedAt.IsZero() {
		return participant.CreatedAt
	}
	if tournament.StartTime != nil && !tournament.StartTime.IsZero() {
		return *tournament.StartTime
	}
	if tournament.StartDate != nil && !tournament.StartDate.IsZero() {
		return *tournament.StartDate
	}
	return tournament.CreatedAt
}

func careerTournamentFinishOccurredAt(tournament model.Tournament) time.Time {
	if tournament.EndTime != nil && !tournament.EndTime.IsZero() {
		return *tournament.EndTime
	}
	if tournament.EndDate != nil && !tournament.EndDate.IsZero() {
		return *tournament.EndDate
	}
	if tournament.StartTime != nil && !tournament.StartTime.IsZero() {
		return *tournament.StartTime
	}
	if tournament.StartDate != nil && !tournament.StartDate.IsZero() {
		return *tournament.StartDate
	}
	return tournament.CreatedAt
}

func careerEventOccurredAt(event model.AchievementProgressEvent) time.Time {
	if !event.OccurredAt.IsZero() {
		return event.OccurredAt
	}
	return event.CreatedAt
}

func sortedCareerRebuildUserIDs(set map[int64]struct{}) []int64 {
	list := make([]int64, 0, len(set))
	for userId := range set {
		list = append(list, userId)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	return list
}
