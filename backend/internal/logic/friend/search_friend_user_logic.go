package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchFriendUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索用户
func NewSearchFriendUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchFriendUserLogic {
	return &SearchFriendUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchFriendUserLogic) SearchFriendUser(req *types.SearchUserReq) (resp *types.SearchUserResp, err error) {
	// todo: add your logic here and delete this line

	return
}
