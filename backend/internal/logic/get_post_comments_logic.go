package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostCommentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取动态评论
func NewGetPostCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostCommentsLogic {
	return &GetPostCommentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostCommentsLogic) GetPostComments(req *types.GetPostCommentsReq) (resp *types.GetPostCommentsResp, err error) {
	comments, total, err := l.svcCtx.SocialPostModel.GetComments(req.PostId, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询动态评论失败: postId=%d err=%v", req.PostId, err)
		return &types.GetPostCommentsResp{Success: false}, nil
	}

	list := make([]types.PostCommentInfo, 0, len(comments))
	for _, comment := range comments {
		nickname := ""
		avatar := ""
		user, userErr := l.svcCtx.UserModel.FindById(comment.UserId)
		if userErr != nil {
			l.Logger.Errorf("查询评论用户失败: commentId=%d userId=%d err=%v", comment.Id, comment.UserId, userErr)
			continue
		}
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		list = append(list, types.PostCommentInfo{
			Id:        comment.Id,
			UserId:    comment.UserId,
			Nickname:  nickname,
			Avatar:    avatar,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetPostCommentsResp{Success: true, Total: total, List: list}, nil
}
