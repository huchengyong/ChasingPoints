package stats

import (
	"context"
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentStrengthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 强弱对手分析
func NewGetOpponentStrengthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentStrengthLogic {
	return &GetOpponentStrengthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func opponentStrengthRange(bucket string) string {
	switch bucket {
	case ranking.StrengthBucketStronger:
		return "强于我"
	case ranking.StrengthBucketEven:
		return "实力相当"
	case ranking.StrengthBucketWeaker:
		return "弱于我"
	case "score_2001_plus", "level_5", "level_6":
		return "高级(2001+)"
	case "score_1001_2000", "level_3", "level_4":
		return "中级(1001-2000)"
	default:
		return "初级(0-1000)"
	}
}

func (l *GetOpponentStrengthLogic) GetOpponentStrength(req *types.GetOpponentStrengthReq) (resp *types.GetOpponentStrengthResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetOpponentStrengthResp{Success: false}, nil
	}

	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return &types.GetOpponentStrengthResp{Success: false}, nil
	}
	rules := ranking.RulesFor(l.svcCtx.Config)
	ratingV2 := rules.V2ActiveAt(time.Now())
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if ratingV2 {
			return l.getLegacyRelativeOpponentStrength(userIdInt, req, rules.V2StartAt())
		}
		return l.getLegacyOpponentStrength(userIdInt, req)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.GetOpponentStrengthResp{Success: false}, nil
	}
	gameType := 0
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}
	// 排位规则 V2 后只展示结算时按分差记录的相对分档，旧的绝对分档保留但不再展示。
	prefix := ""
	if ratingV2 {
		prefix = "rating_"
	}
	buckets, err := l.svcCtx.CompetitiveReadModel.ListOpponentStrengthBucketsWithPrefix(userIdInt, gameType, prefix)
	if err != nil {
		return nil, err
	}
	if ratingV2 {
		sort.SliceStable(buckets, func(i, j int) bool {
			return relativeStrengthOrder[buckets[i].RankBucket] < relativeStrengthOrder[buckets[j].RankBucket]
		})
	}

	list := make([]types.OpponentStrengthItem, 0, len(buckets))
	for _, bucket := range buckets {
		list = append(list, types.OpponentStrengthItem{
			RankRange: opponentStrengthRange(bucket.RankBucket),
			Matches:   bucket.Matches,
			Wins:      bucket.Wins,
			WinRate:   calculateWinRatePercent(bucket.Wins, bucket.Matches),
		})
	}

	return &types.GetOpponentStrengthResp{
		Success: true,
		List:    list,
	}, nil
}

var relativeStrengthOrder = map[string]int{
	ranking.StrengthBucketStronger: 0,
	ranking.StrengthBucketEven:     1,
	ranking.StrengthBucketWeaker:   2,
}

// getLegacyRelativeOpponentStrength 是竞技读模型未切读时的 V2 回退：只统计 V2 起点之后的排位，按双方当前分差近似相对分档。
func (l *GetOpponentStrengthLogic) getLegacyRelativeOpponentStrength(userID int64, req *types.GetOpponentStrengthReq, since time.Time) (*types.GetOpponentStrengthResp, error) {
	type row struct {
		Bucket  string
		Matches int
		Wins    int
	}
	var rows []row
	query := l.svcCtx.DB.WithContext(l.ctx).Table("matches m").
		Select(`
			CASE
				WHEN COALESCE(opp.rank_score, ?) - COALESCE(me.rank_score, ?) > ? THEN ?
				WHEN COALESCE(opp.rank_score, ?) - COALESCE(me.rank_score, ?) < ? THEN ?
				ELSE ?
			END AS bucket,
			COUNT(*) AS matches,
			SUM(CASE WHEN (m.user_id = ? AND m.result = 1) OR (m.opponent_id = ? AND m.result = 2) THEN 1 ELSE 0 END) AS wins`,
			ranking.InitialRating, ranking.InitialRating, ranking.StrengthBucketGap, ranking.StrengthBucketStronger,
			ranking.InitialRating, ranking.InitialRating, -ranking.StrengthBucketGap, ranking.StrengthBucketWeaker,
			ranking.StrengthBucketEven,
			userID, userID).
		Joins("LEFT JOIN user_ranking opp ON opp.user_id = CASE WHEN m.user_id = ? THEN m.opponent_id ELSE m.user_id END AND opp.game_type = m.game_type", userID).
		Joins("LEFT JOIN user_ranking me ON me.user_id = ? AND me.game_type = m.game_type", userID).
		Where("(m.user_id = ? OR m.opponent_id = ?) AND m.status = ? AND m.result IN ?", userID, userID, 2, []int{1, 2}).
		Where("m.match_mode = ? OR m.match_mode = '' OR m.match_mode IS NULL", model.MatchModeRanked).
		Where("m.end_time >= ?", since)
	if req != nil && req.GameType > 0 {
		query = query.Where("m.game_type = ?", req.GameType)
	}
	if err := query.Group("bucket").Scan(&rows).Error; err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return relativeStrengthOrder[rows[i].Bucket] < relativeStrengthOrder[rows[j].Bucket]
	})
	list := make([]types.OpponentStrengthItem, 0, len(rows))
	for _, item := range rows {
		list = append(list, types.OpponentStrengthItem{
			RankRange: opponentStrengthRange(item.Bucket),
			Matches:   item.Matches,
			Wins:      item.Wins,
			WinRate:   calculateWinRatePercent(item.Wins, item.Matches),
		})
	}
	return &types.GetOpponentStrengthResp{Success: true, List: list}, nil
}

func (l *GetOpponentStrengthLogic) getLegacyOpponentStrength(userID int64, req *types.GetOpponentStrengthReq) (*types.GetOpponentStrengthResp, error) {
	type row struct {
		RankRange string
		Matches   int
		Wins      int
	}
	var rows []row
	query := l.svcCtx.DB.WithContext(l.ctx).Table("matches m").
		Select(`
			CASE
				WHEN ur.rank_score IS NULL OR ur.rank_score <= 1000 THEN '初级(0-1000)'
				WHEN ur.rank_score <= 2000 THEN '中级(1001-2000)'
				ELSE '高级(2001+)'
			END AS rank_range,
			COUNT(*) AS matches,
			SUM(CASE WHEN (m.user_id = ? AND m.result = 1) OR (m.opponent_id = ? AND m.result = 2) THEN 1 ELSE 0 END) AS wins`, userID, userID).
		Joins("LEFT JOIN user_ranking ur ON ur.user_id = CASE WHEN m.user_id = ? THEN m.opponent_id ELSE m.user_id END AND ur.game_type = m.game_type", userID).
		Where("(m.user_id = ? OR m.opponent_id = ?) AND m.status = ?", userID, userID, 2).
		Where("m.match_mode = ? OR m.match_mode = '' OR m.match_mode IS NULL", model.MatchModeRanked)
	if req != nil && req.GameType > 0 {
		query = query.Where("m.game_type = ?", req.GameType)
	}
	if err := query.Group("rank_range").Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]types.OpponentStrengthItem, 0, len(rows))
	for _, item := range rows {
		list = append(list, types.OpponentStrengthItem{
			RankRange: item.RankRange,
			Matches:   item.Matches,
			Wins:      item.Wins,
			WinRate:   calculateWinRatePercent(item.Wins, item.Matches),
		})
	}
	return &types.GetOpponentStrengthResp{Success: true, List: list}, nil
}
