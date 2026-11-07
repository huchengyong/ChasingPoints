package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
)

const defaultSeasonRepairBatchSize = 100

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
	Cursor                      string
	NextCursor                  string
	HasMore                     bool
	Conflicts                   []string
	Failures                    []string
}

type SeasonLifecycleRepairOptions struct {
	DryRun         bool
	BatchSize      int
	AfterStartDate string
	Checkpoint     func(string) error
}

type SeasonLifecycleRepairService struct {
	svcCtx *svc.ServiceContext
	settle *SeasonSettlementService
}

type seasonRepairCandidate struct {
	window seasonx.Window
	season *model.Season
}

func NewSeasonLifecycleRepairService(svcCtx *svc.ServiceContext) *SeasonLifecycleRepairService {
	return &SeasonLifecycleRepairService{svcCtx: svcCtx, settle: NewSeasonSettlementService(svcCtx)}
}

func (s *SeasonLifecycleRepairService) DryRun(ctx context.Context) (*SeasonLifecycleRepairSummary, error) {
	return s.DryRunAt(ctx, time.Now())
}

func (s *SeasonLifecycleRepairService) DryRunAt(ctx context.Context, now time.Time) (*SeasonLifecycleRepairSummary, error) {
	return s.RunBatchAt(ctx, now, SeasonLifecycleRepairOptions{DryRun: true, BatchSize: defaultSeasonRepairBatchSize})
}

func (s *SeasonLifecycleRepairService) Repair(ctx context.Context) (*SeasonLifecycleRepairSummary, error) {
	return s.RepairAt(ctx, time.Now())
}

func (s *SeasonLifecycleRepairService) RepairAt(ctx context.Context, now time.Time) (*SeasonLifecycleRepairSummary, error) {
	return s.RunBatchAt(ctx, now, SeasonLifecycleRepairOptions{BatchSize: defaultSeasonRepairBatchSize})
}

func (s *SeasonLifecycleRepairService) RunBatchAt(ctx context.Context, now time.Time, options SeasonLifecycleRepairOptions) (*SeasonLifecycleRepairSummary, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if s == nil || s.svcCtx == nil || s.svcCtx.SeasonModel == nil {
		return nil, fmt.Errorf("season lifecycle repair service is unavailable")
	}
	policy, err := seasonx.NewPolicy(s.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return &SeasonLifecycleRepairSummary{
			State: seasonx.StateUnavailable, Problem: err.Error(), Windows: []SeasonLifecycleRepairWindow{},
			Conflicts: []string{err.Error()}, Failures: []string{},
		}, nil
	}
	summary := &SeasonLifecycleRepairSummary{
		State: seasonx.StateActive, Policy: repairPolicy(policy), Windows: []SeasonLifecycleRepairWindow{},
		Cursor: strings.TrimSpace(options.AfterStartDate), Conflicts: []string{}, Failures: []string{},
	}
	if !policy.Enabled || !policy.StartedAt(now) {
		summary.State = seasonx.StateNotStarted
		return summary, nil
	}
	windows, total, hasMore, err := seasonRepairWindows(policy, now, summary.Cursor, options.BatchSize)
	if err != nil {
		return repairFailure(summary, err)
	}
	summary.WindowsPlanned = total
	summary.HasMore = hasMore
	candidates := make([]seasonRepairCandidate, 0, len(windows))
	for _, window := range windows {
		candidate := seasonRepairCandidate{window: window}
		rows, findErr := s.svcCtx.SeasonModel.FindByStartDateCandidates(window.StartDate)
		if findErr != nil {
			return repairFailure(summary, findErr)
		}
		detail := SeasonLifecycleRepairWindow{
			Name: window.Name, StartDate: repairWindowKey(window.StartDate), EndDate: repairWindowKey(window.EndDate),
			Exists: len(rows) == 1, Settlement: "not_due",
		}
		if len(rows) == 1 {
			candidate.season = &rows[0]
			detail.SeasonId = rows[0].Id
			detail.Status = rows[0].Status
			if !seasonx.MatchesWindow(rows[0], window, policy.Location) {
				summary.Conflicts = append(summary.Conflicts, fmt.Sprintf("season %q conflicts with deterministic window %s", rows[0].Name, window.Name))
			}
		} else if len(rows) > 1 {
			summary.Conflicts = append(summary.Conflicts, fmt.Sprintf("multiple seasons conflict with deterministic window %s", window.Name))
		}
		summary.Windows = append(summary.Windows, detail)
		candidates = append(candidates, candidate)
	}
	if len(summary.Conflicts) > 0 {
		summary.State = seasonx.StateUnavailable
		summary.Problem = strings.Join(summary.Conflicts, "; ")
		return summary, nil
	}

	for index := range candidates {
		window := candidates[index].window
		detail := &summary.Windows[index]
		endExclusive := window.EndDate.AddDate(0, 0, 1)
		coverageEnd := endExclusive
		if now.Before(coverageEnd) {
			coverageEnd = now
		}
		if coverageEnd.After(window.StartDate) && s.svcCtx.AchievementProgressEventModel != nil {
			events, users, countErr := s.svcCtx.AchievementProgressEventModel.CountUsersBetween(window.StartDate, coverageEnd)
			if countErr != nil {
				return repairFailure(summary, countErr)
			}
			summary.EventsCovered += events
			summary.UsersCovered += users
		}
		if !now.Before(endExclusive) {
			detail.Settlement = "pending"
			if candidates[index].season != nil && s.svcCtx.SeasonSettlementModel != nil {
				settlement, findErr := s.svcCtx.SeasonSettlementModel.FindBySeasonId(candidates[index].season.Id)
				if findErr != nil {
					return repairFailure(summary, findErr)
				}
				if settlement != nil && settlement.Status == model.SeasonSettlementStatusCompleted {
					detail.Settlement = "completed"
				}
			}
			if detail.Settlement != "completed" {
				summary.SeasonsToSettle++
				if s.svcCtx.AchievementProgressEventModel != nil {
					pairs, pairErr := s.svcCtx.AchievementProgressEventModel.CountUserGamesBetween(
						achievementx.SeasonChallengeMetricKeys(), window.StartDate, endExclusive,
					)
					if pairErr != nil {
						return repairFailure(summary, pairErr)
					}
					summary.EstimatedChallengeSnapshots += int(pairs) * len(achievementx.FixedSeasonChallengeDefinitions())
					summary.EstimatedSeasonRecords += int(pairs)
					summary.EstimatedSeasonTitles += estimateSeasonTitlesFromPairCount(int(pairs))
				}
			}
		}
	}
	if options.DryRun {
		for _, candidate := range candidates {
			if candidate.season == nil {
				summary.State = seasonx.StateUnavailable
				summary.Problem = "season schedule is missing required windows"
				break
			}
		}
		if len(windows) > 0 {
			summary.NextCursor = repairWindowKey(windows[len(windows)-1].StartDate)
		}
		return summary, nil
	}
	if s.svcCtx.DB == nil || s.svcCtx.SeasonSettlementModel == nil || s.settle == nil {
		return repairFailure(summary, fmt.Errorf("season lifecycle repair write service is unavailable"))
	}

	for index := range candidates {
		if candidates[index].season != nil {
			continue
		}
		created, createErr := s.svcCtx.SeasonModel.CreateIfAbsentWithTx(nil, seasonPointer(seasonx.WindowSeason(candidates[index].window)))
		if createErr != nil {
			return repairFailure(summary, createErr)
		}
		rows, findErr := s.svcCtx.SeasonModel.FindByStartDateCandidates(candidates[index].window.StartDate)
		if findErr != nil || len(rows) != 1 || !seasonx.MatchesWindow(rows[0], candidates[index].window, policy.Location) {
			if findErr != nil {
				return repairFailure(summary, findErr)
			}
			return repairFailure(summary, fmt.Errorf("season %s missing after lifecycle repair create", candidates[index].window.Name))
		}
		candidates[index].season = &rows[0]
		summary.Windows[index].Exists = true
		summary.Windows[index].SeasonId = rows[0].Id
		summary.Windows[index].Status = rows[0].Status
		summary.Windows[index].Created = created
		if created {
			summary.WindowsCreated++
		}
	}

	for index := range candidates {
		candidate := candidates[index]
		detail := &summary.Windows[index]
		endExclusive := candidate.window.EndDate.AddDate(0, 0, 1)
		if !now.Before(endExclusive) && detail.Settlement != "completed" {
			settled, settleErr := s.settle.SettleSeasonWithOptionsAt(ctx, candidate.season.Id, now, SeasonSettlementOptions{Notify: false})
			if settleErr != nil {
				detail.Settlement = "failed"
				return repairFailure(summary, settleErr)
			}
			if settled.Skipped {
				detail.Settlement = "completed"
			} else {
				detail.Settlement = "settled"
				summary.SeasonsSettled++
				summary.ChallengeSnapshotsWritten += settled.ChallengeSnapshots
				summary.SeasonRecordsWritten += settled.SeasonRecords
				summary.SeasonTitlesGranted += settled.SeasonTitles
			}
		}
		status := lifecycleStatus(candidate.window, now)
		if _, updateErr := s.svcCtx.SeasonModel.UpdateStatusToWithTx(nil, candidate.season.Id, status); updateErr != nil {
			return repairFailure(summary, updateErr)
		}
		detail.Status = status
		summary.NextCursor = repairWindowKey(candidate.window.StartDate)
		if options.Checkpoint != nil {
			if checkpointErr := options.Checkpoint(summary.NextCursor); checkpointErr != nil {
				return repairFailure(summary, checkpointErr)
			}
		}
	}
	if len(candidates) > 0 {
		invalidateSeasonInfoCache(ctx, s.svcCtx)
	}
	return summary, nil
}

func seasonRepairWindows(policy seasonx.Policy, now time.Time, cursor string, batchSize int) ([]seasonx.Window, int, bool, error) {
	current, started := policy.WindowAt(now)
	if !started {
		return []seasonx.Window{}, 0, false, nil
	}
	if batchSize <= 0 {
		batchSize = defaultSeasonRepairBatchSize
	}
	if batchSize > 1000 {
		batchSize = 1000
	}
	total := current.Number - policy.InitialNumber + 2
	start := policy.Anchor
	number := policy.InitialNumber
	if strings.TrimSpace(cursor) != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(cursor), policy.Location)
		if err != nil {
			return nil, total, false, fmt.Errorf("invalid season repair cursor %q: %w", cursor, err)
		}
		months := (parsed.Year()-policy.Anchor.Year())*12 + int(parsed.Month()-policy.Anchor.Month())
		if months < 0 || months%policy.CycleMonths != 0 || !policy.Anchor.AddDate(0, months, 0).Equal(parsed) {
			return nil, total, false, fmt.Errorf("season repair cursor %q is not aligned with policy", cursor)
		}
		start = parsed.AddDate(0, policy.CycleMonths, 0)
		number = policy.InitialNumber + months/policy.CycleMonths + 1
	}
	lastStart := current.StartDate.AddDate(0, policy.CycleMonths, 0)
	windows := make([]seasonx.Window, 0, batchSize)
	for len(windows) < batchSize && !start.After(lastStart) {
		windows = append(windows, seasonx.Window{
			Number: number, Name: fmt.Sprintf("S%d", number), StartDate: start,
			EndDate: start.AddDate(0, policy.CycleMonths, 0).AddDate(0, 0, -1),
		})
		start = start.AddDate(0, policy.CycleMonths, 0)
		number++
	}
	return windows, total, !start.After(lastStart), nil
}

func repairPolicy(policy seasonx.Policy) SeasonLifecycleRepairPolicy {
	anchorDate := ""
	if !policy.Anchor.IsZero() {
		anchorDate = policy.Anchor.Format(time.DateOnly)
	}
	return SeasonLifecycleRepairPolicy{
		AnchorDate: anchorDate, InitialNumber: policy.InitialNumber,
		CycleMonths: policy.CycleMonths, Timezone: policy.Location.String(),
	}
}

func repairWindowKey(value time.Time) string {
	return value.Format(time.DateOnly)
}

func repairFailure(summary *SeasonLifecycleRepairSummary, err error) (*SeasonLifecycleRepairSummary, error) {
	if summary != nil && err != nil {
		summary.Failures = append(summary.Failures, err.Error())
	}
	return summary, err
}

func estimateSeasonTitlesFromPairCount(pairs int) int {
	if pairs <= 0 {
		return 0
	}
	if pairs > 40 {
		return 40
	}
	return pairs
}
