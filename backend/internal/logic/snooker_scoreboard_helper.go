package logic

import "chasing_points/internal/model"

func applySnookerScore(match *model.Match, actor int, score int) {
	if match == nil {
		return
	}
	match.CurrentFrameStarted = true
	if actor == 1 {
		match.CurrentFrameMyScore += score
		return
	}
	match.CurrentFrameOpponentScore += score
}

func applySnookerFoul(match *model.Match, actor int, score int) {
	if match == nil {
		return
	}
	match.CurrentFrameStarted = true
	if actor == 1 {
		match.CurrentFrameOpponentScore += score
		return
	}
	match.CurrentFrameMyScore += score
}

func finalizeSnookerFrame(match *model.Match, winner int) model.MatchRound {
	round := model.MatchRound{}
	if match == nil {
		return round
	}

	round.MyScore = match.CurrentFrameMyScore
	round.OpponentScore = match.CurrentFrameOpponentScore

	if winner == 1 {
		match.MyScore++
	} else if winner == 2 {
		match.OpponentScore++
	}

	match.CurrentFrameMyScore = 0
	match.CurrentFrameOpponentScore = 0
	match.CurrentFrameStarted = false

	return round
}

func reopenSnookerFrame(match *model.Match, round *model.MatchRound, winner int) {
	if match == nil || round == nil {
		return
	}

	if winner == 1 && match.MyScore > 0 {
		match.MyScore--
	} else if winner == 2 && match.OpponentScore > 0 {
		match.OpponentScore--
	}

	match.CurrentFrameMyScore = round.MyScore
	match.CurrentFrameOpponentScore = round.OpponentScore
	match.CurrentFrameStarted = true
}
