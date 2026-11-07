package admin

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetGearPendingListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取待收录的未收录型号（管理员）
func NewAdminGetGearPendingListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetGearPendingListLogic {
	return &AdminGetGearPendingListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetGearPendingListLogic) AdminGetGearPendingList(req *types.AdminGearPendingListReq) (resp *types.AdminGearPendingListResp, err error) {
	fail := func(code int, message string) *types.AdminGearPendingListResp {
		return &types.AdminGearPendingListResp{Code: code, Success: false, Message: message, List: []types.AdminGearPendingInfo{}}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	if req == nil {
		req = &types.AdminGearPendingListReq{}
	}
	if req.Category != "" && !isPendingCategory(req.Category) {
		return fail(400, "装备类别无效"), nil
	}

	page, pageSize := normalizeAdminGearPage(req.Page, req.PageSize)
	groups, total, err := l.svcCtx.GearModel.ListPendingGroups(req.Category, page, pageSize)
	if err != nil {
		l.Logger.Errorf("读取待收录分组失败: %v", err)
		return fail(500, "读取待收录列表失败"), nil
	}

	latestIds := make([]int64, 0, len(groups))
	for _, group := range groups {
		latestIds = append(latestIds, group.LatestId)
	}
	items, err := l.svcCtx.GearModel.FindItemsByIds(latestIds)
	if err != nil {
		l.Logger.Errorf("读取待收录登记失败: %v", err)
		return fail(500, "读取待收录列表失败"), nil
	}
	brandIds := make([]int64, 0, len(items))
	for _, item := range items {
		brandIds = append(brandIds, item.CustomBrandId)
	}
	brands, err := l.svcCtx.GearModel.FindBrandsByIds(brandIds)
	if err != nil {
		l.Logger.Errorf("读取待收录品牌失败: %v", err)
		return fail(500, "读取待收录列表失败"), nil
	}

	list := make([]types.AdminGearPendingInfo, 0, len(groups))
	for _, group := range groups {
		item := items[group.LatestId]
		brandName := item.CustomBrandName
		if item.CustomBrandId > 0 {
			brandName = brands[item.CustomBrandId].Name
		}
		list = append(list, types.AdminGearPendingInfo{
			CustomKey: group.CustomKey,
			Category:  group.Category,
			BrandName: brandName,
			ModelName: item.CustomModelName,
			CueStyle:  item.CueStyle,
			Roles:     model.ParseGearRoles(item.Roles),
			Count:     group.Count,
			LastAt:    logicx.FormatUTC8Time(item.UpdatedAt),
		})
	}
	return &types.AdminGearPendingListResp{Code: 0, Success: true, Message: "ok", Total: total, List: list}, nil
}
