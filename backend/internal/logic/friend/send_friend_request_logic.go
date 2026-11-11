package friend

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	friendRequestMessageRuneLimit = 200
	friendRequestCooldown         = 7 * 24 * time.Hour
	friendRequestDailyLimit       = 30
	friendLimit                   = 500
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *SendFriendRequestLogic) SendFriendRequest(req *types.SendFriendRequestReq) (resp *types.CommonResp, err error) {
	resp = &types.CommonResp{}
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.ToUserId == userIdInt {
		return &types.CommonResp{Success: false, Message: "不能添加自己为好友"}, nil
	}

	message := strings.TrimSpace(req.Message)
	if utf8.RuneCountInString(message) > friendRequestMessageRuneLimit {
		return &types.CommonResp{Success: false, Message: "留言不能超过 200 字"}, nil
	}

	toUser, err := l.svcCtx.UserModel.FindById(req.ToUserId)
	if err != nil {
		l.Logger.Errorf("查询目标用户失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if toUser == nil {
		return &types.CommonResp{Success: false, Message: "用户不存在"}, nil
	}

	var friendRequest *model.FriendRequest
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, []int64{userIdInt, req.ToUserId}); err != nil {
			return err
		}
		now := time.Now()

		hasBlacklistRelation, err := l.svcCtx.FriendModel.HasBlacklistRelationWithTx(tx, userIdInt, req.ToUserId)
		if err != nil {
			return err
		}
		if hasBlacklistRelation {
			resp.Message = "由于隐私设置，无法发送好友申请"
			return nil
		}

		areFriends, err := l.svcCtx.FriendModel.AreFriendsWithTx(tx, userIdInt, req.ToUserId)
		if err != nil {
			return err
		}
		if areFriends {
			resp.Message = "你们已经是好友"
			return nil
		}

		hasPending, err := l.svcCtx.FriendModel.HasPendingRequestWithTx(tx, userIdInt, req.ToUserId)
		if err != nil {
			return err
		}
		if hasPending {
			resp.Message = "已有待处理的好友请求"
			return nil
		}

		latest, err := l.svcCtx.FriendModel.FindLatestRequestWithTx(tx, userIdInt, req.ToUserId)
		if err != nil {
			return err
		}
		if latest != nil && latest.Status == 2 && latest.UpdatedAt.After(now.Add(-friendRequestCooldown)) {
			resp.Message = "近期已申请过，请过几天再试"
			return nil
		}

		dailyCount, err := l.svcCtx.FriendModel.CountRequestsSinceWithTx(tx, userIdInt, now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		if dailyCount >= friendRequestDailyLimit {
			resp.Message = "今天发送的好友申请太多了，请明天再试"
			return nil
		}

		friendCount, err := l.svcCtx.FriendModel.GetFriendCountWithTx(tx, userIdInt)
		if err != nil {
			return err
		}
		if friendCount >= friendLimit {
			resp.Message = "好友数量已达上限"
			return nil
		}

		friendRequest, err = l.svcCtx.FriendModel.SendRequestWithTx(tx, userIdInt, req.ToUserId, message)
		return err
	})
	if err != nil {
		l.Logger.Errorf("发送好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "发送失败"}, nil
	}
	if friendRequest == nil {
		return resp, nil
	}

	fromUserName := "有球友"
	if fromUser, fromUserErr := l.svcCtx.UserModel.FindById(userIdInt); fromUserErr == nil && fromUser != nil && fromUser.Nickname != "" {
		fromUserName = fromUser.Nickname
	}

	content := buildFriendRequestNotificationContent(fromUserName, message)
	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      req.ToUserId,
		Type:        "friend_request",
		Title:       "收到好友申请",
		Content:     content,
		Data:        buildNotificationPayload("/subPages/social/friendRequests", 0, friendRequest.Id),
		PushTitle:   "收到好友申请",
		PushContent: content,
		PushData: map[string]interface{}{
			"url": "/subPages/social/friendRequests",
		},
		WSCategory: "friend_request",
	}); notifyErr != nil {
		l.Logger.Errorf("分发好友申请通知失败: %v", notifyErr)
	}
	sendFriendRequestCountUpdated(l.svcCtx, req.ToUserId, "friend_requests")

	return &types.CommonResp{Success: true, Message: "发送成功"}, nil
}

func buildFriendRequestNotificationContent(fromUserName, message string) string {
	content := fmt.Sprintf("%s 向你发送了好友申请", fromUserName)
	if message != "" {
		content = fmt.Sprintf("%s：%s", content, message)
	}
	return content
}
