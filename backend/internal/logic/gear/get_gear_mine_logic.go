package gear

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGearMineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的装备与主力设置
func NewGetGearMineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGearMineLogic {
	return &GetGearMineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGearMineLogic) GetGearMine() (resp *types.GetGearMineResp, err error) {
	failed := func(message string) *types.GetGearMineResp {
		return &types.GetGearMineResp{
			Success: false, Message: message, MaxItems: model.GearMaxItemsPerUser,
			Items: []types.GearItemInfo{}, Defaults: []types.GearDefaultInfo{},
		}
	}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return failed("用户未登录"), nil
	}

	items, err := l.svcCtx.GearModel.ListUserItems(userId, []int{model.GearItemStatusActive, model.GearItemStatusRetired})
	if err != nil {
		l.Logger.Errorf("读取我的装备失败: userId=%d err=%v", userId, err)
		return failed("读取装备失败"), nil
	}
	catalog, err := loadGearCatalog(l.svcCtx.GearModel, items)
	if err != nil {
		l.Logger.Errorf("读取装备目录信息失败: userId=%d err=%v", userId, err)
		return failed("读取装备失败"), nil
	}
	defaults, err := l.svcCtx.GearModel.ListDefaults(userId)
	if err != nil {
		l.Logger.Errorf("读取主力设置失败: userId=%d err=%v", userId, err)
		return failed("读取装备失败"), nil
	}

	return &types.GetGearMineResp{
		Success:  true,
		Message:  "ok",
		MaxItems: model.GearMaxItemsPerUser,
		Items:    catalog.itemInfos(items, true),
		Defaults: defaultInfos(defaults),
	}, nil
}
