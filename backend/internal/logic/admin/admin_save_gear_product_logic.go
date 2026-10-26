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

type AdminSaveGearProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 保存装备型号（管理员）
func NewAdminSaveGearProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSaveGearProductLogic {
	return &AdminSaveGearProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminSaveGearProductLogic) AdminSaveGearProduct(req *types.AdminGearProductSaveReq) (resp *types.AdminGearSaveResp, err error) {
	fail := func(code int, message string) *types.AdminGearSaveResp {
		return &types.AdminGearSaveResp{Code: code, Success: false, Message: message}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	if req == nil {
		return fail(400, "请求不能为空"), nil
	}

	product, message := l.buildProduct(req)
	if message != "" {
		return fail(400, message), nil
	}

	if req.Id > 0 {
		existing, err := l.svcCtx.GearModel.FindProduct(req.Id)
		if err != nil {
			l.Logger.Errorf("读取装备型号失败: id=%d err=%v", req.Id, err)
			return fail(500, "保存失败"), nil
		}
		if existing == nil {
			return fail(404, "型号不存在"), nil
		}
		// 已有用户装备和对局记录引用这个型号，类别不能改。
		if existing.Category != product.Category {
			return fail(400, "不能修改型号类别"), nil
		}
		product.Id, product.CreatedAt = existing.Id, existing.CreatedAt
	}

	if product.BrandId > 0 {
		brand, err := l.svcCtx.GearModel.FindBrand(product.BrandId)
		if err != nil {
			l.Logger.Errorf("读取装备品牌失败: id=%d err=%v", product.BrandId, err)
			return fail(500, "保存失败"), nil
		}
		if brand == nil {
			return fail(400, "品牌不存在"), nil
		}
	}

	if err := l.svcCtx.GearModel.SaveProduct(product); err != nil {
		if errors.Is(err, model.ErrGearDuplicate) {
			return fail(409, "该品牌下已有同名型号"), nil
		}
		l.Logger.Errorf("保存装备型号失败: err=%v", err)
		return fail(500, "保存失败"), nil
	}
	return &types.AdminGearSaveResp{Code: 0, Success: true, Message: "保存成功", Id: product.Id}, nil
}

// buildProduct 校验并构造型号，校验失败时返回给管理员看的提示。
func (l *AdminSaveGearProductLogic) buildProduct(req *types.AdminGearProductSaveReq) (*model.GearProduct, string) {
	if !model.IsGearCategory(req.Category) {
		return nil, "装备类别无效"
	}
	name := strings.TrimSpace(req.Name)
	if n := runeLen(name); n == 0 || n > adminGearProductNameMaxRunes {
		return nil, fmt.Sprintf("型号名称需为 1 到 %d 个字", adminGearProductNameMaxRunes)
	}
	if req.Status != 0 && req.Status != 1 {
		return nil, "状态只能是启用或停用"
	}
	if req.BrandId < 0 {
		return nil, "品牌无效"
	}
	house := req.BrandId == 0
	if house && req.Category != model.GearCategoryCue && req.Category != model.GearCategoryChalk {
		return nil, "只有球杆和巧克可以是球房公用条目（不选品牌）"
	}

	product := &model.GearProduct{
		Category: req.Category, BrandId: req.BrandId, Name: name, House: house, Sort: req.Sort, Status: req.Status,
	}

	switch req.Category {
	case model.GearCategoryCue:
		if req.CueStyle != "" || !house {
			if !model.IsGearCueStyle(req.CueStyle) {
				return nil, "请选择球杆杆型"
			}
		}
		product.CueStyle = req.CueStyle
		if len(req.Roles) > 0 {
			return nil, "球杆不能设置冲跳身份"
		}
	case model.GearCategoryBreakJump:
		roles, ok := model.NormalizeGearRoles(req.Roles)
		if !ok || roles == "" {
			return nil, "请至少选择冲杆或跳杆身份"
		}
		if req.CueStyle != "" {
			return nil, "冲杆和跳杆不设置杆型"
		}
		product.Roles = roles
	default:
		if req.CueStyle != "" || len(req.Roles) > 0 {
			return nil, "皮头和巧克不设置杆型与身份"
		}
	}

	if req.LengthMm < 0 || req.LengthMm > adminGearMaxLengthMm {
		return nil, fmt.Sprintf("长度需在 0 到 %d 毫米之间，0 表示未提供", adminGearMaxLengthMm)
	}
	if req.WeightG < 0 || req.WeightG > adminGearMaxWeightG {
		return nil, fmt.Sprintf("重量需在 0 到 %d 克之间，0 表示未提供", adminGearMaxWeightG)
	}
	if req.TipDiameterMm < 0 || req.TipDiameterMm > adminGearMaxTipDiameterMm {
		return nil, fmt.Sprintf("皮头直径需在 0 到 %.0f 毫米之间，0 表示未提供", adminGearMaxTipDiameterMm)
	}
	product.LengthMm, product.WeightG, product.TipDiameterMm = req.LengthMm, req.WeightG, roundTo2(req.TipDiameterMm)

	imagePath := strings.TrimSpace(req.ImagePath)
	if !isValidGearImagePath(imagePath) {
		return nil, "目录图只能使用 /static/images/gear/ 下的 png 或 webp 文件"
	}
	product.ImagePath = imagePath
	return product, ""
}
