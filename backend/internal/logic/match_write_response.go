package logic

import "chasing_points/internal/model"

type matchWriteScoreView struct {
	MyScore                   int
	OpponentScore             int
	CurrentFrameMyScore       int
	CurrentFrameOpponentScore int
}

func buildMatchWriteScoreView(userId int64, match *model.Match) matchWriteScoreView {
	view := matchWriteScoreView{}
	if match == nil {
		return view
	}

	view.MyScore = match.MyScore
	view.OpponentScore = match.OpponentScore
	view.CurrentFrameMyScore = match.CurrentFrameMyScore
	view.CurrentFrameOpponentScore = match.CurrentFrameOpponentScore

	if match.UserId != userId {
		view.MyScore = match.OpponentScore
		view.OpponentScore = match.MyScore
		view.CurrentFrameMyScore = match.CurrentFrameOpponentScore
		view.CurrentFrameOpponentScore = match.CurrentFrameMyScore
	}

	return view
}
