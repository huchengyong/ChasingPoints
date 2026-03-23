package logic

import (
	"context"
	"fmt"
	"time"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 接受挑战
func NewAcceptChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptChallengeLogic {
	return &AcceptChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AcceptChallengeLogic) AcceptChallenge(req *types.HandleChallengeReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if err = l.svcCtx.ChallengeModel.ExpireOld(); err != nil {
		l.Logger.Errorf("过期挑战清理失败: err=%v", err)
	}

	challenge, err := l.svcCtx.ChallengeModel.FindById(req.ChallengeId)
	if err != nil {
		l.Logger.Errorf("查询挑战失败: challengeId=%d err=%v", req.ChallengeId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if challenge == nil || challenge.ToUserId != userIdInt || challenge.Status != 0 {
		return &types.CommonResp{Success: false, Message: "挑战不存在或已处理"}, nil
	}
	if challenge.ExpiresAt.Before(time.Now()) {
		_ = l.svcCtx.ChallengeModel.ExpireOld()
		return &types.CommonResp{Success: false, Message: "挑战已过期"}, nil
	}

	var matchId int64
	if err = l.svcCtx.ChallengeModel.Accept(req.ChallengeId, userIdInt, matchId); err != nil {
		l.Logger.Errorf("接受挑战失败: challengeId=%d userId=%d err=%v", req.ChallengeId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	myName := "好友"
	if me, userErr := l.svcCtx.UserModel.FindById(userIdInt); userErr == nil && me != nil && me.Nickname != "" {
		myName = me.Nickname
	}

	if notifyErr := l.svcCtx.NotificationModel.Create(&model.Notification{
		UserId:  challenge.FromUserId,
		Type:    "challenge",
		Title:   "挑战已接受",
		Content: fmt.Sprintf("%s 接受了你的挑战", myName),
		IsRead:  0,
	}); notifyErr != nil {
	l.Logger.Errorf("创建挑战接受通知失败: from=%d err=%v", challenge.FromUserId, notifyErr)
	}

	// 推送通知
	if targetUser, pushErr := l.svcCtx.UserModel.FindById(challenge.FromUserId); pushErr == nil && targetUser != nil && targetUser.PushToken != "" {
		l.svcCtx.PushService.SendPush(targetUser.PushToken, "挑战已接受", fmt.Sprintf("%s 接受了你的挑战", myName), nil)
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
