package social

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSocialReviewTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareSocialSchema(db); err != nil {
		t.Fatalf("prepare social schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:              db,
		UserModel:       model.NewUserModel(db),
		FollowModel:     model.NewFollowModel(db),
		SocialPostModel: model.NewSocialPostModel(db),
	}
}

func socialUserCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedSocialUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create social user: %v", err)
	}
}

func seedSocialFollow(t *testing.T, svcCtx *svc.ServiceContext, followerID, followingID int64) {
	t.Helper()
	if err := svcCtx.FollowModel.Follow(followerID, followingID); err != nil {
		t.Fatalf("create social follow: %v", err)
	}
}

func seedSocialPost(t *testing.T, svcCtx *svc.ServiceContext, post *model.SocialPost) {
	t.Helper()
	if err := svcCtx.SocialPostModel.Create(post); err != nil {
		t.Fatalf("create social post: %v", err)
	}
}

func TestSocialPostReviewCreateDefaultsToPending(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 101, Nickname: "作者A", Avatar: "a.png"})

	logic := NewCreatePostLogic(socialUserCtx(101), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{
		Content:  "今天打得不错",
		PostType: 3,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected create success, got %#v", resp)
	}

	post, err := svcCtx.SocialPostModel.FindById(resp.PostId)
	if err != nil {
		t.Fatalf("find created post: %v", err)
	}
	if post == nil {
		t.Fatal("expected created post, got nil")
	}
	if post.Status != model.SocialPostStatusPending {
		t.Fatalf("expected pending status, got %d", post.Status)
	}
}

func TestSocialPostReviewCreateBlockedInComplianceMode(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	svcCtx.Config = config.Config{
		Compliance: config.ComplianceConfig{
			RestrictedMode: true,
		},
	}

	logic := NewCreatePostLogic(socialUserCtx(101), svcCtx)
	resp, err := logic.CreatePost(&types.CreatePostReq{
		Content:  "今天打得不错",
		PostType: 3,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected compliance mode to block create post, got %#v", resp)
	}
}

func TestSocialPostReviewPublicFeedOnlyReturnsPublished(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 102, Nickname: "作者B", Avatar: "b.png"})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        201,
		UserId:    102,
		Content:   "已发布动态",
		PostType:  3,
		Status:    model.SocialPostStatusPublished,
		CreatedAt: time.Now().Add(-time.Hour),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        202,
		UserId:    102,
		Content:   "待审核动态",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:           203,
		UserId:       102,
		Content:      "已拒绝动态",
		PostType:     3,
		Status:       model.SocialPostStatusRejected,
		RejectReason: "包含辱骂内容",
		CreatedAt:    time.Now().Add(time.Minute),
	})

	logic := NewGetPublicPostsLogic(context.Background(), svcCtx)
	resp, err := logic.GetPublicPosts(&types.GetPostListReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get public posts: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected public feed success, got %#v", resp)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected exactly one published post, got %#v", resp.List)
	}
	if resp.List[0].Id != 201 {
		t.Fatalf("expected published post 201, got %#v", resp.List[0])
	}
}

func TestSocialPostReviewFollowingFeedOnlyReturnsPublished(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 103, Nickname: "读者", Avatar: "reader.png"})
	seedSocialUser(t, svcCtx, &model.User{Id: 104, Nickname: "作者C", Avatar: "c.png"})
	seedSocialFollow(t, svcCtx, 103, 104)
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        204,
		UserId:    104,
		Content:   "作者C 已发布",
		PostType:  1,
		Status:    model.SocialPostStatusPublished,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        205,
		UserId:    104,
		Content:   "作者C 待审核",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})

	logic := NewGetPostListLogic(socialUserCtx(103), svcCtx)
	resp, err := logic.GetPostList(&types.GetPostListReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get following posts: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected following feed success, got %#v", resp)
	}
	if len(resp.List) != 1 || resp.List[0].Id != 204 {
		t.Fatalf("expected only published followed post, got %#v", resp.List)
	}
}

func TestSocialPostReviewMyPostsReturnsOwnStatuses(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 105, Nickname: "作者D", Avatar: "d.png"})
	seedSocialUser(t, svcCtx, &model.User{Id: 106, Nickname: "别人", Avatar: "other.png"})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        206,
		UserId:    105,
		Content:   "我的已发布",
		PostType:  3,
		Status:    model.SocialPostStatusPublished,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        207,
		UserId:    105,
		Content:   "我的待审核",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now().Add(-time.Hour),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:           208,
		UserId:       105,
		Content:      "我的已拒绝",
		PostType:     3,
		Status:       model.SocialPostStatusRejected,
		RejectReason: "包含违法敏感话题",
		CreatedAt:    time.Now(),
	})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        209,
		UserId:    106,
		Content:   "别人的已发布",
		PostType:  3,
		Status:    model.SocialPostStatusPublished,
		CreatedAt: time.Now().Add(time.Minute),
	})

	logic := NewGetMyPostListLogic(socialUserCtx(105), svcCtx)
	resp, err := logic.GetMyPostList(&types.GetPostListReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get my posts: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected my-posts success, got %#v", resp)
	}
	if len(resp.List) != 3 {
		t.Fatalf("expected exactly my three posts, got %#v", resp.List)
	}
	if resp.List[0].Id != 208 || resp.List[1].Id != 207 || resp.List[2].Id != 206 {
		t.Fatalf("expected descending own posts only, got %#v", resp.List)
	}
	if resp.List[0].RejectReason != "包含违法敏感话题" {
		t.Fatalf("expected reject reason on my post, got %#v", resp.List[0])
	}
	if !resp.List[0].IsMine || !resp.List[1].IsMine || !resp.List[2].IsMine {
		t.Fatalf("expected all my posts marked is_mine, got %#v", resp.List)
	}
}

func TestSocialPostReviewBlocksLikeCommentAndCommentListForPendingPost(t *testing.T) {
	svcCtx := newSocialReviewTestSvc(t)
	seedSocialUser(t, svcCtx, &model.User{Id: 107, Nickname: "互动者", Avatar: "fan.png"})
	seedSocialUser(t, svcCtx, &model.User{Id: 108, Nickname: "作者E", Avatar: "e.png"})
	seedSocialPost(t, svcCtx, &model.SocialPost{
		Id:        210,
		UserId:    108,
		Content:   "审核中内容",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})

	likeLogic := NewLikePostLogic(socialUserCtx(107), svcCtx)
	likeResp, err := likeLogic.LikePost(&types.PostIdReq{PostId: 210})
	if err != nil {
		t.Fatalf("like post: %v", err)
	}
	if likeResp.Success {
		t.Fatalf("expected like pending post to fail, got %#v", likeResp)
	}

	commentLogic := NewCommentPostLogic(socialUserCtx(107), svcCtx)
	commentResp, err := commentLogic.CommentPost(&types.CommentPostReq{PostId: 210, Content: "不该成功"})
	if err != nil {
		t.Fatalf("comment post: %v", err)
	}
	if commentResp.Success {
		t.Fatalf("expected comment pending post to fail, got %#v", commentResp)
	}

	commentListLogic := NewGetPostCommentsLogic(context.Background(), svcCtx)
	commentListResp, err := commentListLogic.GetPostComments(&types.GetPostCommentsReq{PostId: 210, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if commentListResp.Success {
		t.Fatalf("expected comment list for pending post to fail, got %#v", commentListResp)
	}
}
