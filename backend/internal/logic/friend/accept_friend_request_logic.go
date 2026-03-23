package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
