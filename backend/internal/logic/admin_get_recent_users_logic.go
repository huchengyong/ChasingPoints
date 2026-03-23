package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetRecentUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取最近注册用户
func NewAdminGetRecentUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetRecentUsersLogic {
	return &AdminGetRecentUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetRecentUsersLogic) AdminGetRecentUsers() (resp *types.AdminRecentUsersResp, err error) {
	users, err := l.svcCtx.UserModel.FindRecent(10)
	if err != nil {
		l.Logger.Errorf("获取最近用户失败: %v", err)
		return &types.AdminRecentUsersResp{
			Code:    500,
			Success: false,
			Message: "获取失败",
			List:    []types.AdminRecentUser{},
		}, nil
	}

	list := make([]types.AdminRecentUser, 0, len(users))
	for _, user := range users {
		phone := ""
		if user.Phone != nil {
			p := *user.Phone
			if len(p) == 11 {
				phone = p[:3] + "****" + p[7:]
			} else {
				phone = p
			}
		}
		list = append(list, types.AdminRecentUser{
			Id:        user.Id,
			Nickname:  user.Nickname,
			Phone:     phone,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	return &types.AdminRecentUsersResp{
		Code:    0,
		Success: true,
		Message: "success",
		List:    list,
	}, nil
}
