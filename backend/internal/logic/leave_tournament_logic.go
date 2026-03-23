package logic

import (
	"context"
	"fmt"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type LeaveTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 退出赛事
func NewLeaveTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveTournamentLogic {
	return &LeaveTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LeaveTournamentLogic) LeaveTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentId <= 0 {
		return &types.CommonResp{Success: false, Message: "赛事参数无效"}, nil
	}

	txErr := l.svcCtx.TournamentModel.Transaction(func(tx *gorm.DB) error {
		// Check tournament is still recruiting
		tournament, findErr := l.svcCtx.TournamentModel.FindByIdWithDB(tx, req.TournamentId)
		if findErr != nil {
			return findErr
		}
		if tournament == nil {
			return fmt.Errorf("赛事不存在")
		}
		if tournament.Status != 0 {
			return fmt.Errorf("赛事已开始，无法退出")
		}

		// Delete participant record
		deleted, delErr := l.svcCtx.TournamentParticipantModel.DeleteByTournamentAndUserWithDB(tx, req.TournamentId, userIdInt)
		if delErr != nil {
			return delErr
		}
		if !deleted {
			return fmt.Errorf("未报名该赛事")
		}

		// Decrement current_players
		_, decrErr := l.svcCtx.TournamentModel.DecrementCurrentPlayersWithDB(tx, req.TournamentId)
		if decrErr != nil {
			return decrErr
		}

		return nil
	})

	if txErr != nil {
		l.Logger.Errorf("退出赛事失败: tournamentId=%d userId=%d err=%v", req.TournamentId, userIdInt, txErr)
		return &types.CommonResp{Success: false, Message: txErr.Error()}, nil
	}

	return &types.CommonResp{Success: true, Message: "已退出赛事"}, nil
}
