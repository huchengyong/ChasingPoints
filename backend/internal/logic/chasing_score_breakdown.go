package logic

import (
	"encoding/json"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchtext"
	"chasing_points/internal/types"
)

// MaxChasingScoreEvents 是九球追分得分明细最多返回的条数；超出时只保留最近的明细。
const MaxChasingScoreEvents = 200

// ChasingScoreSide 是九球追分一方在整场的得分构成。
type ChasingScoreSide struct {
	Score           int
	NormalWins      int
	SmallGolds      int
	BigGolds        int
	FoulCount       int
	FoulPointsGiven int // 本方犯规送给对手的分
	OtherPoints     int
}

// ChasingScoreEvent 是一条得分明细，分数是记分后的双方累计总分。
type ChasingScoreEvent struct {
	Seq          int
	Kind         string // normal / small_gold / big_gold / foul / other
	Scorer       int    // 得分方：1=选手1 2=选手2
	Points       int
	Player1Score int
	Player2Score int
	CreatedAt    time.Time
}

// ChasingScoreBreakdown 是九球追分的得分构成；动作与最终比分对不上时 Available=false，只保留总分。
type ChasingScoreBreakdown struct {
	Available       bool
	Player1         ChasingScoreSide
	Player2         ChasingScoreSide
	Events          []ChasingScoreEvent
	EventsTruncated bool
}

// BuildChasingScoreBreakdown 从有效动作统计九球追分的得分构成与明细，不访问数据库。
// 动作需按 id 升序；分值一律取动作实际记录的分值，不按胜法反推。
func BuildChasingScoreBreakdown(match *model.Match, rounds []model.MatchRound, actions []model.MatchAction) *ChasingScoreBreakdown {
	if match == nil || match.GameType != 2 {
		return nil
	}

	roundWinTypes := make(map[int]string, len(rounds))
	for _, round := range rounds {
		if _, ok := roundWinTypes[round.RoundNo]; !ok {
			roundWinTypes[round.RoundNo] = round.WinType
		}
	}

	var sides [3]ChasingScoreSide
	events := []ChasingScoreEvent{}
	for _, action := range actions {
		if action.IsUndone != 0 || (action.Actor != 1 && action.Actor != 2) {
			continue
		}

		kind, scorer, points := "", action.Actor, action.ScoreChange
		switch action.ActionType {
		case "win":
			kind = normalizeChasingWinType(chasingWinType(action, roundWinTypes))
		case "foul":
			kind, scorer = "foul", opponentActor(action.Actor)
			sides[action.Actor].FoulCount++
			if points > 0 {
				sides[action.Actor].FoulPointsGiven += points
			}
		case "score":
			kind = "other"
		default:
			continue
		}
		if points <= 0 {
			continue
		}

		sides[scorer].Score += points
		switch kind {
		case "normal":
			sides[scorer].NormalWins++
		case "small_gold":
			sides[scorer].SmallGolds++
		case "big_gold":
			sides[scorer].BigGolds++
		case "foul":
			// 犯规得分已计入犯规方的送分，不再归入其他得分。
		default:
			sides[scorer].OtherPoints += points
		}
		events = append(events, ChasingScoreEvent{
			Seq:          len(events) + 1,
			Kind:         kind,
			Scorer:       scorer,
			Points:       points,
			Player1Score: sides[1].Score,
			Player2Score: sides[2].Score,
			CreatedAt:    action.CreatedAt,
		})
	}

	// 宁缺勿错：累计总分与最终比分对不上时，只保留总分，不给出可能错误的次数与明细。
	if sides[1].Score != match.MyScore || sides[2].Score != match.OpponentScore {
		return UnavailableChasingScoreBreakdown(match)
	}

	truncated := len(events) > MaxChasingScoreEvents
	if truncated {
		events = events[len(events)-MaxChasingScoreEvents:]
	}
	return &ChasingScoreBreakdown{
		Available:       true,
		Player1:         sides[1],
		Player2:         sides[2],
		Events:          events,
		EventsTruncated: truncated,
	}
}

// UnavailableChasingScoreBreakdown 返回只含双方总分的不可用得分构成；非九球追分返回 nil。
func UnavailableChasingScoreBreakdown(match *model.Match) *ChasingScoreBreakdown {
	if match == nil || match.GameType != 2 {
		return nil
	}
	return &ChasingScoreBreakdown{
		Player1: ChasingScoreSide{Score: match.MyScore},
		Player2: ChasingScoreSide{Score: match.OpponentScore},
		Events:  []ChasingScoreEvent{},
	}
}

// ChasingScoreBreakdownPayload 把得分构成转换成接口结构；withEvents=false 时不带明细。
func ChasingScoreBreakdownPayload(b *ChasingScoreBreakdown, withEvents bool) *types.ChasingScoreBreakdown {
	if b == nil {
		return nil
	}
	payload := &types.ChasingScoreBreakdown{
		Available: b.Available,
		Player1:   chasingScoreSidePayload(b.Player1),
		Player2:   chasingScoreSidePayload(b.Player2),
		Events:    []types.ChasingScoreEvent{},
	}
	if !withEvents {
		return payload
	}
	payload.EventsTruncated = b.EventsTruncated
	for _, event := range b.Events {
		payload.Events = append(payload.Events, types.ChasingScoreEvent{
			Seq:          event.Seq,
			Kind:         event.Kind,
			Scorer:       event.Scorer,
			Points:       event.Points,
			Player1Score: event.Player1Score,
			Player2Score: event.Player2Score,
			CreatedAt:    formatChasingScoreTime(event.CreatedAt),
		})
	}
	return payload
}

func chasingScoreSidePayload(side ChasingScoreSide) types.ChasingScoreSide {
	return types.ChasingScoreSide{
		Score:           side.Score,
		NormalWins:      side.NormalWins,
		SmallGolds:      side.SmallGolds,
		BigGolds:        side.BigGolds,
		FoulCount:       side.FoulCount,
		FoulPointsGiven: side.FoulPointsGiven,
		OtherPoints:     side.OtherPoints,
	}
}

// formatChasingScoreTime 输出详情页使用的本地时间格式；零值时间输出空串。
func formatChasingScoreTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// chasingWinType 优先取动作附加信息里的胜法，缺失或无法解析时回退到同局号局记录。
func chasingWinType(action model.MatchAction, roundWinTypes map[int]string) string {
	if action.ExtraData != nil {
		var payload struct {
			WinType string `json:"win_type"`
		}
		if err := json.Unmarshal([]byte(*action.ExtraData), &payload); err == nil && payload.WinType != "" {
			return payload.WinType
		}
	}
	return roundWinTypes[action.RoundNo]
}

// normalizeChasingWinType 把胜法归一到 normal / small_gold / big_gold，其余计为 other。
func normalizeChasingWinType(winType string) string {
	return matchtext.NormalizeChasingWinType(winType)
}
