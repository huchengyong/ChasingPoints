package admin

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateReputationConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新信誉制度配置
func NewAdminUpdateReputationConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateReputationConfigLogic {
	return &AdminUpdateReputationConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminUpdateReputationConfigLogic) AdminUpdateReputationConfig(req *types.AdminReputationConfigUpdateReq) (resp *types.AdminWriteResp, err error) {
	if err := validateReputationConfigUpdateReq(req); err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: err.Error(),
		}, nil
	}

	adminID, _ := utils.GetAdminIDFromCtx(l.ctx)
	cfg := buildReputationConfigModel(req, int64(adminID))
	if err := l.svcCtx.ReputationConfigModel.Upsert(cfg); err != nil {
		l.Logger.Errorf("保存信誉制度配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存配置失败",
		}, nil
	}
	logicx.InvalidateRuntimeConfigCache(l.ctx, l.svcCtx, logicx.ReputationRuntimeConfigCacheKey)

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "保存成功",
	}, nil
}
