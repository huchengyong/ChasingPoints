package tournament

import (
	"context"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建赛事
func NewCreateTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTournamentLogic {
	return &CreateTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreateTournamentLogic) CreateTournament(req *types.CreateTournamentReq) (resp *types.CreateTournamentResp, err error) {
	// 只允许管理员创建赛事
	adminId, err := utils.GetAdminIDFromCtx(l.ctx)
	if err != nil {
		return &types.CreateTournamentResp{Success: false}, nil
	}

	if strings.TrimSpace(req.Name) == "" || req.GameType < 1 || req.GameType > 3 {
		return &types.CreateTournamentResp{Success: false}, nil
	}
	if req.Format != 1 && req.Format != 3 {
		return &types.CreateTournamentResp{Success: false}, nil
	}
	if req.MaxPlayers < 2 || req.MaxPlayers > 64 {
		return &types.CreateTournamentResp{Success: false}, nil
	}

	startTime, err := parseTournamentTime(req.StartTime)
	if err != nil {
		return &types.CreateTournamentResp{Success: false}, nil
	}

	tournament := &model.Tournament{
		CreatorId:   -int64(adminId), // 使用负数表示管理员创建的赛事
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		GameType:    req.GameType,
		Format:      req.Format,
		MaxPlayers:  req.MaxPlayers,
		Status:      0,
		City:        strings.TrimSpace(req.City),
		VenueName:   strings.TrimSpace(req.VenueName),
		StartTime:   startTime,
	}

	if err = l.svcCtx.TournamentModel.Create(tournament); err != nil {
		l.Logger.Errorf("创建赛事失败: adminId=%d err=%v", adminId, err)
		return &types.CreateTournamentResp{Success: false}, nil
	}

	return &types.CreateTournamentResp{Success: true, TournamentId: tournament.Id}, nil
}
