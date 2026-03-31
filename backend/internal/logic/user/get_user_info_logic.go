package user

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前用户信息
func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserInfoLogic) GetUserInfo() (resp *types.GetUserInfoResp, err error) {
	// 从上下文获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserInfoResp{
			Success: false,
		}, nil
	}

	// 查询用户信息
	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil {
		l.Logger.Errorf("查询用户失败: %v", err)
		return &types.GetUserInfoResp{
			Success: false,
		}, nil
	}

	if user == nil {
		return &types.GetUserInfoResp{
			Success: false,
		}, nil
	}

	// 脱敏手机号
	maskedPhone := ""
	if user.Phone != nil {
		phone := *user.Phone
		if len(phone) == 11 {
			maskedPhone = phone[:3] + "****" + phone[7:]
		}
	}

	return &types.GetUserInfoResp{
		Success: true,
		UserInfo: &types.UserInfo{
			Id:        user.Id,
			Phone:     maskedPhone,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Status:    user.Status,
			CreatedAt: logicx.FormatUTC8Time(user.CreatedAt),
		},
	}, nil
}
