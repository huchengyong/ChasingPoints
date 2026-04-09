package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetMatchListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局列表（管理员）
func NewAdminGetMatchListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetMatchListLogic {
	return &AdminGetMatchListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetMatchListLogic) AdminGetMatchList(req *types.AdminMatchListReq) (resp *types.AdminMatchListResp, err error) {
	matches, total, err := l.svcCtx.MatchModel.FindListForAdmin(req.Page, req.PageSize, req.Status, req.GameType)
	if err != nil {
		l.Logger.Errorf("获取对局列表失败: %v", err)
		return &types.AdminMatchListResp{
			Code:    500,
			Success: false,
			Message: "获取对局列表失败",
			List:    []types.AdminMatchInfo{},
		}, nil
	}

	list := make([]types.AdminMatchInfo, 0, len(matches))
	playerNameMap := buildAdminPlayerNameMap(l.svcCtx, matches)
	for _, match := range matches {
		gameTypeName := getGameTypeName(match.GameType)
		statusText := getMatchStatusText(match.Status)
		player1Name, player2Name := resolveAdminPlayerNames(match, playerNameMap)
		winnerName := resolveAdminWinnerName(match.Result, player1Name, player2Name)

		list = append(list, types.AdminMatchInfo{
			Id:            match.Id,
			GameType:      match.GameType,
			GameTypeName:  gameTypeName,
			Player1Name:   player1Name,
			Player2Name:   player2Name,
			MyScore:       match.MyScore,
			OpponentScore: match.OpponentScore,
			Status:        match.Status,
			StatusText:    statusText,
			Result:        match.Result,
			WinnerName:    winnerName,
			MatchTime:     match.MatchTime.Format("2006-01-02 15:04"),
			CreatedAt:     match.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	return &types.AdminMatchListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    list,
	}, nil
}

func getMatchStatusText(status int) string {
	switch status {
	case 1:
		return "进行中"
	case 2:
		return "已完成"
	case 3:
		return "已取消"
	default:
		return "未知"
	}
}
