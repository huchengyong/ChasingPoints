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

	content := fmt.Sprintf("%s 接受了你的挑战", myName)
	if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
		UserId:      challenge.FromUserId,
		Type:        "challenge",
		Title:       "挑战已接受",
		Content:     content,
		PushTitle:   "挑战已接受",
		PushContent: content,
		WSCategory:  "challenge",
	}); notifyErr != nil {
		l.Logger.Errorf("分发挑战接受通知失败: from=%d err=%v", challenge.FromUserId, notifyErr)
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
