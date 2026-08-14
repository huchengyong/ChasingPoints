package tournament

import (
	"context"
	"fmt"

	"chasing_points/internal/config"
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type JoinTournamentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 报名赛事
func NewJoinTournamentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinTournamentLogic {
	return &JoinTournamentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *JoinTournamentLogic) JoinTournament(req *types.TournamentIdReq) (resp *types.CommonResp, err error) {
	if !l.svcCtx.Config.UserTournamentEnabled() {
		return &types.CommonResp{Success: false, Message: config.DisabledFeatureMessage("tournament_user_action")}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req.TournamentId <= 0 {
		return &types.CommonResp{Success: false, Message: "赛事参数无效"}, nil
	}

	// Use transaction with optimistic locking
	txErr := l.svcCtx.TournamentModel.Transaction(func(tx *gorm.DB) error {
		// Check tournament exists and is recruiting
		tournament, findErr := l.svcCtx.TournamentModel.FindByIdWithDB(tx, req.TournamentId)
		if findErr != nil {
			return findErr
		}
		if tournament == nil {
			return fmt.Errorf("赛事不存在")
		}
		if tournament.Status != 0 {
			return fmt.Errorf("赛事不在招募中")
		}
		if tournament.MaxPlayers <= 0 || tournament.MaxPlayers > 64 {
			return fmt.Errorf("赛事人数配置异常")
		}

		// Check if already registered
		existing, findErr := l.svcCtx.TournamentParticipantModel.FindByTournamentAndUserWithDB(tx, req.TournamentId, userIdInt)
		if findErr != nil {
			return findErr
		}
		if existing != nil {
			return fmt.Errorf("已报名该赛事")
		}

		// Optimistic lock: increment current_players only if < max_players
		ok, incrErr := l.svcCtx.TournamentModel.IncrementCurrentPlayersWithDB(tx, req.TournamentId)
		if incrErr != nil {
			return incrErr
		}
		if !ok {
			return fmt.Errorf("赛事已满员")
		}

		// Create participant record
		participant := &model.TournamentParticipant{
			TournamentId: req.TournamentId,
			UserId:       userIdInt,
			Status:       0, // 已报名
		}
		if createErr := l.svcCtx.TournamentParticipantModel.CreateWithDB(tx, participant); createErr != nil {
			return createErr
		}

		return nil
	})

	if txErr != nil {
		l.Logger.Errorf("报名赛事失败: tournamentId=%d userId=%d err=%v", req.TournamentId, userIdInt, txErr)
		return &types.CommonResp{Success: false, Message: txErr.Error()}, nil
	}

	// Send notification to tournament creator (outside transaction)
	tournament, _ := l.svcCtx.TournamentModel.FindById(req.TournamentId)
	if tournament != nil {
		l.syncTournamentJoinAchievement(userIdInt, tournament)
	}
	if tournament != nil && tournament.CreatorId != userIdInt {
		userName := "球友"
		if user, userErr := l.svcCtx.UserModel.FindById(userIdInt); userErr == nil && user != nil && user.Nickname != "" {
			userName = user.Nickname
		}
		content := fmt.Sprintf("%s 报名了你的赛事「%s」", userName, tournament.Name)
		if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
			UserId:      tournament.CreatorId,
			Type:        "tournament",
			Title:       "有新选手报名",
			Content:     content,
			PushTitle:   "有新选手报名",
			PushContent: content,
			WSCategory:  "tournament",
		}); notifyErr != nil {
			l.Logger.Errorf("分发赛事报名通知失败: tournamentId=%d creator=%d err=%v", req.TournamentId, tournament.CreatorId, notifyErr)
		}
	}

	return &types.CommonResp{Success: true, Message: "报名成功"}, nil
}
