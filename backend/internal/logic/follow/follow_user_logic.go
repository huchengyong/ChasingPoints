package follow

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 关注用户
func NewFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowUserLogic {
	return &FollowUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowUserLogic) FollowUser(req *types.FollowReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.TargetUserId <= 0 {
		return &types.CommonResp{Success: false, Message: "目标用户无效"}, nil
	}

	if userIdInt == req.TargetUserId {
		return &types.CommonResp{Success: false, Message: "不能关注自己"}, nil
	}

	if err = l.svcCtx.FollowModel.Follow(userIdInt, req.TargetUserId); err != nil {
		l.Logger.Errorf("关注用户失败: follower=%d following=%d err=%v", userIdInt, req.TargetUserId, err)
		return &types.CommonResp{Success: false, Message: "关注失败"}, nil
	}

	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      req.TargetUserId,
		Type:        "follow",
		Title:       "你有新的关注",
		Content:     "有用户关注了你",
		PushTitle:   "你有新的关注",
		PushContent: "有用户关注了你",
		WSCategory:  "follow",
	}); notifyErr != nil {
		l.Logger.Errorf("分发关注通知失败: target=%d err=%v", req.TargetUserId, notifyErr)
	}

	return &types.CommonResp{Success: true, Message: "关注成功"}, nil
}
