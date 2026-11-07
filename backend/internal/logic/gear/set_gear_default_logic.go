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

type SetGearDefaultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 设置或清除主力装备
func NewSetGearDefaultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetGearDefaultLogic {
	return &SetGearDefaultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *SetGearDefaultLogic) SetGearDefault(req *types.SetGearDefaultReq) (resp *types.CommonResp, err error) {
	fail := func(message string) *types.CommonResp {
		return &types.CommonResp{Success: false, Message: message}
	}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return fail("用户未登录"), nil
	}
	if req == nil || !isValidGameType(req.GameType) {
		return fail("球种无效"), nil
	}
	if req.Slot != model.GearSlotCue && req.Slot != model.GearSlotBreak {
		return fail("主力槽位无效"), nil
	}
	if req.Slot == model.GearSlotBreak && !supportsBreak(req.GameType) {
		return fail("该球种不记录冲杆"), nil
	}

	gears := l.svcCtx.GearModel
	err = withUserLock(l.svcCtx, userId, func(tx *gorm.DB) error {
		if req.ItemId == 0 {
			return gears.DeleteDefaultWithTx(tx, userId, req.GameType, req.Slot)
		}
		item, err := gears.FindItemWithTx(tx, req.ItemId)
		if err != nil {
			return err
		}
		if !isOwnedAvailableItem(item, userId) {
			return newUserError("装备不存在")
		}
		if item.Status != model.GearItemStatusActive {
			return newUserError("已退役的装备不能设为主力")
		}
		if err := checkDefaultSlot(gears, item, req.Slot); err != nil {
			return err
		}
		return gears.UpsertDefaultWithTx(tx, &model.UserGearDefault{
			UserId: userId, GameType: req.GameType, Slot: req.Slot, ItemId: item.Id,
		})
	})
	if err != nil {
		message := userMessage(err, "设置失败")
		if message == "设置失败" {
			l.Logger.Errorf("设置主力装备失败: userId=%d err=%v", userId, err)
		}
		return fail(message), nil
	}
	return &types.CommonResp{Success: true, Message: "已保存"}, nil
}

// checkDefaultSlot 校验装备是否适合该槽位：球杆槽位只能是球杆，冲杆槽位只能是带冲杆身份的冲跳装备。
func checkDefaultSlot(gears *model.GearModel, item *model.UserGearItem, slot string) error {
	switch slot {
	case model.GearSlotCue:
		if item.Category != model.GearCategoryCue {
			return newUserError("只能把球杆设为主力球杆")
		}
	case model.GearSlotBreak:
		if item.Category != model.GearCategoryBreakJump {
			return newUserError("只能把冲杆设为主力冲杆")
		}
		roles, err := itemRoles(gears, item)
		if err != nil {
			return err
		}
		if !model.GearRolesContain(roles, model.GearRoleBreak) {
			return newUserError("这件装备不是冲杆，不能设为主力冲杆")
		}
	}
	return nil
}
