package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTournamentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事详情
func NewGetTournamentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTournamentDetailLogic {
	return &GetTournamentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTournamentDetailLogic) GetTournamentDetail(req *types.GetTournamentDetailReq) (resp *types.GetTournamentDetailResp, err error) {
	if req.TournamentId <= 0 {
		return &types.GetTournamentDetailResp{Success: false}, nil
	}

	tournament, err := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事详情失败: id=%d err=%v", req.TournamentId, err)
		return &types.GetTournamentDetailResp{Success: false}, nil
	}
	if tournament == nil {
		return &types.GetTournamentDetailResp{Success: false}, nil
	}

	info := &types.TournamentInfo{
		Id:             tournament.Id,
		CreatorId:      tournament.CreatorId,
		Name:           tournament.Name,
		Description:    tournament.Description,
		GameType:       tournament.GameType,
		Format:         tournament.Format,
		MaxPlayers:     tournament.MaxPlayers,
		CurrentPlayers: tournament.CurrentPlayers,
		Status:         tournament.Status,
		City:           tournament.City,
		VenueName:      tournament.VenueName,
		CreatedAt:      tournament.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if tournament.StartTime != nil {
		info.StartTime = tournament.StartTime.Format("2006-01-02 15:04:05")
	}
	if tournament.EndTime != nil {
		info.EndTime = tournament.EndTime.Format("2006-01-02 15:04:05")
	}
	if creator, userErr := l.svcCtx.UserModel.FindById(tournament.CreatorId); userErr == nil && creator != nil {
		info.CreatorName = creator.Nickname
	}

	// Get participants
	participants, err := l.svcCtx.TournamentParticipantModel.FindListByTournamentId(req.TournamentId)
	if err != nil {
		l.Logger.Errorf("查询赛事参赛者失败: id=%d err=%v", req.TournamentId, err)
		return &types.GetTournamentDetailResp{Success: false}, nil
	}

	var participantItems []types.TournamentParticipant
	for _, p := range participants {
		item := types.TournamentParticipant{
			UserId:    p.UserId,
			Seed:      p.Seed,
			Status:    p.Status,
			FinalRank: p.FinalRank,
		}
		if user, userErr := l.svcCtx.UserModel.FindById(p.UserId); userErr == nil && user != nil {
			item.Nickname = user.Nickname
			item.Avatar = user.Avatar
		}
		participantItems = append(participantItems, item)
	}

	if participantItems == nil {
		participantItems = []types.TournamentParticipant{}
	}

	return &types.GetTournamentDetailResp{
		Success:      true,
		Tournament:   info,
		Participants: participantItems,
	}, nil
}
