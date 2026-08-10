package logic

import (
	"context"
	"fmt"
	"sort"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
)

type SeasonLifecycleRepairPolicy struct {
	AnchorDate    string
	InitialNumber int
	CycleMonths   int
	Timezone      string
}

type SeasonLifecycleRepairWindow struct {
	Name       string
	StartDate  string
	EndDate    string
	SeasonId   int64
	Status     int
	Exists     bool
	Created    bool
	Settlement string
}

type SeasonLifecycleRepairSummary struct {
	State                       string
	Problem                     string
	Policy                      SeasonLifecycleRepairPolicy
	Windows                     []SeasonLifecycleRepairWindow
	WindowsPlanned              int
	WindowsCreated              int
	SeasonsToSettle             int
	SeasonsSettled              int
	EventsCovered               int64
	UsersCovered                int64
	EstimatedChallengeSnapshots int
	EstimatedSeasonRecords      int
	EstimatedSeasonTitles       int
	ChallengeSnapshotsWritten   int
	SeasonRecordsWritten        int
	SeasonTitlesGranted         int
	Conflicts                   []string
	Failures                    []string
}

type SeasonLifecycleRepairService struct {
	svcCtx    *svc.ServiceContext
	lifecycle *SeasonLifecycleService
	settle    *SeasonSettlementService
}

func NewSeasonLifecycleRepairService(svcCtx *svc.ServiceContext) *SeasonLifecycleRepairService {
	return &SeasonLifecycleRepairService{
		svcCtx:    svcCtx,
		lifecycle: NewSeasonLifecycleService(svcCtx),
		settle:    NewSeasonSettlementService(svcCtx),
	}
}

func (s *SeasonLifecycleRepairService) DryRun(ctx context.Context) (*SeasonLifecycleRepairSummary, error) {
	return s.DryRunAt(ctx, time.Now())
}

func (s *SeasonLifecycleRepairService) DryRunAt(_ context.Context, now time.Time) (*SeasonLifecycleRepairSummary, error) {
	summary, _, _, err := s.inspect(now)
	return summary, err
}

func (s *SeasonLifecycleRepairService) Repair(ctx context.Context) (*SeasonLifecycleRepairSummary, error) {
	return s.RepairAt(ctx, time.Now())
}

func (s *SeasonLifecycleRepairService) RepairAt(ctx context.Context, now time.Time) (*SeasonLifecycleRepairSummary, error) {
	summary, policy, planned, err := s.inspect(now)
	if err != nil || summary.State == seasonx.StateNotStarted || len(summary.Conflicts) > 0 {
		return summary, err
	}
	if s == nil || s.lifecycle == nil || s.settle == nil || s.svcCtx == nil || s.svcCtx.SeasonModel == nil || s.svcCtx.SeasonSettlementModel == nil {
		return repairFailure(summary, fmt.Errorf("season lifecycle repair service is unavailable"))
	}

	ensured, err := s.lifecycle.EnsureAt(ctx, now)
	if err != nil {
		return repairFailure(summary, err)
	}
	if ensured.State != seasonx.StateActive {
		summary.State = ensured.State
		summary.Problem = ensured.Problem
		summary.Conflicts = append(summary.Conflicts, ensured.Plan.Conflicts...)
		return summary, nil
	}
	summary.State = ensured.State
	summary.Problem = ensured.Problem
	summary.WindowsCreated = ensured.Created
	createdByStart := make(map[string]bool, len(ensured.CreatedWindows))
	for _, window := range ensured.CreatedWindows {
		createdByStart[repairWindowKey(window.StartDate)] = true
	}

	seasons, err := s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return repairFailure(summary, err)
	}
	byStart := refreshRepairWindows(summary, policy, seasons)
	for index := range summary.Windows {
		summary.Windows[index].Created = createdByStart[summary.Windows[index].StartDate]
	}
	for _, window := range planned.Plan.Windows {
		endExclusive := window.EndDate.AddDate(0, 0, 1)
		if now.Before(endExclusive) {
			continue
		}
		item, found := byStart[repairWindowKey(window.StartDate)]
		if !found {
			return repairFailure(summary, fmt.Errorf("season %s missing after lifecycle ensure", window.Name))
		}
		settlement, findErr := s.svcCtx.SeasonSettlementModel.FindBySeasonId(item.Id)
		if findErr != nil {
			return repairFailure(summary, findErr)
		}
		if settlement != nil && settlement.Status == model.SeasonSettlementStatusCompleted {
			setRepairWindowSettlement(summary, window.StartDate, "completed")
			continue
		}
		settled, settleErr := s.settle.SettleSeasonWithOptionsAt(ctx, item.Id, now, SeasonSettlementOptions{Notify: false})
		if settleErr != nil {
			setRepairWindowSettlement(summary, window.StartDate, "failed")
			return repairFailure(summary, settleErr)
		}
		if settled.Skipped {
			setRepairWindowSettlement(summary, window.StartDate, "completed")
			continue
		}
		summary.SeasonsSettled++
		summary.ChallengeSnapshotsWritten += settled.ChallengeSnapshots
		summary.SeasonRecordsWritten += settled.SeasonRecords
		summary.SeasonTitlesGranted += settled.SeasonTitles
		setRepairWindowSettlement(summary, window.StartDate, "settled")
	}

	converged, convergeErr := s.lifecycle.ConvergeStatusesAt(ctx, now)
	if convergeErr != nil {
		return repairFailure(summary, convergeErr)
	}
	summary.State = converged.State
	summary.Problem = converged.Problem
	summary.Conflicts = append(summary.Conflicts, converged.Plan.Conflicts...)
	if converged.State != seasonx.StateActive {
		return summary, nil
	}
	seasons, err = s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return repairFailure(summary, err)
	}
	refreshRepairWindows(summary, policy, seasons)
	return summary, nil
}

func (s *SeasonLifecycleRepairService) inspect(now time.Time) (*SeasonLifecycleRepairSummary, seasonx.Policy, *SeasonLifecycleResult, error) {
	if s == nil || s.svcCtx == nil || s.lifecycle == nil {
		return nil, seasonx.Policy{}, nil, fmt.Errorf("season lifecycle repair service is unavailable")
	}
	result, err := s.lifecycle.PlanAt(now)
	if err != nil {
		return nil, seasonx.Policy{}, nil, err
	}
	summary := &SeasonLifecycleRepairSummary{
		State:     result.State,
		Problem:   repairProblem(result),
		Windows:   []SeasonLifecycleRepairWindow{},
		Conflicts: append([]string(nil), result.Plan.Conflicts...),
		Failures:  []string{},
	}
	policy, policyErr := seasonx.NewPolicy(s.svcCtx.Config.SeasonLifecycle)
	if policyErr != nil {
		summary.State = seasonx.StateUnavailable
		summary.Problem = policyErr.Error()
		summary.Conflicts = append(summary.Conflicts, policyErr.Error())
		return summary, seasonx.Policy{}, result, nil
	}
	summary.Policy = repairPolicy(policy)
	if summary.State == seasonx.StateNotStarted {
		return summary, policy, result, nil
	}

	summary.WindowsPlanned = len(result.Plan.Windows)
	seasons, seasonsErr := s.svcCtx.SeasonModel.ListAll()
	if seasonsErr != nil {
		return nil, seasonx.Policy{}, nil, seasonsErr
	}
	seasonsByStart := make(map[string]model.Season, len(seasons))
	for _, item := range seasons {
		start, _ := seasonx.Bounds(&item, policy.Location)
		seasonsByStart[repairWindowKey(start)] = item
	}
	pendingSettlements := make(map[string]bool)
	for _, window := range result.Plan.Windows {
		key := repairWindowKey(window.StartDate)
		item, found := seasonsByStart[key]
		detail := SeasonLifecycleRepairWindow{
			Name:       window.Name,
			StartDate:  key,
			EndDate:    window.EndDate.Format(time.DateOnly),
			Exists:     found,
			Settlement: "not_due",
		}
		if found {
			detail.SeasonId = item.Id
			detail.Status = item.Status
		}
		if !now.Before(window.EndDate.AddDate(0, 0, 1)) {
			detail.Settlement = "pending"
			if found && s.svcCtx.SeasonSettlementModel != nil {
				settlement, settlementErr := s.svcCtx.SeasonSettlementModel.FindBySeasonId(item.Id)
				if settlementErr != nil {
					return nil, seasonx.Policy{}, nil, settlementErr
				}
				if settlement != nil && settlement.Status == model.SeasonSettlementStatusCompleted {
					detail.Settlement = "completed"
				}
			}
			if detail.Settlement != "completed" {
				pendingSettlements[key] = true
				summary.SeasonsToSettle++
			}
		}
		summary.Windows = append(summary.Windows, detail)
	}
	if s.svcCtx.AchievementProgressEventModel == nil || result.Plan.Current == nil {
		return summary, policy, result, nil
	}
	startAt := policy.Anchor
	endExclusive := now
	if endExclusive.Before(startAt) || endExclusive.Equal(startAt) {
		return summary, policy, result, nil
	}
	events, users, countErr := s.svcCtx.AchievementProgressEventModel.CountUsersBetween(startAt, endExclusive)
	if countErr != nil {
		return nil, seasonx.Policy{}, nil, countErr
	}
	summary.EventsCovered = events
	summary.UsersCovered = users

	for _, window := range result.Plan.Windows {
		if !pendingSettlements[repairWindowKey(window.StartDate)] {
			continue
		}
		pairs, pairErr := s.svcCtx.AchievementProgressEventModel.ListUserGamesBetween(
			achievementx.SeasonChallengeMetricKeys(),
			window.StartDate,
			window.EndDate.AddDate(0, 0, 1),
		)
		if pairErr != nil {
			return nil, seasonx.Policy{}, nil, pairErr
		}
		summary.EstimatedChallengeSnapshots += len(pairs) * len(achievementx.FixedSeasonChallengeDefinitions())
		summary.EstimatedSeasonRecords += len(pairs)
		summary.EstimatedSeasonTitles += estimateSeasonTitles(pairs)
	}
	return summary, policy, result, nil
}

func repairPolicy(policy seasonx.Policy) SeasonLifecycleRepairPolicy {
	anchorDate := ""
	if !policy.Anchor.IsZero() {
		anchorDate = policy.Anchor.Format(time.DateOnly)
	}
	return SeasonLifecycleRepairPolicy{
		AnchorDate:    anchorDate,
		InitialNumber: policy.InitialNumber,
		CycleMonths:   policy.CycleMonths,
		Timezone:      policy.Location.String(),
	}
}

func repairProblem(result *SeasonLifecycleResult) string {
	if result == nil {
		return "season lifecycle result is missing"
	}
	if result.Problem != "" {
		return result.Problem
	}
	if result.State == seasonx.StateUnavailable && len(result.Plan.Missing) > 0 {
		return "season schedule is missing required windows"
	}
	return ""
}

func repairWindowKey(value time.Time) string {
	return value.Format(time.DateOnly)
}

func refreshRepairWindows(summary *SeasonLifecycleRepairSummary, policy seasonx.Policy, seasons []model.Season) map[string]model.Season {
	byStart := make(map[string]model.Season, len(seasons))
	for _, item := range seasons {
		start, _ := seasonx.Bounds(&item, policy.Location)
		byStart[repairWindowKey(start)] = item
	}
	for index := range summary.Windows {
		item, found := byStart[summary.Windows[index].StartDate]
		summary.Windows[index].Exists = found
		if !found {
			summary.Windows[index].SeasonId = 0
			summary.Windows[index].Status = 0
			continue
		}
		summary.Windows[index].SeasonId = item.Id
		summary.Windows[index].Status = item.Status
	}
	return byStart
}

func setRepairWindowSettlement(summary *SeasonLifecycleRepairSummary, startDate time.Time, settlement string) {
	if summary == nil {
		return
	}
	key := repairWindowKey(startDate)
	for index := range summary.Windows {
		if summary.Windows[index].StartDate == key {
			summary.Windows[index].Settlement = settlement
			return
		}
	}
}

func repairFailure(summary *SeasonLifecycleRepairSummary, err error) (*SeasonLifecycleRepairSummary, error) {
	if summary != nil && err != nil {
		summary.Failures = append(summary.Failures, err.Error())
	}
	return summary, err
}

func estimateSeasonTitles(pairs []model.AchievementProgressUserGame) int {
	byGameType := make(map[int]int)
	for _, pair := range pairs {
		byGameType[pair.GameType]++
	}
	gameTypes := make([]int, 0, len(byGameType))
	for gameType := range byGameType {
		gameTypes = append(gameTypes, gameType)
	}
	sort.Ints(gameTypes)
	count := 0
	for _, gameType := range gameTypes {
		if byGameType[gameType] > 10 {
			count += 10
		} else {
			count += byGameType[gameType]
		}
	}
	return count
}
