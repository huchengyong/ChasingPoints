package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消对局
func NewCancelMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelMatchLogic {
	return &CancelMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CancelMatchLogic) CancelMatch(req *types.CancelMatchReq) (resp *types.CommonResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CommonResp{Success: false, Message: "用户未登录"}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		return &types.CommonResp{Success: false, Message: "对局不存在"}, nil
	}

	// 验证用户权限
	if match.UserId != userId {
		return &types.CommonResp{Success: false, Message: "无权操作"}, nil
	}

	// 只能取消进行中的对局
	if match.Status != 1 {
		return &types.CommonResp{Success: false, Message: "对局已结束"}, nil
	}

	// 更新状态为已取消，记录结束时间
	now := time.Now()
	match.Status = 3
	match.EndTime = &now
	match.CompletedByUserId = nil
	match.CompletionSource = "unknown"

	if err := l.svcCtx.MatchModel.Update(match); err != nil {
		l.Logger.Errorf("取消对局失败: %v", err)
		return &types.CommonResp{Success: false, Message: "取消失败"}, nil
	}

	l.Logger.Infof("用户 %d 取消对局 %d", userId, match.Id)

	return &types.CommonResp{
		Success: true,
		Message: "对局已取消",
	}, nil
}
