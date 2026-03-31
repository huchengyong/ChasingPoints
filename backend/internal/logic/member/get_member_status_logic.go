package member

import (
	"context"

	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会员状态
func NewGetMemberStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberStatusLogic {
	return &GetMemberStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMemberStatusLogic) GetMemberStatus() (resp *types.GetMemberStatusResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMemberStatusResp{Success: false}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
		return &types.GetMemberStatusResp{Success: false}, nil
	}
	if user == nil {
		return &types.GetMemberStatusResp{Success: false}, nil
	}

	return paymentlogic.BuildMemberStatusResp(user), nil
}
