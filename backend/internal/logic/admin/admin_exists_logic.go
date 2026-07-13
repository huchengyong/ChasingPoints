package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminExistsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 检查管理员账号是否存在
func NewAdminExistsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminExistsLogic {
	return &AdminExistsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminExistsLogic) AdminExists() (resp *types.AdminExistsResp, err error) {
	count, err := l.svcCtx.AdminModel.Count(l.ctx)
	if err != nil {
		l.Logger.Errorf("检查管理员存在性失败: %v", err)
		return &types.AdminExistsResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
			Exists:  false,
		}, nil
	}

	return &types.AdminExistsResp{
		Code:    0,
		Success: true,
		Message: "success",
		Exists:  count > 0,
	}, nil
}
