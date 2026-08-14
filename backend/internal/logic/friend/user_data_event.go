package friend

import (
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

func sendFriendRequestCountUpdated(svcCtx *svc.ServiceContext, userID int64, scopes ...string) {
	if svcCtx == nil || svcCtx.FriendModel == nil || userID <= 0 {
		return
	}
	count, err := svcCtx.FriendModel.GetPendingRequestCount(userID)
	if err != nil {
		logx.Errorf("查询好友申请精确计数失败: userId=%d err=%v", userID, err)
		return
	}
	logicx.SendUserDataUpdated(userID, logicx.UserDataUpdatedEvent{
		Scopes:                    scopes,
		PendingFriendRequestCount: &count,
	})
}
