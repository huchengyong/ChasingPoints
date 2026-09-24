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
		svcCtx: svcCtx,
	}
}

func matchDurationSecondsExpr(dialect string) string {
	if dialect == "sqlite" {
		return "(strftime('%s', end_time) - strftime('%s', match_time))"
	}
	return "TIMESTAMPDIFF(SECOND, match_time, end_time)"
}

func (l *GetMatchDurationStatsLogic) GetMatchDurationStats(req *types.GetMatchDurationStatsReq) (resp *types.GetMatchDurationStatsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchDurationStatsResp{Success: false}, nil
	}

	type durationStatsRow struct {
		AverageSeconds float64
		FastestSeconds int
		LongestSeconds int
		TotalMatches   int
	}

	durationExpr := matchDurationSecondsExpr(l.svcCtx.DB.Dialector.Name())
	query := l.svcCtx.DB.Table("matches").
		Select(fmt.Sprintf(`
			COALESCE(AVG(%[1]s), 0) AS average_seconds,
			COALESCE(MIN(%[1]s), 0) AS fastest_seconds,
			COALESCE(MAX(%[1]s), 0) AS longest_seconds,
			COUNT(*) AS total_matches`, durationExpr)).
		Where("(user_id = ? OR opponent_id = ?) AND status = 2 AND end_time IS NOT NULL", userIdInt, userIdInt)
	query = query.Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", model.MatchModeRanked)

	if req != nil && req.GameType > 0 {
		query = query.Where("game_type = ?", req.GameType)
	}

	var row durationStatsRow
	err = query.Scan(&row).Error
	if err != nil {
		return nil, err
	}

	return &types.GetMatchDurationStatsResp{
		Success: true,
		Stats: &types.DurationStats{
			AverageSeconds: int(row.AverageSeconds),
			FastestSeconds: row.FastestSeconds,
			LongestSeconds: row.LongestSeconds,
			TotalMatches:   row.TotalMatches,
		},
	}, nil
}
