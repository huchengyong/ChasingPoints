package gear

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateGearItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改装备状态或皮头巧克
func NewUpdateGearItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGearItemLogic {
	return &UpdateGearItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdateGearItemLogic) UpdateGearItem(req *types.UpdateGearItemReq) (resp *types.GearItemResp, err error) {
	fail := func(message string) *types.GearItemResp {
		return &types.GearItemResp{Success: false, Message: message}
	}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return fail("用户未登录"), nil
	}
	if req == nil || req.ItemId <= 0 {
		return fail("装备不存在"), nil
	}
	if req.Status != 0 && req.Status != model.GearItemStatusActive && req.Status != model.GearItemStatusRetired {
		return fail("装备状态无效"), nil
	}

	gears := l.svcCtx.GearModel
	var updated model.UserGearItem
	err = withUserLock(l.svcCtx, userId, func(tx *gorm.DB) error {
		item, err := gears.FindItemWithTx(tx, req.ItemId)
		if err != nil {
			return err
		}
		if !isOwnedAvailableItem(item, userId) {
			return newUserError("装备不存在")
		}

		if req.UpdateTip || req.UpdateChalk {
			if item.Category != model.GearCategoryCue {
				return newUserError("冲杆和跳杆不能设置皮头与巧克")
			}
			if req.UpdateTip {
				if err := checkSubProduct(gears, req.TipProductId, model.GearCategoryTip, true); err != nil {
					return err
				}
				item.TipProductId = req.TipProductId
			}
			if req.UpdateChalk {
				if err := checkSubProduct(gears, req.ChalkProductId, model.GearCategoryChalk, true); err != nil {
					return err
				}
				item.ChalkProductId = req.ChalkProductId
			}
		}

		if req.Status != 0 && req.Status != item.Status {
			item.Status = req.Status
			if req.Status == model.GearItemStatusRetired {
				// 退役的装备不能继续作为主力。
				if err := gears.DeleteDefaultsByItemWithTx(tx, userId, item.Id); err != nil {
					return err
				}
			}
		}

		if err := gears.SaveItemWithTx(tx, item); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		message := userMessage(err, "修改失败")
		if message == "修改失败" {
			l.Logger.Errorf("修改装备失败: userId=%d itemId=%d err=%v", userId, req.ItemId, err)
		}
		return fail(message), nil
	}

	catalog, err := loadGearCatalog(gears, []model.UserGearItem{updated})
	if err != nil {
		l.Logger.Errorf("修改后读取装备展示信息失败: userId=%d itemId=%d err=%v", userId, updated.Id, err)
		return &types.GearItemResp{Success: true, Message: "已保存"}, nil
	}
	info := catalog.itemInfo(updated, true)
	return &types.GearItemResp{Success: true, Message: "已保存", Item: &info}, nil
}
