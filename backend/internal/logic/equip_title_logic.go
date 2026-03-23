package logic

import (
	"context"
	"errors"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type EquipTitleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 装备/卸下称号
func NewEquipTitleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EquipTitleLogic {
	return &EquipTitleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EquipTitleLogic) EquipTitle(req *types.EquipTitleReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	err = l.svcCtx.UserTitleModel.EquipTitle(userIdInt, req.TitleId, req.Equip)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &types.CommonResp{Success: false, Message: "称号不存在"}, nil
		}
		l.Logger.Errorf("更新称号装备状态失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	message := "称号已卸下"
	if req.Equip {
		message = "称号已装备"
	}

	return &types.CommonResp{
		Success: true,
		Message: message,
	}, nil
}
