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

// poolWinTypeLabels 与复盘里的局胜法命名保持一致，只在最近操作文案中展示非普胜负法。
var poolWinTypeLabels = map[string]string{
	"break_clear":    "炸清",
	"continue_clear": "接清",
	"small_gold":     "小金",
	"big_gold":       "大金",
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

// actionActorName 返回动作发起方在查看者视角下的称呼；裁判或无身份时使用选手1/选手2。
func actionActorName(viewerRole string, actor int) string {
	if viewerRole != "player1" && viewerRole != "player2" {
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

// FoulDescription 生成犯规文案；action.Actor 是犯规方，得分归另一方。
func FoulDescription(viewerRole string, action *model.MatchAction) string {
	return fmt.Sprintf("%s犯规，%s得%d分", actionActorName(viewerRole, action.Actor), actionActorName(viewerRole, 3-action.Actor), action.ScoreChange)
}

// LastActionDescription 生成记分、局开始、对局结束、犯规与胜局的统一文案；其他动作返回空串。
func LastActionDescription(viewerRole string, gameType int, action *model.MatchAction) string {
	if action == nil {
		return ""
	}
	switch action.ActionType {
	case "score":
		return fmt.Sprintf("%s加%d分", actionActorName(viewerRole, action.Actor), action.ScoreChange)
	case "round_start":
		if action.RoundNo > 0 {
			return fmt.Sprintf("开始第%d局", action.RoundNo)
		}
		return "开始新一局"
	case "match_end":
		return "对局结束"
	case "foul":
		if gameType == 2 {
			return NineBallLastActionDescription(viewerRole, action)
		}
		return FoulDescription(viewerRole, action)
	case "win":
		if gameType == 2 {
			return NineBallLastActionDescription(viewerRole, action)
		}
		winner := actionActorName(viewerRole, action.Actor)
		if label, ok := poolWinTypeLabels[winTypeOfAction(action)]; ok {
			return fmt.Sprintf("%s%s，赢下本局", winner, label)
		}
		return fmt.Sprintf("%s赢下本局", winner)
	}
	return ""
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
