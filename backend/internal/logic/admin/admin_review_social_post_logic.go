package admin

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminReviewSocialPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核动态
func NewAdminReviewSocialPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewSocialPostLogic {
	return &AdminReviewSocialPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminReviewSocialPostLogic) AdminReviewSocialPost(req *types.AdminSocialPostReviewReq) (resp *types.AdminWriteResp, err error) {
	adminID, err := utils.GetAdminIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取管理员ID失败: %v", err)
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "获取管理员信息失败"}, nil
	}
	if req.PostId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "动态ID无效"}, nil
	}
	if req.Status != model.SocialPostStatusPublished && req.Status != model.SocialPostStatusRejected {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "无效的审核状态"}, nil
	}

	rejectReason := strings.TrimSpace(req.RejectReason)
	if req.Status == model.SocialPostStatusRejected && rejectReason == "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请填写拒绝原因"}, nil
	}

	post, err := l.svcCtx.SocialPostModel.FindById(req.PostId)
	if err != nil {
		l.Logger.Errorf("查询动态失败: postId=%d err=%v", req.PostId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "系统错误"}, nil
	}
	if post == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "动态不存在"}, nil
	}
	if post.Status != model.SocialPostStatusPending {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "当前动态不可审核"}, nil
	}

	if err := l.svcCtx.SocialPostModel.UpdateReview(req.PostId, req.Status, rejectReason, int64(adminID), time.Now()); err != nil {
		l.Logger.Errorf("审核动态失败: postId=%d status=%d err=%v", req.PostId, req.Status, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "审核失败"}, nil
	}

	message := "审核通过"
	if req.Status == model.SocialPostStatusRejected {
		message = "审核已拒绝"
	}
	return &types.AdminWriteResp{Code: 0, Success: true, Message: message}, nil
}
