package logic

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentSeasonLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前赛季
func NewGetCurrentSeasonLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentSeasonLogic {
	return &GetCurrentSeasonLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentSeasonLogic) GetCurrentSeason() (resp *types.GetCurrentSeasonResp, err error) {
	season, err := l.svcCtx.SeasonModel.FindCurrent()
	if err != nil {
		l.Logger.Errorf("查询当前赛季失败: err=%v", err)
		return &types.GetCurrentSeasonResp{Success: false}, nil
	}

	if season == nil {
		return &types.GetCurrentSeasonResp{Success: true, Season: nil}, nil
	}

	return &types.GetCurrentSeasonResp{
		Success: true,
		Season:  buildSeasonInfo(season),
	}, nil
}

func buildSeasonInfo(season *model.Season) *types.SeasonInfo {
	if season == nil {
		return nil
	}

	return &types.SeasonInfo{
		Id:             season.Id,
		Name:           season.Name,
		StartDate:      season.StartDate.Format("2006-01-02"),
		EndDate:        season.EndDate.Format("2006-01-02"),
		Status:         season.Status,
		RankResetRatio: season.RankResetRatio,
	}
}
