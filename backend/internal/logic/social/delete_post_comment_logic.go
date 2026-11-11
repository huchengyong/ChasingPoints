package social

import (
	"context"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const deleteCommentDeniedMessage = "评论不存在或无权删除"

type DeletePostCommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除评论
func NewDeletePostCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePostCommentLogic {
	return &DeletePostCommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *DeletePostCommentLogic) DeletePostComment(req *types.DeleteCommentReq) (resp *types.CommonResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.CommonResp{Success: false, Message: config.DisabledFeatureMessage("social")}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	if req.CommentId <= 0 {
		return &types.CommonResp{Success: false, Message: deleteCommentDeniedMessage}, nil
	}

	comment, err := l.svcCtx.SocialPostModel.FindCommentById(req.CommentId)
	if err != nil {
		l.Logger.Errorf("查询评论失败: commentId=%d err=%v", req.CommentId, err)
		return &types.CommonResp{Success: false, Message: "删除评论失败"}, nil
	}
	if comment == nil {
		return &types.CommonResp{Success: false, Message: deleteCommentDeniedMessage}, nil
	}

	post, err := l.svcCtx.SocialPostModel.FindById(comment.PostId)
	if err != nil {
		l.Logger.Errorf("查询评论所属动态失败: postId=%d err=%v", comment.PostId, err)
		return &types.CommonResp{Success: false, Message: "删除评论失败"}, nil
	}
	if post == nil || (userIdInt != comment.UserId && userIdInt != post.UserId) {
		return &types.CommonResp{Success: false, Message: deleteCommentDeniedMessage}, nil
	}

	if err = l.svcCtx.SocialPostModel.DeleteComment(comment.Id); err != nil {
		l.Logger.Errorf("删除评论失败: commentId=%d userId=%d err=%v", comment.Id, userIdInt, err)
		return &types.CommonResp{Success: false, Message: "删除评论失败"}, nil
	}

	return &types.CommonResp{Success: true, Message: "删除成功"}, nil
}
