package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateSnookerFormatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新未记分斯诺克对局赛制
func NewUpdateSnookerFormatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSnookerFormatLogic {
	return &UpdateSnookerFormatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdateSnookerFormatLogic) UpdateSnookerFormat(req *types.UpdateSnookerFormatReq) (resp *types.UpdateSnookerFormatResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.UpdateSnookerFormatResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req == nil || req.MatchId <= 0 {
		return &types.UpdateSnookerFormatResp{Success: false, Message: "对局不存在"}, nil
	}
	format, targetWins, valid := model.NormalizeSnookerFormat(req.SnookerFormat, req.SnookerTargetWins, 0)
	if !valid || format == model.SnookerFormatLegacy {
		return &types.UpdateSnookerFormatResp{Success: false, Message: "请选择有效的斯诺克赛制"}, nil
	}

	var (
		match          *model.Match
		failureMessage string
	)
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		locked, findErr := l.svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		match = locked
		if locked == nil {
			failureMessage = "对局不存在"
			return nil
		}
		if locked.GameType != 1 || locked.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
			failureMessage = "当前对局不支持修改斯诺克赛制"
			return nil
		}
		if locked.SyncRevision != req.BaseRevision {
			failureMessage = "对局状态已更新"
			return nil
		}
		if !canChangeSnookerFormatWithTx(l.svcCtx, tx, userID, locked) {
			failureMessage = "对局已开始记分，赛制不可修改"
			return nil
		}

		locked.SnookerFormat = format
		locked.SnookerTargetWins = targetWins
		locked.BestOfFrames = 0
		if locked.StartingActor != 1 && locked.StartingActor != 2 {
			locked.StartingActor = 1
		}
		_, bumpErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		return bumpErr
	})
	if err != nil {
		l.Logger.Errorf("更新斯诺克赛制失败: matchId=%d userId=%d err=%v", req.MatchId, userID, err)
		return &types.UpdateSnookerFormatResp{Success: false, Message: "更新赛制失败"}, nil
	}
	if match == nil {
		return &types.UpdateSnookerFormatResp{Success: false, Message: failureMessage}, nil
	}

	view, viewErr := loadMatchWriteState(l.svcCtx, userID, match)
	if viewErr != nil {
		return &types.UpdateSnookerFormatResp{Success: false, Message: "加载对局状态失败"}, nil
	}
	if failureMessage != "" {
		return &types.UpdateSnookerFormatResp{
			Accepted:       false,
			Success:        false,
			Message:        failureMessage,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
		}, nil
	}

	broadcastSnookerFormatChange(l.svcCtx, match)
	return &types.UpdateSnookerFormatResp{
		Accepted:       true,
		Success:        true,
		ServerRevision: view.Snapshot.ServerRevision,
		Snapshot:       view.Snapshot,
	}, nil
}

func broadcastSnookerFormatChange(svcCtx *svc.ServiceContext, match *model.Match) {
	if ws.GlobalHub == nil || svcCtx == nil || match == nil {
		return
	}
	messages := make(map[int64]*ws.Message, 2)
	userIDs := []int64{match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		userIDs = append(userIDs, *match.OpponentId)
	}
	for _, userID := range userIDs {
		view, err := loadMatchWriteState(svcCtx, userID, match)
		if err != nil {
			continue
		}
		messages[userID] = &ws.Message{Type: "sync", Data: view.Snapshot}
	}
	ws.GlobalHub.BroadcastToMatchForUsers(match.Id, messages)
}
