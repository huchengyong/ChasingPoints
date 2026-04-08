package match

import "chasing_points/internal/model"

const (
	matchViewerRoleUnknown  = ""
	matchViewerRolePlayer1  = "player1"
	matchViewerRolePlayer2  = "player2"
	matchViewerRoleReferee  = "referee"
)

type matchViewerCapabilities struct {
	ViewerRole   string
	RefereeBound bool
	RefereeUserId int64
	CanScore     bool
	CanUndo      bool
	CanFinish    bool
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
		capabilities.CanScore = !capabilities.RefereeBound
		capabilities.CanUndo = !capabilities.RefereeBound
		capabilities.CanFinish = !capabilities.RefereeBound
	case match.OpponentId != nil && *match.OpponentId == userId:
		capabilities.ViewerRole = matchViewerRolePlayer2
		capabilities.CanScore = !capabilities.RefereeBound
		capabilities.CanUndo = !capabilities.RefereeBound
		capabilities.CanFinish = !capabilities.RefereeBound
	}

	return capabilities
}

func shouldUsePlayer2Perspective(role string) bool {
	return role == matchViewerRolePlayer2
}
