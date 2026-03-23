package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MatchUndoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销操作
func NewMatchUndoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchUndoLogic {
	return &MatchUndoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MatchUndoLogic) MatchUndo(req *types.MatchUndoReq) (resp *types.MatchUndoResp, err error) {
	// todo: add your logic here and delete this line

	return
}
