package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateNicknameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新昵称
func NewUpdateNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateNicknameLogic {
	return &UpdateNicknameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateNicknameLogic) UpdateNickname(req *types.UpdateNicknameReq) (resp *types.UpdateNicknameResp, err error) {
	// todo: add your logic here and delete this line

	return
}
