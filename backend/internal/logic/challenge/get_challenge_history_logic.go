package challenge

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChallengeHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 约球历史分页（含终态原因与赛果）
func NewGetChallengeHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChallengeHistoryLogic {
	return &GetChallengeHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetChallengeHistoryLogic) GetChallengeHistory(req *types.GetChallengeHistoryReq) (resp *types.GetChallengeHistoryResp, err error) {
	resp = &types.GetChallengeHistoryResp{List: []types.ChallengeInfo{}}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return resp, nil
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	} else if pageSize > 50 {
		pageSize = 50
	}
	rows, err := l.svcCtx.ChallengeModel.ListHistoryWithRows(userId, req.BeforeId, pageSize)
	if err != nil {
		l.Logger.Errorf("查询约球历史失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	now := time.Now()
	resp.List = buildChallengeInfoList(rows, now)
	if len(rows) >= pageSize {
		resp.NextBeforeId = rows[len(rows)-1].Id
	}
	resp.Success = true
	return resp, nil
}
