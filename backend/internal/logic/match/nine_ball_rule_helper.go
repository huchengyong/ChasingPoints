package match

import (
	"errors"

	"chasing_points/internal/model"
)

// 九球追分（game_type=2）：每局按获胜方式计固定分，犯规固定给对方 1 分；
// 选手只能记录对手的胜局和犯规（与客户端一致），裁判可以记双方。
var nineBallWinScores = map[string]int{
	"normal":     4,
	"small_gold": 7,
	"big_gold":   10,
}

const nineBallFoulScore = 1

var (
	errNineBallScoreUnsupported = errors.New("九球追分请使用结束一局或犯规记分")
	errNineBallInvalidWin       = errors.New("九球追分获胜方式与得分不匹配")
	errNineBallInvalidFoulScore = errors.New("九球追分犯规固定计1分")
	errNineBallOpponentOnly     = errors.New("九球追分只能记录对手的胜局或犯规")
)

func isNineBallMatch(match *model.Match) bool {
	return match != nil && match.GameType == 2
}

func validateNineBallWin(viewerRole string, winner int, winType string, score int) error {
	if expected, ok := nineBallWinScores[winType]; !ok || score != expected {
		return errNineBallInvalidWin
	}
	if !viewerMayTargetActor(viewerRole, winner) {
		return errNineBallOpponentOnly
	}
	return nil
}

// resolveNineBallFoulScore 兼容客户端不传分数（0），其他取值必须等于固定犯规分。
func resolveNineBallFoulScore(viewerRole string, foulActor int, score int) (int, error) {
	if score != 0 && score != nineBallFoulScore {
		return 0, errNineBallInvalidFoulScore
	}
	if !viewerMayTargetActor(viewerRole, foulActor) {
		return 0, errNineBallOpponentOnly
	}
	return nineBallFoulScore, nil
}
