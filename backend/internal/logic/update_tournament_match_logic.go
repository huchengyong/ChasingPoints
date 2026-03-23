package logic

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTournamentMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTournamentMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTournamentMatchLogic {
	return &UpdateTournamentMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTournamentMatchLogic) UpdateTournamentMatch(req *types.UpdateTournamentMatchReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false}, nil
	}
	if userIdInt <= 0 {
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentMatchId <= 0 || req.WinnerId <= 0 {
		return &types.CommonResp{Success: false, Message: "参数无效"}, nil
	}

	tournamentMatch, err := l.svcCtx.TournamentMatchModel.FindById(req.TournamentMatchId)
	if err != nil {
		l.Logger.Errorf("查询赛事对局失败: matchId=%d err=%v", req.TournamentMatchId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if tournamentMatch == nil {
		return &types.CommonResp{Success: false, Message: "赛事对局不存在"}, nil
	}

	tournament, err := l.svcCtx.TournamentModel.FindById(tournamentMatch.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", tournamentMatch.TournamentId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if tournament == nil {
		return &types.CommonResp{Success: false, Message: "赛事不存在"}, nil
	}
	if tournament.CreatorId != userIdInt {
		return &types.CommonResp{Success: false, Message: "仅创建者可更新对局"}, nil
	}
	if tournament.Status != 1 {
		return &types.CommonResp{Success: false, Message: "赛事不在进行中"}, nil
	}
	if tournamentMatch.Status == 2 {
		return &types.CommonResp{Success: false, Message: "该对局已完成"}, nil
	}
	if req.WinnerId != tournamentMatch.Player1Id && req.WinnerId != tournamentMatch.Player2Id {
		return &types.CommonResp{Success: false, Message: "胜者不在对阵双方中"}, nil
	}

	if err := l.svcCtx.TournamentMatchModel.UpdateWinner(tournamentMatch.Id, req.WinnerId, 2); err != nil {
		l.Logger.Errorf("更新赛事对局胜者失败: matchId=%d winnerId=%d err=%v", tournamentMatch.Id, req.WinnerId, err)
		return &types.CommonResp{Success: false, Message: "更新对局失败"}, nil
	}

	if req.MatchId > 0 {
		if err := l.svcCtx.TournamentMatchModel.UpdateMatchId(tournamentMatch.Id, req.MatchId); err != nil {
			l.Logger.Errorf("更新赛事对局关联比赛ID失败: matchId=%d linkedMatchId=%d err=%v", tournamentMatch.Id, req.MatchId, err)
			return &types.CommonResp{Success: false, Message: "更新对局失败"}, nil
		}
	}

	if tournament.Format == 1 {
		if err := l.advanceSingleEliminationWinner(tournamentMatch.TournamentId, tournamentMatch.RoundNumber, tournamentMatch.MatchOrder, req.WinnerId); err != nil {
			l.Logger.Errorf("推进淘汰赛下一轮失败: tournamentId=%d matchId=%d winnerId=%d err=%v", tournamentMatch.TournamentId, tournamentMatch.Id, req.WinnerId, err)
			return &types.CommonResp{Success: false, Message: "更新对局失败"}, nil
		}
	}

	return &types.CommonResp{Success: true, Message: "更新成功"}, nil
}

func (l *UpdateTournamentMatchLogic) advanceSingleEliminationWinner(tournamentId int64, roundNumber, matchOrder int, winnerId int64) error {
	if winnerId <= 0 {
		return nil
	}

	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(tournamentId)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}

	matchMap := make(map[int]map[int]*model.TournamentMatch)
	for i := range matches {
		tm := &matches[i]
		if _, ok := matchMap[tm.RoundNumber]; !ok {
			matchMap[tm.RoundNumber] = make(map[int]*model.TournamentMatch)
		}
		matchMap[tm.RoundNumber][tm.MatchOrder] = tm
	}

	currentRound := roundNumber
	currentOrder := matchOrder
	currentWinner := winnerId

	for {
		nextRound := currentRound + 1
		roundMatches, ok := matchMap[nextRound]
		if !ok || len(roundMatches) == 0 {
			return nil
		}

		nextOrder := (currentOrder + 1) / 2
		nextMatch, ok := roundMatches[nextOrder]
		if !ok || nextMatch == nil {
			return nil
		}

		playerField := "player1_id"
		siblingOrder := currentOrder + 1
		if currentOrder%2 == 0 {
			playerField = "player2_id"
			siblingOrder = currentOrder - 1
		}

		if err := l.svcCtx.TournamentMatchModel.UpdatePlayer(nextMatch.Id, playerField, currentWinner); err != nil {
			return err
		}

		if playerField == "player1_id" {
			nextMatch.Player1Id = currentWinner
		} else {
			nextMatch.Player2Id = currentWinner
		}

		memo := make(map[[2]int]bool)
		if tournamentMatchBranchHasPotential(matchMap, currentRound, siblingOrder, memo) {
			return nil
		}

		if err := l.svcCtx.TournamentMatchModel.UpdateWinner(nextMatch.Id, currentWinner, 2); err != nil {
			return err
		}
		nextMatch.WinnerId = currentWinner
		nextMatch.Status = 2

		currentRound = nextRound
		currentOrder = nextOrder
	}
}

func tournamentMatchBranchHasPotential(matchMap map[int]map[int]*model.TournamentMatch, roundNumber, matchOrder int, memo map[[2]int]bool) bool {
	key := [2]int{roundNumber, matchOrder}
	if value, ok := memo[key]; ok {
		return value
	}

	roundMatches, ok := matchMap[roundNumber]
	if !ok {
		memo[key] = false
		return false
	}

	tm, ok := roundMatches[matchOrder]
	if !ok || tm == nil {
		memo[key] = false
		return false
	}

	if roundNumber == 1 {
		hasPotential := tm.Player1Id > 0 || tm.Player2Id > 0
		memo[key] = hasPotential
		return hasPotential
	}

	if tm.WinnerId > 0 || tm.Player1Id > 0 || tm.Player2Id > 0 {
		memo[key] = true
		return true
	}

	leftPotential := tournamentMatchBranchHasPotential(matchMap, roundNumber-1, matchOrder*2-1, memo)
	rightPotential := tournamentMatchBranchHasPotential(matchMap, roundNumber-1, matchOrder*2, memo)
	hasPotential := leftPotential || rightPotential
	memo[key] = hasPotential
	return hasPotential
}
