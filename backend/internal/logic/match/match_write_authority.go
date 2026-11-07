package match

import (
	"errors"

	"chasing_points/internal/model"
)

var (
	errMatchWriteForbiddenByReferee = errors.New("本场已由裁判接管记分")
	errMatchViewerNotParticipant    = errors.New("用户不是对局参与者")
	errMatchFinishPending           = errors.New("本场正在等待结束确认")
)

func validateMatchWriteAuthority(match *model.Match, userId int64) (matchViewerCapabilities, error) {
	capabilities := resolveMatchViewerCapabilities(match, userId)
	if capabilities.ViewerRole == matchViewerRoleUnknown {
		return capabilities, errMatchViewerNotParticipant
	}
	if capabilities.RefereeBound && capabilities.ViewerRole != matchViewerRoleReferee {
		return capabilities, errMatchWriteForbiddenByReferee
	}
	if match.FinishState == model.FinishStatePendingConfirmation && capabilities.ViewerRole != matchViewerRoleReferee {
		return capabilities, errMatchFinishPending
	}
	return capabilities, nil
}

// viewerMayTargetActor 判断记分对象是否允许：胜局的胜方、犯规的犯规方都必须是对手，裁判不受限。
func viewerMayTargetActor(viewerRole string, actor int) bool {
	switch viewerRole {
	case matchViewerRoleReferee:
		return actor == 1 || actor == 2
	case matchViewerRolePlayer1:
		return actor == 2
	case matchViewerRolePlayer2:
		return actor == 1
	default:
		return false
	}
}
