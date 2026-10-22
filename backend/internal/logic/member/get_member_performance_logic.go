package member

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberPerformanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	now    func() time.Time
}

// 获取会员表现分析
func NewGetMemberPerformanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberPerformanceLogic {
	return &GetMemberPerformanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
		now:    time.Now,
	}
}

func (l *GetMemberPerformanceLogic) GetMemberPerformance(req *types.GetMemberPerformanceReq) (resp *types.GetMemberPerformanceResp, err error) {
	resp = &types.GetMemberPerformanceResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return resp, nil
	}
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}
	if gameType > 4 {
		return resp, nil
	}
	if l.svcCtx == nil || l.svcCtx.UserModel == nil || l.svcCtx.RankingModel == nil {
		l.Logger.Errorf("表现分析依赖未就绪: userId=%d", userId)
		return resp, nil
	}

	// 会员资格只在服务端按请求时刻判断，不信任客户端。
	now := l.now()
	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil || user == nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	isMember := logicx.MemberActiveAt(user.MemberExpiresAt, now)

	// 非会员只需要 90 天样本数与本月预览；会员需要完整的 6 个自然月。读取行数都有上限。
	since := now.AddDate(0, 0, -performanceSummaryDays)
	if isMember {
		since = sixMonthsStart(now)
	}
	rows, err := l.svcCtx.RankingModel.ListMatchRankChangesForAnalysis(userId, gameType, since, model.MaxAnalysisRankChangeRows)
	if err != nil {
		l.Logger.Errorf("读取表现分析样本失败: userId=%d gameType=%d err=%v", userId, gameType, err)
		return resp, nil
	}
	truncated := len(rows) >= model.MaxAnalysisRankChangeRows
	oldestAt := time.Time{}
	if truncated {
		oldestAt = rows[len(rows)-1].EffectiveAt
	}

	candidates, monthEnd := parsePerformanceRows(rows)
	matchIDs := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		matchIDs = append(matchIDs, candidate.matchID)
	}
	refs, err := l.svcCtx.RankingModel.ListOpponentBeforeScores(matchIDs, userId, gameType)
	if err != nil {
		l.Logger.Errorf("读取对手赛前分失败: userId=%d gameType=%d err=%v", userId, gameType, err)
		return resp, nil
	}
	samples, dropped := attachOpponents(candidates, refs)
	if dropped > 0 {
		l.Logger.Infof("表现分析剔除缺少对手结算记录的样本: userId=%d gameType=%d dropped=%d", userId, gameType, dropped)
	}
	window := summaryWindow(samples, now)

	resp.GameType = gameType
	resp.SampleCount = len(window)
	if !isMember {
		resp.Locked = true
		resp.MonthPreview = buildMonthPreview(samples, now)
		resp.Success = true
		return resp, nil
	}

	summary, bestWin, worstLoss := summarize(window)
	resp.Summary = summary
	resp.Months = buildMonths(samples, monthEnd, now, truncated, oldestAt)
	resp.BestWin, resp.WorstLoss = l.buildNotableMatches(bestWin, worstLoss)
	resp.Success = true
	return resp, nil
}

// buildNotableMatches 补上最佳胜利与爆冷失利的对手昵称和头像。昵称读取失败只影响展示，不影响分析结果。
func (l *GetMemberPerformanceLogic) buildNotableMatches(bestWin, worstLoss *performanceSample) (*types.PerformanceMatchItem, *types.PerformanceMatchItem) {
	opponentIDs := make([]int64, 0, 2)
	for _, sample := range []*performanceSample{bestWin, worstLoss} {
		if sample != nil {
			opponentIDs = append(opponentIDs, sample.opponentUserID)
		}
	}
	var users map[int64]model.User
	if len(opponentIDs) > 0 {
		var err error
		if users, err = l.svcCtx.UserModel.FindByIds(opponentIDs); err != nil {
			l.Logger.Errorf("读取表现分析对手信息失败: err=%v", err)
		}
	}
	build := func(sample *performanceSample) *types.PerformanceMatchItem {
		if sample == nil {
			return nil
		}
		item := &types.PerformanceMatchItem{
			MatchId:             sample.matchID,
			OpponentName:        "对手",
			OpponentBeforeScore: sample.opponentRating,
			ExpectedWinRate:     round(sample.expected*100, 1),
			PlayedAt:            logicx.FormatUTC8Time(sample.playedAt),
		}
		if user, ok := users[sample.opponentUserID]; ok {
			item.OpponentName, item.OpponentAvatar = user.Nickname, user.Avatar
		}
		return item
	}
	return build(bestWin), build(worstLoss)
}
