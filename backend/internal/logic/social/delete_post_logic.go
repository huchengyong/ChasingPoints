package social

import (
	"context"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除动态
func NewDeletePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePostLogic {
	return &DeletePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeletePostLogic) DeletePost(req *types.PostIdReq) (resp *types.CommonResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.CommonResp{Success: false, Message: config.DisabledFeatureMessage("social")}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.PostId <= 0 {
		return &types.CommonResp{Success: false, Message: "动态不存在"}, nil
	}

	if err = l.svcCtx.SocialPostModel.Delete(userIdInt, req.PostId); err != nil {
		l.Logger.Errorf("删除动态失败: postId=%d userId=%d err=%v", req.PostId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "删除失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "删除成功"}, nil
}
