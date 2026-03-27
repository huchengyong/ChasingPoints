package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetRecentMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取最近对局
func NewAdminGetRecentMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetRecentMatchesLogic {
	return &AdminGetRecentMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetRecentMatchesLogic) AdminGetRecentMatches() (resp *types.AdminRecentMatchesResp, err error) {
	matches, err := l.svcCtx.MatchModel.FindRecent(10)
	if err != nil {
		l.Logger.Errorf("获取最近对局失败: %v", err)
		return &types.AdminRecentMatchesResp{
			Code:    500,
			Success: false,
			Message: "获取失败",
			List:    []types.AdminRecentMatch{},
		}, nil
	}

	list := make([]types.AdminRecentMatch, 0, len(matches))
	for _, match := range matches {
		gameTypeName := getGameTypeName(match.GameType)
		list = append(list, types.AdminRecentMatch{
			Id:            match.Id,
			GameType:      match.GameType,
			GameTypeName:  gameTypeName,
			Player1Name:   "我",
			Player2Name:   match.OpponentName,
			MyScore:       match.MyScore,
			OpponentScore: match.OpponentScore,
			Status:        match.Status,
			CreatedAt:     match.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	return &types.AdminRecentMatchesResp{
		Code:    0,
		Success: true,
		Message: "success",
		List:    list,
	}, nil
}

func getGameTypeName(gameType int) string {
	switch gameType {
	case 1:
		return "斯诺克"
	case 2:
		return "九球追分"
	case 3:
		return "中式八球"
	case 4:
		return "美式九球"
	default:
		return "未知"
	}
}
