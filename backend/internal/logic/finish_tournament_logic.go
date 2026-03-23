package logic

import (
	"context"
	"sort"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type FinishTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type tournamentSettlement struct {
	FinalRank int
	Status    int
}

func NewFinishTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FinishTournamentLogic {
	return &FinishTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FinishTournamentLogic) FinishTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false}, nil
	}
	if userIdInt <= 0 {
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentId <= 0 {
		return &types.CommonResp{Success: false, Message: "赛事参数无效"}, nil
	}

	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if tournament == nil {
		return &types.CommonResp{Success: false, Message: "赛事不存在"}, nil
	}
	if tournament.CreatorId != userIdInt {
		return &types.CommonResp{Success: false, Message: "仅创建者可结束赛事"}, nil
	}
	if tournament.Status != 1 {
		return &types.CommonResp{Success: false, Message: "赛事不在进行中"}, nil
	}

	participants, err := l.svcCtx.TournamentParticipantModel.FindListByTournamentId(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事参赛者失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事对阵失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	settlement := make(map[int64]tournamentSettlement, len(participants))
	switch tournament.Format {
	case 1:
		settlement = buildSingleEliminationSettlement(participants, matches)
	case 3:
		settlement = buildRoundRobinSettlement(participants, matches)
	default:
		return &types.CommonResp{Success: false, Message: "暂不支持该赛制结算"}, nil
	}

	fallbackRank := len(participants)
	if fallbackRank == 0 {
		fallbackRank = 1
	}

	for _, participant := range participants {
		result, ok := settlement[participant.UserId]
		if !ok {
			result = tournamentSettlement{FinalRank: fallbackRank, Status: 2}
		}
		if result.FinalRank <= 0 {
			result.FinalRank = fallbackRank
		}
		if result.Status != 3 {
			result.Status = 2
		}

		updated, updateErr := l.svcCtx.TournamentParticipantModel.UpdateFinalRankAndStatus(req.TournamentId, participant.UserId, result.FinalRank, result.Status)
		if updateErr != nil {
			l.Logger.Errorf("更新参赛者结算结果失败: tournamentId=%d userId=%d err=%v", req.TournamentId, participant.UserId, updateErr)
			return &types.CommonResp{Success: false, Message: "赛事结算失败"}, nil
		}
		if !updated {
			l.Logger.Errorf("更新参赛者结算结果失败: tournamentId=%d userId=%d not found", req.TournamentId, participant.UserId)
			return &types.CommonResp{Success: false, Message: "赛事结算失败"}, nil
		}
	}

	updated, err := l.svcCtx.TournamentModel.UpdateStatusByCreator(req.TournamentId, userIdInt, 2)
	if err != nil {
		l.Logger.Errorf("更新赛事状态失败: tournamentId=%d err=%v", req.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "赛事结算失败"}, nil
	}
	if !updated {
		return &types.CommonResp{Success: false, Message: "赛事结算失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "赛事已结束"}, nil
}

func buildSingleEliminationSettlement(participants []model.TournamentParticipant, matches []model.TournamentMatch) map[int64]tournamentSettlement {
	result := make(map[int64]tournamentSettlement, len(participants))
	if len(participants) == 0 {
		return result
	}

	fallbackRank := len(participants)
	for _, participant := range participants {
		result[participant.UserId] = tournamentSettlement{FinalRank: fallbackRank, Status: 2}
	}

	if len(participants) == 1 {
		result[participants[0].UserId] = tournamentSettlement{FinalRank: 1, Status: 3}
		return result
	}

	maxRound := 0
	for _, tm := range matches {
		if tm.RoundNumber > maxRound {
			maxRound = tm.RoundNumber
		}
	}

	eliminatedRound := make(map[int64]int)
	for _, tm := range matches {
		if tm.WinnerId <= 0 {
			continue
		}
		if tm.WinnerId != tm.Player1Id && tm.WinnerId != tm.Player2Id {
			continue
		}

		loserId := int64(0)
		if tm.WinnerId == tm.Player1Id {
			loserId = tm.Player2Id
		} else {
			loserId = tm.Player1Id
		}

		if loserId > 0 && tm.RoundNumber > eliminatedRound[loserId] {
			eliminatedRound[loserId] = tm.RoundNumber
		}
	}

	championId := int64(0)
	runnerUpId := int64(0)
	finalMatch := findTournamentFinalMatch(matches, maxRound)
	if finalMatch != nil && finalMatch.WinnerId > 0 {
		championId = finalMatch.WinnerId
		if finalMatch.WinnerId == finalMatch.Player1Id {
			runnerUpId = finalMatch.Player2Id
		} else if finalMatch.WinnerId == finalMatch.Player2Id {
			runnerUpId = finalMatch.Player1Id
		}
	}

	if championId == 0 {
		for _, participant := range participants {
			if _, eliminated := eliminatedRound[participant.UserId]; !eliminated {
				championId = participant.UserId
				break
			}
		}
	}

	if championId > 0 {
		result[championId] = tournamentSettlement{FinalRank: 1, Status: 3}
	}

	if runnerUpId == 0 && maxRound > 0 {
		for _, participant := range participants {
			if participant.UserId == championId {
				continue
			}
			if eliminatedRound[participant.UserId] == maxRound {
				runnerUpId = participant.UserId
				break
			}
		}
	}

	if runnerUpId > 0 && runnerUpId != championId {
		result[runnerUpId] = tournamentSettlement{FinalRank: 2, Status: 2}
	}

	for userId, roundNumber := range eliminatedRound {
		if userId == championId || userId == runnerUpId {
			continue
		}

		rank := fallbackRank
		if maxRound > 0 && roundNumber > 0 && roundNumber <= maxRound {
			rank = (1 << (maxRound - roundNumber)) + 1
			if rank > fallbackRank {
				rank = fallbackRank
			}
		}

		result[userId] = tournamentSettlement{FinalRank: rank, Status: 2}
	}

	return result
}

func buildRoundRobinSettlement(participants []model.TournamentParticipant, matches []model.TournamentMatch) map[int64]tournamentSettlement {
	result := make(map[int64]tournamentSettlement, len(participants))
	if len(participants) == 0 {
		return result
	}

	wins := make(map[int64]int, len(participants))
	for _, participant := range participants {
		wins[participant.UserId] = 0
	}

	for _, tm := range matches {
		if tm.WinnerId > 0 {
			wins[tm.WinnerId]++
		}
	}

	sortedParticipants := make([]model.TournamentParticipant, len(participants))
	copy(sortedParticipants, participants)

	sort.SliceStable(sortedParticipants, func(i, j int) bool {
		leftWins := wins[sortedParticipants[i].UserId]
		rightWins := wins[sortedParticipants[j].UserId]
		if leftWins != rightWins {
			return leftWins > rightWins
		}
		if sortedParticipants[i].Seed != sortedParticipants[j].Seed {
			return sortedParticipants[i].Seed < sortedParticipants[j].Seed
		}
		if !sortedParticipants[i].CreatedAt.Equal(sortedParticipants[j].CreatedAt) {
			return sortedParticipants[i].CreatedAt.Before(sortedParticipants[j].CreatedAt)
		}
		return sortedParticipants[i].UserId < sortedParticipants[j].UserId
	})

	for index, participant := range sortedParticipants {
		status := 2
		if index == 0 {
			status = 3
		}
		result[participant.UserId] = tournamentSettlement{FinalRank: index + 1, Status: status}
	}

	return result
}

func findTournamentFinalMatch(matches []model.TournamentMatch, maxRound int) *model.TournamentMatch {
	if maxRound <= 0 {
		return nil
	}

	var finalMatch *model.TournamentMatch
	for i := range matches {
		tm := &matches[i]
		if tm.RoundNumber != maxRound {
			continue
		}
		if finalMatch == nil || tm.MatchOrder < finalMatch.MatchOrder {
			finalMatch = tm
		}
	}

	return finalMatch
}
