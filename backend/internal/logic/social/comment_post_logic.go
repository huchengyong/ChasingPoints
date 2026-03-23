package social

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CommentPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 评论动态
func NewCommentPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentPostLogic {
	return &CommentPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentPostLogic) CommentPost(req *types.CommentPostReq) (resp *types.CommonResp, err error) {
	// todo: add your logic here and delete this line

	return
}
