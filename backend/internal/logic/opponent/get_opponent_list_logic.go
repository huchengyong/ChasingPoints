package opponent

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对手列表
func NewGetOpponentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentListLogic {
	return &GetOpponentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetOpponentListLogic) GetOpponentList(req *types.GetOpponentListReq) (resp *types.GetOpponentListResp, err error) {
	if req == nil {
		req = &types.GetOpponentListReq{}
	}
	// 获取用户ID
	viewerUserId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetOpponentListResp{
			Success: false,
			List:    []types.OpponentRecordItem{},
		}, nil
	}

	subjectUserId := viewerUserId
	if req.TargetUserId > 0 && req.TargetUserId != viewerUserId {
		areFriends, friendErr := l.svcCtx.FriendModel.AreFriends(viewerUserId, req.TargetUserId)
		if friendErr != nil {
			l.Logger.Errorf("检查好友关系失败: %v", friendErr)
			return &types.GetOpponentListResp{
				Success: false,
				Message: "获取对方战绩失败",
				List:    []types.OpponentRecordItem{},
			}, nil
		}
		if !areFriends {
			return &types.GetOpponentListResp{
				Success: false,
				Message: "仅可查看好友的对方战绩",
				List:    []types.OpponentRecordItem{},
			}, nil
		}

		subjectUserId = req.TargetUserId
	}

	if subjectUserId != viewerUserId {
		targetUser, findErr := l.svcCtx.UserModel.FindById(subjectUserId)
		if findErr != nil {
			l.Logger.Errorf("查询目标用户失败: %v", findErr)
			return &types.GetOpponentListResp{
				Success: false,
				Message: "获取对方战绩失败",
				List:    []types.OpponentRecordItem{},
			}, nil
		}
		if targetUser == nil {
			return &types.GetOpponentListResp{
				Success: false,
				Message: "仅可查看好友的对方战绩",
				List:    []types.OpponentRecordItem{},
			}, nil
		}
		if targetUser.HideMatchRecord {
			return &types.GetOpponentListResp{
				Success: true,
				Message: "对方已隐藏战绩",
				Hidden:  true,
				List:    []types.OpponentRecordItem{},
			}, nil
		}
	}

	// 分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if l.svcCtx == nil {
		return &types.GetOpponentListResp{Success: false, Message: "竞技读模型不可用", List: []types.OpponentRecordItem{}}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if l.svcCtx.MatchModel == nil {
			return &types.GetOpponentListResp{Success: false, Message: "竞技读模型不可用", List: []types.OpponentRecordItem{}}, nil
		}
		return l.getLegacyOpponentList(subjectUserId, req, offset, pageSize)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.GetOpponentListResp{Success: false, Message: "竞技读模型不可用", List: []types.OpponentRecordItem{}}, nil
	}
	totalOpponents, totalMatches, totalWins, err := l.svcCtx.CompetitiveReadModel.GetOpponentSummary(subjectUserId)
	if err != nil {
		l.Logger.Errorf("获取整体对手统计失败: %v", err)
		return &types.GetOpponentListResp{Success: false, Message: "获取对方战绩失败", List: []types.OpponentRecordItem{}}, nil
	}
	stats, total, err := l.svcCtx.CompetitiveReadModel.ListOpponentStatsPage(subjectUserId, req.Keyword, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("获取对手列表失败: %v", err)
		return &types.GetOpponentListResp{Success: false, Message: "获取对方战绩失败", List: []types.OpponentRecordItem{}}, nil
	}

	list := make([]types.OpponentRecordItem, 0, len(stats))
	for _, stat := range stats {
		winRate := 0.0
		if stat.TotalMatches > 0 {
			winRate = float64(int(float64(stat.Wins)/float64(stat.TotalMatches)*1000)) / 10
		}
		lastMatchAt := ""
		if stat.LastMatchAt != nil {
			lastMatchAt = stat.LastMatchAt.Format("2006-01-02T15:04:05+08:00")
		}
		list = append(list, types.OpponentRecordItem{
			Id:           stat.OpponentUserId,
			Name:         stat.OpponentName,
			Avatar:       stat.OpponentAvatar,
			TotalMatches: stat.TotalMatches,
			Wins:         stat.Wins,
			Losses:       stat.Losses,
			WinRate:      winRate,
			LastMatchAt:  lastMatchAt,
		})
	}

	return &types.GetOpponentListResp{
		Success:        true,
		Hidden:         false,
		TotalOpponents: int(totalOpponents),
		TotalMatches:   totalMatches,
		TotalWins:      totalWins,
		Total:          total,
		List:           list,
	}, nil
}

func (l *GetOpponentListLogic) getLegacyOpponentList(subjectUserID int64, req *types.GetOpponentListReq, offset, pageSize int) (*types.GetOpponentListResp, error) {
	totalOpponents, totalWins, err := l.svcCtx.MatchModel.GetOverallOpponentStats(subjectUserID)
	if err != nil {
		l.Logger.Errorf("获取历史整体对手统计失败: %v", err)
		return &types.GetOpponentListResp{Success: false, Message: "获取对方战绩失败", List: []types.OpponentRecordItem{}}, nil
	}
	stats, total, err := l.svcCtx.MatchModel.ListOpponentsWithStats(subjectUserID, req.Keyword, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("获取历史对手列表失败: %v", err)
		return &types.GetOpponentListResp{Success: false, Message: "获取对方战绩失败", List: []types.OpponentRecordItem{}}, nil
	}
	list := make([]types.OpponentRecordItem, 0, len(stats))
	for _, stat := range stats {
		winRate := 0.0
		if stat.TotalMatches > 0 {
			winRate = float64(int(float64(stat.Wins)/float64(stat.TotalMatches)*1000)) / 10
		}
		list = append(list, types.OpponentRecordItem{
			Id:           stat.OpponentId,
			Name:         stat.OpponentName,
			Avatar:       stat.Avatar,
			TotalMatches: stat.TotalMatches,
			Wins:         stat.Wins,
			Losses:       stat.Losses,
			WinRate:      winRate,
			LastMatchAt:  stat.LastMatchAt.Format("2006-01-02T15:04:05+08:00"),
		})
	}
	return &types.GetOpponentListResp{
		Success:        true,
		TotalOpponents: totalOpponents,
		TotalWins:      totalWins,
		Total:          total,
		List:           list,
	}, nil
}
