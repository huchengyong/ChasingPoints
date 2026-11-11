package social

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetPostCommentsLogic) GetPostComments(req *types.GetPostCommentsReq) (resp *types.GetPostCommentsResp, err error) {
	if req.PostId <= 0 {
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}

	post, err := l.svcCtx.SocialPostModel.FindById(req.PostId)
	if err != nil {
		l.Logger.Errorf("查询动态失败: postId=%d err=%v", req.PostId, err)
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}
	if post == nil || post.Status != model.SocialPostStatusPublished {
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}

	comments, total, err := l.svcCtx.SocialPostModel.GetComments(req.PostId, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询动态评论失败: postId=%d err=%v", req.PostId, err)
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}

	userIds := make([]int64, 0, len(comments))
	for _, comment := range comments {
		userIds = append(userIds, comment.UserId)
	}
	users, err := l.svcCtx.UserModel.FindByIds(userIds)
	if err != nil {
		l.Logger.Errorf("批量查询评论用户失败: postId=%d err=%v", req.PostId, err)
		return &types.GetPostCommentsResp{Success: false, Total: 0, List: []types.PostCommentInfo{}}, nil
	}

	list := make([]types.PostCommentInfo, 0, len(comments))
	for _, comment := range comments {
		nickname := deletedUserNickname
		avatar := ""
		if user, ok := users[comment.UserId]; ok {
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
