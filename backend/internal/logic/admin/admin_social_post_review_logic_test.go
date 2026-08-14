package admin

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminSocialReviewTestSvc(t *testing.T) *svc.ServiceContext {
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
		SocialPostModel: model.NewSocialPostModel(db),
	}
}

func adminCtx(adminID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", -adminID)
}

func seedAdminSocialUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("create admin social user: %v", err)
	}
}

func seedAdminSocialPost(t *testing.T, svcCtx *svc.ServiceContext, post *model.SocialPost) {
	t.Helper()
	if err := svcCtx.SocialPostModel.Create(post); err != nil {
		t.Fatalf("create admin social post: %v", err)
	}
}

func TestAdminSocialPostReviewListsByStatus(t *testing.T) {
	svcCtx := newAdminSocialReviewTestSvc(t)
	seedAdminSocialUser(t, svcCtx, &model.User{Id: 201, Nickname: "作者F", Avatar: "f.png"})
	seedAdminSocialPost(t, svcCtx, &model.SocialPost{
		Id:        301,
		UserId:    201,
		Content:   "待审核动态",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})
	seedAdminSocialPost(t, svcCtx, &model.SocialPost{
		Id:        302,
		UserId:    201,
		Content:   "已发布动态",
		PostType:  1,
		Status:    model.SocialPostStatusPublished,
		CreatedAt: time.Now().Add(-time.Hour),
	})

	logic := NewAdminGetSocialPostListLogic(adminCtx(9001), svcCtx)
	resp, err := logic.AdminGetSocialPostList(&types.AdminSocialPostListReq{
		Page:     1,
		PageSize: 20,
		Status:   model.SocialPostStatusPending,
	})
	if err != nil {
		t.Fatalf("admin get social posts: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected admin list success, got %#v", resp)
	}
	if len(resp.List) != 1 || resp.List[0].Id != 301 {
		t.Fatalf("expected only pending social post, got %#v", resp.List)
	}
}

func TestAdminSocialPostReviewApproveTransitionsToPublished(t *testing.T) {
	svcCtx := newAdminSocialReviewTestSvc(t)
	seedAdminSocialUser(t, svcCtx, &model.User{Id: 202, Nickname: "作者G", Avatar: "g.png"})
	seedAdminSocialPost(t, svcCtx, &model.SocialPost{
		Id:        303,
		UserId:    202,
		Content:   "待审核要通过",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})

	logic := NewAdminReviewSocialPostLogic(adminCtx(9002), svcCtx)
	resp, err := logic.AdminReviewSocialPost(&types.AdminSocialPostReviewReq{
		PostId: 303,
		Status: model.SocialPostStatusPublished,
	})
	if err != nil {
		t.Fatalf("approve social post: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected approve success, got %#v", resp)
	}

	post, err := svcCtx.SocialPostModel.FindById(303)
	if err != nil {
		t.Fatalf("find approved post: %v", err)
	}
	if post == nil || post.Status != model.SocialPostStatusPublished {
		t.Fatalf("expected published post, got %#v", post)
	}
	if post.RejectReason != "" {
		t.Fatalf("expected cleared reject reason, got %#v", post)
	}
	if post.ReviewedAt == nil || post.ReviewedBy == nil || *post.ReviewedBy != 9002 {
		t.Fatalf("expected review stamps, got %#v", post)
	}
}

func TestAdminSocialPostReviewRejectStoresReason(t *testing.T) {
	svcCtx := newAdminSocialReviewTestSvc(t)
	seedAdminSocialUser(t, svcCtx, &model.User{Id: 203, Nickname: "作者H", Avatar: "h.png"})
	seedAdminSocialPost(t, svcCtx, &model.SocialPost{
		Id:        304,
		UserId:    203,
		Content:   "待审核要拒绝",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})

	logic := NewAdminReviewSocialPostLogic(adminCtx(9003), svcCtx)
	resp, err := logic.AdminReviewSocialPost(&types.AdminSocialPostReviewReq{
		PostId:       304,
		Status:       model.SocialPostStatusRejected,
		RejectReason: "包含辱骂内容",
	})
	if err != nil {
		t.Fatalf("reject social post: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected reject success, got %#v", resp)
	}

	post, err := svcCtx.SocialPostModel.FindById(304)
	if err != nil {
		t.Fatalf("find rejected post: %v", err)
	}
	if post == nil || post.Status != model.SocialPostStatusRejected || post.RejectReason != "包含辱骂内容" {
		t.Fatalf("expected rejected post with reason, got %#v", post)
	}
}

func TestAdminSocialPostReviewRejectRequiresReason(t *testing.T) {
	svcCtx := newAdminSocialReviewTestSvc(t)
	seedAdminSocialUser(t, svcCtx, &model.User{Id: 204, Nickname: "作者I", Avatar: "i.png"})
	seedAdminSocialPost(t, svcCtx, &model.SocialPost{
		Id:        305,
		UserId:    204,
		Content:   "待审核缺原因",
		PostType:  3,
		Status:    model.SocialPostStatusPending,
		CreatedAt: time.Now(),
	})

	logic := NewAdminReviewSocialPostLogic(adminCtx(9004), svcCtx)
	resp, err := logic.AdminReviewSocialPost(&types.AdminSocialPostReviewReq{
		PostId: 305,
		Status: model.SocialPostStatusRejected,
	})
	if err != nil {
		t.Fatalf("reject social post without reason: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected reject without reason to fail, got %#v", resp)
	}
}
