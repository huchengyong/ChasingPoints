package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectFriendRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拒绝好友请求
func NewRejectFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectFriendRequestLogic {
	return &RejectFriendRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RejectFriendRequestLogic) RejectFriendRequest(req *types.HandleFriendRequestReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	request, err := l.svcCtx.FriendModel.GetRequestById(req.RequestId)
	if err != nil {
		l.Logger.Errorf("查询好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if request == nil || request.ToUserId != userIdInt || request.Status != 0 {
		return &types.CommonResp{Success: false, Message: "好友请求不存在或已处理"}, nil
	}

	if err = l.svcCtx.FriendModel.RejectRequest(req.RequestId, userIdInt); err != nil {
		l.Logger.Errorf("拒绝好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	sendFriendRequestCountUpdated(l.svcCtx, userIdInt, "friend_requests")

	// 拒绝后清理对应的“收到好友申请”通知；失败只记日志，不回滚已完成的拒绝。
	if l.svcCtx.NotificationModel != nil {
		senderName := "有球友"
		if fromUser, userErr := l.svcCtx.UserModel.FindById(request.FromUserId); userErr != nil {
			l.Logger.Errorf("查询好友申请发起人失败: fromUserId=%d err=%v", request.FromUserId, userErr)
		} else if fromUser != nil && fromUser.Nickname != "" {
			senderName = fromUser.Nickname
		}
		legacyContent := buildFriendRequestNotificationContent(senderName, request.Message)
		if notifyErr := l.svcCtx.NotificationModel.DeleteFriendRequestNotification(request.ToUserId, request.Id, legacyContent); notifyErr != nil {
			l.Logger.Errorf("清理好友申请通知失败: requestId=%d toUserId=%d err=%v", request.Id, request.ToUserId, notifyErr)
		}
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
