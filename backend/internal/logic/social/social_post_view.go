package social

import (
	"encoding/json"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

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

func buildSocialPostInfo(svcCtx *svc.ServiceContext, post model.SocialPost, currentUserID int64, includePrivate bool) (types.SocialPostInfo, bool, error) {
	nickname := ""
	avatar := ""
	user, err := svcCtx.UserModel.FindById(post.UserId)
	if err != nil {
		return types.SocialPostInfo{}, false, err
	}
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	isLiked := false
	if currentUserID > 0 && post.Status == model.SocialPostStatusPublished {
		isLiked, err = svcCtx.SocialPostModel.HasLiked(post.Id, currentUserID)
		if err != nil {
			return types.SocialPostInfo{}, false, err
		}
	}

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

	return item, true, nil
}
