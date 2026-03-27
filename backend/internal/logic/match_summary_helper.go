package logic

import (
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildMatchSummary(gameType int, myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, myMaxScore int, opponentMaxScore int, redBallCount int, createdAt string, rounds []model.MatchRound, actions []model.MatchAction, achievements types.MatchAchievement) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	switch gameType {
	case 1:
		snookerActions := actions
		if len(rounds) > 0 {
			snookerActions = filterSnookerActionsToCompletedRounds(actions, rounds)
		}
		return buildSnookerSummary(myActor, myWinRate, opponentWinRate, myMaxScore, opponentMaxScore, redBallCount, createdAt, snookerActions)
	case 2:
		return buildNineBallSummary(myActor, myScore, opponentScore, myWinRate, opponentWinRate, createdAt, rounds, actions)
	case 3:
		return buildEightBallSummary(myActor, myScore, opponentScore, myWinRate, opponentWinRate, createdAt, rounds)
	case 4:
		return buildAmericanNineSummary(myActor, myScore, opponentScore, myWinRate, opponentWinRate, createdAt, rounds)
	default:
		return []types.MatchSummaryItem{}, []types.MatchSummaryItem{}
	}
}

func BuildMatchSummary(gameType int, myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, myMaxScore int, opponentMaxScore int, redBallCount int, createdAt string, rounds []model.MatchRound, actions []model.MatchAction, achievements types.MatchAchievement) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	return buildMatchSummary(gameType, myActor, myScore, opponentScore, myWinRate, opponentWinRate, myMaxScore, opponentMaxScore, redBallCount, createdAt, rounds, actions, achievements)
}

func buildSnookerSummary(myActor int, myWinRate float64, opponentWinRate float64, myMaxScore int, opponentMaxScore int, _ int, createdAt string, actions []model.MatchAction) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	myBreakStats, opponentBreakStats := calculateSnookerBreakStatsForActors(actions, myActor)
	if myBreakStats.Highest > 0 || opponentBreakStats.Highest > 0 {
		myMaxScore = myBreakStats.Highest
		opponentMaxScore = opponentBreakStats.Highest
	}

	highlights := []types.MatchSummaryItem{}
	if myMaxScore > 0 {
		highlights = append(highlights, buildSummaryItem("最高单杆", fmt.Sprintf("%d 分", myMaxScore), "本场个人连续得分上限", "flag", "emerald"))
	}
	if myBreakStats.FiftyPlus > 0 {
		highlights = append(highlights, buildSummaryItem("50+次数", fmt.Sprintf("%d 次", myBreakStats.FiftyPlus), "本场个人打出 50+ 的杆数", "fire", "gold"))
	}
	if myBreakStats.Centuries > 0 {
		highlights = append(highlights, buildSummaryItem("破百次数", fmt.Sprintf("%d 次", myBreakStats.Centuries), "本场个人完成破百的次数", "medal", "purple"))
	}

	stats := []types.MatchSummaryItem{
		buildSummaryItem("我的胜率", formatPercent(myWinRate), "当前历史胜率表现", "person", "blue"),
		buildSummaryItem("对手胜率", formatPercent(opponentWinRate), "用于感知本场对手强度", "person-filled", "purple"),
		buildSummaryItem("我的最高单杆", fmt.Sprintf("%d 分", myMaxScore), "斯诺克单杆得分上限", "flag", "emerald"),
		buildSummaryItem("对手最高单杆", fmt.Sprintf("%d 分", opponentMaxScore), "对手斯诺克单杆表现", "flag-filled", "orange"),
		buildSummaryItem("我的50+次数", fmt.Sprintf("%d 次", myBreakStats.FiftyPlus), "本场个人打出 50+ 的杆数", "fire", "gold"),
		buildSummaryItem("对手50+次数", fmt.Sprintf("%d 次", opponentBreakStats.FiftyPlus), "对手本场打出 50+ 的杆数", "fire", "orange"),
		buildSummaryItem("我的破百次数", fmt.Sprintf("%d 次", myBreakStats.Centuries), "本场个人完成破百的次数", "medal", "purple"),
		buildSummaryItem("对手破百次数", fmt.Sprintf("%d 次", opponentBreakStats.Centuries), "对手本场完成破百的次数", "medal", "slate"),
		buildSummaryItem("记录时间", createdAt, "本场总结生成时间", "calendar", "slate"),
	}
	return highlights, stats
}

func buildEightBallSummary(myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, createdAt string, rounds []model.MatchRound) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	highestStreak := calculateHighestWinStreak(rounds, myActor)
	myNormalWins := countRoundWinsByType(rounds, myActor, "normal")
	myBreakClears := countRoundWinsByType(rounds, myActor, "break_clear")
	myContinueClears := countRoundWinsByType(rounds, myActor, "continue_clear")
	mySpecialWins := myBreakClears + myContinueClears

	highlights := []types.MatchSummaryItem{}
	if myBreakClears > 0 {
		highlights = append(highlights, buildSummaryItem("炸清次数", fmt.Sprintf("%d 次", myBreakClears), "直接清台收下胜局", "fire", "gold"))
	}
	if myContinueClears > 0 {
		highlights = append(highlights, buildSummaryItem("接清次数", fmt.Sprintf("%d 次", myContinueClears), "连续进攻终结对局", "star", "blue"))
	}
	if highestStreak > 0 {
		highlights = append(highlights, buildSummaryItem("最高连胜", fmt.Sprintf("%d 局", highestStreak), "本场连下局数峰值", "top", "green"))
	}

	stats := []types.MatchSummaryItem{
		buildSummaryItem("我的胜率", formatPercent(myWinRate), "当前历史胜率表现", "person", "blue"),
		buildSummaryItem("对手胜率", formatPercent(opponentWinRate), "用于感知本场对手强度", "person-filled", "purple"),
		buildSummaryItem("总局数", fmt.Sprintf("%d 局", len(rounds)), "本场已完成局数", "bars", "cyan"),
		buildSummaryItem("本场分差", fmt.Sprintf("%d 局", absInt(myScore-opponentScore)), "以当前视角统计的领先/落后局差", "minus", "slate"),
		buildSummaryItem("我的普胜局", fmt.Sprintf("%d 局", myNormalWins), "常规胜局数量", "checkbox", "green"),
		buildSummaryItem("我的特殊胜局", fmt.Sprintf("%d 局", mySpecialWins), "炸清与接清合计", "medal", "gold"),
		buildSummaryItem("记录时间", createdAt, "本场总结生成时间", "calendar", "slate"),
	}
	return highlights, stats
}

func buildNineBallSummary(myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, createdAt string, rounds []model.MatchRound, actions []model.MatchAction) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	mySmallGold := countRoundWinsByType(rounds, myActor, "small_gold")
	myBigGold := countRoundWinsByType(rounds, myActor, "big_gold")
	myNormalWins := countRoundWinsByType(rounds, myActor, "normal")
	highestRun := calculateHighestScoringRun(actions, myActor)

	highlights := []types.MatchSummaryItem{}
	if mySmallGold > 0 {
		highlights = append(highlights, buildSummaryItem("小金次数", fmt.Sprintf("%d 次", mySmallGold), "九球关键收分回合", "medal", "blue"))
	}
	if myBigGold > 0 {
		highlights = append(highlights, buildSummaryItem("大金次数", fmt.Sprintf("%d 次", myBigGold), "高价值一杆制胜", "medal", "gold"))
	}
	if highestRun > 0 {
		highlights = append(highlights, buildSummaryItem("最高连续得分", fmt.Sprintf("%d 分", highestRun), "不被对手打断的连续得分峰值", "fire", "red"))
	}

	stats := []types.MatchSummaryItem{
		buildSummaryItem("我的胜率", formatPercent(myWinRate), "当前历史胜率表现", "person", "blue"),
		buildSummaryItem("对手胜率", formatPercent(opponentWinRate), "用于感知本场对手强度", "person-filled", "purple"),
		buildSummaryItem("总得分", fmt.Sprintf("%d 分", myScore+opponentScore), "双方本场合计得分", "bars", "cyan"),
		buildSummaryItem("我的普胜", fmt.Sprintf("%d 次", myNormalWins), "常规胜局得分次数", "checkbox", "green"),
		buildSummaryItem("我的小金", fmt.Sprintf("%d 次", mySmallGold), "小金完成次数", "star", "blue"),
		buildSummaryItem("我的大金", fmt.Sprintf("%d 次", myBigGold), "大金完成次数", "medal", "gold"),
		buildSummaryItem("记录时间", createdAt, "本场总结生成时间", "calendar", "slate"),
	}
	return highlights, stats
}

func buildAmericanNineSummary(myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, createdAt string, rounds []model.MatchRound) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	mySmallGold := countRoundWinsByType(rounds, myActor, "small_gold")
	myBigGold := countRoundWinsByType(rounds, myActor, "big_gold")
	myNormalWins := countRoundWinsByType(rounds, myActor, "normal")
	highestStreak := calculateHighestWinStreak(rounds, myActor)

	highlights := []types.MatchSummaryItem{}
	if myBigGold > 0 {
		highlights = append(highlights, buildSummaryItem("大金次数", fmt.Sprintf("%d 次", myBigGold), "开球直接终结本局", "medal", "gold"))
	}
	if mySmallGold > 0 {
		highlights = append(highlights, buildSummaryItem("小金次数", fmt.Sprintf("%d 次", mySmallGold), "关键金球收下胜局", "medal", "blue"))
	}
	if highestStreak > 0 {
		highlights = append(highlights, buildSummaryItem("最高连胜", fmt.Sprintf("%d 局", highestStreak), "本场连续赢下局数峰值", "top", "green"))
	}

	stats := []types.MatchSummaryItem{
		buildSummaryItem("我的胜率", formatPercent(myWinRate), "当前历史胜率表现", "person", "blue"),
		buildSummaryItem("对手胜率", formatPercent(opponentWinRate), "用于感知本场对手强度", "person-filled", "purple"),
		buildSummaryItem("总局数", fmt.Sprintf("%d 局", len(rounds)), "本场已完成局数", "bars", "cyan"),
		buildSummaryItem("本场分差", fmt.Sprintf("%d 局", absInt(myScore-opponentScore)), "以当前视角统计的领先/落后局差", "minus", "slate"),
		buildSummaryItem("我的普胜局", fmt.Sprintf("%d 局", myNormalWins), "常规赢下的局数", "checkbox", "green"),
		buildSummaryItem("我的小金", fmt.Sprintf("%d 次", mySmallGold), "小金完成次数", "star", "blue"),
		buildSummaryItem("我的大金", fmt.Sprintf("%d 次", myBigGold), "大金完成次数", "medal", "gold"),
		buildSummaryItem("记录时间", createdAt, "本场总结生成时间", "calendar", "slate"),
	}
	return highlights, stats
}

func buildSummaryItem(label string, value string, desc string, icon string, tone string) types.MatchSummaryItem {
	return types.MatchSummaryItem{
		Label: label,
		Value: value,
		Desc:  desc,
		Icon:  icon,
		Tone:  tone,
	}
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.0f%%", value)
}

func countRoundWinsByType(rounds []model.MatchRound, actor int, winType string) int {
	count := 0
	for _, round := range rounds {
		if round.Winner != nil && *round.Winner == actor && round.WinType == winType {
			count++
		}
	}
	return count
}

func calculateHighestWinStreak(rounds []model.MatchRound, actor int) int {
	best := 0
	current := 0
	for _, round := range rounds {
		if round.Winner != nil && *round.Winner == actor {
			current++
			if current > best {
				best = current
			}
			continue
		}
		current = 0
	}
	return best
}

func calculateHighestScoringRun(actions []model.MatchAction, actor int) int {
	best := 0
	current := 0
	for _, action := range actions {
		if action.ActionType == "round_start" {
			current = 0
			continue
		}
		if action.ScoreChange <= 0 || (action.ActionType != "score" && action.ActionType != "foul" && action.ActionType != "win") {
			continue
		}

		scoringActor := action.Actor
		if action.ActionType == "foul" {
			scoringActor = opponentActor(action.Actor)
		}

		if scoringActor == actor {
			current += action.ScoreChange
			if current > best {
				best = current
			}
			continue
		}
		current = 0
	}
	return best
}

func calculateSnookerHighestBreaks(actions []model.MatchAction, myActor int) (int, int) {
	return calculateSnookerHighestBreak(actions, myActor), calculateSnookerHighestBreak(actions, opponentActor(myActor))
}

func CalculateSnookerHighestBreaks(actions []model.MatchAction, myActor int) (int, int) {
	return calculateSnookerHighestBreaks(actions, myActor)
}

type snookerBreakStats struct {
	Highest   int
	FiftyPlus int
	Centuries int
}

func calculateSnookerBreakStatsForActors(actions []model.MatchAction, myActor int) (snookerBreakStats, snookerBreakStats) {
	return calculateSnookerBreakStats(actions, myActor), calculateSnookerBreakStats(actions, opponentActor(myActor))
}

func calculateSnookerBreakStats(actions []model.MatchAction, actor int) snookerBreakStats {
	stats := snookerBreakStats{}
	current := 0
	currentRound := 0
	finalize := func() {
		if current <= 0 {
			current = 0
			return
		}
		if current > stats.Highest {
			stats.Highest = current
		}
		if current >= 100 {
			stats.Centuries++
		} else if current >= 50 {
			stats.FiftyPlus++
		}
		current = 0
	}

	for _, action := range actions {
		if action.RoundNo != currentRound {
			finalize()
			currentRound = action.RoundNo
		}

		switch action.ActionType {
		case "score":
			if action.Actor == actor && action.ScoreChange > 0 {
				current += action.ScoreChange
				continue
			}
			finalize()
		case "foul", "win", "round_start", "match_end":
			finalize()
		default:
			if action.Actor != actor {
				finalize()
			}
		}
	}

	finalize()
	return stats
}

func calculateSnookerHighestBreak(actions []model.MatchAction, actor int) int {
	best := 0
	current := 0
	currentRound := 0

	for _, action := range actions {
		if action.RoundNo != currentRound {
			currentRound = action.RoundNo
			current = 0
		}

		switch action.ActionType {
		case "score":
			if action.Actor == actor && action.ScoreChange > 0 {
				current += action.ScoreChange
				if current > best {
					best = current
				}
				continue
			}
			current = 0
		case "foul", "win", "round_start":
			current = 0
		default:
			if action.Actor != actor {
				current = 0
			}
		}
	}

	return best
}

func calculateSnookerRedBallPots(actions []model.MatchAction, actor int) int {
	count := 0
	for _, action := range actions {
		if action.Actor == actor && action.ActionType == "score" && action.ScoreChange == 1 {
			count++
		}
	}
	return count
}

func opponentActor(actor int) int {
	if actor == 2 {
		return 1
	}
	return 2
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
