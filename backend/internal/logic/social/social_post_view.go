package social

import (
	"encoding/json"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

const deletedUserNickname = "已注销用户"

func socialPostStatusText(status int) string {
	switch status {
	case model.SocialPostStatusPublished:
		return "已发布"
	case model.SocialPostStatusPending:
		return "审核中，仅自己可见"
	case model.SocialPostStatusRejected:
		return "审核未通过"
	default:
		return "未知状态"
	}
}

// socialPostViewData 是一页动态共用的组装数据，避免逐条查作者和点赞。
type socialPostViewData struct {
	users map[int64]model.User
	liked map[int64]struct{}
}

func loadSocialPostViewData(svcCtx *svc.ServiceContext, posts []model.SocialPost, currentUserID int64) (*socialPostViewData, error) {
	userIds := make([]int64, 0, len(posts))
	postIds := make([]int64, 0, len(posts))
	for _, post := range posts {
		userIds = append(userIds, post.UserId)
		if post.Status == model.SocialPostStatusPublished {
			postIds = append(postIds, post.Id)
		}
	}

	users, err := svcCtx.UserModel.FindByIds(userIds)
	if err != nil {
		return nil, err
	}
	liked, err := svcCtx.SocialPostModel.FindLikedPostIds(currentUserID, postIds)
	if err != nil {
		return nil, err
	}
	return &socialPostViewData{users: users, liked: liked}, nil
}

func buildSocialPostInfo(post model.SocialPost, view *socialPostViewData, currentUserID int64, includePrivate bool) types.SocialPostInfo {
	nickname := deletedUserNickname
	avatar := ""
	if user, ok := view.users[post.UserId]; ok {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	_, isLiked := view.liked[post.Id]

	images := make([]string, 0)
	if post.Images != nil && *post.Images != "" {
		if unmarshalErr := json.Unmarshal([]byte(*post.Images), &images); unmarshalErr != nil {
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
		Status:        post.Status,
		StatusText:    socialPostStatusText(post.Status),
		CreatedAt:     post.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if post.MatchId != nil {
		item.MatchId = *post.MatchId
	}
	if includePrivate {
		item.RejectReason = post.RejectReason
		item.IsMine = currentUserID > 0 && currentUserID == post.UserId
	}

	return item
}
