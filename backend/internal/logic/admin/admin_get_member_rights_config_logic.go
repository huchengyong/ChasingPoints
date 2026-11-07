package admin

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetMemberRightsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会员成长与排位权益配置
func NewAdminGetMemberRightsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetMemberRightsConfigLogic {
	return &AdminGetMemberRightsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetMemberRightsConfigLogic) AdminGetMemberRightsConfig() (resp *types.AdminMemberRightsConfigResp, err error) {
	config, err := logicx.NewMemberRightsConfigService(l.svcCtx).GetConfig()
	if err != nil {
		l.Logger.Errorf("获取会员体系配置失败: %v", err)
		return &types.AdminMemberRightsConfigResp{
			Code:    500,
			Success: false,
			Message: "获取配置失败",
		}, nil
	}
	return buildAdminMemberRightsConfigResp(config), nil
}
