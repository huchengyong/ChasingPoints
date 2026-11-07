package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetLlmConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取大模型调用配置
func NewAdminGetLlmConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetLlmConfigLogic {
	return &AdminGetLlmConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetLlmConfigLogic) AdminGetLlmConfig() (resp *types.AdminLlmConfigResp, err error) {
	cfg, err := l.svcCtx.LlmConfigModel.FindByKey(model.DefaultLlmConfigKey)
	if err != nil {
		l.Logger.Errorf("获取大模型调用配置失败: %v", err)
		return &types.AdminLlmConfigResp{
			Code:    500,
			Success: false,
			Message: "获取配置失败",
		}, nil
	}

	return buildAdminLlmConfigResp(cfg), nil
}
