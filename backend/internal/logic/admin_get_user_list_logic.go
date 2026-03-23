package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户列表（管理员）
func NewAdminGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUserListLogic {
	return &AdminGetUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetUserListLogic) AdminGetUserList(req *types.AdminUserListReq) (resp *types.AdminUserListResp, err error) {
	users, total, err := l.svcCtx.UserModel.FindListForAdmin(req.Page, req.PageSize, req.Status)
	if err != nil {
		l.Logger.Errorf("获取用户列表失败: err=%v", err)
		return &types.AdminUserListResp{
			Code:    500,
			Success: false,
			Message: "获取用户列表失败",
		}, nil
	}

	items := make([]types.AdminUserInfo, 0, len(users))
	for _, user := range users {
		// 脱敏手机号
		phone := ""
		if user.Phone != nil && len(*user.Phone) == 11 {
			phone = (*user.Phone)[:3] + "****" + (*user.Phone)[7:]
		}

		items = append(items, types.AdminUserInfo{
			Id:        user.Id,
			Phone:     phone,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.AdminUserListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
