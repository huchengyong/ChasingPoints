package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除好友
func NewDeleteFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFriendLogic {
	return &DeleteFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteFriendLogic) DeleteFriend(req *types.DeleteFriendReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	areFriends, err := l.svcCtx.FriendModel.AreFriends(userIdInt, req.FriendUserId)
	if err != nil {
		l.Logger.Errorf("检查好友关系失败: %v", err)
		return &types.CommonResp{Success: false, Message: "操作失败"}, nil
	}
	if !areFriends {
		return &types.CommonResp{Success: false, Message: "对方不是你的好友"}, nil
	}

	if err = l.svcCtx.FriendModel.DeleteFriend(userIdInt, req.FriendUserId); err != nil {
		l.Logger.Errorf("删除好友失败: %v", err)
		return &types.CommonResp{Success: false, Message: "删除失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "删除成功"}, nil
}
