package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyTournamentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的赛事
func NewGetMyTournamentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyTournamentsLogic {
	return &GetMyTournamentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyTournamentsLogic) GetMyTournaments(req *types.GetMyTournamentsReq) (resp *types.GetTournamentListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetTournamentListResp{Success: false}, nil
	}

	useStatusFilter := req.Status >= 0 && req.Status <= 3
	list, total, err := l.svcCtx.TournamentModel.FindListByParticipant(userIdInt, req.Page, req.PageSize, req.Status, useStatusFilter)
	if err != nil {
		l.Logger.Errorf("获取我的赛事失败: userId=%d err=%v", userIdInt, err)
		return &types.GetTournamentListResp{Success: false}, nil
	}

	var items []types.TournamentInfo
	for _, t := range list {
		item := types.TournamentInfo{
			Id:             t.Id,
			CreatorId:      t.CreatorId,
			Name:           t.Name,
			Description:    t.Description,
			GameType:       t.GameType,
			Format:         t.Format,
			MaxPlayers:     t.MaxPlayers,
			CurrentPlayers: t.CurrentPlayers,
			Status:         t.Status,
			City:           t.City,
			VenueName:      t.VenueName,
			CreatedAt:      t.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if t.StartTime != nil {
			item.StartTime = t.StartTime.Format("2006-01-02 15:04:05")
		}
		if t.EndTime != nil {
			item.EndTime = t.EndTime.Format("2006-01-02 15:04:05")
		}
		if creator, userErr := l.svcCtx.UserModel.FindById(t.CreatorId); userErr == nil && creator != nil {
			item.CreatorName = creator.Nickname
		}
		items = append(items, item)
	}

	if items == nil {
		items = []types.TournamentInfo{}
	}

	return &types.GetTournamentListResp{Success: true, Total: total, List: items}, nil
}
