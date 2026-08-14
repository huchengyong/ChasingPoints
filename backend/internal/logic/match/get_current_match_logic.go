package match

import (
	"context"
	"time"

	"chasing_points/internal/requestctx"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取进行中对局
func NewGetCurrentMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentMatchLogic {
	return &GetCurrentMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetCurrentMatchLogic) GetCurrentMatch() (resp *types.GetCurrentMatchResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetCurrentMatchResp{Success: false}, nil
	}

	// 查询进行中的对局
	match, err := l.svcCtx.MatchModel.FindCurrentByUserId(userId)
	if err != nil {
		l.Logger.Errorf("查询进行中对局失败: %v", err)
		return &types.GetCurrentMatchResp{Success: false}, nil
	}

	if match == nil {
		return &types.GetCurrentMatchResp{
			Success: true,
			Match:   nil,
		}, nil
	}
	match = effectiveMatchForRead(match, time.Now())

	return &types.GetCurrentMatchResp{
		Success: true,
		Match:   BuildCurrentMatchInfoWithKnownUser(l.svcCtx, userId, match, requestctx.ActiveUser(l.ctx)),
	}, nil
}
