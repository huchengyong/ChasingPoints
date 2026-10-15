package logic

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

const competitiveReadModelRebuildJobName = "competitive-read-model-v1"

type CompetitiveReadModelRebuildOptions struct {
	BatchSize  int
	MaxBatches int
	RateLimit  time.Duration
	DryRun     bool
}

type CompetitiveReadModelRebuildSummary struct {
	DryRun                      bool
	Paused                      bool
	Batches                     int
	BackfillBatches             int
	MatchesScanned              int
	CompletionTimesBackfilled   int
	ParticipantRows             int
	CompetitiveUsers            int
	LastCursorMatchID           int64
	LastCursorCompletedAt       *time.Time
	CompletedAtBackfillComplete bool
}

type CompetitiveReadModelRebuildService struct {
	svcCtx *svc.ServiceContext
}

func NewCompetitiveReadModelRebuildService(svcCtx *svc.ServiceContext) *CompetitiveReadModelRebuildService {
	return &CompetitiveReadModelRebuildService{svcCtx: svcCtx}
}

func (s *CompetitiveReadModelRebuildService) Pause() error {
	checkpoint, err := s.checkpoint()
	if err != nil {
		return err
	}
	return s.saveCheckpoint(rebuildCheckpointState(checkpoint), true, "")
}

func (s *CompetitiveReadModelRebuildService) Resume() error {
	checkpoint, err := s.checkpoint()
	if err != nil {
		return err
	}
	return s.saveCheckpoint(rebuildCheckpointState(checkpoint), false, "")
}

func (s *CompetitiveReadModelRebuildService) DryRun(ctx context.Context, options CompetitiveReadModelRebuildOptions) (*CompetitiveReadModelRebuildSummary, error) {
	options.DryRun = true
	return s.Rebuild(ctx, options)
}

func (s *CompetitiveReadModelRebuildService) Rebuild(ctx context.Context, options CompetitiveReadModelRebuildOptions) (*CompetitiveReadModelRebuildSummary, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.MatchModel == nil || s.svcCtx.CompetitiveReadModel == nil {
		return nil, fmt.Errorf("competitive read model rebuild infrastructure is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	options = normalizeCompetitiveReadModelRebuildOptions(options)
	if options.DryRun {
		return s.dryRun(ctx, options)
	}

	checkpoint, err := s.checkpoint()
	if err != nil {
		return nil, err
	}
	state := rebuildCheckpointState(checkpoint)
	summary := rebuildSummaryFromCheckpoint(state)
	if summary.Paused {
		return summary, nil
	}

	remainingBatches := options.MaxBatches
	if !state.BackfillCompleted {
		if err := s.backfillCompletedAt(ctx, options, state, summary, &remainingBatches); err != nil {
			return summary, err
		}
		if !state.BackfillCompleted || remainingBatches == 0 && options.MaxBatches > 0 {
			return summary, nil
		}
	}
	return s.projectCompletedMatches(ctx, options, state, summary, &remainingBatches)
}

func (s *CompetitiveReadModelRebuildService) dryRun(ctx context.Context, options CompetitiveReadModelRebuildOptions) (*CompetitiveReadModelRebuildSummary, error) {
	summary := &CompetitiveReadModelRebuildSummary{DryRun: true}
	cursor := int64(0)
	for options.MaxBatches == 0 || summary.BackfillBatches < options.MaxBatches {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		matches, err := s.svcCtx.MatchModel.ListCompletedByIDAfterWithTx(s.svcCtx.DB.WithContext(ctx), cursor, options.BatchSize)
		if err != nil {
			return summary, err
		}
		if len(matches) == 0 {
			summary.CompletedAtBackfillComplete = true
			break
		}
		for index := range matches {
			match := matches[index]
			cursor = match.Id
			summary.MatchesScanned++
			if match.CompletedAt == nil {
				summary.CompletionTimesBackfilled++
			}
			summary.ParticipantRows++
			if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
				summary.ParticipantRows++
			}
			if isEligibleCompetitiveProjection(&match) {
				summary.CompetitiveUsers++
				if match.OpponentId != nil && *match.OpponentId > 0 {
					summary.CompetitiveUsers++
				}
			}
		}
		summary.BackfillBatches++
		summary.LastCursorMatchID = cursor
		if len(matches) < options.BatchSize {
			summary.CompletedAtBackfillComplete = true
			break
		}
		if err := waitBetweenRebuildBatches(ctx, options.RateLimit); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func (s *CompetitiveReadModelRebuildService) backfillCompletedAt(
	ctx context.Context,
	options CompetitiveReadModelRebuildOptions,
	state *model.CompetitiveReadModelRebuildCheckpoint,
	summary *CompetitiveReadModelRebuildSummary,
	remainingBatches *int,
) error {
	for hasRebuildBatchBudget(options, *remainingBatches) {
		if err := ctx.Err(); err != nil {
			return err
		}
		matches, err := s.svcCtx.MatchModel.ListCompletedByIDAfterWithTx(s.svcCtx.DB.WithContext(ctx), state.BackfillCursorMatchId, options.BatchSize)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			state.BackfillCompleted = true
			if err := s.saveCheckpoint(state, false, ""); err != nil {
				return err
			}
			summary.CompletedAtBackfillComplete = true
			return nil
		}

		for index := range matches {
			match := matches[index]
			if match.CompletedAt == nil {
				completedAt := resolveMatchCompletedAt(&match)
				if err := s.backfillMatchCompletedAt(ctx, match.Id, completedAt); err != nil {
					_ = s.saveCheckpoint(state, false, err.Error())
					return err
				}
				summary.CompletionTimesBackfilled++
			}
			state.BackfillCursorMatchId = match.Id
			summary.LastCursorMatchID = state.BackfillCursorMatchId
			summary.MatchesScanned++
		}
		summary.BackfillBatches++
		consumeRebuildBatch(remainingBatches)
		if len(matches) < options.BatchSize {
			state.BackfillCompleted = true
			summary.CompletedAtBackfillComplete = true
		}
		if err := s.saveCheckpoint(state, false, ""); err != nil {
			return err
		}
		if state.BackfillCompleted {
			return nil
		}
		if err := waitBetweenRebuildBatches(ctx, options.RateLimit); err != nil {
			return err
		}
	}
	return nil
}

func (s *CompetitiveReadModelRebuildService) projectCompletedMatches(
	ctx context.Context,
	options CompetitiveReadModelRebuildOptions,
	state *model.CompetitiveReadModelRebuildCheckpoint,
	summary *CompetitiveReadModelRebuildSummary,
	remainingBatches *int,
) (*CompetitiveReadModelRebuildSummary, error) {
	for hasRebuildBatchBudget(options, *remainingBatches) {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		matches, err := s.svcCtx.MatchModel.ListCompletedByCompletionAfterWithTx(s.svcCtx.DB.WithContext(ctx), state.CursorCompletedAt, state.CursorMatchId, options.BatchSize)
		if err != nil {
			return summary, err
		}
		if len(matches) == 0 {
			return summary, nil
		}

		for index := range matches {
			match := matches[index]
			result, projectErr := s.projectMatch(ctx, &match)
			if projectErr != nil {
				_ = s.saveCheckpoint(state, false, projectErr.Error())
				return summary, projectErr
			}
			advanceRebuildProjectionCursor(state, &match)
			summary.MatchesScanned++
			summary.ParticipantRows += len(result.AppliedUsers)
			if isEligibleCompetitiveProjection(&match) {
				summary.CompetitiveUsers += len(result.AppliedUsers)
			}
		}
		summary.Batches++
		consumeRebuildBatch(remainingBatches)
		summary.LastCursorMatchID = state.CursorMatchId
		summary.LastCursorCompletedAt = copyTimePointer(state.CursorCompletedAt)
		if err := s.saveCheckpoint(state, false, ""); err != nil {
			return summary, err
		}
		if len(matches) < options.BatchSize {
			return summary, nil
		}
		if err := waitBetweenRebuildBatches(ctx, options.RateLimit); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func (s *CompetitiveReadModelRebuildService) backfillMatchCompletedAt(ctx context.Context, matchID int64, completedAt time.Time) error {
	return s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.svcCtx.MatchModel.SetCompletedAtIfEmptyWithTx(tx, matchID, completedAt)
	})
}

func (s *CompetitiveReadModelRebuildService) projectMatch(ctx context.Context, match *model.Match) (CompetitiveProjectionResult, error) {
	if match == nil || match.CompletedAt == nil {
		return CompetitiveProjectionResult{}, fmt.Errorf("match completion time is not backfilled")
	}
	result := CompetitiveProjectionResult{}
	err := s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		input, err := s.buildProjectionInput(match)
		if err != nil {
			return err
		}
		result, err = NewCompetitiveProjector(s.svcCtx).ProjectWithTx(tx, input)
		return err
	})
	return result, err
}

func (s *CompetitiveReadModelRebuildService) buildProjectionInput(match *model.Match) (CompetitiveProjectionInput, error) {
	input := CompetitiveProjectionInput{Match: match}
	if !isEligibleCompetitiveProjection(match) || s.svcCtx.RankingModel == nil {
		return input, nil
	}
	logs, err := s.svcCtx.RankingModel.ListRankChangesByMatchAndGameType(match.Id, match.GameType)
	if err != nil {
		return input, err
	}
	for _, rankLog := range logs {
		before := &model.UserRanking{UserId: rankLog.UserId, GameType: rankLog.GameType, RankScore: rankLog.BeforeScore, RankLevel: rankLog.BeforeLevel}
		after := &model.UserRanking{UserId: rankLog.UserId, GameType: rankLog.GameType, RankScore: rankLog.AfterScore, RankLevel: rankLog.AfterLevel}
		switch rankLog.UserId {
		case match.UserId:
			input.Player1Before, input.Player1After = before, after
		case opponentID(match):
			input.Player2Before, input.Player2After = before, after
		}
	}
	return input, nil
}

func isEligibleCompetitiveProjection(match *model.Match) bool {
	return match != nil && match.Result != nil && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked && (*match.Result == 1 || *match.Result == 2)
}

func opponentID(match *model.Match) int64 {
	if match == nil || match.OpponentId == nil {
		return 0
	}
	return *match.OpponentId
}

func (s *CompetitiveReadModelRebuildService) checkpoint() (*model.CompetitiveReadModelRebuildCheckpoint, error) {
	return s.svcCtx.CompetitiveReadModel.FindCheckpoint(competitiveReadModelRebuildJobName)
}

func (s *CompetitiveReadModelRebuildService) saveCheckpoint(state *model.CompetitiveReadModelRebuildCheckpoint, paused bool, lastError string) error {
	checkpoint := rebuildCheckpointState(state)
	checkpoint.Paused = paused
	checkpoint.LastError = lastError
	return s.svcCtx.CompetitiveReadModel.UpsertCheckpoint(checkpoint)
}

func rebuildCheckpointState(checkpoint *model.CompetitiveReadModelRebuildCheckpoint) *model.CompetitiveReadModelRebuildCheckpoint {
	if checkpoint == nil {
		return &model.CompetitiveReadModelRebuildCheckpoint{JobName: competitiveReadModelRebuildJobName}
	}
	copy := *checkpoint
	copy.CursorCompletedAt = copyTimePointer(checkpoint.CursorCompletedAt)
	return &copy
}

func rebuildSummaryFromCheckpoint(checkpoint *model.CompetitiveReadModelRebuildCheckpoint) *CompetitiveReadModelRebuildSummary {
	return &CompetitiveReadModelRebuildSummary{
		Paused:                      checkpoint.Paused,
		LastCursorMatchID:           checkpoint.CursorMatchId,
		LastCursorCompletedAt:       copyTimePointer(checkpoint.CursorCompletedAt),
		CompletedAtBackfillComplete: checkpoint.BackfillCompleted,
	}
}

func advanceRebuildProjectionCursor(checkpoint *model.CompetitiveReadModelRebuildCheckpoint, match *model.Match) {
	if checkpoint == nil || match == nil || match.CompletedAt == nil {
		return
	}
	completedAt := *match.CompletedAt
	checkpoint.CursorCompletedAt = &completedAt
	checkpoint.CursorMatchId = match.Id
}

func copyTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func hasRebuildBatchBudget(options CompetitiveReadModelRebuildOptions, remaining int) bool {
	return options.MaxBatches == 0 || remaining > 0
}

func consumeRebuildBatch(remaining *int) {
	if remaining != nil && *remaining > 0 {
		*remaining--
	}
}

func waitBetweenRebuildBatches(ctx context.Context, rateLimit time.Duration) error {
	if rateLimit <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(rateLimit):
		return nil
	}
}

func normalizeCompetitiveReadModelRebuildOptions(options CompetitiveReadModelRebuildOptions) CompetitiveReadModelRebuildOptions {
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}
	if options.BatchSize > 1000 {
		options.BatchSize = 1000
	}
	if options.MaxBatches < 0 {
		options.MaxBatches = 0
	}
	if options.RateLimit < 0 {
		options.RateLimit = 0
	}
	return options
}
