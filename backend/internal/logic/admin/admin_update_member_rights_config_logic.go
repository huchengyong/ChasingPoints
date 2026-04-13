package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateMemberRightsConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新会员成长与排位权益配置
func NewAdminUpdateMemberRightsConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateMemberRightsConfigLogic {
	return &AdminUpdateMemberRightsConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateMemberRightsConfigLogic) AdminUpdateMemberRightsConfig(req *types.AdminMemberRightsConfigUpdateReq) (resp *types.AdminWriteResp, err error) {
	if err := validateMemberRightsConfigUpdateReq(req); err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: err.Error(),
		}, nil
	}

	adminID, _ := utils.GetAdminIDFromCtx(l.ctx)
	cfg := buildMemberRightsConfigModel(req, int64(adminID))
	if err := l.svcCtx.MemberRightsConfigModel.Upsert(cfg); err != nil {
		l.Logger.Errorf("保存会员体系配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存配置失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "保存成功",
	}, nil
}
