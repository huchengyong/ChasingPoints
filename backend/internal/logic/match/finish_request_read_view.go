package match

import (
	"time"

	"chasing_points/internal/model"
)

func effectiveMatchForRead(match *model.Match, now time.Time) *model.Match {
	if match == nil {
		return nil
	}
	view := *match
	if isFinishRequestExpired(&view, now) {
		clearFinishRequest(&view)
	}
	return &view
}

func isFinishRequestExpired(match *model.Match, now time.Time) bool {
	return match != nil &&
		match.FinishState == model.FinishStatePendingConfirmation &&
		match.FinishRequestedAt != nil &&
		!match.FinishRequestedAt.Add(finishRequestTTL).After(now)
}
