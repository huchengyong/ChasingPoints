package logic

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentBracketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事对阵图
func NewGetTournamentBracketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentBracketLogic {
	return &GetTournamentBracketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentBracketLogic) GetTournamentBracket(req *types.GetTournamentBracketReq) (resp *types.GetTournamentBracketResp, err error) {
	if req.TournamentId <= 0 {
		return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
	}

	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
	}
	if tournament == nil {
		return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
	}

	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事对阵失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
	}

	if len(matches) == 0 && tournament.Status == 1 {
		if err := l.generateBracket(tournament); err != nil {
			l.Logger.Errorf("生成赛事对阵失败: tournamentId=%d format=%d err=%v", req.TournamentId, tournament.Format, err)
			return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
		}

		matches, err = l.svcCtx.TournamentMatchModel.FindByTournament(req.TournamentId)
		if err != nil {
			l.Logger.Errorf("查询赛事对阵失败: tournamentId=%d err=%v", req.TournamentId, err)
			return &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}, nil
		}
	}

	items := make([]types.TournamentBracketMatch, 0, len(matches))
	totalRounds := 0
	nameCache := make(map[int64]string)

	for _, tm := range matches {
		if tm.RoundNumber > totalRounds {
			totalRounds = tm.RoundNumber
		}

		items = append(items, types.TournamentBracketMatch{
			Id:              tm.Id,
			RoundNumber:     tm.RoundNumber,
			MatchOrder:      tm.MatchOrder,
			Player1Id:       tm.Player1Id,
			Player1Name:     l.getTournamentPlayerName(tm.Player1Id, nameCache),
			Player2Id:       tm.Player2Id,
			Player2Name:     l.getTournamentPlayerName(tm.Player2Id, nameCache),
			WinnerId:        tm.WinnerId,
			BracketPosition: tm.BracketPosition,
			Status:          tm.Status,
		})
	}

	return &types.GetTournamentBracketResp{
		Success:     true,
		Matches:     items,
		TotalRounds: totalRounds,
	}, nil
}

func (l *GetTournamentBracketLogic) generateBracket(tournament *model.Tournament) error {
	participants, err := l.svcCtx.TournamentParticipantModel.FindListByTournamentId(tournament.Id)
	if err != nil {
		return err
	}

	if err := l.svcCtx.TournamentMatchModel.DeleteByTournament(tournament.Id); err != nil {
		return err
	}

	var matches []model.TournamentMatch
	switch tournament.Format {
	case 1:
		matches = buildSingleEliminationBracket(tournament.Id, participants)
	case 3:
		matches = buildRoundRobinBracket(tournament.Id, participants)
	default:
		return fmt.Errorf("unsupported tournament format: %d", tournament.Format)
	}

	return l.svcCtx.TournamentMatchModel.CreateBatch(matches)
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

	roundMatches := make([][]*model.TournamentMatch, totalRounds+1)
	roundStates := make([][]bracketState, totalRounds+1)
	allMatches := make([]*model.TournamentMatch, 0, totalSlots-1)

	for round := 1; round <= totalRounds; round++ {
		matchCount := totalSlots >> round
		roundMatches[round] = make([]*model.TournamentMatch, matchCount)
		roundStates[round] = make([]bracketState, matchCount)

		for order := 1; order <= matchCount; order++ {
			tm := &model.TournamentMatch{
				TournamentId:    tournamentId,
				RoundNumber:     round,
				MatchOrder:      order,
				BracketPosition: fmt.Sprintf("R%d-M%d", round, order),
				Status:          0,
			}

			state := bracketState{}
			if round == 1 {
				slotIndex := (order - 1) * 2
				tm.Player1Id = slots[slotIndex]
				tm.Player2Id = slots[slotIndex+1]

				hasPlayer1 := tm.Player1Id > 0
				hasPlayer2 := tm.Player2Id > 0
				state.hasPotential = hasPlayer1 || hasPlayer2

				if hasPlayer1 && !hasPlayer2 {
					state.winnerKnown = true
					state.winnerId = tm.Player1Id
					tm.WinnerId = tm.Player1Id
					tm.Status = 2
				} else if !hasPlayer1 && hasPlayer2 {
					state.winnerKnown = true
					state.winnerId = tm.Player2Id
					tm.WinnerId = tm.Player2Id
					tm.Status = 2
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
					state.winnerKnown = true
					state.winnerId = leftState.winnerId
					tm.WinnerId = leftState.winnerId
					tm.Status = 2
				} else if !leftState.hasPotential && rightState.hasPotential && rightState.winnerKnown {
					state.winnerKnown = true
					state.winnerId = rightState.winnerId
					tm.WinnerId = rightState.winnerId
					tm.Status = 2
				}
			}

			roundMatches[round][order-1] = tm
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

	count := len(participants)
	matches := make([]model.TournamentMatch, 0, count*(count-1)/2)
	order := 1

	for i := 0; i < count; i++ {
		for j := i + 1; j < count; j++ {
			matches = append(matches, model.TournamentMatch{
				TournamentId:    tournamentId,
				RoundNumber:     1,
				MatchOrder:      order,
				Player1Id:       participants[i].UserId,
				Player2Id:       participants[j].UserId,
				BracketPosition: fmt.Sprintf("R1-M%d", order),
				Status:          0,
			})
			order++
		}
	}

	return matches
}

func (l *GetTournamentBracketLogic) getTournamentPlayerName(userId int64, cache map[int64]string) string {
	if userId <= 0 {
		return ""
	}

	if name, ok := cache[userId]; ok {
		return name
	}

	name := ""
	if user, err := l.svcCtx.UserModel.FindById(userId); err == nil && user != nil {
		name = user.Nickname
	}

	cache[userId] = name
	return name
}
