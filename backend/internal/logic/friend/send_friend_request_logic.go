package friend

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendFriendRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送好友请求
func NewSendFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendFriendRequestLogic {
	return &SendFriendRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendFriendRequestLogic) SendFriendRequest(req *types.SendFriendRequestReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.ToUserId == userIdInt {
		return &types.CommonResp{Success: false, Message: "不能添加自己为好友"}, nil
	}

	toUser, err := l.svcCtx.UserModel.FindById(req.ToUserId)
	if err != nil {
		l.Logger.Errorf("查询目标用户失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if toUser == nil {
		return &types.CommonResp{Success: false, Message: "用户不存在"}, nil
	}

	hasBlacklistRelation, err := l.svcCtx.FriendModel.HasBlacklistRelation(userIdInt, req.ToUserId)
	if err != nil {
		l.Logger.Errorf("检查黑名单关系失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if hasBlacklistRelation {
		return &types.CommonResp{Success: false, Message: "由于隐私设置，无法发送好友申请"}, nil
	}

	areFriends, err := l.svcCtx.FriendModel.AreFriends(userIdInt, req.ToUserId)
	if err != nil {
		l.Logger.Errorf("检查好友关系失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if areFriends {
		return &types.CommonResp{Success: false, Message: "你们已经是好友"}, nil
	}

	friendCount, err := l.svcCtx.FriendModel.GetFriendCount(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询好友数量失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if friendCount >= 500 {
		return &types.CommonResp{Success: false, Message: "好友数量已达上限"}, nil
	}

	hasPending, err := l.svcCtx.FriendModel.HasPendingRequest(userIdInt, req.ToUserId)
	if err != nil {
		l.Logger.Errorf("检查好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if hasPending {
		return &types.CommonResp{Success: false, Message: "已有待处理的好友请求"}, nil
	}

	friendRequest, err := l.svcCtx.FriendModel.SendRequest(userIdInt, req.ToUserId, req.Message)
	if err != nil {
		l.Logger.Errorf("发送好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}

	fromUserName := "有球友"
	if fromUser, fromUserErr := l.svcCtx.UserModel.FindById(userIdInt); fromUserErr == nil && fromUser != nil && fromUser.Nickname != "" {
		fromUserName = fromUser.Nickname
	}

	content := buildFriendRequestNotificationContent(fromUserName, req.Message)

	if notifyErr := l.svcCtx.NotificationModel.Create(&model.Notification{
		UserId:  req.ToUserId,
		Type:    "friend_request",
		Title:   "收到好友申请",
		Content: content,
		Data:    buildNotificationPayload("/subPages/social/friendRequests", 0, friendRequest.Id),
		IsRead:  0,
	}); notifyErr != nil {
		l.Logger.Errorf("创建好友申请通知失败: %v", notifyErr)
	}

	if toUser.PushToken != "" {
		l.svcCtx.PushService.SendPush(toUser.PushToken, "收到好友申请", content, map[string]interface{}{
			"url": "/subPages/social/friendRequests",
		})
	}

	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(req.ToUserId, &ws.Message{
			Type: "notification_update",
			Data: map[string]interface{}{
				"category": "friend_request",
			},
		})
	}

	return &types.CommonResp{Success: true, Message: "发送成功"}, nil
}

func buildFriendRequestNotificationContent(fromUserName, message string) string {
	content := fmt.Sprintf("%s 向你发送了好友申请", fromUserName)
	if message != "" {
		content = fmt.Sprintf("%s：%s", content, message)
	}
	return content
}
