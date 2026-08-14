package season

import (
	"context"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetCurrentSeasonLogic) GetCurrentSeason() (resp *types.GetCurrentSeasonResp, err error) {
	result, err := cachedCurrentSeasonResponse(l.ctx, l.svcCtx, time.Now())
	if err != nil {
		l.Logger.Errorf("解析当前赛季失败: err=%v", err)
		return &types.GetCurrentSeasonResp{Success: true, SeasonState: "unavailable", Season: nil}, nil
	}
	return &result, nil
}

func cachedCurrentSeasonResponse(ctx context.Context, svcCtx *svc.ServiceContext, now time.Time) (types.GetCurrentSeasonResp, error) {
	return loadCurrentSeasonResponse(ctx, svcCtx, now, func() (types.GetCurrentSeasonResp, error) {
		season, state, err := ResolveCurrentSeasonLifecycle(svcCtx, now)
		if err != nil {
			return types.GetCurrentSeasonResp{}, err
		}
		if season == nil {
			return types.GetCurrentSeasonResp{Success: true, SeasonState: state, Season: nil}, nil
		}
		return types.GetCurrentSeasonResp{Success: true, SeasonState: state, Season: buildSeasonInfo(svcCtx, season)}, nil
	})
}

func buildSeasonInfo(svcCtx *svc.ServiceContext, season *model.Season) *types.SeasonInfo {
	if season == nil {
		return nil
	}

	info := &types.SeasonInfo{
		Id:             season.Id,
		Name:           season.Name,
		StartDate:      season.StartDate.Format(time.DateOnly),
		EndDate:        season.EndDate.Format(time.DateOnly),
		Status:         season.Status,
		RankResetRatio: season.RankResetRatio,
	}
	if svcCtx == nil {
		return info
	}
	startAt, endExclusive, err := seasonx.BoundsForConfig(svcCtx.Config.SeasonLifecycle, season)
	if err != nil {
		return info
	}
	info.StartDate = startAt.Format(time.DateOnly)
	info.EndDate = endExclusive.AddDate(0, 0, -1).Format(time.DateOnly)
	info.StartAt = startAt.Format(time.RFC3339)
	info.EndAtExclusive = endExclusive.Format(time.RFC3339)
	return info
}
