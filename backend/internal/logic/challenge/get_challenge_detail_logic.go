package challenge

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChallengeDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 约球详情：旧互邀 ID 解别名后读取权威主记录
func NewGetChallengeDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChallengeDetailLogic {
	return &GetChallengeDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetChallengeDetailLogic) GetChallengeDetail(req *types.GetChallengeDetailReq) (resp *types.GetChallengeDetailResp, err error) {
	resp = &types.GetChallengeDetailResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || req.Id <= 0 {
		return resp, nil
	}
	main, err := l.svcCtx.ChallengeModel.FindMainById(req.Id)
	if err != nil {
		l.Logger.Errorf("查询约球详情失败: id=%d err=%v", req.Id, err)
		return resp, nil
	}
	if main == nil {
		resp.Success = true
		return resp, nil
	}
	if main.FromUserId != userId && main.ToUserId != userId {
		resp.Success = true
		return resp, nil
	}
	now := time.Now()
	row, err := l.svcCtx.ChallengeModel.FindRowById(main.Id)
	if err != nil || row == nil {
		l.Logger.Errorf("查询约球详情失败: id=%d err=%v", main.Id, err)
		return resp, nil
	}
	matchId, err := l.svcCtx.MatchModel.FindIdByChallengeId(main.Id)
	if err != nil {
		l.Logger.Errorf("查询约球比赛失败: id=%d err=%v", main.Id, err)
		return resp, nil
	}
	resp.Success = true
	info := buildChallengeInfo(*row, matchId, now)
	resp.Challenge = &info
	return resp, nil
}
