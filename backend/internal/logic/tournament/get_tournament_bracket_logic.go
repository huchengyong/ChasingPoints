package tournament

import (
	"context"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetTournamentBracketLogic) GetTournamentBracket(req *types.GetTournamentBracketReq) (resp *types.GetTournamentBracketResp, err error) {
	empty := &types.GetTournamentBracketResp{Success: false, Matches: []types.TournamentBracketMatch{}}
	if req == nil || req.TournamentId <= 0 || l.svcCtx == nil || l.svcCtx.TournamentModel == nil || l.svcCtx.TournamentMatchModel == nil {
		return empty, nil
	}
	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil || tournament == nil {
		if err != nil {
			l.Logger.Errorf("查询赛事失败: tournamentId=%d err=%v", req.TournamentId, err)
		}
		return empty, nil
	}
	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事对阵失败: tournamentId=%d err=%v", req.TournamentId, err)
		return empty, nil
	}
	userIDs := make([]int64, 0, len(matches)*2)
	seen := make(map[int64]struct{})
	for _, match := range matches {
		for _, userID := range []int64{match.Player1Id, match.Player2Id} {
			if userID > 0 {
				if _, exists := seen[userID]; !exists {
					seen[userID] = struct{}{}
					userIDs = append(userIDs, userID)
				}
			}
		}
	}
	names := map[int64]string{}
	if len(userIDs) > 0 && l.svcCtx.UserModel != nil {
		if users, usersErr := l.svcCtx.UserModel.FindByIds(userIDs); usersErr != nil {
			l.Logger.Errorf("查询赛事选手资料失败: tournamentId=%d err=%v", req.TournamentId, usersErr)
		} else {
			for id, user := range users {
				names[id] = user.Nickname
			}
		}
	}
	items := make([]types.TournamentBracketMatch, 0, len(matches))
	totalRounds := 0
	for _, match := range matches {
		if match.RoundNumber > totalRounds {
			totalRounds = match.RoundNumber
		}
		items = append(items, types.TournamentBracketMatch{
			Id:              match.Id,
			RoundNumber:     match.RoundNumber,
			MatchOrder:      match.MatchOrder,
			Player1Id:       match.Player1Id,
			Player1Name:     names[match.Player1Id],
			Player2Id:       match.Player2Id,
			Player2Name:     names[match.Player2Id],
			WinnerId:        match.WinnerId,
			BracketPosition: match.BracketPosition,
			Status:          match.Status,
		})
	}
	return &types.GetTournamentBracketResp{Success: true, Matches: items, TotalRounds: totalRounds}, nil
}
