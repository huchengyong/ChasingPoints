package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetSocialPostListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取动态列表（管理员）
func NewAdminGetSocialPostListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetSocialPostListLogic {
	return &AdminGetSocialPostListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetSocialPostListLogic) AdminGetSocialPostList(req *types.AdminSocialPostListReq) (resp *types.AdminSocialPostListResp, err error) {
	list, total, err := l.svcCtx.SocialPostModel.FindListForAdmin(req.Page, req.PageSize, req.Status)
	if err != nil {
		l.Logger.Errorf("获取动态审核列表失败: err=%v", err)
		return &types.AdminSocialPostListResp{
			Code:    500,
			Success: false,
			Message: "获取动态列表失败",
		}, nil
	}

	items := make([]types.AdminSocialPostInfo, 0, len(list))
	for _, post := range list {
		item, buildErr := buildAdminSocialPostInfo(l.svcCtx, post)
		if buildErr != nil {
			l.Logger.Errorf("构建动态审核项失败: postId=%d err=%v", post.Id, buildErr)
			continue
		}
		items = append(items, item)
	}

	return &types.AdminSocialPostListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
