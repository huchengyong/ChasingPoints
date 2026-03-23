package logic

import (
	"context"
	"strings"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CommentPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 评论动态
func NewCommentPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentPostLogic {
	return &CommentPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentPostLogic) CommentPost(req *types.CommentPostReq) (resp *types.CommonResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.PostId <= 0 {
		return &types.CommonResp{Success: false, Message: "动态不存在"}, nil
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return &types.CommonResp{Success: false, Message: "评论内容不能为空"}, nil
	}

	comment := &model.SocialPostComment{
		PostId:  req.PostId,
		UserId:  userIdInt,
		Content: content,
	}
	if err = l.svcCtx.SocialPostModel.AddComment(comment); err != nil {
		l.Logger.Errorf("评论动态失败: postId=%d userId=%d err=%v", req.PostId, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "评论失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "评论成功"}, nil
}
