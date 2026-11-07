package logic

import (
	"sort"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const (
	// MaxReplayActions 是单场复盘最多处理的动作数；超过时只返回局级数据，防止异常数据拖慢接口。
	MaxReplayActions = 5000
	// MaxReplayTrendPoints 是每局得分走势最多返回的点数。
	MaxReplayTrendPoints = 60

	replayKeyTagDecider  = "decider"
	replayKeyTagComeback = "comeback"
)

// MatchReplayInput 是复盘构建的输入，全部来自同一场已完成对局。
type MatchReplayInput struct {
	Match       *model.Match
	Rounds      []model.MatchRound  // 已完成的局，按局号升序
	Actions     []model.MatchAction // 有效（未撤销）动作，按局号、id 升序；Truncated 时可为空
	ViewerActor int                 // 查看者在本场的位置：1=创建者 2=对手
	Truncated   bool                // 动作数超过 MaxReplayActions，只返回局级数据
}

// BuildMatchReplay 按球种从局记录与动作日志生成单场复盘，统一为「我 / 对手」视角。
// 只使用该场数据，宁缺勿错：某一局的动作与局记录对不上时，只丢掉这一局的明细，局级数据仍保留。
func BuildMatchReplay(in MatchReplayInput) *types.GetMemberMatchReplayResp {
	resp := &types.GetMemberMatchReplayResp{Frames: []types.ReplayFrame{}}
	if in.Match == nil {
		return resp
	}
	resp.MatchId, resp.GameType, resp.FrameCount = in.Match.Id, in.Match.GameType, len(in.Rounds)
	resp.DetailTruncated = in.Truncated

	me := in.ViewerActor
	if me != 2 {
		me = 1
	}
	opp := opponentActor(me)
	mode := replayModeOf(in)
	actionsByRound := groupReplayActions(in.Actions)
	targetWins := replayTargetWins(in.Match)

	var wins [3]int       // 按位置累计已赢局数，用于判断决胜局
	var cumulative [3]int // 非斯诺克的局记录保存累计比分，单局得分由相邻两局相减得到
	var biggestComeback int
	comebackIndex := -1
	detailFrames := 0
	var longestStreak [3]int
	var streak [3]int
	var winTypeCounts = map[string]*types.ReplayWinTypeStat{}
	var winTypeOrder []string
	var maxRun [3]int

	for index, round := range in.Rounds {
		p1, p2 := round.MyScore, round.OpponentScore
		if in.Match.GameType != 1 {
			p1, p2 = round.MyScore-cumulative[1], round.OpponentScore-cumulative[2]
			cumulative[1], cumulative[2] = round.MyScore, round.OpponentScore
		}
		framePoints := [3]int{0, p1, p2}

		frame := types.ReplayFrame{
			FrameNo:       round.RoundNo,
			MyScore:       framePoints[me],
			OpponentScore: framePoints[opp],
			WinType:       round.WinType,
		}
		winner := 0
		if round.Winner != nil {
			winner = *round.Winner
		}
		switch winner {
		case me:
			frame.Winner = 1
		case opp:
			frame.Winner = 2
		}
		frame.DurationSeconds = replayDurationSeconds(actionsByRound[round.RoundNo])

		if targetWins >= 2 && wins[1] >= targetWins-1 && wins[2] >= targetWins-1 {
			frame.KeyTags = append(frame.KeyTags, replayKeyTagDecider)
		}
		if winner == 1 || winner == 2 {
			wins[winner]++
		}

		if mode.detailed() && !in.Truncated {
			if acc, ok := mode.accumulate(actionsByRound[round.RoundNo], me, framePoints); ok {
				detailFrames++
				frame.Trend = downsampleReplayTrend(acc.trend, MaxReplayTrendPoints)
				frame.MaxDeficit = acc.maxDeficit(frame.Winner)
				frame.MyMaxRun, frame.OpponentMaxRun = acc.bestRun[me], acc.bestRun[opp]
				frame.MyFoulCount, frame.OpponentFoulCount = acc.foulCount[me], acc.foulCount[opp]
				frame.MyFoulPoints, frame.OpponentFoulPoints = acc.foulPoints[me], acc.foulPoints[opp]
				maxRun[1], maxRun[2] = max(maxRun[1], acc.bestRun[me]), max(maxRun[2], acc.bestRun[opp])
				if frame.MaxDeficit > biggestComeback {
					biggestComeback, comebackIndex = frame.MaxDeficit, index
				}
			}
		}

		if model.IsPoolMatchFormatGameType(in.Match.GameType) {
			for side := 1; side <= 2; side++ {
				if frame.Winner == side {
					streak[side]++
					longestStreak[side] = max(longestStreak[side], streak[side])
					streak[3-side] = 0
				}
			}
			if frame.Winner != 0 && round.WinType != "" {
				stat, ok := winTypeCounts[round.WinType]
				if !ok {
					stat = &types.ReplayWinTypeStat{WinType: round.WinType, Label: replayWinTypeLabel(round.WinType)}
					winTypeCounts[round.WinType] = stat
					winTypeOrder = append(winTypeOrder, round.WinType)
				}
				if frame.Winner == 1 {
					stat.MyCount++
				} else {
					stat.OpponentCount++
				}
			}
		}
		resp.Frames = append(resp.Frames, frame)
	}

	if comebackIndex >= 0 {
		resp.Frames[comebackIndex].KeyTags = append(resp.Frames[comebackIndex].KeyTags, replayKeyTagComeback)
	}
	resp.FrameOnly = (in.Match.GameType == 1 || in.Match.GameType == 2) && detailFrames == 0

	summary := &types.ReplaySummary{MyMaxRun: maxRun[1], OpponentMaxRun: maxRun[2]}
	for _, frame := range resp.Frames {
		switch frame.Winner {
		case 1:
			summary.MyFrameWins++
		case 2:
			summary.OpponentFrameWins++
		}
	}
	if model.IsPoolMatchFormatGameType(in.Match.GameType) {
		summary.MyLongestStreak, summary.OpponentLongestStreak = longestStreak[1], longestStreak[2]
		for _, key := range winTypeOrder {
			summary.WinTypeStats = append(summary.WinTypeStats, *winTypeCounts[key])
		}
	}
	resp.Summary = summary
	return resp
}

// replayMode 区分各球种的明细口径。
type replayMode int

const (
	replayModeFramesOnly replayMode = iota // 旧斯诺克、中八、美九：没有逐动作明细
	replayModeSnookerV2
	replayModeChasingPoints
)

func replayModeOf(in MatchReplayInput) replayMode {
	switch in.Match.GameType {
	case 1:
		for _, action := range in.Actions {
			if action.ActionType == model.MatchActionTypeSnookerStroke || action.ActionType == model.MatchActionTypeSnookerFrameAction {
				return replayModeSnookerV2
			}
		}
	case 2:
		return replayModeChasingPoints
	}
	return replayModeFramesOnly
}

func (m replayMode) detailed() bool { return m == replayModeSnookerV2 || m == replayModeChasingPoints }

// replayAccumulator 逐个动作累计一局的得分走势、连续得分、犯规。
type replayAccumulator struct {
	me         int
	points     [3]int
	diff       int   // 我方减对手
	trend      []int // 每次得分变化后的分差，第一个元素是开局的 0
	bestRun    [3]int
	runScorer  int
	runPoints  int
	foulCount  [3]int
	foulPoints [3]int
}

func newReplayAccumulator(me int) *replayAccumulator {
	return &replayAccumulator{me: me, trend: []int{0}}
}

func (a *replayAccumulator) score(scorer, points int) {
	if (scorer != 1 && scorer != 2) || points <= 0 {
		return
	}
	a.points[scorer] += points
	if scorer == a.me {
		a.diff += points
	} else {
		a.diff -= points
	}
	a.trend = append(a.trend, a.diff)
}

// consecutive 累计同一方不间断的连续得分（九球追分）；对手得分会打断。
func (a *replayAccumulator) consecutive(scorer, points int) {
	if points <= 0 {
		return
	}
	if scorer != a.runScorer {
		a.runScorer, a.runPoints = scorer, 0
	}
	a.runPoints += points
	a.bestRun[scorer] = max(a.bestRun[scorer], a.runPoints)
}

func (a *replayAccumulator) foul(offender, points int) {
	if offender != 1 && offender != 2 {
		return
	}
	a.foulCount[offender]++
	a.foulPoints[offender] += max(points, 0)
}

// maxDeficit 是胜方在本局曾落后的最大分差（从胜方视角，未落后过为 0）。winner：1=我 2=对手。
func (a *replayAccumulator) maxDeficit(winner int) int {
	deficit := 0
	for _, diff := range a.trend {
		behind := -diff
		if winner == 2 {
			behind = diff
		}
		deficit = max(deficit, behind)
	}
	if winner != 1 && winner != 2 {
		return 0
	}
	return deficit
}

// accumulate 复盘一局的动作。动作与局记录对不上、或出现无法解析的事件时返回 ok=false，调用方丢弃这一局的明细。
func (m replayMode) accumulate(actions []model.MatchAction, me int, expected [3]int) (*replayAccumulator, bool) {
	acc := newReplayAccumulator(me)
	switch m {
	case replayModeSnookerV2:
		type visitKey struct{ actor, visit int }
		visits := map[visitKey]int{}
		for _, action := range actions {
			switch action.ActionType {
			case model.MatchActionTypeSnookerStroke:
				event, err := model.DecodeSnookerEvent(action.ExtraData)
				if err != nil || event.Kind != model.SnookerEventKindStroke || event.Actor != action.Actor {
					return nil, false
				}
				switch event.Outcome {
				case model.SnookerOutcomePot:
					if action.ScoreChange <= 0 {
						return nil, false
					}
					acc.score(action.Actor, action.ScoreChange)
					visits[visitKey{action.Actor, event.VisitNo}] += action.ScoreChange
				case model.SnookerOutcomeFoul:
					if action.ScoreChange <= 0 {
						return nil, false
					}
					acc.foul(action.Actor, action.ScoreChange)
					acc.score(opponentActor(action.Actor), action.ScoreChange)
				}
			case model.MatchActionTypeSnookerFrameAction:
				// 认输、判负等不产生比分变化。
			case "score", "foul", "win":
				return nil, false // 旧版动作混入 V2 对局，数据不可信
			}
		}
		for key, total := range visits {
			acc.bestRun[key.actor] = max(acc.bestRun[key.actor], total)
		}
	case replayModeChasingPoints:
		for _, action := range actions {
			switch action.ActionType {
			case "score", "win":
				if action.ScoreChange > 0 {
					acc.score(action.Actor, action.ScoreChange)
					acc.consecutive(action.Actor, action.ScoreChange)
				}
			case "foul":
				acc.foul(action.Actor, action.ScoreChange)
				if action.ScoreChange > 0 {
					scorer := opponentActor(action.Actor)
					acc.score(scorer, action.ScoreChange)
					acc.consecutive(scorer, action.ScoreChange)
				}
			}
		}
	default:
		return nil, false
	}
	if acc.points[1] != expected[1] || acc.points[2] != expected[2] {
		return nil, false
	}
	return acc, true
}

// replayActionTypes 是会进入复盘的有效动作；撤销、开局、结束确认等流程动作全部剔除。
var replayActionTypes = map[string]bool{
	"score": true, "foul": true, "win": true,
	model.MatchActionTypeSnookerStroke: true, model.MatchActionTypeSnookerFrameAction: true,
}

func groupReplayActions(actions []model.MatchAction) map[int][]model.MatchAction {
	grouped := make(map[int][]model.MatchAction)
	for _, action := range actions {
		if action.IsUndone != 0 || !replayActionTypes[action.ActionType] {
			continue
		}
		grouped[action.RoundNo] = append(grouped[action.RoundNo], action)
	}
	return grouped
}

// replayDurationSeconds 是本局首个与最后一个有效动作的时间差；少于两个动作或缺少时间时为 0。
func replayDurationSeconds(actions []model.MatchAction) int {
	if len(actions) < 2 {
		return 0
	}
	first, last := actions[0].CreatedAt, actions[len(actions)-1].CreatedAt
	if first.IsZero() || last.IsZero() || last.Before(first) {
		return 0
	}
	return int(last.Sub(first).Seconds())
}

// replayTargetWins 返回抢 N 赛制的目标局数；自由局数、旧赛制与九球追分没有决胜局。
func replayTargetWins(match *model.Match) int {
	switch {
	case match.GameType == 1:
		if format, target, ok := model.NormalizeSnookerFormat(match.SnookerFormat, match.SnookerTargetWins, match.BestOfFrames); ok && format == model.SnookerFormatRaceTo {
			return target
		}
	case model.IsPoolMatchFormatGameType(match.GameType):
		if format, target, ok := model.NormalizePoolMatchFormat(match.GameType, match.MatchFormat, match.TargetWins); ok && format == model.MatchFormatRaceTo {
			return target
		}
	}
	return 0
}

func replayWinTypeLabel(winType string) string {
	switch winType {
	case "normal":
		return "普胜"
	case "break_clear":
		return "炸清"
	case "continue_clear":
		return "接清"
	case "small_gold":
		return "小金"
	case "big_gold":
		return "大金"
	default:
		return winType
	}
}

// downsampleReplayTrend 把分差序列降采样到最多 limit 个点：保留首点、末点与分差的最大、最小点，其余按等间隔补足。
// Step 保留在原序列中的位置，所以降采样后 Step 不再连续。
func downsampleReplayTrend(diffs []int, limit int) []types.ReplayTrendPoint {
	points := func(indexes []int) []types.ReplayTrendPoint {
		result := make([]types.ReplayTrendPoint, 0, len(indexes))
		for _, index := range indexes {
			result = append(result, types.ReplayTrendPoint{Step: index, Diff: diffs[index]})
		}
		return result
	}
	n := len(diffs)
	if n == 0 {
		return []types.ReplayTrendPoint{}
	}
	if limit < 4 || n <= limit {
		all := make([]int, n)
		for i := range all {
			all[i] = i
		}
		return points(all)
	}

	chosen := make(map[int]bool, limit)
	for i := 0; i < limit; i++ {
		chosen[i*(n-1)/(limit-1)] = true
	}
	maxIndex, minIndex := 0, 0
	for i, diff := range diffs {
		if diff > diffs[maxIndex] {
			maxIndex = i
		}
		if diff < diffs[minIndex] {
			minIndex = i
		}
	}
	// 极值点没被选中时，替换离它最近的、非端点的已选点，保证总数不变。
	for _, extreme := range []int{maxIndex, minIndex} {
		if chosen[extreme] {
			continue
		}
		nearest, nearestGap := -1, n
		for index := range chosen {
			if index == 0 || index == n-1 || index == maxIndex || index == minIndex {
				continue
			}
			if gap := absInt(index - extreme); gap < nearestGap || (gap == nearestGap && index < nearest) {
				nearest, nearestGap = index, gap
			}
		}
		if nearest >= 0 {
			delete(chosen, nearest)
		}
		chosen[extreme] = true
	}
	indexes := make([]int, 0, len(chosen))
	for index := range chosen {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return points(indexes)
}
