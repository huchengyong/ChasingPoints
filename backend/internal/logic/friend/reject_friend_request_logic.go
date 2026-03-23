package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
		svcCtx: svcCtx,
	}
}

func (l *RejectFriendRequestLogic) RejectFriendRequest(req *types.HandleFriendRequestReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
