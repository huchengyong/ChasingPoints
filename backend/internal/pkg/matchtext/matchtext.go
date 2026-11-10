// Package matchtext 生成按查看者视角组织的对局动作文案，供 HTTP 详情与 WS 同步共用。
package matchtext

import (
	"encoding/json"
	"fmt"

	"chasing_points/internal/model"
)

var chasingWinTypeLabels = map[string]string{
	"normal":     "普胜",
	"small_gold": "小金",
	"big_gold":   "大金",
}

// NormalizeChasingWinType 把九球追分胜法归一到 normal / small_gold / big_gold，其余计为 other。
func NormalizeChasingWinType(winType string) string {
	switch winType {
	case "normal":
		return "normal"
	case "small_gold", "golden_break": // golden_break 是旧版小金的代码
		return "small_gold"
	case "big_gold", "nine_on_break": // nine_on_break 是旧版大金的代码
		return "big_gold"
	default:
		return "other"
	}
}

// actionActorName 返回动作发起方在查看者视角下的称呼；裁判视角使用选手1/选手2。
func actionActorName(viewerRole string, actor int) string {
	if viewerRole == "referee" {
		if actor == 1 {
			return "选手1"
		}
		return "选手2"
	}
	if (viewerRole == "player1" && actor == 1) || (viewerRole == "player2" && actor == 2) {
		return "我方"
	}
	return "对手"
}

// NineBallLastActionDescription 生成九球追分胜局/犯规动作的"最近操作"文案；其他动作返回空串。
// win 动作的 Actor 是得分方，foul 动作的 Actor 是犯规方。
func NineBallLastActionDescription(viewerRole string, action *model.MatchAction) string {
	if action == nil {
		return ""
	}
	switch action.ActionType {
	case "win":
		scorer := actionActorName(viewerRole, action.Actor)
		if label, ok := chasingWinTypeLabels[NormalizeChasingWinType(winTypeOfAction(action))]; ok {
			return fmt.Sprintf("%s%s，得%d分", scorer, label, action.ScoreChange)
		}
		return fmt.Sprintf("%s得%d分", scorer, action.ScoreChange)
	case "foul":
		fouler := actionActorName(viewerRole, action.Actor)
		scorer := actionActorName(viewerRole, 3-action.Actor)
		return fmt.Sprintf("%s犯规，%s得%d分", fouler, scorer, action.ScoreChange)
	}
	return ""
}

func winTypeOfAction(action *model.MatchAction) string {
	if action.ExtraData == nil {
		return ""
	}
	var payload struct {
		WinType string `json:"win_type"`
	}
	if err := json.Unmarshal([]byte(*action.ExtraData), &payload); err != nil {
		return ""
	}
	return payload.WinType
}
