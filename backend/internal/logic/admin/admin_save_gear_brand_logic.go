package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSaveGearBrandLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 保存装备品牌（管理员）
func NewAdminSaveGearBrandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSaveGearBrandLogic {
	return &AdminSaveGearBrandLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminSaveGearBrandLogic) AdminSaveGearBrand(req *types.AdminGearBrandSaveReq) (resp *types.AdminGearSaveResp, err error) {
	fail := func(code int, message string) *types.AdminGearSaveResp {
		return &types.AdminGearSaveResp{Code: code, Success: false, Message: message}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	if req == nil {
		return fail(400, "请求不能为空"), nil
	}

	name := strings.TrimSpace(req.Name)
	if n := runeLen(name); n == 0 || n > adminGearBrandNameMaxRunes {
		return fail(400, fmt.Sprintf("品牌名称需为 1 到 %d 个字", adminGearBrandNameMaxRunes)), nil
	}
	aliases := normalizeGearAliases(req.Aliases)
	if runeLen(aliases) > adminGearAliasesMaxRunes {
		return fail(400, fmt.Sprintf("别名总长度不能超过 %d 个字", adminGearAliasesMaxRunes)), nil
	}
	if req.Status != 0 && req.Status != 1 {
		return fail(400, "状态只能是启用或停用"), nil
	}

	brand := &model.GearBrand{Name: name, Aliases: aliases, Sort: req.Sort, Status: req.Status}
	if req.Id > 0 {
		existing, err := l.svcCtx.GearModel.FindBrand(req.Id)
		if err != nil {
			l.Logger.Errorf("读取装备品牌失败: id=%d err=%v", req.Id, err)
			return fail(500, "保存失败"), nil
		}
		if existing == nil {
			return fail(404, "品牌不存在"), nil
		}
		brand.Id, brand.CreatedAt = existing.Id, existing.CreatedAt
	}

	if err := l.svcCtx.GearModel.SaveBrand(brand); err != nil {
		if errors.Is(err, model.ErrGearDuplicate) {
			return fail(409, "已有同名品牌"), nil
		}
		l.Logger.Errorf("保存装备品牌失败: err=%v", err)
		return fail(500, "保存失败"), nil
	}
	return &types.AdminGearSaveResp{Code: 0, Success: true, Message: "保存成功", Id: brand.Id}, nil
}
