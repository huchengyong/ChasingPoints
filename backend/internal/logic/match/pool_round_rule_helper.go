package match

import (
	"errors"

	"chasing_points/internal/model"
)

// 中式八球（game_type=3）与美式九球（game_type=4）：legacy 与灵活赛制都只通过结束一局记分，每局固定计1胜；
// 选手只能记录对手的胜局（与客户端一致），裁判可以记双方。
const poolRoundWinScore = 1

// 每局可记录的获胜方式；空值兼容不传 win_type 的请求，按普胜处理。
// 其他取值（如 start、其他球种的特殊成绩）会污染局数统计、撤销与成就结算，必须拒绝。
var poolRoundWinTypes = map[int]map[string]bool{
	3: {"": true, "normal": true, "break_clear": true, "continue_clear": true},
	4: {"": true, "normal": true, "small_gold": true, "big_gold": true},
}

var (
	errPoolRoundScoreUnsupported = errors.New("中式八球和美式九球请使用结束一局记录胜负")
	errPoolRoundInvalidWinType   = errors.New("获胜方式不适用于当前球种")
	errPoolRoundInvalidWin       = errors.New("每局只能计1胜")
	errPoolRoundOpponentOnly     = errors.New("只能记录对手的胜局")
)

func isPoolRoundMatch(match *model.Match) bool {
	return match != nil && model.IsPoolMatchFormatGameType(match.GameType)
}

func validatePoolRoundWin(gameType int, viewerRole string, winner int, winType string, score int) error {
	if !poolRoundWinTypes[gameType][winType] {
		return errPoolRoundInvalidWinType
	}
	if score != poolRoundWinScore {
		return errPoolRoundInvalidWin
	}
	if !viewerMayTargetActor(viewerRole, winner) {
		return errPoolRoundOpponentOnly
	}
	return nil
}
