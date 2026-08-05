package match

import (
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func buildMatchLastAction(svcCtx *svc.ServiceContext, userId int64, match *model.Match) *types.MatchLastAction {
	if svcCtx == nil || svcCtx.MatchModel == nil || match == nil {
		return nil
	}
	action, err := svcCtx.MatchModel.GetLastAction(match.Id)
	if err != nil || action == nil {
		return nil
	}
	result := &types.MatchLastAction{
		ActionType:     action.ActionType,
		Actor:          action.Actor,
		ScoreChange:    action.ScoreChange,
		ServerRevision: action.ServerRevision,
	}
	role := resolveMatchViewerCapabilities(match, userId).ViewerRole
	actorName := "对手"
	if role == matchViewerRoleReferee {
		actorName = "选手2"
		if action.Actor == 1 {
			actorName = "选手1"
		}
	} else if (role == matchViewerRolePlayer1 && action.Actor == 1) || (role == matchViewerRolePlayer2 && action.Actor == 2) {
		actorName = "我方"
	}
	switch action.ActionType {
	case "score":
		result.Description = fmt.Sprintf("%s加%d分", actorName, action.ScoreChange)
	case "foul":
		result.Description = fmt.Sprintf("%s犯规，计%d分", actorName, action.ScoreChange)
	case "win":
		result.Description = fmt.Sprintf("%s结束本局", actorName)
	case "undo":
		result.Description = "撤销了最近一次操作"
	case "finish_request":
		result.Description = fmt.Sprintf("%s发起结束确认", actorName)
	case "finish_confirm":
		result.Description = fmt.Sprintf("%s确认结束", actorName)
	case "finish_dispute":
		result.Description = fmt.Sprintf("%s提出异议", actorName)
	case "finish_withdraw":
		result.Description = fmt.Sprintf("%s撤回结束请求", actorName)
	case model.MatchActionTypeSnookerStroke:
		if event, err := model.DecodeSnookerEvent(action.ExtraData); err == nil {
			switch event.Outcome {
			case model.SnookerOutcomePot:
				result.Description = fmt.Sprintf("%s本杆得%d分", actorName, action.ScoreChange)
			case model.SnookerOutcomeNoScore:
				result.Description = fmt.Sprintf("%s未进球，上手结束", actorName)
			case model.SnookerOutcomeFoul:
				result.Description = fmt.Sprintf("%s犯规，对方获%d分", actorName, action.ScoreChange)
			}
		}
	case model.MatchActionTypeSnookerFrameAction:
		if event, err := model.DecodeSnookerEvent(action.ExtraData); err == nil {
			switch event.FrameAction {
			case model.SnookerFrameActionStartRespottedBlack:
				result.Description = fmt.Sprintf("%s先打重置黑球", actorName)
			case model.SnookerFrameActionOfferConcession:
				result.Description = fmt.Sprintf("%s提出认输", actorName)
			case model.SnookerFrameActionAcceptConcession:
				result.Description = fmt.Sprintf("%s接受认输", actorName)
			case model.SnookerFrameActionRejectConcession:
				result.Description = fmt.Sprintf("%s拒绝认输", actorName)
			case model.SnookerFrameActionAwardFrame:
				result.Description = "裁判完成判局"
			}
		}
	default:
		result.Description = action.ActionType
	}
	return result
}
