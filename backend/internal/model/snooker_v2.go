package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	SnookerEventVersion = 2

	MatchActionTypeSnookerStroke      = "snooker_stroke"
	MatchActionTypeSnookerFrameAction = "snooker_frame_action"

	SnookerEventKindStroke      = "stroke"
	SnookerEventKindFrameAction = "frame_action"

	SnookerOutcomePot     = "pot"
	SnookerOutcomeNoScore = "no_score"
	SnookerOutcomeFoul    = "foul"

	SnookerPhaseReds                  = "reds"
	SnookerPhaseColorAfterRed         = "color_after_red"
	SnookerPhaseColors                = "colors"
	SnookerPhaseRespottedBlackPending = "respotted_black_pending"
	SnookerPhaseRespottedBlack        = "respotted_black"
	SnookerPhaseEnded                 = "ended"

	SnookerBallRed         = "red"
	SnookerBallColorChoice = "color_choice"
	SnookerBallYellow      = "yellow"
	SnookerBallGreen       = "green"
	SnookerBallBrown       = "brown"
	SnookerBallBlue        = "blue"
	SnookerBallPink        = "pink"
	SnookerBallBlack       = "black"

	SnookerFoulIncomingPlays           = "incoming_plays"
	SnookerFoulOffenderPlaysFromLeft   = "offender_plays_from_left"
	SnookerFoulOffenderReplaysOriginal = "offender_replays_original"

	SnookerFrameActionStartRespottedBlack = "start_respotted_black"
	SnookerFrameActionOfferConcession     = "offer_concession"
	SnookerFrameActionAcceptConcession    = "accept_concession"
	SnookerFrameActionRejectConcession    = "reject_concession"
	SnookerFrameActionAwardFrame          = "award_frame"

	SnookerConcessionScopeFrame = "frame"
	SnookerConcessionScopeMatch = "match"

	SnookerFrameEndClearance       = "clearance"
	SnookerFrameEndRespottedBlack  = "respotted_black"
	SnookerFrameEndConcession      = "concession"
	SnookerFrameEndMatchConcession = "match_concession"
	SnookerFrameEndRepeatedMiss    = "repeated_miss"
	SnookerFrameEndRefereeAward    = "referee_award"
)

var ErrInvalidSnookerEvent = errors.New("invalid snooker event")

// SnookerEvent is the versioned event stored in MatchAction.ExtraData.
type SnookerEvent struct {
	Version              int    `json:"version"`
	Kind                 string `json:"kind"`
	Actor                int    `json:"actor"`
	VisitNo              int    `json:"visit_no,omitempty"`
	Outcome              string `json:"outcome,omitempty"`
	BallOnValue          int    `json:"ball_on_value,omitempty"`
	PottedReds           int    `json:"potted_reds,omitempty"`
	BallOnPotted         bool   `json:"ball_on_potted,omitempty"`
	FreeBallValue        int    `json:"free_ball_value,omitempty"`
	FreeBallPotted       bool   `json:"free_ball_potted,omitempty"`
	Penalty              int    `json:"penalty,omitempty"`
	RedsRemoved          int    `json:"reds_removed,omitempty"`
	FoulAndMiss          bool   `json:"foul_and_miss,omitempty"`
	MissSequenceEligible bool   `json:"miss_sequence_eligible,omitempty"`
	FoulResolution       string `json:"foul_resolution,omitempty"`
	FreeBallAwarded      bool   `json:"free_ball_awarded,omitempty"`
	CueBallInHand        bool   `json:"cue_ball_in_hand,omitempty"`
	FrameAction          string `json:"frame_action,omitempty"`
	Scope                string `json:"scope,omitempty"`
	Winner               int    `json:"winner,omitempty"`
	Reason               string `json:"reason,omitempty"`
}

type SnookerTransition struct {
	State       SnookerRoundState
	ScoreChange int
	ScoreActor  int
}

func EncodeSnookerEvent(event SnookerEvent) (string, error) {
	if err := validateSnookerEventShape(event); err != nil {
		return "", err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidSnookerEvent, err)
	}
	return string(data), nil
}

func DecodeSnookerEvent(raw *string) (SnookerEvent, error) {
	if raw == nil || *raw == "" {
		return SnookerEvent{}, fmt.Errorf("%w: missing extra_data", ErrInvalidSnookerEvent)
	}
	decoder := json.NewDecoder(bytes.NewBufferString(*raw))
	decoder.DisallowUnknownFields()
	var event SnookerEvent
	if err := decoder.Decode(&event); err != nil {
		return SnookerEvent{}, fmt.Errorf("%w: %v", ErrInvalidSnookerEvent, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return SnookerEvent{}, fmt.Errorf("%w: trailing data", ErrInvalidSnookerEvent)
	}
	if err := validateSnookerEventShape(event); err != nil {
		return SnookerEvent{}, err
	}
	return event, nil
}

func validateSnookerEventShape(event SnookerEvent) error {
	if event.Version != SnookerEventVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidSnookerEvent, event.Version)
	}
	if event.Actor != 1 && event.Actor != 2 {
		return fmt.Errorf("%w: actor must be 1 or 2", ErrInvalidSnookerEvent)
	}
	switch event.Kind {
	case SnookerEventKindStroke:
		if event.VisitNo <= 0 {
			return fmt.Errorf("%w: missing visit", ErrInvalidSnookerEvent)
		}
		if event.Outcome != SnookerOutcomePot && event.Outcome != SnookerOutcomeNoScore && event.Outcome != SnookerOutcomeFoul {
			return fmt.Errorf("%w: invalid stroke outcome", ErrInvalidSnookerEvent)
		}
		if event.FrameAction != "" || event.Scope != "" || event.Winner != 0 || event.Reason != "" {
			return fmt.Errorf("%w: stroke contains frame fields", ErrInvalidSnookerEvent)
		}
	case SnookerEventKindFrameAction:
		if event.VisitNo != 0 || event.Outcome != "" || event.BallOnValue != 0 || event.PottedReds != 0 || event.BallOnPotted ||
			event.FreeBallValue != 0 || event.FreeBallPotted || event.Penalty != 0 || event.RedsRemoved != 0 ||
			event.FoulAndMiss || event.MissSequenceEligible || event.FoulResolution != "" || event.FreeBallAwarded || event.CueBallInHand {
			return fmt.Errorf("%w: frame action contains stroke fields", ErrInvalidSnookerEvent)
		}
		switch event.FrameAction {
		case SnookerFrameActionStartRespottedBlack:
			if event.Scope != "" || event.Winner != 0 || event.Reason != "" {
				return fmt.Errorf("%w: invalid respotted black fields", ErrInvalidSnookerEvent)
			}
		case SnookerFrameActionOfferConcession, SnookerFrameActionAcceptConcession, SnookerFrameActionRejectConcession:
			if event.Scope != "" && event.Scope != SnookerConcessionScopeFrame && event.Scope != SnookerConcessionScopeMatch {
				return fmt.Errorf("%w: invalid concession scope", ErrInvalidSnookerEvent)
			}
			if event.Winner != 0 || event.Reason != "" {
				return fmt.Errorf("%w: invalid concession fields", ErrInvalidSnookerEvent)
			}
		case SnookerFrameActionAwardFrame:
			if event.Winner != 1 && event.Winner != 2 || event.Actor != event.Winner || event.Reason == "" {
				return fmt.Errorf("%w: invalid frame award fields", ErrInvalidSnookerEvent)
			}
			if event.Scope != "" && event.Scope != SnookerConcessionScopeFrame && event.Scope != SnookerConcessionScopeMatch {
				return fmt.Errorf("%w: invalid frame award scope", ErrInvalidSnookerEvent)
			}
		default:
			return fmt.Errorf("%w: invalid frame action", ErrInvalidSnookerEvent)
		}
	default:
		return fmt.Errorf("%w: invalid event kind", ErrInvalidSnookerEvent)
	}
	return nil
}

func ValidateSnookerMatchFormat(bestOfFrames, startingActor int) error {
	if bestOfFrames <= 0 || bestOfFrames%2 == 0 {
		return errors.New("斯诺克总局数必须为正奇数")
	}
	if startingActor != 1 && startingActor != 2 {
		return errors.New("请选择首局开球方")
	}
	return nil
}

func SnookerStartingActor(firstActor, roundNo int) int {
	if (firstActor != 1 && firstActor != 2) || roundNo <= 0 {
		return 0
	}
	if roundNo%2 == 1 {
		return firstActor
	}
	return otherSnookerActor(firstActor)
}

func NewSnookerRoundStateV2(roundNo, startingActor int) (SnookerRoundState, error) {
	if roundNo <= 0 || (startingActor != 1 && startingActor != 2) {
		return SnookerRoundState{}, errors.New("无效的斯诺克局初始化参数")
	}
	state := SnookerRoundState{
		RulesVersion:  SnookerRulesVersionWPBSA,
		RoundNo:       roundNo,
		Phase:         SnookerPhaseReds,
		BallOn:        SnookerBallRed,
		Striker:       startingActor,
		StartingActor: startingActor,
		VisitNo:       1,
		RedsRemaining: 15,
		ClearedColors: make([]int, 0, 6),
	}
	syncSnookerLegacyFields(&state)
	return state, nil
}

func ApplySnookerEvent(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	if state.RulesVersion != SnookerRulesVersionWPBSA {
		return SnookerTransition{}, errors.New("当前局不是斯诺克规则版本2")
	}
	if err := validateSnookerEventShape(event); err != nil {
		return SnookerTransition{}, err
	}
	if state.FrameEnded {
		return SnookerTransition{}, errors.New("当前局已结束")
	}
	if event.Kind == SnookerEventKindStroke {
		return applySnookerStroke(state, event)
	}
	return applySnookerFrameAction(state, event)
}

func applySnookerStroke(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	if state.PendingConcessionActor != 0 {
		return SnookerTransition{}, errors.New("请先处理认输提议")
	}
	if state.Phase == SnookerPhaseRespottedBlackPending {
		return SnookerTransition{}, errors.New("请先确定重置黑球先打方")
	}
	if event.Actor != state.Striker {
		return SnookerTransition{}, errors.New("只能记录当前击球方的击球")
	}
	if event.VisitNo != state.VisitNo {
		return SnookerTransition{}, errors.New("上手编号已过期")
	}

	switch event.Outcome {
	case SnookerOutcomePot:
		return applySnookerPot(state, event)
	case SnookerOutcomeNoScore:
		return applySnookerNoScore(state, event)
	case SnookerOutcomeFoul:
		return applySnookerFoul(state, event)
	default:
		return SnookerTransition{}, errors.New("无效的击球结果")
	}
}

func applySnookerPot(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	if event.Penalty != 0 || event.RedsRemoved != 0 || event.FoulAndMiss || event.MissSequenceEligible ||
		event.FoulResolution != "" || event.FreeBallAwarded || event.CueBallInHand {
		return SnookerTransition{}, errors.New("进球动作包含犯规字段")
	}
	ballOnValue, err := snookerBallOnValue(state, event.BallOnValue)
	if err != nil {
		return SnookerTransition{}, err
	}
	usingFreeBall := event.FreeBallValue != 0
	if usingFreeBall {
		if !state.FreeBallAvailable {
			return SnookerTransition{}, errors.New("当前没有自由球")
		}
		if event.FreeBallValue < 2 || event.FreeBallValue > 7 || event.FreeBallValue == ballOnValue {
			return SnookerTransition{}, errors.New("请选择合法的自由球")
		}
	} else if event.FreeBallPotted {
		return SnookerTransition{}, errors.New("缺少自由球指定球")
	}

	points := 0
	switch state.BallOn {
	case SnookerBallRed:
		if event.BallOnPotted {
			return SnookerTransition{}, errors.New("红球请使用入袋红球数")
		}
		if event.PottedReds < 0 || event.PottedReds > state.RedsRemaining {
			return SnookerTransition{}, errors.New("入袋红球数不合法")
		}
		if !usingFreeBall && event.PottedReds == 0 {
			return SnookerTransition{}, errors.New("进球动作至少需要一颗红球")
		}
		if usingFreeBall && !event.FreeBallPotted && event.PottedReds == 0 {
			return SnookerTransition{}, errors.New("进球动作没有入袋球")
		}
		points = event.PottedReds
		if event.FreeBallPotted {
			points++
		}
		state.RedsRemaining -= event.PottedReds
		state.Phase = SnookerPhaseColorAfterRed
		state.BallOn = SnookerBallColorChoice
	case SnookerBallColorChoice:
		if event.PottedReds != 0 || (!event.BallOnPotted && !event.FreeBallPotted) {
			return SnookerTransition{}, errors.New("自选彩球进球结果不合法")
		}
		if !usingFreeBall && !event.BallOnPotted {
			return SnookerTransition{}, errors.New("目标彩球未入袋")
		}
		points = ballOnValue
		advanceAfterColorFollowingRed(&state)
	default:
		if event.PottedReds != 0 || (!event.BallOnPotted && !event.FreeBallPotted) {
			return SnookerTransition{}, errors.New("清彩进球结果不合法")
		}
		if !usingFreeBall && !event.BallOnPotted {
			return SnookerTransition{}, errors.New("目标彩球未入袋")
		}
		points = ballOnValue
		if event.BallOnPotted {
			if err := advanceClearanceColor(&state, ballOnValue); err != nil {
				return SnookerTransition{}, err
			}
		}
	}

	if points <= 0 {
		return SnookerTransition{}, errors.New("进球分值必须大于0")
	}
	addSnookerScore(&state, event.Actor, points)
	state.CurrentBreak += points
	state.FreeBallAvailable = false
	state.CueBallInHand = false
	resetSnookerMissSequence(&state)

	if state.Phase == SnookerPhaseRespottedBlack {
		finishSnookerFrame(&state, event.Actor, SnookerFrameEndRespottedBlack)
	} else if state.Phase == SnookerPhaseEnded {
		finishAfterFinalBlack(&state)
	}
	syncSnookerLegacyFields(&state)
	return SnookerTransition{State: state, ScoreChange: points, ScoreActor: event.Actor}, nil
}

func applySnookerNoScore(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	if event.PottedReds != 0 || event.BallOnPotted || event.FreeBallPotted || event.Penalty != 0 ||
		event.RedsRemoved != 0 || event.FoulAndMiss || event.MissSequenceEligible || event.FoulResolution != "" || event.FreeBallAwarded || event.CueBallInHand {
		return SnookerTransition{}, errors.New("未进球动作包含无效字段")
	}
	ballOnValue, err := snookerBallOnValue(state, event.BallOnValue)
	if err != nil {
		return SnookerTransition{}, err
	}
	if event.FreeBallValue != 0 {
		if !state.FreeBallAvailable || event.FreeBallValue < 2 || event.FreeBallValue > 7 || event.FreeBallValue == ballOnValue {
			return SnookerTransition{}, errors.New("自由球指定不合法")
		}
	}

	if state.Phase == SnookerPhaseColorAfterRed {
		advanceAfterColorFollowingRed(&state)
	}
	state.FreeBallAvailable = false
	state.CueBallInHand = false
	resetSnookerMissSequence(&state)
	startNextSnookerVisit(&state, otherSnookerActor(event.Actor))
	syncSnookerLegacyFields(&state)
	return SnookerTransition{State: state}, nil
}

func applySnookerFoul(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	if event.PottedReds != 0 || event.BallOnPotted || event.FreeBallValue != 0 || event.FreeBallPotted {
		return SnookerTransition{}, errors.New("犯规动作不得包含合法入袋分")
	}
	if event.Penalty < 4 || event.Penalty > 7 {
		return SnookerTransition{}, errors.New("斯诺克犯规罚分必须为4至7分")
	}
	ballOnValue, err := snookerBallOnValue(state, event.BallOnValue)
	if err != nil {
		return SnookerTransition{}, err
	}
	minimumPenalty := 4
	if ballOnValue > minimumPenalty {
		minimumPenalty = ballOnValue
	}
	if event.Penalty < minimumPenalty {
		return SnookerTransition{}, fmt.Errorf("当前目标球犯规至少罚%d分", minimumPenalty)
	}
	if event.RedsRemoved < 0 || event.RedsRemoved > state.RedsRemaining {
		return SnookerTransition{}, errors.New("离台红球数不合法")
	}
	if event.MissSequenceEligible && (!event.FoulAndMiss || event.FoulResolution != SnookerFoulOffenderReplaysOriginal) {
		return SnookerTransition{}, errors.New("连续Miss资格仅用于原位重打")
	}
	if event.FreeBallAwarded && event.FoulResolution != SnookerFoulIncomingPlays {
		return SnookerTransition{}, errors.New("只有非犯规方从现状击球时可判自由球")
	}
	if event.CueBallInHand && event.FoulResolution == SnookerFoulOffenderReplaysOriginal {
		return SnookerTransition{}, errors.New("原位重打不得同时标记D区手中球")
	}
	validResolution := event.FoulResolution == SnookerFoulIncomingPlays || event.FoulResolution == SnookerFoulOffenderPlaysFromLeft || event.FoulResolution == SnookerFoulOffenderReplaysOriginal
	finalBlack := state.Phase == SnookerPhaseRespottedBlack || (state.Phase == SnookerPhaseColors && state.BallOn == SnookerBallBlack)
	if (!finalBlack && !validResolution) || (finalBlack && event.FoulResolution != "" && !validResolution) {
		return SnookerTransition{}, errors.New("请选择犯规后的继续方式")
	}

	nonOffender := otherSnookerActor(event.Actor)
	addSnookerScore(&state, nonOffender, event.Penalty)
	state.CurrentBreak = 0
	state.FreeBallAvailable = false
	state.CueBallInHand = false

	if finalBlack {
		if event.Penalty != 7 {
			return SnookerTransition{}, errors.New("黑球为目标时犯规罚7分")
		}
		if state.Phase == SnookerPhaseRespottedBlack {
			finishSnookerFrame(&state, nonOffender, SnookerFrameEndRespottedBlack)
		} else {
			finishAfterFinalBlack(&state)
		}
		syncSnookerLegacyFields(&state)
		return SnookerTransition{State: state, ScoreChange: event.Penalty, ScoreActor: nonOffender}, nil
	}

	switch event.FoulResolution {
	case SnookerFoulIncomingPlays, SnookerFoulOffenderPlaysFromLeft:
		state.RedsRemaining -= event.RedsRemoved
		advanceAfterFoulFromLeft(&state)
		nextActor := nonOffender
		if event.FoulResolution == SnookerFoulOffenderPlaysFromLeft {
			nextActor = event.Actor
		}
		startNextSnookerVisit(&state, nextActor)
		state.FreeBallAvailable = event.FreeBallAwarded
		state.CueBallInHand = event.CueBallInHand
		resetSnookerMissSequence(&state)
	case SnookerFoulOffenderReplaysOriginal:
		startNextSnookerVisit(&state, event.Actor)
		if event.FoulAndMiss && event.MissSequenceEligible {
			state.MissSequenceCount++
			state.MissWarningActive = state.MissSequenceCount >= 2
			if state.MissSequenceCount >= 3 {
				finishSnookerFrame(&state, nonOffender, SnookerFrameEndRepeatedMiss)
			}
		} else {
			resetSnookerMissSequence(&state)
		}
	default:
		return SnookerTransition{}, errors.New("请选择犯规后的继续方式")
	}

	syncSnookerLegacyFields(&state)
	return SnookerTransition{State: state, ScoreChange: event.Penalty, ScoreActor: nonOffender}, nil
}

func applySnookerFrameAction(state SnookerRoundState, event SnookerEvent) (SnookerTransition, error) {
	switch event.FrameAction {
	case SnookerFrameActionStartRespottedBlack:
		if state.PendingConcessionActor != 0 {
			return SnookerTransition{}, errors.New("请先处理认输提议")
		}
		if state.Phase != SnookerPhaseRespottedBlackPending {
			return SnookerTransition{}, errors.New("当前不需要重置黑球")
		}
		state.Phase = SnookerPhaseRespottedBlack
		state.BallOn = SnookerBallBlack
		state.CueBallInHand = true
		state.FreeBallAvailable = false
		startNextSnookerVisit(&state, event.Actor)
	case SnookerFrameActionOfferConcession:
		if state.PendingConcessionActor != 0 {
			return SnookerTransition{}, errors.New("已有待处理的认输提议")
		}
		scope := event.Scope
		if scope == "" {
			scope = SnookerConcessionScopeFrame
		}
		if scope != SnookerConcessionScopeFrame && scope != SnookerConcessionScopeMatch {
			return SnookerTransition{}, errors.New("认输范围不合法")
		}
		if scope == SnookerConcessionScopeFrame && !CanOfferSnookerFrameConcession(state, event.Actor) {
			return SnookerTransition{}, errors.New("当前无需罚分即可反超，不能认输本局")
		}
		state.PendingConcessionActor = event.Actor
		state.PendingConcessionScope = scope
	case SnookerFrameActionAcceptConcession:
		if state.PendingConcessionActor == 0 || event.Actor != otherSnookerActor(state.PendingConcessionActor) {
			return SnookerTransition{}, errors.New("没有可接受的认输提议")
		}
		if event.Scope != "" && event.Scope != state.PendingConcessionScope {
			return SnookerTransition{}, errors.New("认输范围与待处理提议不一致")
		}
		reason := SnookerFrameEndConcession
		if state.PendingConcessionScope == SnookerConcessionScopeMatch {
			reason = SnookerFrameEndMatchConcession
		}
		finishSnookerFrame(&state, event.Actor, reason)
	case SnookerFrameActionRejectConcession:
		if state.PendingConcessionActor == 0 || event.Actor != otherSnookerActor(state.PendingConcessionActor) {
			return SnookerTransition{}, errors.New("没有可拒绝的认输提议")
		}
		if event.Scope != "" && event.Scope != state.PendingConcessionScope {
			return SnookerTransition{}, errors.New("认输范围与待处理提议不一致")
		}
		state.PendingConcessionActor = 0
		state.PendingConcessionScope = ""
	case SnookerFrameActionAwardFrame:
		if event.Winner != 1 && event.Winner != 2 {
			return SnookerTransition{}, errors.New("请选择判局胜方")
		}
		if event.Reason == "" {
			return SnookerTransition{}, errors.New("请填写判局原因")
		}
		finishSnookerFrame(&state, event.Winner, SnookerFrameEndRefereeAward)
	default:
		return SnookerTransition{}, errors.New("无效的斯诺克局动作")
	}
	syncSnookerLegacyFields(&state)
	return SnookerTransition{State: state}, nil
}

func ReplaySnookerRoundV2(actions []MatchAction, roundNo, startingActor int) (SnookerRoundState, error) {
	state, err := NewSnookerRoundStateV2(roundNo, startingActor)
	if err != nil {
		return SnookerRoundState{}, err
	}
	for _, action := range actions {
		if action.RoundNo != roundNo || action.IsUndone != 0 {
			continue
		}
		if action.ActionType == "round_start" {
			if action.Actor != 0 || action.ScoreChange != 0 || action.ExtraData != nil {
				return SnookerRoundState{}, fmt.Errorf("action %d: invalid round_start", action.Id)
			}
			continue
		}
		if action.ActionType != MatchActionTypeSnookerStroke && action.ActionType != MatchActionTypeSnookerFrameAction {
			return SnookerRoundState{}, fmt.Errorf("action %d: unsupported action type %s", action.Id, action.ActionType)
		}
		event, err := DecodeSnookerEvent(action.ExtraData)
		if err != nil {
			return SnookerRoundState{}, fmt.Errorf("action %d: %w", action.Id, err)
		}
		if event.Actor != action.Actor {
			return SnookerRoundState{}, fmt.Errorf("action %d: actor mismatch", action.Id)
		}
		if action.ActionType == MatchActionTypeSnookerStroke && event.Kind != SnookerEventKindStroke {
			return SnookerRoundState{}, fmt.Errorf("action %d: kind mismatch", action.Id)
		}
		if action.ActionType == MatchActionTypeSnookerFrameAction && event.Kind != SnookerEventKindFrameAction {
			return SnookerRoundState{}, fmt.Errorf("action %d: kind mismatch", action.Id)
		}
		transition, err := ApplySnookerEvent(state, event)
		if err != nil {
			return SnookerRoundState{}, fmt.Errorf("action %d: %w", action.Id, err)
		}
		if action.ScoreChange != transition.ScoreChange {
			return SnookerRoundState{}, fmt.Errorf("action %d: score mismatch", action.Id)
		}
		state = transition.State
	}
	return state, nil
}

func SnookerRemainingPoints(state SnookerRoundState) int {
	switch state.Phase {
	case SnookerPhaseReds:
		points := state.RedsRemaining*8 + 27
		if state.FreeBallAvailable {
			points += 8
		}
		return points
	case SnookerPhaseColorAfterRed:
		return state.RedsRemaining*8 + 34
	case SnookerPhaseColors:
		points := 0
		value := snookerBallNameValue(state.BallOn)
		for score := value; score <= 7; score++ {
			points += score
		}
		if state.FreeBallAvailable {
			points += value
		}
		return points
	case SnookerPhaseRespottedBlackPending, SnookerPhaseRespottedBlack:
		return 7
	default:
		return 0
	}
}

func CanOfferSnookerFrameConcession(state SnookerRoundState, actor int) bool {
	if actor != 1 && actor != 2 || state.FrameEnded {
		return false
	}
	actorScore := state.Player1Score
	opponentScore := state.Player2Score
	if actor == 2 {
		actorScore, opponentScore = opponentScore, actorScore
	}
	return opponentScore > actorScore && opponentScore-actorScore > SnookerRemainingPoints(state)
}

func snookerBallOnValue(state SnookerRoundState, selected int) (int, error) {
	switch state.BallOn {
	case SnookerBallRed:
		if selected != 0 && selected != 1 {
			return 0, errors.New("当前目标球是红球")
		}
		return 1, nil
	case SnookerBallColorChoice:
		if selected < 2 || selected > 7 {
			return 0, errors.New("请选择目标彩球")
		}
		return selected, nil
	default:
		value := snookerBallNameValue(state.BallOn)
		if value == 0 || (selected != 0 && selected != value) {
			return 0, errors.New("当前目标球不匹配")
		}
		return value, nil
	}
}

func snookerBallNameValue(ball string) int {
	switch ball {
	case SnookerBallYellow:
		return 2
	case SnookerBallGreen:
		return 3
	case SnookerBallBrown:
		return 4
	case SnookerBallBlue:
		return 5
	case SnookerBallPink:
		return 6
	case SnookerBallBlack:
		return 7
	default:
		return 0
	}
}

func snookerBallValueName(value int) string {
	switch value {
	case 2:
		return SnookerBallYellow
	case 3:
		return SnookerBallGreen
	case 4:
		return SnookerBallBrown
	case 5:
		return SnookerBallBlue
	case 6:
		return SnookerBallPink
	case 7:
		return SnookerBallBlack
	default:
		return ""
	}
}

func advanceAfterColorFollowingRed(state *SnookerRoundState) {
	if state.RedsRemaining > 0 {
		state.Phase = SnookerPhaseReds
		state.BallOn = SnookerBallRed
		return
	}
	state.Phase = SnookerPhaseColors
	state.BallOn = SnookerBallYellow
}

func advanceAfterFoulFromLeft(state *SnookerRoundState) {
	switch state.Phase {
	case SnookerPhaseReds:
		if state.RedsRemaining == 0 {
			state.Phase = SnookerPhaseColors
			state.BallOn = SnookerBallYellow
		}
	case SnookerPhaseColorAfterRed:
		advanceAfterColorFollowingRed(state)
	}
}

func advanceClearanceColor(state *SnookerRoundState, value int) error {
	if state.Phase == SnookerPhaseRespottedBlack {
		return nil
	}
	if state.Phase != SnookerPhaseColors || snookerBallNameValue(state.BallOn) != value {
		return errors.New("清彩顺序不合法")
	}
	state.ClearedColors = append(state.ClearedColors, value)
	if value == 7 {
		state.Phase = SnookerPhaseEnded
		state.BallOn = ""
		return nil
	}
	state.BallOn = snookerBallValueName(value + 1)
	return nil
}

func finishAfterFinalBlack(state *SnookerRoundState) {
	if state.Player1Score == state.Player2Score {
		state.Phase = SnookerPhaseRespottedBlackPending
		state.BallOn = SnookerBallBlack
		state.Striker = 0
		state.CurrentBreak = 0
		state.FreeBallAvailable = false
		state.CueBallInHand = false
		return
	}
	winner := 1
	if state.Player2Score > state.Player1Score {
		winner = 2
	}
	finishSnookerFrame(state, winner, SnookerFrameEndClearance)
}

func finishSnookerFrame(state *SnookerRoundState, winner int, reason string) {
	state.Phase = SnookerPhaseEnded
	state.BallOn = ""
	state.Striker = 0
	state.CurrentBreak = 0
	state.FreeBallAvailable = false
	state.CueBallInHand = false
	state.PendingConcessionActor = 0
	state.PendingConcessionScope = ""
	state.FrameEnded = true
	state.FrameWinner = winner
	state.FrameEndReason = reason
	resetSnookerMissSequence(state)
}

func startNextSnookerVisit(state *SnookerRoundState, striker int) {
	state.VisitNo++
	state.Striker = striker
	state.CurrentBreak = 0
}

func addSnookerScore(state *SnookerRoundState, actor, score int) {
	if actor == 1 {
		state.Player1Score += score
	} else if actor == 2 {
		state.Player2Score += score
	}
}

func resetSnookerMissSequence(state *SnookerRoundState) {
	state.MissSequenceCount = 0
	state.MissWarningActive = false
}

func otherSnookerActor(actor int) int {
	if actor == 1 {
		return 2
	}
	if actor == 2 {
		return 1
	}
	return 0
}

func syncSnookerLegacyFields(state *SnookerRoundState) {
	if state.ClearedColors == nil {
		state.ClearedColors = make([]int, 0, 6)
	}
	state.RedBallCount = 15 - state.RedsRemaining
	if state.RedBallCount < 0 {
		state.RedBallCount = 0
	}
	state.ClearanceStarted = state.Phase == SnookerPhaseColors || state.Phase == SnookerPhaseRespottedBlackPending ||
		state.Phase == SnookerPhaseRespottedBlack || (state.FrameEnded && state.RedsRemaining == 0)
	state.ExpectedClearanceScore = 0
	if state.Phase == SnookerPhaseColors {
		state.ExpectedClearanceScore = snookerBallNameValue(state.BallOn)
	}
	state.ClearanceCompleted = len(state.ClearedColors) == 6
}
