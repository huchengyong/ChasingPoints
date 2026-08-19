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

type UpdateMatchFormatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新未记分中八/美九对局赛制
func NewUpdateMatchFormatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMatchFormatLogic {
	return &UpdateMatchFormatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdateMatchFormatLogic) UpdateMatchFormat(req *types.UpdateMatchFormatReq) (resp *types.UpdateMatchFormatResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.UpdateMatchFormatResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req == nil || req.MatchId <= 0 {
		return &types.UpdateMatchFormatResp{Success: false, Message: "对局不存在"}, nil
	}
	format, _, valid := model.NormalizePoolMatchFormat(3, req.MatchFormat, req.TargetWins)
	if !valid || format == model.MatchFormatLegacy {
		return &types.UpdateMatchFormatResp{Success: false, Message: "请选择有效的赛制"}, nil
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
		normalizedFormat, normalizedTargetWins, ok := model.NormalizePoolMatchFormat(locked.GameType, req.MatchFormat, req.TargetWins)
		if !ok || normalizedFormat == model.MatchFormatLegacy {
			failureMessage = "当前对局不支持该赛制"
			return nil
		}
		if !model.IsFlexiblePoolMatch(locked) {
			failureMessage = "legacy 对局不支持修改赛制"
			return nil
		}
		if locked.SyncRevision != req.BaseRevision {
			failureMessage = "对局状态已更新"
			return nil
		}
		if !canChangeMatchFormatWithTx(l.svcCtx, tx, userID, locked) {
			failureMessage = "对局已开始记分，赛制不可修改"
			return nil
		}

		locked.MatchFormat = normalizedFormat
		locked.TargetWins = normalizedTargetWins
		_, bumpErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, locked)
		return bumpErr
	})
	if err != nil {
		l.Logger.Errorf("更新中八/美九赛制失败: matchId=%d userId=%d err=%v", req.MatchId, userID, err)
		return &types.UpdateMatchFormatResp{Success: false, Message: "更新赛制失败"}, nil
	}
	if match == nil {
		return &types.UpdateMatchFormatResp{Success: false, Message: failureMessage}, nil
	}

	view, viewErr := loadMatchWriteState(l.svcCtx, userID, match)
	if viewErr != nil {
		return &types.UpdateMatchFormatResp{Success: false, Message: "加载对局状态失败"}, nil
	}
	if failureMessage != "" {
		return &types.UpdateMatchFormatResp{
			Accepted:       false,
			Success:        false,
			Message:        failureMessage,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
		}, nil
	}

	broadcastMatchFormatChange(l.svcCtx, match)
	return &types.UpdateMatchFormatResp{
		Accepted:       true,
		Success:        true,
		ServerRevision: view.Snapshot.ServerRevision,
		Snapshot:       view.Snapshot,
	}, nil
}

func broadcastMatchFormatChange(svcCtx *svc.ServiceContext, match *model.Match) {
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
