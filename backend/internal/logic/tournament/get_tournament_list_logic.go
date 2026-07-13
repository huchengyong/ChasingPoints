package tournament

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事列表
func NewGetTournamentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentListLogic {
	return &GetTournamentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentListLogic) GetTournamentList(req *types.GetTournamentListReq) (resp *types.GetTournamentListResp, err error) {
	useStatusFilter := req.Status >= 0 && req.Status <= 3

	list, total, err := l.svcCtx.TournamentModel.FindList(req.Page, req.PageSize, req.City, req.GameType, req.Status, useStatusFilter)
	if err != nil {
		l.Logger.Errorf("获取赛事列表失败: err=%v", err)
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
		// Look up creator name
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
