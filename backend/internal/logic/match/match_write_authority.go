package match

import (
	"errors"

	"chasing_points/internal/model"
)

var (
	errMatchWriteForbiddenByReferee = errors.New("本场已由裁判接管记分")
	errMatchViewerNotParticipant    = errors.New("用户不是对局参与者")
)

func validateMatchWriteAuthority(match *model.Match, userId int64) (matchViewerCapabilities, error) {
	capabilities := resolveMatchViewerCapabilities(match, userId)
	if capabilities.ViewerRole == matchViewerRoleUnknown {
		return capabilities, errMatchViewerNotParticipant
	}
	if capabilities.RefereeBound && capabilities.ViewerRole != matchViewerRoleReferee {
		return capabilities, errMatchWriteForbiddenByReferee
	}
	return capabilities, nil
}
