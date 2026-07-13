package friend

import (
	"context"
	"fmt"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type BlacklistFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 加入黑名单
func NewBlacklistFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BlacklistFriendLogic {
	return &BlacklistFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BlacklistFriendLogic) BlacklistFriend(req *types.BlacklistFriendReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.FriendUserId == userIdInt {
		return &types.CommonResp{Success: false, Message: "不能将自己加入黑名单"}, nil
	}

	targetUser, err := l.svcCtx.UserModel.FindById(req.FriendUserId)
	if err != nil {
		l.Logger.Errorf("查询目标用户失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if targetUser == nil {
		return &types.CommonResp{Success: false, Message: "用户不存在"}, nil
	}

	pendingRequests, err := l.svcCtx.FriendModel.GetPendingRequestsBetweenUsers(userIdInt, req.FriendUserId)
	if err != nil {
		l.Logger.Errorf("查询待处理好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	if err = l.svcCtx.FriendModel.BlacklistFriend(userIdInt, req.FriendUserId); err != nil {
		l.Logger.Errorf("加入黑名单失败: %v", err)
		return &types.CommonResp{Success: false, Message: "加入黑名单失败"}, nil
	}

	for _, request := range pendingRequests {
		senderName := "有球友"
		fromUser, userErr := l.svcCtx.UserModel.FindById(request.FromUserId)
		if userErr != nil {
			l.Logger.Errorf("查询好友申请发起人失败: fromUserId=%d err=%v", request.FromUserId, userErr)
		} else if fromUser != nil && fromUser.Nickname != "" {
			senderName = fromUser.Nickname
		}

		legacyContent := buildFriendRequestNotificationContent(senderName, request.Message)
		if notifyErr := l.svcCtx.NotificationModel.DeleteFriendRequestNotification(request.ToUserId, request.Id, legacyContent); notifyErr != nil {
			l.Logger.Errorf("清理好友申请通知失败: requestId=%d toUserId=%d err=%v", request.Id, request.ToUserId, notifyErr)
			return &types.CommonResp{Success: false, Message: fmt.Sprintf("已加入黑名单，但清理通知失败")}, nil
		}
	}

	return &types.CommonResp{Success: true, Message: "已加入黑名单"}, nil
}
