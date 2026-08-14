package follow

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnfollowUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消关注
func NewUnfollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnfollowUserLogic {
	return &UnfollowUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UnfollowUserLogic) UnfollowUser(req *types.FollowReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.TargetUserId <= 0 {
		return &types.CommonResp{Success: false, Message: "目标用户无效"}, nil
	}

	if err = l.svcCtx.FollowModel.Unfollow(userIdInt, req.TargetUserId); err != nil {
		l.Logger.Errorf("取消关注失败: follower=%d following=%d err=%v", userIdInt, req.TargetUserId, err)
		return &types.CommonResp{Success: false, Message: "取消关注失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "取消关注成功"}, nil
}
