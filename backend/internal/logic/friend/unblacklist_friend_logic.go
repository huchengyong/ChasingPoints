package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnblacklistFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 移出黑名单
func NewUnblacklistFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnblacklistFriendLogic {
	return &UnblacklistFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UnblacklistFriendLogic) UnblacklistFriend(req *types.UnblacklistFriendReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if req == nil || req.FriendUserId <= 0 {
		return &types.CommonResp{Success: false, Message: "对方不在黑名单中"}, nil
	}

	removed, err := l.svcCtx.FriendModel.RemoveBlacklist(userIdInt, req.FriendUserId)
	if err != nil {
		l.Logger.Errorf("移出黑名单失败: user=%d blocked=%d err=%v", userIdInt, req.FriendUserId, err)
		return &types.CommonResp{Success: false, Message: "移出黑名单失败"}, nil
	}
	if !removed {
		return &types.CommonResp{Success: false, Message: "对方不在黑名单中"}, nil
	}

	return &types.CommonResp{Success: true, Message: "已移出黑名单"}, nil
}
