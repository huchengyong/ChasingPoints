package logic

import (
	"errors"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
)

// RatingReplayKey 标识一个用户在一个球种上的排位分。
type RatingReplayKey struct {
	UserId   int64
	GameType int
}

// RatingReplayPlayer 是重放过程中的排位状态。
type RatingReplayPlayer struct {
	Rating       int
	RatedMatches int
}

// RatingReplayMatchResult 是重放中一场有效排位的双方结果。
type RatingReplayMatchResult struct {
	MatchId   int64
	GameType  int
	Player1Id int64
	Player2Id int64
	Player1   ranking.Outcome
	Player2   ranking.Outcome
}

type ratingReplayPairDay struct {
	PlayerA  int64
	PlayerB  int64
	GameType int
	DayStart int64
}

type ratingReplayUserDay struct {
	UserId   int64
	GameType int
	DayStart int64
}

// RatingReplay 按完成时间顺序重放已完成比赛，口径与线上 V2 结算一致：
// 同对手计数包含当日此前全部已完成排位（含平局），每日上限只累计正向变化，期望、定级与调整都由 ranking.Settle 计算。
type RatingReplay struct {
	rules     ranking.Rules
	dailyCap  int
	players   map[RatingReplayKey]*RatingReplayPlayer
	pairDaily map[ratingReplayPairDay]int
	gainDaily map[ratingReplayUserDay]int
	Applied   int
	Skipped   int
}

// NewRatingReplay 创建重放；start 为各段位行的起点，缺失的用户从 V2 初始分开始。
func NewRatingReplay(rules ranking.Rules, dailyCap int, start map[RatingReplayKey]RatingReplayPlayer) *RatingReplay {
	players := make(map[RatingReplayKey]*RatingReplayPlayer, len(start))
	for key, value := range start {
		player := value
		players[key] = &player
	}
	return &RatingReplay{
		rules:     rules,
		dailyCap:  dailyCap,
		players:   players,
		pairDaily: map[ratingReplayPairDay]int{},
		gainDaily: map[ratingReplayUserDay]int{},
	}
}

// Apply 按完成顺序处理一场已完成比赛；只有双方均为注册用户的排位胜负局返回结果。
func (r *RatingReplay) Apply(match *model.Match, rounds []model.MatchRound) (*RatingReplayMatchResult, bool) {
	if match == nil || match.Status != 2 || match.CompletedAt == nil || match.Result == nil ||
		model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked ||
		match.OpponentId == nil || *match.OpponentId <= 0 || *match.OpponentId == match.UserId {
		r.Skipped++
		return nil, false
	}
	player1Id, player2Id := match.UserId, *match.OpponentId
	completedAt := *match.CompletedAt
	dayStart, _ := r.rules.DayRange(completedAt)
	pairKey := ratingReplayPairDay{PlayerA: min(player1Id, player2Id), PlayerB: max(player1Id, player2Id), GameType: match.GameType, DayStart: dayStart.Unix()}
	prior := r.pairDaily[pairKey]
	r.pairDaily[pairKey]++
	if *match.Result != 1 && *match.Result != 2 {
		r.Skipped++
		return nil, false
	}

	player1 := r.player(player1Id, match.GameType)
	player2 := r.player(player2Id, match.GameType)
	gain1Key := ratingReplayUserDay{UserId: player1Id, GameType: match.GameType, DayStart: dayStart.Unix()}
	gain2Key := ratingReplayUserDay{UserId: player2Id, GameType: match.GameType, DayStart: dayStart.Unix()}
	outcome1, outcome2 := ranking.Settle(ranking.Input{
		Player1:               ranking.PlayerState{Rating: player1.Rating, RatedMatches: player1.RatedMatches, TodayPositiveGain: r.gainDaily[gain1Key]},
		Player2:               ranking.PlayerState{Rating: player2.Rating, RatedMatches: player2.RatedMatches, TodayPositiveGain: r.gainDaily[gain2Key]},
		Player1Won:            *match.Result == 1,
		RaceLength:            RatingV2RaceLength(match, rounds),
		PriorPairMatchesToday: prior,
		DailyPositiveCap:      r.dailyCap,
	})
	for _, item := range []struct {
		player  *RatingReplayPlayer
		outcome ranking.Outcome
		gainKey ratingReplayUserDay
	}{
		{player: player1, outcome: outcome1, gainKey: gain1Key},
		{player: player2, outcome: outcome2, gainKey: gain2Key},
	} {
		item.player.Rating = item.outcome.After
		if item.outcome.Rated {
			item.player.RatedMatches++
		}
		if item.outcome.Change > 0 {
			r.gainDaily[item.gainKey] += item.outcome.Change
		}
	}
	r.Applied++
	return &RatingReplayMatchResult{
		MatchId:   match.Id,
		GameType:  match.GameType,
		Player1Id: player1Id,
		Player2Id: player2Id,
		Player1:   outcome1,
		Player2:   outcome2,
	}, true
}

func (r *RatingReplay) player(userId int64, gameType int) *RatingReplayPlayer {
	key := RatingReplayKey{UserId: userId, GameType: gameType}
	player, ok := r.players[key]
	if !ok {
		player = &RatingReplayPlayer{Rating: ranking.InitialRating}
		r.players[key] = player
	}
	return player
}

// Players 返回重放后的全部排位状态。
func (r *RatingReplay) Players() map[RatingReplayKey]RatingReplayPlayer {
	out := make(map[RatingReplayKey]RatingReplayPlayer, len(r.players))
	for key, player := range r.players {
		out[key] = *player
	}
	return out
}

// RatingReplayBatchSize 是离线重放每批读取的比赛数。
const RatingReplayBatchSize = 500

// ReplayCompletedMatches 按 (completed_at, id) 游标分批读取 [from, to) 内的已完成比赛与局记录并逐场重放，
// onResult 收到每场有效排位的结果（可为 nil）。
func ReplayCompletedMatches(svcCtx *svc.ServiceContext, replay *RatingReplay, from, to time.Time, onResult func(*RatingReplayMatchResult) error) error {
	if svcCtx == nil || svcCtx.MatchModel == nil || replay == nil {
		return errors.New("rating replay infrastructure is unavailable")
	}
	var afterCompletedAt *time.Time
	afterID := int64(0)
	for {
		matches, err := svcCtx.MatchModel.ListCompletedByCompletionBetweenWithTx(nil, from, to, afterCompletedAt, afterID, RatingReplayBatchSize)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return nil
		}
		matchIds := make([]int64, 0, len(matches))
		for _, match := range matches {
			matchIds = append(matchIds, match.Id)
		}
		rounds, err := svcCtx.MatchModel.ListCompletedRoundsByMatchIDs(matchIds)
		if err != nil {
			return err
		}
		roundsByMatch := make(map[int64][]model.MatchRound, len(matches))
		for _, round := range rounds {
			roundsByMatch[round.MatchId] = append(roundsByMatch[round.MatchId], round)
		}
		for index := range matches {
			result, ok := replay.Apply(&matches[index], roundsByMatch[matches[index].Id])
			if ok && onResult != nil {
				if err := onResult(result); err != nil {
					return err
				}
			}
		}
		last := matches[len(matches)-1]
		lastCompletedAt := *last.CompletedAt
		afterCompletedAt, afterID = &lastCompletedAt, last.Id
		if len(matches) < RatingReplayBatchSize {
			return nil
		}
	}
}

// RatingV2DailyPositiveCap 返回 V2 结算使用的每日上涨上限，供离线重放与线上保持一致。
func RatingV2DailyPositiveCap(svcCtx *svc.ServiceContext) int {
	return ratingV2DailyPositiveCap(svcCtx)
}
