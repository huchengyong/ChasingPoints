package challenge

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发起挑战
func NewSendChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendChallengeLogic {
	return &SendChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *SendChallengeLogic) SendChallenge(req *types.SendChallengeReq) (resp *types.SendChallengeResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.SendChallengeResp{Success: false}, nil
	}

	if req.ToUserId <= 0 || req.ToUserId == userIdInt {
		return &types.SendChallengeResp{Success: false}, nil
	}
	if req.GameType < 1 || req.GameType > 3 {
		return &types.SendChallengeResp{Success: false}, nil
	}

	areFriends, err := l.svcCtx.FriendModel.AreFriends(userIdInt, req.ToUserId)
	if err != nil {
		l.Logger.Errorf("检查好友关系失败: from=%d to=%d err=%v", userIdInt, req.ToUserId, err)
		return &types.SendChallengeResp{Success: false}, nil
	}
	if !areFriends {
		return &types.SendChallengeResp{Success: false}, nil
	}

	toUser, err := l.svcCtx.UserModel.FindById(req.ToUserId)
	if err != nil {
		l.Logger.Errorf("查询挑战目标用户失败: to=%d err=%v", req.ToUserId, err)
		return &types.SendChallengeResp{Success: false}, nil
	}
	if toUser == nil {
		return &types.SendChallengeResp{Success: false}, nil
	}

	challenge := &model.Challenge{
		FromUserId: userIdInt,
		ToUserId:   req.ToUserId,
		GameType:   req.GameType,
		Message:    req.Message,
		Status:     0,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	if err = l.svcCtx.ChallengeModel.Create(challenge); err != nil {
		l.Logger.Errorf("创建挑战失败: from=%d to=%d err=%v", userIdInt, req.ToUserId, err)
		return &types.SendChallengeResp{Success: false}, nil
	}

	fromName := "好友"
	if fromUser, userErr := l.svcCtx.UserModel.FindById(userIdInt); userErr == nil && fromUser != nil && fromUser.Nickname != "" {
		fromName = fromUser.Nickname
	}

	content := fromName + " 向你发起了挑战"
	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      req.ToUserId,
		Type:        "challenge",
		Title:       "你收到了一条挑战",
		Content:     content,
		PushTitle:   "你收到了一条挑战",
		PushContent: content,
		WSCategory:  "challenge",
	}); notifyErr != nil {
		l.Logger.Errorf("分发挑战通知失败: to=%d err=%v", req.ToUserId, notifyErr)
	}

	return &types.SendChallengeResp{Success: true, ChallengeId: challenge.Id}, nil
}
