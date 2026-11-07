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

type DeleteGearItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除装备
func NewDeleteGearItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteGearItemLogic {
	return &DeleteGearItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *DeleteGearItemLogic) DeleteGearItem(req *types.DeleteGearItemReq) (resp *types.CommonResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.CommonResp{Success: false, Message: "用户未登录"}, nil
	}
	if req == nil || req.ItemId <= 0 {
		return &types.CommonResp{Success: false, Message: "装备不存在"}, nil
	}

	gears := l.svcCtx.GearModel
	err = withUserLock(l.svcCtx, userId, func(tx *gorm.DB) error {
		item, err := gears.FindItemWithTx(tx, req.ItemId)
		if err != nil {
			return err
		}
		if !isOwnedAvailableItem(item, userId) {
			return newUserError("装备不存在")
		}
		// 软删除：历史对局的用杆记录仍要能解析这件装备。
		item.Status = model.GearItemStatusDeleted
		if err := gears.SaveItemWithTx(tx, item); err != nil {
			return err
		}
		return gears.DeleteDefaultsByItemWithTx(tx, userId, item.Id)
	})
	if err != nil {
		message := userMessage(err, "删除失败")
		if message == "删除失败" {
			l.Logger.Errorf("删除装备失败: userId=%d itemId=%d err=%v", userId, req.ItemId, err)
		}
		return &types.CommonResp{Success: false, Message: message}, nil
	}
	return &types.CommonResp{Success: true, Message: "已删除"}, nil
}
