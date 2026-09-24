package match

import "chasing_points/internal/model"

const (
	matchViewerRoleUnknown = ""
	matchViewerRolePlayer1 = "player1"
	matchViewerRolePlayer2 = "player2"
	matchViewerRoleReferee = "referee"
)

type matchViewerCapabilities struct {
	ViewerRole        string
	RefereeBound      bool
	RefereeUserId     int64
	CanScore          bool
	CanUndo           bool
	CanFinish         bool
	CanRequestFinish  bool
	CanConfirmFinish  bool
	CanDisputeFinish  bool
	CanWithdrawFinish bool
}

func resolveMatchViewerCapabilities(match *model.Match, userId int64) matchViewerCapabilities {
	capabilities := matchViewerCapabilities{}
	if match == nil || userId <= 0 {
		return capabilities
	}

	// 约球创建的比赛：任一参赛方可在合法结束点单方结束，无需对方确认。
	challengeMatch := match.ChallengeId != nil && *match.ChallengeId > 0

	if match.RefereeUserId != nil && *match.RefereeUserId > 0 {
		capabilities.RefereeBound = true
		capabilities.RefereeUserId = *match.RefereeUserId
	}

	switch {
	case capabilities.RefereeBound && capabilities.RefereeUserId == userId:
		capabilities.ViewerRole = matchViewerRoleReferee
		capabilities.CanScore = true
		capabilities.CanUndo = true
		capabilities.CanFinish = true
	case match.UserId == userId:
		capabilities.ViewerRole = matchViewerRolePlayer1
		capabilities.CanScore = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		capabilities.CanUndo = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		// 约球创建的比赛：任一参赛方在合法结束点单方结束；非约球比赛仍不得绕过裁判。
		capabilities.CanFinish = match.Status == 1 && match.FinishState != model.FinishStatePendingConfirmation &&
			((!capabilities.RefereeBound && (model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice || !match.FinishConfirmationRequired)) || challengeMatch)
		capabilities.CanRequestFinish = !capabilities.RefereeBound && match.FinishConfirmationRequired && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked && match.FinishState != model.FinishStatePendingConfirmation && match.Status == 1
	case match.OpponentId != nil && *match.OpponentId == userId:
		capabilities.ViewerRole = matchViewerRolePlayer2
		capabilities.CanScore = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		capabilities.CanUndo = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		capabilities.CanFinish = match.Status == 1 && match.FinishState != model.FinishStatePendingConfirmation &&
			((!capabilities.RefereeBound && (model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice || !match.FinishConfirmationRequired)) || challengeMatch)
		capabilities.CanRequestFinish = !capabilities.RefereeBound && match.FinishConfirmationRequired && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked && match.FinishState != model.FinishStatePendingConfirmation && match.Status == 1
	}

	if !capabilities.RefereeBound && match.FinishState == model.FinishStatePendingConfirmation && capabilities.ViewerRole != matchViewerRoleReferee {
		requestedBy := int64(0)
		if match.FinishRequestedBy != nil {
			requestedBy = *match.FinishRequestedBy
		}
		capabilities.CanConfirmFinish = requestedBy > 0 && requestedBy != userId
		capabilities.CanDisputeFinish = capabilities.CanConfirmFinish
		capabilities.CanWithdrawFinish = requestedBy == userId
	}
	if isSnookerV2Match(match) && match.FinishState != model.FinishStatePendingConfirmation && !snookerNormalFinishEligible(match) {
		capabilities.CanFinish = false
		capabilities.CanRequestFinish = false
	}
	if model.IsFlexiblePoolMatch(match) && match.FinishState != model.FinishStatePendingConfirmation && !poolNormalFinishEligible(match) {
		capabilities.CanFinish = false
		capabilities.CanRequestFinish = false
	}

	return capabilities
}

func shouldUsePlayer2Perspective(role string) bool {
	return role == matchViewerRolePlayer2
}

func resolveFinishRequestedBy(match *model.Match) int64 {
	if match == nil || match.FinishRequestedBy == nil {
		return 0
	}
	return *match.FinishRequestedBy
}

func challengeIdOf(match *model.Match) int64 {
	if match == nil || match.ChallengeId == nil {
		return 0
	}
	if *match.ChallengeId <= 0 {
		return 0
	}
	return *match.ChallengeId
}

// isChallengeFinisher 约球创建的比赛：任一参赛方（含裁判接管时）可单方结束或重放已完成的结束操作。
func isChallengeFinisher(match *model.Match, userId int64) bool {
	if match == nil || userId <= 0 || (match.Status != 1 && match.Status != 2) {
		return false
	}
	if match.ChallengeId == nil || *match.ChallengeId <= 0 {
		return false
	}
	isParticipant := match.UserId == userId || (match.OpponentId != nil && *match.OpponentId == userId)
	return isParticipant && match.FinishState != model.FinishStatePendingConfirmation
}
