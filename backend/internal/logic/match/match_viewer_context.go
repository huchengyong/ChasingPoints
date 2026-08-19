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
		capabilities.CanFinish = !capabilities.RefereeBound && match.Status == 1 && (model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice || !match.FinishConfirmationRequired)
		capabilities.CanRequestFinish = !capabilities.RefereeBound && match.FinishConfirmationRequired && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked && match.FinishState != model.FinishStatePendingConfirmation && match.Status == 1
	case match.OpponentId != nil && *match.OpponentId == userId:
		capabilities.ViewerRole = matchViewerRolePlayer2
		capabilities.CanScore = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		capabilities.CanUndo = !capabilities.RefereeBound && match.FinishState != model.FinishStatePendingConfirmation
		capabilities.CanFinish = !capabilities.RefereeBound && match.Status == 1 && (model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice || !match.FinishConfirmationRequired)
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
