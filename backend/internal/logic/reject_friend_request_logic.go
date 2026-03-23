package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

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
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if err = l.svcCtx.FriendModel.RejectRequest(req.RequestId, userIdInt); err != nil {
		l.Logger.Errorf("拒绝好友请求失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "操作成功"}, nil
}
