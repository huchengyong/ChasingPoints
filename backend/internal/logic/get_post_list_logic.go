package logic

import (
	"context"
	"encoding/json"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取关注的人的动态
func NewGetPostListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostListLogic {
	return &GetPostListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostListLogic) GetPostList(req *types.GetPostListReq) (resp *types.GetPostListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetPostListResp{Success: false}, nil
	}

	followingIds := make([]int64, 0)
	if err = l.svcCtx.DB.Table("follows").Where("follower_id = ?", userIdInt).Pluck("following_id", &followingIds).Error; err != nil {
		l.Logger.Errorf("查询关注列表失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPostListResp{Success: false}, nil
	}
	if len(followingIds) == 0 {
		return &types.GetPostListResp{Success: true, Total: 0, List: []types.SocialPostInfo{}}, nil
	}

	posts, total, err := l.svcCtx.SocialPostModel.FindPostsByUserIds(followingIds, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询关注动态失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPostListResp{Success: false}, nil
	}

	list := make([]types.SocialPostInfo, 0, len(posts))
	for _, post := range posts {
		nickname := ""
		avatar := ""
		user, userErr := l.svcCtx.UserModel.FindById(post.UserId)
		if userErr != nil {
			l.Logger.Errorf("查询动态作者失败: postId=%d userId=%d err=%v", post.Id, post.UserId, userErr)
			continue
		}
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		isLiked, likeErr := l.svcCtx.SocialPostModel.HasLiked(post.Id, userIdInt)
		if likeErr != nil {
			l.Logger.Errorf("查询动态点赞状态失败: postId=%d userId=%d err=%v", post.Id, userIdInt, likeErr)
			return &types.GetPostListResp{Success: false}, nil
		}

		images := make([]string, 0)
		if post.Images != nil && *post.Images != "" {
			if unmarshalErr := json.Unmarshal([]byte(*post.Images), &images); unmarshalErr != nil {
				l.Logger.Errorf("解析动态图片失败: postId=%d err=%v", post.Id, unmarshalErr)
				images = []string{}
			}
		}

		item := types.SocialPostInfo{
			Id:            post.Id,
			UserId:        post.UserId,
			Nickname:      nickname,
			Avatar:        avatar,
			Content:       post.Content,
			Images:        images,
			PostType:      post.PostType,
			LikesCount:    post.LikesCount,
			CommentsCount: post.CommentsCount,
			IsLiked:       isLiked,
			CreatedAt:     post.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if post.MatchId != nil {
			item.MatchId = *post.MatchId
		}

		list = append(list, item)
	}

	return &types.GetPostListResp{Success: true, Total: total, List: list}, nil
}
