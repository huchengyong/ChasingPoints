package admin

import (
	"encoding/json"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func adminSocialPostStatusText(status int) string {
	switch status {
	case model.SocialPostStatusPublished:
		return "已发布"
	case model.SocialPostStatusPending:
		return "待审核"
	case model.SocialPostStatusRejected:
		return "已拒绝"
	default:
		return "未知状态"
	}
}

func buildAdminSocialPostInfo(svcCtx *svc.ServiceContext, post model.SocialPost) (types.AdminSocialPostInfo, error) {
	nickname := ""
	avatar := ""
	user, err := svcCtx.UserModel.FindById(post.UserId)
	if err != nil {
		return types.AdminSocialPostInfo{}, err
	}
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	images := make([]string, 0)
	if post.Images != nil && *post.Images != "" {
		if unmarshalErr := json.Unmarshal([]byte(*post.Images), &images); unmarshalErr != nil {
			images = []string{}
		}
	}

	item := types.AdminSocialPostInfo{
		Id:            post.Id,
		UserId:        post.UserId,
		Nickname:      nickname,
		Avatar:        avatar,
		Content:       post.Content,
		Images:        images,
		PostType:      post.PostType,
		LikesCount:    post.LikesCount,
		CommentsCount: post.CommentsCount,
		Status:        post.Status,
		StatusText:    adminSocialPostStatusText(post.Status),
		RejectReason:  post.RejectReason,
		CreatedAt:     post.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if post.MatchId != nil {
		item.MatchId = *post.MatchId
	}
	if post.ReviewedAt != nil {
		item.ReviewedAt = post.ReviewedAt.Format("2006-01-02 15:04:05")
	}
	if post.ReviewedBy != nil {
		item.ReviewedBy = *post.ReviewedBy
	}

	return item, nil
}
