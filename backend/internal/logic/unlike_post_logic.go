package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlikePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消点赞
func NewUnlikePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikePostLogic {
	return &UnlikePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnlikePostLogic) UnlikePost(req *types.PostIdReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.PostId <= 0 {
		return &types.CommonResp{Success: false, Message: "动态不存在"}, nil
	}

	if err = l.svcCtx.SocialPostModel.RemoveLike(req.PostId, userIdInt); err != nil {
		l.Logger.Errorf("取消点赞失败: postId=%d userId=%d err=%v", req.PostId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "取消点赞失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "取消点赞成功"}, nil
}
