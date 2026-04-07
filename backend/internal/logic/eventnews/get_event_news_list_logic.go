package eventnews

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventNewsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报列表
func NewGetEventNewsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventNewsListLogic {
	return &GetEventNewsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventNewsListLogic) GetEventNewsList(req *types.GetEventNewsListReq) (resp *types.GetEventNewsListResp, err error) {
	if req == nil {
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	dateWindow, err := buildEventNewsDateWindow(req, eventNewsNow())
	if err != nil {
		l.Logger.Errorf("获取赛事情报列表参数无效: req=%+v err=%v", req, err)
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	items, err := l.svcCtx.EventNewsModel.FindPublishedMatching(req.GameType, req.Status, req.City)
	if err != nil {
		l.Logger.Errorf("获取赛事情报列表失败: req=%+v err=%v", req, err)
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	tournamentMap, matchMap, err := l.loadTournamentData(items)
	if err != nil {
		l.Logger.Errorf("获取赛事情报赛事数据失败: req=%+v err=%v", req, err)
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	items = filterEventNewsItemsByDateWindow(items, tournamentMap, dateWindow)
	sortEventNewsItems(items, eventNewsNow())
	pageItems := paginateEventNewsItems(items, req.Page, req.PageSize)

	respItems := make([]types.EventNewsInfo, 0, len(pageItems))
	for _, item := range pageItems {
		tournament := tournamentMap[item.TournamentId]
		respItems = append(respItems, mapEventNewsInfo(item, tournament, matchMap[item.TournamentId]))
	}
	if respItems == nil {
		respItems = []types.EventNewsInfo{}
	}

	return &types.GetEventNewsListResp{
		Success: true,
		Total:   int64(len(items)),
		List:    respItems,
	}, nil
}

func (l *GetEventNewsListLogic) loadTournamentData(items []model.EventNews) (map[int64]*model.Tournament, map[int64][]model.TournamentMatch, error) {
	tournamentIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item.TournamentId > 0 {
			tournamentIDs = append(tournamentIDs, item.TournamentId)
		}
	}

	tournamentRecords, err := l.svcCtx.TournamentModel.FindByIds(tournamentIDs)
	if err != nil {
		return nil, nil, err
	}

	tournamentMap := make(map[int64]*model.Tournament, len(tournamentRecords))
	for id, item := range tournamentRecords {
		tournament := item
		tournamentMap[id] = &tournament
	}

	matchMap, err := l.svcCtx.TournamentMatchModel.FindByTournamentIds(tournamentIDs)
	if err != nil {
		return nil, nil, err
	}
	return tournamentMap, matchMap, nil
}
