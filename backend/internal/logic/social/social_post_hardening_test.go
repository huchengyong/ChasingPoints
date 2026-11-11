package social

import (
	"context"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const socialTestPublicDomain = "https://cdn.example.com"

func TestCreatePostTrimsContentAndKeepsValidImages(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config.Qiniu.PublicDomain = socialTestPublicDomain
	seedSocialUser(t, svcCtx, &model.User{Id: 301, Nickname: "作者"})

	logic := NewCreatePostLogic(socialUserCtx(301), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{
		Content:  "  今天手感不错  ",
		Images:   []string{" https://cdn.example.com/a.jpg "},
		PostType: 2,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if !resp.Success || resp.Message != "" {
		t.Fatalf("expected success without message, got %#v", resp)
	}

	post, err := svcCtx.SocialPostModel.FindById(resp.PostId)
	if err != nil || post == nil {
		t.Fatalf("find created post: post=%v err=%v", post, err)
	}
	if post.Content != "今天手感不错" {
		t.Fatalf("expected trimmed content, got %q", post.Content)
	}
	if post.Images == nil || *post.Images != `["https://cdn.example.com/a.jpg"]` {
		t.Fatalf("expected trimmed image url, got %v", post.Images)
	}
	if post.PostType != 2 || post.MatchId != nil {
		t.Fatalf("unexpected post type/match id: %#v", post)
	}
}

func TestCreatePostRejectsLongContent(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 302, Nickname: "作者"})

	logic := NewCreatePostLogic(socialUserCtx(302), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{Content: strings.Repeat("字", 1001), PostType: 3})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "内容不能超过 1000 字" {
		t.Fatalf("expected content length rejection, got %#v", resp)
	}

	resp, err = logic.CreatePost(&types.CreatePostReq{Content: strings.Repeat("字", 1000), PostType: 3})
	if err != nil {
		t.Fatalf("create post at limit: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected 1000 runes to be accepted, got %#v", resp)
	}
}

func TestCreatePostRejectsEmptyContentAndImages(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config.Qiniu.PublicDomain = socialTestPublicDomain
	seedSocialUser(t, svcCtx, &model.User{Id: 303, Nickname: "作者"})

	logic := NewCreatePostLogic(socialUserCtx(303), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{Content: "   ", PostType: 3})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "说点什么再发布吧" {
		t.Fatalf("expected empty content rejection, got %#v", resp)
	}
}

func TestCreatePostRejectsTooManyImages(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config.Qiniu.PublicDomain = socialTestPublicDomain
	seedSocialUser(t, svcCtx, &model.User{Id: 304, Nickname: "作者"})

	images := make([]string, 10)
	for i := range images {
		images[i] = socialTestPublicDomain + "/a.jpg"
	}
	logic := NewCreatePostLogic(socialUserCtx(304), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{Images: images, PostType: 3})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "最多上传 9 张图片" {
		t.Fatalf("expected image count rejection, got %#v", resp)
	}
}

func TestCreatePostRejectsInvalidImageHost(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config.Qiniu.PublicDomain = socialTestPublicDomain
	seedSocialUser(t, svcCtx, &model.User{Id: 305, Nickname: "作者"})

	logic := NewCreatePostLogic(socialUserCtx(305), svcCtx)
	for _, raw := range []string{
		"http://cdn.example.com/a.jpg",
		"https://evil.example.net/a.jpg",
		"not-a-url",
	} {
		resp, err := logic.CreatePost(&types.CreatePostReq{Images: []string{raw}, PostType: 3})
		if err != nil {
			t.Fatalf("create post %q: %v", raw, err)
		}
		if resp.Success || resp.Message != "图片地址无效" {
			t.Fatalf("expected invalid image %q rejection, got %#v", raw, resp)
		}
	}

	svcCtx.Config.Qiniu.PublicDomain = ""
	resp, err := logic.CreatePost(&types.CreatePostReq{
		Images:   []string{socialTestPublicDomain + "/a.jpg"},
		PostType: 3,
	})
	if err != nil {
		t.Fatalf("create post without public domain: %v", err)
	}
	if resp.Success || resp.Message != "图片地址无效" {
		t.Fatalf("expected unconfigured public domain rejection, got %#v", resp)
	}
}

func TestCreatePostNormalizesPostType(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 306, Nickname: "作者"})

	logic := NewCreatePostLogic(socialUserCtx(306), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{Content: "默认类型", PostType: 0})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected default type success, got %#v", resp)
	}
	post, err := svcCtx.SocialPostModel.FindById(resp.PostId)
	if err != nil || post == nil {
		t.Fatalf("find created post: post=%v err=%v", post, err)
	}
	if post.PostType != 3 {
		t.Fatalf("expected post type 3, got %d", post.PostType)
	}

	resp, err = logic.CreatePost(&types.CreatePostReq{Content: "非法类型", PostType: 4})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "动态类型无效" {
		t.Fatalf("expected invalid type rejection, got %#v", resp)
	}
}

func TestCreatePostMatchRules(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 307, Nickname: "作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 308, Nickname: "对手"})
	seedSocialUser(t, svcCtx, &model.User{Id: 309, Nickname: "路人"})
	if err := svcCtx.DB.Exec("INSERT INTO matches (id, user_id, opponent_id) VALUES (?, ?, ?)", 501, 307, 308).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	if err := svcCtx.DB.Exec("INSERT INTO matches (id, user_id, opponent_id) VALUES (?, ?, ?)", 502, 309, nil).Error; err != nil {
		t.Fatalf("seed other match: %v", err)
	}

	logic := NewCreatePostLogic(socialUserCtx(307), svcCtx)

	resp, err := logic.CreatePost(&types.CreatePostReq{Content: "战报", PostType: 1})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "战报需要关联比赛" {
		t.Fatalf("expected missing match rejection, got %#v", resp)
	}

	resp, err = logic.CreatePost(&types.CreatePostReq{Content: "别人的战报", PostType: 1, MatchId: 502})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "只能分享自己参与的比赛" {
		t.Fatalf("expected foreign match rejection, got %#v", resp)
	}
	if resp, err = logic.CreatePost(&types.CreatePostReq{Content: "不存在的战报", PostType: 1, MatchId: 999}); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != "只能分享自己参与的比赛" {
		t.Fatalf("expected missing match rejection, got %#v", resp)
	}

	resp, err = logic.CreatePost(&types.CreatePostReq{Content: "对手视角的战报", PostType: 1, MatchId: 501})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected match participant success, got %#v", resp)
	}
	post, err := svcCtx.SocialPostModel.FindById(resp.PostId)
	if err != nil || post == nil {
		t.Fatalf("find created post: post=%v err=%v", post, err)
	}
	if post.MatchId == nil || *post.MatchId != 501 {
		t.Fatalf("expected match 501, got %v", post.MatchId)
	}

	resp, err = logic.CreatePost(&types.CreatePostReq{Content: "普通动态", PostType: 3, MatchId: 501})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected ignored match success, got %#v", resp)
	}
	post, err = svcCtx.SocialPostModel.FindById(resp.PostId)
	if err != nil || post == nil {
		t.Fatalf("find created post: post=%v err=%v", post, err)
	}
	if post.MatchId != nil {
		t.Fatalf("expected match_id to be ignored, got %v", *post.MatchId)
	}
}

func TestCreatePostDisabledFeatureReturnsMessage(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config = config.Config{Compliance: config.ComplianceConfig{RestrictedMode: true}}

	resp, err := NewCreatePostLogic(socialUserCtx(310), svcCtx).CreatePost(&types.CreatePostReq{Content: "关闭中"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success || resp.Message != config.DisabledFeatureMessage("social") {
		t.Fatalf("expected disabled feature message, got %#v", resp)
	}
}

func TestCommentPostRejectsLongContent(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 321, Nickname: "作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 322, Nickname: "评论者"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 601, UserId: 321, Content: "动态", PostType: 3, Status: model.SocialPostStatusPublished})

	logic := NewCommentPostLogic(socialUserCtx(322), svcCtx)
	resp, err := logic.CommentPost(&types.CommentPostReq{PostId: 601, Content: strings.Repeat("字", 301)})
	if err != nil {
		t.Fatalf("comment post: %v", err)
	}
	if resp.Success || resp.Message != "评论不能超过 300 字" {
		t.Fatalf("expected comment length rejection, got %#v", resp)
	}

	resp, err = logic.CommentPost(&types.CommentPostReq{PostId: 601, Content: strings.Repeat("字", 300)})
	if err != nil {
		t.Fatalf("comment post at limit: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected 300 runes to be accepted, got %#v", resp)
	}
}

func TestCommentPostRateLimit(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 323, Nickname: "作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 324, Nickname: "评论者"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 602, UserId: 323, Content: "动态", PostType: 3, Status: model.SocialPostStatusPublished})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 603, UserId: 323, Content: "另一条", PostType: 3, Status: model.SocialPostStatusPublished})

	for i := 0; i < 10; i++ {
		if err := svcCtx.SocialPostModel.AddComment(&model.SocialPostComment{PostId: 602, UserId: 324, Content: "旧评论"}); err != nil {
			t.Fatalf("seed comment: %v", err)
		}
	}

	logic := NewCommentPostLogic(socialUserCtx(324), svcCtx)
	resp, err := logic.CommentPost(&types.CommentPostReq{PostId: 603, Content: "第十一条"})
	if err != nil {
		t.Fatalf("comment post: %v", err)
	}
	if resp.Success || resp.Message != "评论太频繁了，稍后再试" {
		t.Fatalf("expected rate limit rejection, got %#v", resp)
	}

	if err := svcCtx.DB.Exec("UPDATE social_post_comments SET created_at = ? WHERE user_id = ?", time.Now().Add(-2*time.Minute), 324).Error; err != nil {
		t.Fatalf("age comments: %v", err)
	}
	resp, err = logic.CommentPost(&types.CommentPostReq{PostId: 603, Content: "过一分钟后可以评论"})
	if err != nil {
		t.Fatalf("comment post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected aged comments to release rate limit, got %#v", resp)
	}
}

func TestDeletePostCommentPermissionsAndCount(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 331, Nickname: "动态作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 332, Nickname: "评论作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 333, Nickname: "第三人"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 611, UserId: 331, Content: "动态", PostType: 3, Status: model.SocialPostStatusPublished})

	comment := &model.SocialPostComment{PostId: 611, UserId: 332, Content: "评论"}
	if err := svcCtx.SocialPostModel.AddComment(comment); err != nil {
		t.Fatalf("seed comment: %v", err)
	}

	thirdResp, err := NewDeletePostCommentLogic(socialUserCtx(333), svcCtx).
		DeletePostComment(&types.DeleteCommentReq{CommentId: comment.Id})
	if err != nil {
		t.Fatalf("delete comment: %v", err)
	}
	if thirdResp.Success || thirdResp.Message != "评论不存在或无权删除" {
		t.Fatalf("expected third party denial, got %#v", thirdResp)
	}

	missingResp, err := NewDeletePostCommentLogic(socialUserCtx(331), svcCtx).
		DeletePostComment(&types.DeleteCommentReq{CommentId: 9999})
	if err != nil {
		t.Fatalf("delete comment: %v", err)
	}
	if missingResp.Success || missingResp.Message != "评论不存在或无权删除" {
		t.Fatalf("expected missing comment denial, got %#v", missingResp)
	}

	postAuthorResp, err := NewDeletePostCommentLogic(socialUserCtx(331), svcCtx).
		DeletePostComment(&types.DeleteCommentReq{CommentId: comment.Id})
	if err != nil {
		t.Fatalf("delete comment: %v", err)
	}
	if !postAuthorResp.Success {
		t.Fatalf("expected post author deletion success, got %#v", postAuthorResp)
	}
	post, err := svcCtx.SocialPostModel.FindById(611)
	if err != nil || post == nil {
		t.Fatalf("find post: post=%v err=%v", post, err)
	}
	if post.CommentsCount != 0 {
		t.Fatalf("expected comments_count 0, got %d", post.CommentsCount)
	}

	ownComment := &model.SocialPostComment{PostId: 611, UserId: 332, Content: "自己的评论"}
	if err := svcCtx.SocialPostModel.AddComment(ownComment); err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	ownResp, err := NewDeletePostCommentLogic(socialUserCtx(332), svcCtx).
		DeletePostComment(&types.DeleteCommentReq{CommentId: ownComment.Id})
	if err != nil {
		t.Fatalf("delete own comment: %v", err)
	}
	if !ownResp.Success {
		t.Fatalf("expected comment author deletion success, got %#v", ownResp)
	}
}

func TestDeletePostCleansLikesAndComments(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 341, Nickname: "作者"})
	seedSocialUser(t, svcCtx, &model.User{Id: 342, Nickname: "读者"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 621, UserId: 341, Content: "待删除", PostType: 3, Status: model.SocialPostStatusPublished})
	if err := svcCtx.SocialPostModel.AddLike(621, 342); err != nil {
		t.Fatalf("seed like: %v", err)
	}
	if err := svcCtx.SocialPostModel.AddComment(&model.SocialPostComment{PostId: 621, UserId: 342, Content: "评论"}); err != nil {
		t.Fatalf("seed comment: %v", err)
	}

	resp, err := NewDeletePostLogic(socialUserCtx(341), svcCtx).DeletePost(&types.PostIdReq{PostId: 621})
	if err != nil {
		t.Fatalf("delete post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected delete success, got %#v", resp)
	}

	var likes, comments int64
	if err := svcCtx.DB.Model(&model.SocialPostLike{}).Where("post_id = ?", 621).Count(&likes).Error; err != nil {
		t.Fatalf("count likes: %v", err)
	}
	if err := svcCtx.DB.Model(&model.SocialPostComment{}).Where("post_id = ?", 621).Count(&comments).Error; err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if likes != 0 || comments != 0 {
		t.Fatalf("expected likes/comments cleaned, got likes=%d comments=%d", likes, comments)
	}
}

func TestGetPostCommentsVisibilityAndMissingAuthor(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 351, Nickname: "作者"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 631, UserId: 351, Content: "动态", PostType: 3, Status: model.SocialPostStatusPublished})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 632, UserId: 351, Content: "审核中", PostType: 3, Status: model.SocialPostStatusPending})
	if err := svcCtx.SocialPostModel.AddComment(&model.SocialPostComment{PostId: 631, UserId: 999, Content: "已注销用户的评论"}); err != nil {
		t.Fatalf("seed comment: %v", err)
	}

	logic := NewGetPostCommentsLogic(context.Background(), svcCtx)

	invalidResp, err := logic.GetPostComments(&types.GetPostCommentsReq{PostId: 0})
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if invalidResp.Success {
		t.Fatalf("expected invalid post id failure, got %#v", invalidResp)
	}

	hiddenResp, err := logic.GetPostComments(&types.GetPostCommentsReq{PostId: 632, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if hiddenResp.Success {
		t.Fatalf("expected pending post failure, got %#v", hiddenResp)
	}
	if hiddenResp.List == nil || len(hiddenResp.List) != 0 {
		t.Fatalf("expected empty non-nil list, got %#v", hiddenResp.List)
	}

	resp, err := logic.GetPostComments(&types.GetPostCommentsReq{PostId: 631, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("expected one comment, got %#v", resp)
	}
	if resp.List[0].Nickname != "已注销用户" {
		t.Fatalf("expected deleted user nickname, got %#v", resp.List[0])
	}
}

func TestSocialPostListsKeepMissingAuthorAndCapPageSize(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 361, Nickname: "作者"})
	seedSocialPost(t, svcCtx, &model.SocialPost{Id: 641, UserId: 999, Content: "作者已注销", PostType: 3, Status: model.SocialPostStatusPublished})
	for i := 0; i < 25; i++ {
		seedSocialPost(t, svcCtx, &model.SocialPost{
			Id:        int64(642 + i),
			UserId:    361,
			Content:   "动态",
			PostType:  3,
			Status:    model.SocialPostStatusPublished,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}

	resp, err := NewGetPublicPostsLogic(context.Background(), svcCtx).
		GetPublicPosts(&types.GetPostListReq{Page: 1, PageSize: 1000})
	if err != nil {
		t.Fatalf("get public posts: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if len(resp.List) != 20 {
		t.Fatalf("expected page size capped at 20, got %d", len(resp.List))
	}

	var missingAuthor *types.SocialPostInfo
	for i := range resp.List {
		if resp.List[i].Id == 641 {
			missingAuthor = &resp.List[i]
			break
		}
	}
	if missingAuthor == nil {
		t.Fatalf("expected missing-author post to be kept, got %#v", resp.List)
	}
	if missingAuthor.Nickname != "已注销用户" {
		t.Fatalf("expected deleted user nickname, got %#v", missingAuthor)
	}
}
