package logic

import (
	"fmt"

	"chasing_points/internal/model"
)

func validateSnookerScore(state model.SnookerRoundState, score int) error {
	if score == 1 && state.RedBallCount >= 15 {
		return fmt.Errorf("本局红球已打完")
	}
	if score < 2 || score > 7 {
		return nil
	}
	if !state.ClearanceStarted {
		return nil
	}
	if state.ClearanceCompleted {
		return fmt.Errorf("清彩阶段已结束")
	}
	expected := state.ExpectedClearanceScore
	if expected == 0 {
		return nil
	}
	if score != expected {
		return fmt.Errorf("清彩阶段请先击打%s", getSnookerColorNameByScore(expected))
	}
	return nil
}

func getSnookerColorNameByScore(score int) string {
	switch score {
	case 2:
		return "黄球"
	case 3:
		return "绿球"
	case 4:
		return "咖啡球"
	case 5:
		return "蓝球"
	case 6:
		return "粉球"
	case 7:
		return "黑球"
	default:
		return "目标球"
	}
}
