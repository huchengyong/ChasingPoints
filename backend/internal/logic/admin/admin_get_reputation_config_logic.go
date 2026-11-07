package admin

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetReputationConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取信誉制度配置
func NewAdminGetReputationConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetReputationConfigLogic {
	return &AdminGetReputationConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetReputationConfigLogic) AdminGetReputationConfig() (resp *types.AdminReputationConfigResp, err error) {
	config, err := logicx.NewReputationConfigService(l.svcCtx).GetConfig()
	if err != nil {
		l.Logger.Errorf("获取信誉制度配置失败: %v", err)
		return &types.AdminReputationConfigResp{
			Code:    500,
			Success: false,
			Message: "获取配置失败",
		}, nil
	}

	return buildAdminReputationConfigResp(config), nil
}
