package friend

import (
	"context"
	"errors"
	"fmt"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AcceptFriendRequestLogic) AcceptFriendRequest(req *types.HandleFriendRequestReq) (resp *types.CommonResp, err error) {
	resp = &types.CommonResp{}
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

	// 申请状态与双向好友关系在同一事务内写入，避免留下“已接受但不是好友”。
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, []int64{userIdInt, request.FromUserId}); err != nil {
			return err
		}
		current, err := l.svcCtx.FriendModel.GetRequestByIdWithTx(tx, req.RequestId)
		if err != nil {
			return err
		}
		if current == nil || current.ToUserId != userIdInt || current.Status != 0 {
			resp.Message = "好友请求不存在或已处理"
			return nil
		}

		blacklisted, err := l.svcCtx.FriendModel.HasBlacklistRelationWithTx(tx, userIdInt, current.FromUserId)
		if err != nil {
			return err
		}
		if blacklisted {
			resp.Message = "由于隐私设置，无法添加对方为好友"
			return nil
		}

		myCount, err := l.svcCtx.FriendModel.GetFriendCountWithTx(tx, userIdInt)
		if err != nil {
			return err
		}
		theirCount, err := l.svcCtx.FriendModel.GetFriendCountWithTx(tx, current.FromUserId)
		if err != nil {
			return err
		}
		if myCount >= friendLimit || theirCount >= friendLimit {
			resp.Message = "好友数量已达上限"
			return nil
		}

		if err := l.svcCtx.FriendModel.AcceptRequestWithTx(tx, current.Id, userIdInt); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				resp.Message = "好友请求不存在或已处理"
				return nil
			}
			return err
		}
		if err := l.svcCtx.FriendModel.AddFriendWithTx(tx, userIdInt, current.FromUserId); err != nil {
			return err
		}
		request = current
		return nil
	})
	if err != nil {
		l.Logger.Errorf("接受好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if resp.Message != "" {
		return resp, nil
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
	sendFriendRequestCountUpdated(l.svcCtx, userIdInt, "friend_requests", "opponents")
	logicx.SendUserDataUpdated(request.FromUserId, logicx.UserDataUpdatedEvent{Scopes: []string{"opponents"}})

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
