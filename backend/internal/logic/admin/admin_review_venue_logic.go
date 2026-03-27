package admin

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AdminReviewVenueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核球馆
func NewAdminReviewVenueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewVenueLogic {
	return &AdminReviewVenueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminReviewVenueLogic) AdminReviewVenue(req *types.AdminVenueReviewReq) (resp *types.AdminVenueReviewResp, err error) {
	// 验证参数
	if req.VenueId <= 0 {
		return &types.AdminVenueReviewResp{
			Code:    400,
			Success: false,
			Message: "球馆ID无效",
		}, nil
	}

	// 只允许通过(1)或拒绝(3)
	if req.Status != 1 && req.Status != 3 {
		return &types.AdminVenueReviewResp{
			Code:    400,
			Success: false,
			Message: "无效的审核状态",
		}, nil
	}

	// 查询球馆
	venue, err := l.svcCtx.VenueModel.FindById(req.VenueId)
	if err != nil {
		l.Logger.Errorf("查询球馆失败: venueId=%d err=%v", req.VenueId, err)
		return &types.AdminVenueReviewResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}
	if venue == nil {
		return &types.AdminVenueReviewResp{
			Code:    404,
			Success: false,
			Message: "球馆不存在",
		}, nil
	}

	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		rejectReason := ""
		if req.Status == model.VenueStatusRejected {
			rejectReason = req.RejectReason
		}
		if err := l.svcCtx.VenueModel.UpdateReviewWithTx(tx, req.VenueId, req.Status, rejectReason); err != nil {
			return err
		}
		if req.Status != model.VenueStatusPublished {
			return nil
		}
		return l.grantFavoriteVenueRewardIfEligible(tx, venue, time.Now())
	}); err != nil {
		l.Logger.Errorf("更新球馆审核状态失败: venueId=%d status=%d err=%v", req.VenueId, req.Status, err)
		return &types.AdminVenueReviewResp{
			Code:    500,
			Success: false,
			Message: "审核失败",
		}, nil
	}

	var msg string
	if req.Status == 1 {
		msg = "审核通过"
	} else {
		msg = "审核已拒绝"
	}

	return &types.AdminVenueReviewResp{
		Code:    0,
		Success: true,
		Message: msg,
	}, nil
}

func (l *AdminReviewVenueLogic) grantFavoriteVenueRewardIfEligible(tx *gorm.DB, venue *model.Venue, now time.Time) error {
	if venue == nil || venue.OwnerUserId <= 0 {
		return nil
	}

	config, err := l.svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
	if err != nil {
		return err
	}
	if config == nil {
		config = model.DefaultFavoriteVenueRewardConfig()
	}
	if !model.FavoriteVenueRewardConfigIsActive(config, now) {
		return nil
	}

	existingByUser, err := l.svcCtx.FavoriteVenueRewardRecordModel.FindByActivityAndUser(model.FavoriteVenueRewardActivityKey, venue.OwnerUserId)
	if err != nil {
		return err
	}
	if existingByUser != nil {
		return nil
	}

	existingByVenue, err := l.svcCtx.FavoriteVenueRewardRecordModel.FindByActivityAndVenue(model.FavoriteVenueRewardActivityKey, venue.Id)
	if err != nil {
		return err
	}
	if existingByVenue != nil {
		return nil
	}

	user, err := l.svcCtx.UserModel.FindById(venue.OwnerUserId)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	if !model.FavoriteVenueRewardIsWithinWindow(user.CreatedAt, venue.CreatedAt, config.NewUserWindowDays) {
		return nil
	}

	var before *time.Time
	baseTime := now
	if user.MemberExpiresAt != nil {
		beforeValue := *user.MemberExpiresAt
		before = &beforeValue
		if beforeValue.After(now) {
			baseTime = beforeValue
		}
	}
	after := baseTime.AddDate(0, 0, config.RewardDays)
	if err := l.svcCtx.UserModel.UpdateMemberExpiresAtWithTx(tx, user.Id, &after); err != nil {
		return err
	}

	return l.svcCtx.FavoriteVenueRewardRecordModel.CreateWithTx(tx, &model.FavoriteVenueRewardRecord{
		ActivityKey:           model.FavoriteVenueRewardActivityKey,
		UserId:                user.Id,
		VenueId:               venue.Id,
		RewardDays:            config.RewardDays,
		MemberExpiresAtBefore: before,
		MemberExpiresAtAfter:  after,
		GrantedAt:             now,
	})
}
