package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetEventNewsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报列表（管理员）
func NewAdminGetEventNewsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetEventNewsListLogic {
	return &AdminGetEventNewsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetEventNewsListLogic) AdminGetEventNewsList(req *types.AdminEventNewsListReq) (resp *types.AdminEventNewsListResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminEventNewsListResp{
			Code:    401,
			Success: false,
			Message: "未登录或登录已过期",
			List:    []types.EventNewsInfo{},
		}, nil
	}

	if req == nil {
		return &types.AdminEventNewsListResp{
			Code:    400,
			Success: false,
			Message: "请求参数错误",
			List:    []types.EventNewsInfo{},
		}, nil
	}

	publishedFilter := req.Published
	if publishedFilter != -1 && publishedFilter != 0 && publishedFilter != 1 {
		publishedFilter = -1
	}

	list, total, err := l.svcCtx.EventNewsModel.FindList(req.Page, req.PageSize, req.GameType, req.Status, "", publishedFilter)
	if err != nil {
		l.Logger.Errorf("获取赛事情报列表失败: err=%v", err)
		return &types.AdminEventNewsListResp{
			Code:    500,
			Success: false,
			Message: "获取赛事情报列表失败",
			List:    []types.EventNewsInfo{},
		}, nil
	}

	tournamentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		if item.TournamentId > 0 {
			tournamentIDs = append(tournamentIDs, item.TournamentId)
		}
	}
	tournamentRecords, err := l.svcCtx.TournamentModel.FindByIds(tournamentIDs)
	if err != nil {
		l.Logger.Errorf("获取赛事情报关联赛事失败: err=%v", err)
		return &types.AdminEventNewsListResp{
			Code:    500,
			Success: false,
			Message: "获取赛事情报列表失败",
			List:    []types.EventNewsInfo{},
		}, nil
	}

	matchMap, err := l.svcCtx.TournamentMatchModel.FindByTournamentIds(tournamentIDs)
	if err != nil {
		l.Logger.Errorf("获取赛事情报比赛失败: err=%v", err)
		return &types.AdminEventNewsListResp{
			Code:    500,
			Success: false,
			Message: "获取赛事情报列表失败",
			List:    []types.EventNewsInfo{},
		}, nil
	}

	tournamentMap := make(map[int64]*model.Tournament, len(tournamentRecords))
	for id, item := range tournamentRecords {
		tournament := item
		tournamentMap[id] = &tournament
	}

	items := make([]types.EventNewsInfo, 0, len(list))
	for _, item := range list {
		items = append(items, buildAdminEventNewsInfo(item, tournamentMap[item.TournamentId], matchMap[item.TournamentId]))
	}
	if items == nil {
		items = []types.EventNewsInfo{}
	}

	return &types.AdminEventNewsListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}
