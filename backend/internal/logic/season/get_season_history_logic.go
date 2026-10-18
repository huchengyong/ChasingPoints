package season

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultSeasonHistoryPageSize = 10
	maxSeasonHistoryPageSize     = 20
)

type GetSeasonHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取历届已结算赛季
func NewGetSeasonHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonHistoryLogic {
	return &GetSeasonHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetSeasonHistoryLogic) GetSeasonHistory(req *types.GetSeasonHistoryReq) (resp *types.GetSeasonHistoryResp, err error) {
	result := &types.GetSeasonHistoryResp{List: []types.SeasonHistoryItem{}}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return result, nil
	}
	if l.svcCtx == nil || l.svcCtx.SeasonModel == nil || l.svcCtx.SeasonRecordModel == nil {
		return result, nil
	}
	if req == nil {
		req = &types.GetSeasonHistoryReq{}
	}
	gameType := 3
	if req.GameType > 0 {
		gameType = req.GameType
	}
	page, pageSize := normalizeSeasonHistoryPage(req.Page, req.PageSize)

	total, err := l.svcCtx.SeasonModel.CountSettled()
	if err != nil {
		l.Logger.Errorf("统计历届赛季失败: err=%v", err)
		return result, nil
	}
	seasons, err := l.svcCtx.SeasonModel.ListSettledPage(page, pageSize)
	if err != nil {
		l.Logger.Errorf("查询历届赛季失败: page=%d err=%v", page, err)
		return result, nil
	}
	seasonIDs := make([]int64, 0, len(seasons))
	for _, item := range seasons {
		seasonIDs = append(seasonIDs, item.Id)
	}
	records, err := l.svcCtx.SeasonRecordModel.FindByUserAndGameTypeInSeasons(userID, gameType, seasonIDs)
	if err != nil {
		l.Logger.Errorf("查询历届赛季记录失败: userId=%d gameType=%d err=%v", userID, gameType, err)
		return result, nil
	}

	for index := range seasons {
		season := &seasons[index]
		item := types.SeasonHistoryItem{Season: buildSeasonInfo(l.svcCtx, season)}
		if record, ok := records[season.Id]; ok {
			item.Record = buildSeasonRecordInfo(&record, season.Name)
		}
		result.List = append(result.List, item)
	}
	result.Success = true
	result.Total = total
	return result, nil
}

func normalizeSeasonHistoryPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultSeasonHistoryPageSize
	}
	if pageSize > maxSeasonHistoryPageSize {
		pageSize = maxSeasonHistoryPageSize
	}
	return page, pageSize
}
