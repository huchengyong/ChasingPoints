package user

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const memberTimeLayout = "2006-01-02 15:04:05"

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
		svcCtx: svcCtx,
	}
}

func (l *GetFavoriteVenueRewardStatusLogic) GetFavoriteVenueRewardStatus() (resp *types.FavoriteVenueRewardStatusResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if user == nil {
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}

	config, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
	if err != nil {
		l.Logger.Errorf("查询奖励配置失败: %v", err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if config == nil {
		config = model.DefaultFavoriteVenueRewardConfig()
	}

	resp = &types.FavoriteVenueRewardStatusResp{
		Success:           true,
		Enabled:           config.Enabled,
		PopupEnabled:      config.PopupEnabled,
		RewardDays:        config.RewardDays,
		NewUserWindowDays: config.NewUserWindowDays,
		Status:            "not_started",
	}

	record, err := l.svcCtx.FavoriteVenueRewardRecordModel.FindByActivityAndUser(model.FavoriteVenueRewardActivityKey, userID)
	if err != nil {
		l.Logger.Errorf("查询奖励记录失败: userId=%d err=%v", userID, err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if record != nil {
		resp.Status = "reward_granted"
		resp.SubmittedVenueId = record.VenueId
		if user.MemberExpiresAt != nil {
			resp.MemberExpiresAt = user.MemberExpiresAt.Format(memberTimeLayout)
		} else {
			resp.MemberExpiresAt = record.MemberExpiresAtAfter.Format(memberTimeLayout)
		}
		return resp, nil
	}

	latestVenue, err := l.svcCtx.VenueModel.FindLatestByOwnerUserId(userID)
	if err != nil {
		l.Logger.Errorf("查询用户最近球馆失败: userId=%d err=%v", userID, err)
		return &types.FavoriteVenueRewardStatusResp{Success: false}, nil
	}
	if latestVenue != nil {
		resp.SubmittedVenueId = latestVenue.Id
		resp.SubmittedVenueName = latestVenue.Name

		switch latestVenue.Status {
		case model.VenueStatusPending:
			resp.Status = "pending_review"
			return resp, nil
		case model.VenueStatusRejected:
			resp.Status = "rejected"
			resp.RejectReason = latestVenue.RejectReason
			return resp, nil
		}
	}

	if config.Enabled && model.FavoriteVenueRewardWindowExpired(user.CreatedAt, time.Now(), config.NewUserWindowDays) {
		resp.Status = "expired"
	}

	return resp, nil
}
