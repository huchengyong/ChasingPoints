package gear

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGearMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局的本场用杆
func NewGetGearMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGearMatchLogic {
	return &GetGearMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGearMatchLogic) GetGearMatch(req *types.GetGearMatchReq) (resp *types.GetGearMatchResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return failedMatchResp("", "用户未登录"), nil
	}
	if req == nil {
		return failedMatchResp(reasonMatchNotFound, "对局不存在"), nil
	}

	match, refused, err := loadViewableMatch(l.svcCtx, userId, req.MatchId)
	if err != nil {
		l.Logger.Errorf("读取对局失败: matchId=%d err=%v", req.MatchId, err)
		return failedMatchResp("", "读取本场用杆失败"), nil
	}
	if refused != nil {
		return refused, nil
	}

	resp, err = buildMatchGearResp(l.svcCtx, userId, match, time.Now())
	if err != nil {
		l.Logger.Errorf("读取本场用杆失败: matchId=%d userId=%d err=%v", req.MatchId, userId, err)
		return failedMatchResp("", "读取本场用杆失败"), nil
	}
	return resp, nil
}
