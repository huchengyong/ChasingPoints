package user

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFavoriteVenueRewardStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取常玩球馆奖励状态
func NewGetFavoriteVenueRewardStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFavoriteVenueRewardStatusLogic {
	return &GetFavoriteVenueRewardStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFavoriteVenueRewardStatusLogic) GetFavoriteVenueRewardStatus() (resp *types.FavoriteVenueRewardStatusResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}

	user, err := currentUserFromRequest(l.ctx, l.svcCtx, userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if user == nil {
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}

	config, err := logicx.LoadFavoriteVenueRewardConfig(l.svcCtx)
	if err != nil {
		l.Logger.Errorf("查询奖励配置失败: %v", err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}

	resp = &types.FavoriteVenueRewardStatusResp{
		Success:           true,
		Enabled:           config.Enabled,
		PopupEnabled:      config.PopupEnabled,
		RewardDays:        config.RewardDays,
		NewUserWindowDays: config.NewUserWindowDays,
		Status:            "not_started",
	}

	snapshot, err := l.svcCtx.FavoriteVenueRewardRecordModel.FindStatusByUser(model.FavoriteVenueRewardActivityKey, userID)
	if err != nil {
		l.Logger.Errorf("查询奖励状态失败: userId=%d err=%v", userID, err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if snapshot.RecordId != nil && snapshot.RecordVenueId != nil {
		resp.Status = "reward_granted"
		resp.SubmittedVenueId = *snapshot.RecordVenueId
		if user.MemberExpiresAt != nil {
			resp.MemberExpiresAt = logicx.FormatUTC8TimePtr(user.MemberExpiresAt)
		} else if snapshot.RecordMemberExpiresAtAfter != nil {
			resp.MemberExpiresAt = logicx.FormatUTC8Time(*snapshot.RecordMemberExpiresAtAfter)
		}
		return resp, nil
	}

	if snapshot.VenueId != nil {
		resp.SubmittedVenueId = *snapshot.VenueId
		resp.SubmittedVenueName = snapshot.VenueName

		if snapshot.VenueStatus != nil {
			switch *snapshot.VenueStatus {
			case model.VenueStatusPending:
				resp.Status = "pending_review"
				return resp, nil
			case model.VenueStatusRejected:
				resp.Status = "rejected"
				resp.RejectReason = snapshot.VenueRejectReason
				return resp, nil
			}
		}
	}

	return resp, nil
}
