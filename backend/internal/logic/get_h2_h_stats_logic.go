package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetH2HStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取交锋统计
func NewGetH2HStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetH2HStatsLogic {
	return &GetH2HStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetH2HStatsLogic) GetH2HStats(req *types.H2HStatsReq) (resp *types.H2HStatsResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.H2HStatsResp{Success: false}, nil
	}

	// 获取对手信息
	opponentName := req.OpponentName
	opponentAvatar := ""
	opponentId := req.OpponentId

	// 如果传入了 OpponentId，则通过用户表查询对手信息
	if opponentName == "" && req.OpponentId > 0 {
		user, err := l.svcCtx.UserModel.FindById(req.OpponentId)
		if err != nil {
			l.Logger.Errorf("查询用户失败: %v", err)
			return &types.H2HStatsResp{Success: false}, nil
		}
		if user != nil {
			opponentName = user.Nickname
			opponentAvatar = user.Avatar
		}
	}

	if opponentName == "" {
		return &types.H2HStatsResp{Success: false}, nil
	}

	// 获取交锋统计
	total, myWins, oppWins, avgDiff, err := l.svcCtx.MatchModel.GetH2HStats(userId, opponentName)
	if err != nil {
		l.Logger.Errorf("获取交锋统计失败: %v", err)
		return &types.H2HStatsResp{Success: false}, nil
	}

	// 计算胜率
	var winRate float64
	if total > 0 {
		winRate = float64(myWins) / float64(total) * 100
	}

	// TODO: 计算最长连胜
	maxWinStreak := 0

	return &types.H2HStatsResp{
		Success: true,
		Opponent: &types.H2HOpponent{
			Id:     opponentId,
			Name:   opponentName,
			Avatar: opponentAvatar,
		},
		Stats: &types.H2HStats{
			TotalMatches: total,
			MyWins:       myWins,
			OpponentWins: oppWins,
			WinRate:      winRate,
			AvgScoreDiff: avgDiff,
			MaxWinStreak: maxWinStreak,
		},
	}, nil
}
