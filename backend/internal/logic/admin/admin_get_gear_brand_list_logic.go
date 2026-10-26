package admin

import (
	"context"
	"strings"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetGearBrandListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取装备品牌列表（管理员）
func NewAdminGetGearBrandListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetGearBrandListLogic {
	return &AdminGetGearBrandListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetGearBrandListLogic) AdminGetGearBrandList(req *types.AdminGearBrandListReq) (resp *types.AdminGearBrandListResp, err error) {
	fail := func(code int, message string) *types.AdminGearBrandListResp {
		return &types.AdminGearBrandListResp{Code: code, Success: false, Message: message, List: []types.AdminGearBrandInfo{}}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	if req == nil {
		req = &types.AdminGearBrandListReq{}
	}

	page, pageSize := normalizeAdminGearPage(req.Page, req.PageSize)
	rows, total, err := l.svcCtx.GearModel.ListBrandsPage(strings.TrimSpace(req.Keyword), page, pageSize)
	if err != nil {
		l.Logger.Errorf("读取装备品牌列表失败: %v", err)
		return fail(500, "读取品牌失败"), nil
	}
	list := make([]types.AdminGearBrandInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.AdminGearBrandInfo{
			Id: row.Id, Name: row.Name, Aliases: row.Aliases, Sort: row.Sort, Status: row.Status, ProductCount: row.ProductCount,
		})
	}
	return &types.AdminGearBrandListResp{Code: 0, Success: true, Message: "ok", Total: total, List: list}, nil
}
