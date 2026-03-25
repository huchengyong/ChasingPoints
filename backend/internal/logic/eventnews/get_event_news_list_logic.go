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

	items, err := l.svcCtx.EventNewsModel.FindPublishedMatching(req.GameType, req.Status, req.City)
	if err != nil {
		l.Logger.Errorf("获取赛事情报列表失败: req=%+v err=%v", req, err)
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	sortEventNewsItems(items, eventNewsNow())
	pageItems := paginateEventNewsItems(items, req.Page, req.PageSize)
	stageMap, err := l.loadStageMap(pageItems)
	if err != nil {
		l.Logger.Errorf("获取赛事情报阶段失败: req=%+v err=%v", req, err)
		return &types.GetEventNewsListResp{Success: false, List: []types.EventNewsInfo{}}, nil
	}

	respItems := make([]types.EventNewsInfo, 0, len(pageItems))
	for _, item := range pageItems {
		respItems = append(respItems, mapEventNewsInfo(item, stageMap[item.Id]))
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

func (l *GetEventNewsListLogic) loadStageMap(items []model.EventNews) (map[int64][]model.EventNewsStage, error) {
	eventIDs := make([]int64, 0, len(items))
	for _, item := range items {
		eventIDs = append(eventIDs, item.Id)
	}
	return l.svcCtx.EventNewsStageModel.FindByEventIds(eventIDs)
}
