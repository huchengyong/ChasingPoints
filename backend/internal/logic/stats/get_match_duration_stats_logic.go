package stats

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchDurationStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 对局时长统计
func NewGetMatchDurationStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchDurationStatsLogic {
	return &GetMatchDurationStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMatchDurationStatsLogic) GetMatchDurationStats(req *types.GetMatchDurationStatsReq) (resp *types.GetMatchDurationStatsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchDurationStatsResp{Success: false}, nil
	}

	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return &types.GetMatchDurationStatsResp{Success: false}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		return l.getLegacyMatchDurationStats(userIdInt, req)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.GetMatchDurationStatsResp{Success: false}, nil
	}
	gameType := 0
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}
	stats, err := l.svcCtx.CompetitiveReadModel.FindStatsWithTx(l.svcCtx.DB.WithContext(l.ctx), userIdInt, gameType)
	if err != nil {
		return nil, err
	}
	duration := &types.DurationStats{}
	if stats != nil && stats.DurationCount > 0 {
		duration = &types.DurationStats{
			AverageSeconds: int(stats.DurationSumSeconds / int64(stats.DurationCount)),
			FastestSeconds: int(stats.DurationMinSeconds),
			LongestSeconds: int(stats.DurationMaxSeconds),
			TotalMatches:   stats.DurationCount,
		}
	}
	return &types.GetMatchDurationStatsResp{Success: true, Stats: duration}, nil
}

func matchDurationSecondsExpr(dialect string) string {
	if dialect == "sqlite" {
		return "(strftime('%s', end_time) - strftime('%s', match_time))"
	}
	return "TIMESTAMPDIFF(SECOND, match_time, end_time)"
}

func (l *GetMatchDurationStatsLogic) getLegacyMatchDurationStats(userID int64, req *types.GetMatchDurationStatsReq) (*types.GetMatchDurationStatsResp, error) {
	type row struct {
		AverageSeconds float64
		FastestSeconds int
		LongestSeconds int
		TotalMatches   int
	}
	expression := matchDurationSecondsExpr(l.svcCtx.DB.Dialector.Name())
	query := l.svcCtx.DB.WithContext(l.ctx).Table("matches").
		Select(fmt.Sprintf(`
			COALESCE(AVG(%[1]s), 0) AS average_seconds,
			COALESCE(MIN(%[1]s), 0) AS fastest_seconds,
			COALESCE(MAX(%[1]s), 0) AS longest_seconds,
			COUNT(*) AS total_matches`, expression)).
		Where("(user_id = ? OR opponent_id = ?) AND status = ? AND end_time IS NOT NULL", userID, userID, 2).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", model.MatchModeRanked)
	if req != nil && req.GameType > 0 {
		query = query.Where("game_type = ?", req.GameType)
	}
	var result row
	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}
	return &types.GetMatchDurationStatsResp{Success: true, Stats: &types.DurationStats{
		AverageSeconds: int(result.AverageSeconds),
		FastestSeconds: result.FastestSeconds,
		LongestSeconds: result.LongestSeconds,
		TotalMatches:   result.TotalMatches,
	}}, nil
}
