package tournament

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	tournamentBracketWorkerBatchSize = 20
	tournamentBracketWorkerInterval  = time.Minute
)

// EnsureTournamentBracket creates an active tournament's bracket once. The
// tournament row lock makes repeated worker runs and concurrent instances safe.
func EnsureTournamentBracket(svcCtx *svc.ServiceContext, tournamentID int64) error {
	if svcCtx == nil || svcCtx.DB == nil || tournamentID <= 0 {
		return nil
	}
	return svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var tournament model.Tournament
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tournament, tournamentID).Error; err != nil {
			return err
		}
		if tournament.Status != 1 {
			return nil
		}
		var matchCount int64
		if err := tx.Model(&model.TournamentMatch{}).Where("tournament_id = ?", tournamentID).Count(&matchCount).Error; err != nil {
			return err
		}
		if matchCount > 0 {
			return nil
		}
		var participants []model.TournamentParticipant
		if err := tx.Where("tournament_id = ?", tournamentID).Order("seed ASC, created_at ASC, id ASC").Find(&participants).Error; err != nil {
			return err
		}
		matches, err := tournamentBracketMatches(&tournament, participants)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return nil
		}
		return tx.Create(&matches).Error
	})
}

func tournamentBracketMatches(tournament *model.Tournament, participants []model.TournamentParticipant) ([]model.TournamentMatch, error) {
	switch tournament.Format {
	case 1:
		return buildSingleEliminationBracket(tournament.Id, participants), nil
	case 3:
		return buildRoundRobinBracket(tournament.Id, participants), nil
	default:
		return nil, fmt.Errorf("unsupported tournament format: %d", tournament.Format)
	}
}

type TournamentBracketGenerationWorker struct {
	svcCtx *svc.ServiceContext
}

func NewTournamentBracketGenerationWorker(svcCtx *svc.ServiceContext) *TournamentBracketGenerationWorker {
	return &TournamentBracketGenerationWorker{svcCtx: svcCtx}
}

func (w *TournamentBracketGenerationWorker) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(tournamentBracketWorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			logx.Errorf("赛事对阵生成任务失败: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *TournamentBracketGenerationWorker) RunOnce(contexts ...context.Context) error {
	ctx := context.Background()
	if len(contexts) > 0 && contexts[0] != nil {
		ctx = contexts[0]
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if w == nil || w.svcCtx == nil || w.svcCtx.TournamentModel == nil {
		return nil
	}
	ids, err := w.svcCtx.TournamentModel.ListActiveWithoutBracket(tournamentBracketWorkerBatchSize)
	if err != nil {
		return err
	}
	var firstErr error
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := EnsureTournamentBracket(w.svcCtx, id); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("tournamentId=%d: %w", id, err)
		}
	}
	return firstErr
}

func buildSingleEliminationBracket(tournamentId int64, participants []model.TournamentParticipant) []model.TournamentMatch {
	if len(participants) <= 1 {
		return []model.TournamentMatch{}
	}

	totalRounds := 0
	totalSlots := 1
	for totalSlots < len(participants) {
		totalSlots <<= 1
		totalRounds++
	}

	slots := make([]int64, totalSlots)
	for i, participant := range participants {
		slots[i] = participant.UserId
	}

	type bracketState struct {
		hasPotential bool
		winnerKnown  bool
		winnerId     int64
	}

	roundStates := make([][]bracketState, totalRounds+1)
	allMatches := make([]*model.TournamentMatch, 0, totalSlots-1)
	for round := 1; round <= totalRounds; round++ {
		matchCount := totalSlots >> round
		roundStates[round] = make([]bracketState, matchCount)
		for order := 1; order <= matchCount; order++ {
			tm := &model.TournamentMatch{TournamentId: tournamentId, RoundNumber: round, MatchOrder: order, BracketPosition: fmt.Sprintf("R%d-M%d", round, order), Status: 0}
			state := bracketState{}
			if round == 1 {
				slotIndex := (order - 1) * 2
				tm.Player1Id, tm.Player2Id = slots[slotIndex], slots[slotIndex+1]
				hasPlayer1, hasPlayer2 := tm.Player1Id > 0, tm.Player2Id > 0
				state.hasPotential = hasPlayer1 || hasPlayer2
				if hasPlayer1 && !hasPlayer2 {
					state.winnerKnown, state.winnerId, tm.WinnerId, tm.Status = true, tm.Player1Id, tm.Player1Id, 2
				} else if !hasPlayer1 && hasPlayer2 {
					state.winnerKnown, state.winnerId, tm.WinnerId, tm.Status = true, tm.Player2Id, tm.Player2Id, 2
				}
			} else {
				leftState := roundStates[round-1][(order-1)*2]
				rightState := roundStates[round-1][(order-1)*2+1]
				state.hasPotential = leftState.hasPotential || rightState.hasPotential
				if leftState.winnerKnown {
					tm.Player1Id = leftState.winnerId
				}
				if rightState.winnerKnown {
					tm.Player2Id = rightState.winnerId
				}
				if leftState.hasPotential && !rightState.hasPotential && leftState.winnerKnown {
					state.winnerKnown, state.winnerId, tm.WinnerId, tm.Status = true, leftState.winnerId, leftState.winnerId, 2
				} else if !leftState.hasPotential && rightState.hasPotential && rightState.winnerKnown {
					state.winnerKnown, state.winnerId, tm.WinnerId, tm.Status = true, rightState.winnerId, rightState.winnerId, 2
				}
			}
			roundStates[round][order-1] = state
			allMatches = append(allMatches, tm)
		}
	}
	result := make([]model.TournamentMatch, 0, len(allMatches))
	for _, tm := range allMatches {
		result = append(result, *tm)
	}
	return result
}

func buildRoundRobinBracket(tournamentId int64, participants []model.TournamentParticipant) []model.TournamentMatch {
	if len(participants) <= 1 {
		return []model.TournamentMatch{}
	}
	matches := make([]model.TournamentMatch, 0, len(participants)*(len(participants)-1)/2)
	order := 1
	for i := range participants {
		for j := i + 1; j < len(participants); j++ {
			matches = append(matches, model.TournamentMatch{TournamentId: tournamentId, RoundNumber: 1, MatchOrder: order, Player1Id: participants[i].UserId, Player2Id: participants[j].UserId, BracketPosition: fmt.Sprintf("R1-M%d", order), Status: 0})
			order++
		}
	}
	return matches
}
