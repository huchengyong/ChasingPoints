package eventnews

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventNewsViewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报详情
func NewGetEventNewsViewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventNewsViewLogic {
	return &GetEventNewsViewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventNewsViewLogic) GetEventNewsView(req *types.GetEventNewsViewReq) (resp *types.GetEventNewsViewResp, err error) {
	if req == nil || req.EventId <= 0 {
		return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
	}
	if cached, ok := loadCachedEventNewsViewResp(l.ctx, l.svcCtx, req.EventId); ok {
		return cached, nil
	}

	item, err := l.svcCtx.EventNewsModel.FindPublishedById(req.EventId)
	if err != nil {
		l.Logger.Errorf("获取赛讯视图失败: eventId=%d err=%v", req.EventId, err)
		return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
	}
	if item == nil {
		return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
	}

	var tournament *model.Tournament
	if item.TournamentId > 0 {
		tournament, err = l.svcCtx.TournamentModel.FindById(item.TournamentId)
		if err != nil {
			l.Logger.Errorf("获取关联赛事失败: eventId=%d tournamentId=%d err=%v", req.EventId, item.TournamentId, err)
			return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
		}
	}

	matches := []model.TournamentMatch{}
	if item.TournamentId > 0 {
		matches, err = l.svcCtx.TournamentMatchModel.FindByTournament(item.TournamentId)
		if err != nil {
			l.Logger.Errorf("获取赛事比赛失败: eventId=%d tournamentId=%d err=%v", req.EventId, item.TournamentId, err)
			return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
		}
	}

	playerMap := map[int64]model.Player{}
	if l.svcCtx.PlayerModel != nil {
		playerIDs := collectMatchPlayerIDs(matches)
		playerMap, err = l.svcCtx.PlayerModel.FindByIds(playerIDs)
		if err != nil {
			l.Logger.Errorf("获取赛事球员失败: eventId=%d tournamentId=%d err=%v", req.EventId, item.TournamentId, err)
			return &types.GetEventNewsViewResp{Success: false, Matches: []types.EventNewsMatchInfo{}}, nil
		}
	}

	matchItems := make([]types.EventNewsMatchInfo, 0, len(matches))
	for _, match := range matches {
		matchItems = append(matchItems, mapEventNewsMatchInfo(item.Id, match, playerMap))
	}
	if matchItems == nil {
		matchItems = []types.EventNewsMatchInfo{}
	}

	eventInfo := mapEventNewsInfo(*item, tournament, matches)
	resp = &types.GetEventNewsViewResp{
		Success:    true,
		EventNews:  &eventInfo,
		Tournament: mapTournamentInfo(tournament),
		Matches:    matchItems,
	}
	storeCachedEventNewsViewResp(l.ctx, l.svcCtx, req.EventId, resp)
	return resp, nil
}
