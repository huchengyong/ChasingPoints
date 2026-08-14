package admin

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminDeleteEventNewsMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除赛事比赛
func NewAdminDeleteEventNewsMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminDeleteEventNewsMatchLogic {
	return &AdminDeleteEventNewsMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminDeleteEventNewsMatchLogic) AdminDeleteEventNewsMatch(req *types.AdminEventNewsMatchIdReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.MatchId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}

	match, err := l.svcCtx.TournamentMatchModel.FindById(req.MatchId)
	if err != nil {
		l.Logger.Errorf("查询赛事比赛失败: matchId=%d err=%v", req.MatchId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事比赛失败"}, nil
	}
	if match == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "比赛不存在"}, nil
	}

	if err := l.svcCtx.TournamentMatchModel.DeleteById(req.MatchId); err != nil {
		l.Logger.Errorf("删除赛事比赛失败: matchId=%d err=%v", req.MatchId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "删除赛事比赛失败"}, nil
	}
	return &types.AdminWriteResp{Code: 0, Success: true, Message: "删除成功"}, nil
}
