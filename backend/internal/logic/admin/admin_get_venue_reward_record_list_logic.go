package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetVenueRewardRecordListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取常玩球馆奖励发放记录
func NewAdminGetVenueRewardRecordListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetVenueRewardRecordListLogic {
	return &AdminGetVenueRewardRecordListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetVenueRewardRecordListLogic) AdminGetVenueRewardRecordList(req *types.AdminVenueRewardRecordListReq) (resp *types.AdminVenueRewardRecordListResp, err error) {
	list, total, err := l.svcCtx.FavoriteVenueRewardRecordModel.FindAdminList(model.FavoriteVenueRewardActivityKey, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("获取常玩球馆奖励发放记录失败: %v", err)
		return &types.AdminVenueRewardRecordListResp{
			Code:    500,
			Success: false,
			Message: "获取奖励记录失败",
		}, nil
	}

	items := make([]types.AdminVenueRewardRecordInfo, 0, len(list))
	for _, item := range list {
		record := types.AdminVenueRewardRecordInfo{
			Id:                   item.Id,
			UserId:               item.UserId,
			UserNickname:         item.UserNickname,
			UserPhone:            item.UserPhone,
			VenueId:              item.VenueId,
			VenueName:            item.VenueName,
			RewardDays:           item.RewardDays,
			MemberExpiresAtAfter: item.MemberExpiresAtAfter.Format("2006-01-02 15:04:05"),
			GrantedAt:            item.GrantedAt.Format("2006-01-02 15:04:05"),
		}
		if item.MemberExpiresAtBefore != nil {
			record.MemberExpiresAtBefore = item.MemberExpiresAtBefore.Format("2006-01-02 15:04:05")
		}
		items = append(items, record)
	}

	return &types.AdminVenueRewardRecordListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
