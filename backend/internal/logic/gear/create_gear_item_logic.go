package gear

import (
	"context"
	"fmt"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CreateGearItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 登记装备
func NewCreateGearItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGearItemLogic {
	return &CreateGearItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreateGearItemLogic) CreateGearItem(req *types.CreateGearItemReq) (resp *types.GearItemResp, err error) {
	fail := func(message string) *types.GearItemResp {
		return &types.GearItemResp{Success: false, Message: message}
	}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return fail("用户未登录"), nil
	}
	if req == nil || (req.Category != model.GearCategoryCue && req.Category != model.GearCategoryBreakJump) {
		return fail("装备类别无效"), nil
	}

	item, err := l.buildItem(userId, req)
	if err != nil {
		return l.failed(userId, err, "登记失败"), nil
	}

	err = withUserLock(l.svcCtx, userId, func(tx *gorm.DB) error {
		count, err := l.svcCtx.GearModel.CountUserItemsWithTx(tx, userId)
		if err != nil {
			return err
		}
		if count >= model.GearMaxItemsPerUser {
			return newUserError(fmt.Sprintf("装备最多 %d 件，请先删除不用的装备", model.GearMaxItemsPerUser))
		}
		return l.svcCtx.GearModel.CreateItemWithTx(tx, item)
	})
	if err != nil {
		return l.failed(userId, err, "登记失败"), nil
	}

	catalog, err := loadGearCatalog(l.svcCtx.GearModel, []model.UserGearItem{*item})
	if err != nil {
		// 装备已经登记成功，展示信息读取失败时让客户端刷新列表即可。
		l.Logger.Errorf("登记后读取装备展示信息失败: userId=%d itemId=%d err=%v", userId, item.Id, err)
		return &types.GearItemResp{Success: true, Message: "已添加"}, nil
	}
	info := catalog.itemInfo(*item, true)
	return &types.GearItemResp{Success: true, Message: "已添加", Item: &info}, nil
}

func (l *CreateGearItemLogic) failed(userId int64, err error, fallback string) *types.GearItemResp {
	message := userMessage(err, fallback)
	if message == fallback {
		l.Logger.Errorf("登记装备失败: userId=%d err=%v", userId, err)
	}
	return &types.GearItemResp{Success: false, Message: message}
}

// buildItem 校验请求并构造待写入的装备；所有校验失败都以 userError 返回。
func (l *CreateGearItemLogic) buildItem(userId int64, req *types.CreateGearItemReq) (*model.UserGearItem, error) {
	gears := l.svcCtx.GearModel
	item := &model.UserGearItem{UserId: userId, Category: req.Category, Status: model.GearItemStatusActive}

	if req.ProductId > 0 {
		product, err := gears.FindProduct(req.ProductId)
		if err != nil {
			return nil, err
		}
		if product == nil || product.Category != req.Category {
			return nil, newUserError("型号不存在")
		}
		if err := checkProductAvailable(gears, product, "装备"); err != nil {
			return nil, err
		}
		item.ProductId = product.Id
	} else if err := l.fillUnlisted(item, req); err != nil {
		return nil, err
	}

	if req.TipProductId != 0 || req.ChalkProductId != 0 {
		if req.Category != model.GearCategoryCue {
			return nil, newUserError("冲杆和跳杆不能设置皮头与巧克")
		}
		if err := checkSubProduct(gears, req.TipProductId, model.GearCategoryTip, true); err != nil {
			return nil, err
		}
		if err := checkSubProduct(gears, req.ChalkProductId, model.GearCategoryChalk, true); err != nil {
			return nil, err
		}
		item.TipProductId, item.ChalkProductId = req.TipProductId, req.ChalkProductId
	}
	return item, nil
}

// fillUnlisted 填充未收录型号：品牌（目录品牌或自填）、型号名，以及杆型或冲跳身份。
func (l *CreateGearItemLogic) fillUnlisted(item *model.UserGearItem, req *types.CreateGearItemReq) error {
	modelName := strings.TrimSpace(req.ModelName)
	if n := runeLength(modelName); n == 0 || n > modelNameMaxRunes {
		return newUserError(fmt.Sprintf("请填写型号名称（不超过 %d 字）", modelNameMaxRunes))
	}

	if req.BrandId > 0 {
		brand, err := l.svcCtx.GearModel.FindBrand(req.BrandId)
		if err != nil {
			return err
		}
		if brand == nil || brand.Status != 1 {
			return newUserError("品牌不存在")
		}
		item.CustomBrandId = brand.Id
	} else {
		brandName := strings.TrimSpace(req.BrandName)
		if n := runeLength(brandName); n == 0 || n > brandNameMaxRunes {
			return newUserError(fmt.Sprintf("请填写品牌名称（不超过 %d 字）", brandNameMaxRunes))
		}
		item.CustomBrandName = brandName
	}

	if req.Category == model.GearCategoryCue {
		if !model.IsGearCueStyle(req.CueStyle) {
			return newUserError("请选择杆型")
		}
		item.CueStyle = req.CueStyle
	} else {
		roles, ok := model.NormalizeGearRoles(req.Roles)
		if !ok || roles == "" {
			return newUserError("请选择冲杆或跳杆")
		}
		item.Roles = roles
	}

	item.CustomModelName = modelName
	item.CustomKey = model.BuildGearCustomKey(item.Category, item.CustomBrandId, item.CustomBrandName, modelName)
	return nil
}
