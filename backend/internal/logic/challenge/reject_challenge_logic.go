package challenge

import (
	"context"
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拒绝挑战
func NewRejectChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectChallengeLogic {
	return &RejectChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RejectChallengeLogic) RejectChallenge(req *types.HandleChallengeReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	challenge, err := l.svcCtx.ChallengeModel.FindById(req.ChallengeId)
	if err != nil {
		l.Logger.Errorf("查询挑战失败: challengeId=%d err=%v", req.ChallengeId, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if challenge == nil || challenge.ToUserId != userIdInt || challenge.Status != 0 {
		return &types.CommonResp{Success: false, Message: "挑战不存在或已处理"}, nil
	}
	if !challenge.ExpiresAt.After(time.Now()) {
		return &types.CommonResp{Success: false, Message: "挑战已过期"}, nil
	}

	if err = l.svcCtx.ChallengeModel.Reject(req.ChallengeId, userIdInt); err != nil {
		l.Logger.Errorf("拒绝挑战失败: challengeId=%d userId=%d err=%v", req.ChallengeId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	myName := "好友"
	if me, userErr := l.svcCtx.UserModel.FindById(userIdInt); userErr == nil && me != nil && me.Nickname != "" {
		myName = me.Nickname
	}

	content := fmt.Sprintf("%s 拒绝了你的挑战", myName)
	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      challenge.FromUserId,
		Type:        "challenge",
		Title:       "挑战被拒绝",
		Content:     content,
		PushTitle:   "挑战被拒绝",
		PushContent: content,
		WSCategory:  "challenge",
	}); notifyErr != nil {
		l.Logger.Errorf("分发挑战拒绝通知失败: from=%d err=%v", challenge.FromUserId, notifyErr)
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
