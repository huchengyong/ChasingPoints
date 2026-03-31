package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserPrivacyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户隐私设置
func NewGetUserPrivacyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserPrivacyLogic {
	return &GetUserPrivacyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserPrivacyLogic) GetUserPrivacy() (resp *types.GetUserPrivacyResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserPrivacyResp{Success: false}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户隐私设置失败: userId=%d err=%v", userID, err)
		return &types.GetUserPrivacyResp{Success: false}, nil
	}
	if user == nil {
		return &types.GetUserPrivacyResp{Success: false}, nil
	}

	return &types.GetUserPrivacyResp{
		Success:         true,
		HideMatchRecord: user.HideMatchRecord,
	}, nil
}
