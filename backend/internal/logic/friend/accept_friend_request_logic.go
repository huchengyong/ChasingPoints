package friend

import (
	"context"
	"fmt"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptFriendRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 接受好友请求
func NewAcceptFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptFriendRequestLogic {
	return &AcceptFriendRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AcceptFriendRequestLogic) AcceptFriendRequest(req *types.HandleFriendRequestReq) (resp *types.CommonResp, err error) {
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

	if err = l.svcCtx.FriendModel.AcceptRequest(req.RequestId, userIdInt); err != nil {
		l.Logger.Errorf("接受好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	if err = l.svcCtx.FriendModel.AddFriend(userIdInt, request.FromUserId); err != nil {
		l.Logger.Errorf("添加好友失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	myNickname := "你的好友"
	me, userErr := l.svcCtx.UserModel.FindById(userIdInt)
	if userErr == nil && me != nil && me.Nickname != "" {
		myNickname = me.Nickname
	}

	content := fmt.Sprintf("%s 已通过你的好友请求", myNickname)
	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      request.FromUserId,
		Type:        "friend_request",
		Title:       "好友请求已通过",
		Content:     content,
		Data:        buildNotificationPayload("/subPages/social/friendList", 0, request.Id),
		PushTitle:   "好友请求已通过",
		PushContent: content,
		PushData: map[string]interface{}{
			"url": "/subPages/social/friendList",
		},
		WSCategory: "friend_request",
	}); notifyErr != nil {
		l.Logger.Errorf("分发好友通知失败: %v", notifyErr)
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
