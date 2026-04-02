package eventnews

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFeaturedEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取首页焦点赛事情报
func NewGetFeaturedEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFeaturedEventNewsLogic {
	return &GetFeaturedEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFeaturedEventNewsLogic) GetFeaturedEventNews() (resp *types.GetFeaturedEventNewsResp, err error) {
	items, err := l.svcCtx.EventNewsModel.FindPublishedMatching(-1, -1, "")
	if err != nil {
		l.Logger.Errorf("获取首页焦点赛事情报失败: err=%v", err)
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}
	if len(items) == 0 {
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	item := pickBestFeaturedEventNews(items, eventNewsNow())
	if item == nil {
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	var tournament *model.Tournament
	if item.TournamentId > 0 {
		tournamentRecord, queryErr := l.svcCtx.TournamentModel.FindById(item.TournamentId)
		if queryErr != nil {
			l.Logger.Errorf("获取焦点赛事失败: eventId=%d tournamentId=%d err=%v", item.Id, item.TournamentId, queryErr)
			return &types.GetFeaturedEventNewsResp{Success: false}, nil
		}
		tournament = tournamentRecord
	}

	matches, err := l.svcCtx.TournamentMatchModel.FindByTournament(item.TournamentId)
	if err != nil && item.TournamentId > 0 {
		l.Logger.Errorf("获取焦点赛事比赛失败: eventId=%d tournamentId=%d err=%v", item.Id, item.TournamentId, err)
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	info := mapEventNewsInfo(*item, tournament, matches)
	return &types.GetFeaturedEventNewsResp{
		Success: true,
		Event:   &info,
	}, nil
}
