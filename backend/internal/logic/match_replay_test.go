package logic

import (
	"slices"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

var replayTestStart = time.Date(2026, 10, 20, 20, 0, 0, 0, UTC8Location)

func replayStroke(round, actor, visit int, outcome string, score int, offset time.Duration) model.MatchAction {
	event := model.SnookerEvent{Version: model.SnookerEventVersion, Kind: model.SnookerEventKindStroke, Actor: actor, VisitNo: visit, Outcome: outcome}
	if outcome == model.SnookerOutcomeFoul {
		event.Penalty = score
	}
	extra, err := model.EncodeSnookerEvent(event)
	if err != nil {
		panic(err)
	}
	return model.MatchAction{RoundNo: round, ActionType: model.MatchActionTypeSnookerStroke, Actor: actor, ScoreChange: score, ExtraData: &extra, CreatedAt: replayTestStart.Add(offset)}
}

func replayLegacyAction(round int, actionType string, actor, score int, offset time.Duration) model.MatchAction {
	return model.MatchAction{RoundNo: round, ActionType: actionType, Actor: actor, ScoreChange: score, CreatedAt: replayTestStart.Add(offset)}
}

func replayRound(no, p1, p2, winner int, winType string) model.MatchRound {
	w := winner
	return model.MatchRound{RoundNo: no, MyScore: p1, OpponentScore: p2, Winner: &w, WinType: winType}
}

func snookerReplayMatch(targetWins int) *model.Match {
	return &model.Match{Id: 1, UserId: 1, GameType: 1, Status: 2, SnookerRulesVersion: model.SnookerRulesVersionWPBSA, SnookerFormat: model.SnookerFormatRaceTo, SnookerTargetWins: targetWins}
}

func poolReplayMatch(gameType, targetWins int) *model.Match {
	return &model.Match{Id: 2, UserId: 1, GameType: gameType, Status: 2, MatchFormat: model.MatchFormatRaceTo, TargetWins: targetWins}
}

// snookerReplayFixture：抢 3，第 2 局本人在落后 9 分后打出 68 分单杆。
func snookerReplayFixture() MatchReplayInput {
	frame1 := []model.MatchAction{
		replayStroke(1, 1, 1, model.SnookerOutcomePot, 30, 0),
		replayStroke(1, 2, 2, model.SnookerOutcomePot, 12, 60*time.Second),
	}
	frame2 := []model.MatchAction{
		{RoundNo: 2, ActionType: "round_start", CreatedAt: replayTestStart.Add(10 * time.Minute)}, // 开局：流程动作
		replayStroke(2, 2, 1, model.SnookerOutcomePot, 5, 10*time.Minute+10*time.Second),
		replayStroke(2, 1, 2, model.SnookerOutcomeFoul, 4, 10*time.Minute+20*time.Second), // 本人犯规，罚分给对手
	}
	for i, value := range []int{7, 7, 7, 7, 7, 7, 7, 7, 7, 5} {
		frame2 = append(frame2, replayStroke(2, 1, 3, model.SnookerOutcomePot, value, 10*time.Minute+time.Duration(30+i)*time.Second))
	}
	// 被撤销的一杆及撤销动作本身都不能计入。
	undone := replayStroke(2, 1, 3, model.SnookerOutcomePot, 7, 10*time.Minute+45*time.Second)
	undone.IsUndone = 1
	frame2 = append(frame2, undone,
		model.MatchAction{RoundNo: 2, ActionType: "undo", Actor: 1, CreatedAt: replayTestStart.Add(10*time.Minute + 46*time.Second)},
		replayStroke(2, 2, 4, model.SnookerOutcomeNoScore, 0, 10*time.Minute+95*time.Second),
	)
	frame3 := []model.MatchAction{ // 局记录是 10:40 而动作为空，对不上，只丢掉这一局的明细
		{RoundNo: 3, ActionType: "round_start", CreatedAt: replayTestStart.Add(20 * time.Minute)},
	}
	var actions []model.MatchAction
	actions = append(actions, frame1...)
	actions = append(actions, frame2...)
	actions = append(actions, frame3...)
	actions = append(actions, model.MatchAction{RoundNo: 0, ActionType: "match_end", CreatedAt: replayTestStart.Add(30 * time.Minute)})
	return MatchReplayInput{
		Match: snookerReplayMatch(3),
		Rounds: []model.MatchRound{
			replayRound(1, 30, 12, 1, "clearance"),
			replayRound(2, 68, 9, 1, "clearance"),
			replayRound(3, 10, 40, 2, "concession"),
		},
		Actions:     actions,
		ViewerActor: 1,
	}
}

func TestBuildMatchReplaySnookerV2ReturnsBreaksTrendFoulsAndDeficit(t *testing.T) {
	resp := BuildMatchReplay(snookerReplayFixture())
	if resp.FrameCount != 3 || len(resp.Frames) != 3 || resp.FrameOnly || resp.DetailTruncated {
		t.Fatalf("unexpected shape: %+v", resp)
	}
	frame2 := resp.Frames[1]
	if frame2.FrameNo != 2 || frame2.Winner != 1 || frame2.MyScore != 68 || frame2.OpponentScore != 9 || frame2.WinType != "clearance" {
		t.Fatalf("frame 2 basics: %+v", frame2)
	}
	if frame2.MyMaxRun != 68 || frame2.OpponentMaxRun != 5 {
		t.Fatalf("frame 2 must report a 68 break and ignore the undone pot: me=%d opponent=%d", frame2.MyMaxRun, frame2.OpponentMaxRun)
	}
	if frame2.MyFoulCount != 1 || frame2.MyFoulPoints != 4 || frame2.OpponentFoulCount != 0 || frame2.OpponentFoulPoints != 0 {
		t.Fatalf("frame 2 fouls: %+v", frame2)
	}
	if frame2.MaxDeficit != 9 {
		t.Fatalf("the winner trailed by 9 (5 + 4 foul points): %d", frame2.MaxDeficit)
	}
	// 走势只在得分变化处取点：开局、对手 5 分、犯规罚分、十次得分，共 13 个点，不含撤销的一杆。
	if len(frame2.Trend) != 13 || frame2.Trend[0] != (types.ReplayTrendPoint{Step: 0, Diff: 0}) || frame2.Trend[12] != (types.ReplayTrendPoint{Step: 12, Diff: 59}) {
		t.Fatalf("frame 2 trend must start at 0 and end at the final lead of 59: %+v", frame2.Trend)
	}
	if frame2.Trend[1].Diff != -5 || frame2.Trend[2].Diff != -9 || frame2.Trend[3].Diff != -2 {
		t.Fatalf("frame 2 trend start: %+v", frame2.Trend[:4])
	}
	if frame2.DurationSeconds != 85 { // 首个有效动作 +10s，最后一个有效动作 +95s
		t.Fatalf("duration is measured between the first and last valid action: %d", frame2.DurationSeconds)
	}
	if !slices.Equal(frame2.KeyTags, []string{replayKeyTagComeback}) {
		t.Fatalf("frame 2 is the biggest comeback: %v", frame2.KeyTags)
	}

	// 第 3 局动作与局记录对不上：只丢明细，局级数据保留。
	frame3 := resp.Frames[2]
	if frame3.Winner != 2 || frame3.MyScore != 10 || frame3.OpponentScore != 40 || frame3.WinType != "concession" || len(frame3.Trend) != 0 || frame3.MyMaxRun != 0 {
		t.Fatalf("frame 3 keeps frame-level data only: %+v", frame3)
	}
	if s := resp.Summary; s.MyFrameWins != 2 || s.OpponentFrameWins != 1 || s.MyMaxRun != 68 || s.OpponentMaxRun != 12 {
		t.Fatalf("summary: %+v", s)
	}
}

func TestBuildMatchReplaySwapsEverythingForTheOtherParticipant(t *testing.T) {
	in := snookerReplayFixture()
	in.ViewerActor = 2
	resp := BuildMatchReplay(in)
	frame2 := resp.Frames[1]
	if frame2.Winner != 2 || frame2.MyScore != 9 || frame2.OpponentScore != 68 || frame2.MyMaxRun != 5 || frame2.OpponentMaxRun != 68 {
		t.Fatalf("frame 2 from the opponent's view: %+v", frame2)
	}
	if frame2.MyFoulCount != 0 || frame2.OpponentFoulCount != 1 || frame2.OpponentFoulPoints != 4 {
		t.Fatalf("fouls must swap: %+v", frame2)
	}
	// 对手是胜方，从对手视角看「我」一直领先；对手落后 9 分对应分差 +9。
	if frame2.Trend[2].Diff != 9 || frame2.Trend[12].Diff != -59 || frame2.MaxDeficit != 9 {
		t.Fatalf("trend must be signed from the viewer's side: %+v deficit=%d", frame2.Trend, frame2.MaxDeficit)
	}
	if s := resp.Summary; s.MyFrameWins != 1 || s.OpponentFrameWins != 2 || s.MyMaxRun != 12 || s.OpponentMaxRun != 68 {
		t.Fatalf("summary from the opponent's view: %+v", s)
	}
}

func TestBuildMatchReplayMarksTheDeciderFrame(t *testing.T) {
	// 抢 5 打到 4:4，第 9 局是决胜局；之前任何一局都不是。
	winners := []int{1, 2, 1, 2, 1, 2, 1, 2, 1}
	var rounds []model.MatchRound
	p1, p2 := 0, 0
	for i, winner := range winners {
		if winner == 1 {
			p1++
		} else {
			p2++
		}
		rounds = append(rounds, replayRound(i+1, p1, p2, winner, "normal"))
	}
	resp := BuildMatchReplay(MatchReplayInput{Match: poolReplayMatch(3, 5), Rounds: rounds, ViewerActor: 1})
	for i, frame := range resp.Frames {
		isDecider := slices.Contains(frame.KeyTags, replayKeyTagDecider)
		if want := i == 8; isDecider != want {
			t.Fatalf("frame %d decider=%v want %v", frame.FrameNo, isDecider, want)
		}
	}

	// 自由局数没有决胜局。
	free := poolReplayMatch(3, 0)
	free.MatchFormat = model.MatchFormatFree
	for _, frame := range BuildMatchReplay(MatchReplayInput{Match: free, Rounds: rounds, ViewerActor: 1}).Frames {
		if len(frame.KeyTags) != 0 {
			t.Fatalf("free format has no decider: %+v", frame)
		}
	}
}

func TestBuildMatchReplayEightBallStreaksAndWinTypes(t *testing.T) {
	// 5:3：对手赢第 1、6、7 局，本人连赢第 2–5 局并赢第 8 局。
	plan := []struct {
		winner  int
		winType string
	}{{2, "normal"}, {1, "break_clear"}, {1, "normal"}, {1, "continue_clear"}, {1, "normal"}, {2, "break_clear"}, {2, "normal"}, {1, "normal"}}
	var rounds []model.MatchRound
	p1, p2 := 0, 0
	for i, item := range plan {
		if item.winner == 1 {
			p1++
		} else {
			p2++
		}
		rounds = append(rounds, replayRound(i+1, p1, p2, item.winner, item.winType))
	}
	resp := BuildMatchReplay(MatchReplayInput{Match: poolReplayMatch(3, 5), Rounds: rounds, ViewerActor: 1})

	if resp.FrameOnly {
		t.Fatalf("pool games are complete at frame level, they are not degraded: %+v", resp)
	}
	if resp.ViewerActor != 1 {
		t.Fatalf("other game types report the viewer actor too: %+v", resp)
	}
	s := resp.Summary
	if s.MyFrameWins != 5 || s.OpponentFrameWins != 3 || s.MyLongestStreak != 4 || s.OpponentLongestStreak != 2 {
		t.Fatalf("5:3 with a four-rack run: %+v", s)
	}
	wantStats := []types.ReplayWinTypeStat{
		{WinType: "normal", Label: "普胜", MyCount: 3, OpponentCount: 2},
		{WinType: "break_clear", Label: "炸清", MyCount: 1, OpponentCount: 1},
		{WinType: "continue_clear", Label: "接清", MyCount: 1},
	}
	if len(s.WinTypeStats) != 3 {
		t.Fatalf("win type stats: %+v", s.WinTypeStats)
	}
	for _, want := range wantStats {
		if !slices.Contains(s.WinTypeStats, want) {
			t.Fatalf("missing %+v in %+v", want, s.WinTypeStats)
		}
	}
	for i, frame := range resp.Frames { // 每局得分是累计比分相邻两局相减
		wantMine := 0
		if plan[i].winner == 1 {
			wantMine = 1
		}
		if frame.MyScore != wantMine || frame.OpponentScore != 1-wantMine || len(frame.Trend) != 0 {
			t.Fatalf("frame %d: %+v", frame.FrameNo, frame)
		}
	}
	// 本人是第 2 位参与者时连胜互换。
	swapped := BuildMatchReplay(MatchReplayInput{Match: poolReplayMatch(3, 5), Rounds: rounds, ViewerActor: 2}).Summary
	if swapped.MyLongestStreak != 2 || swapped.OpponentLongestStreak != 4 {
		t.Fatalf("streaks must swap: %+v", swapped)
	}
}

// chasingReplayFixture：整场 34:21。选手1 三次大金加一次普胜，选手2 一次大金、一次小金、一次普胜。
func chasingReplayFixture() MatchReplayInput {
	actions := []model.MatchAction{
		chasingWin(1, 2, 10, "big_gold", 0),
		chasingWin(2, 1, 10, "big_gold", time.Minute),
		chasingWin(3, 1, 10, "big_gold", 2*time.Minute),
		chasingWin(4, 2, 7, "small_gold", 3*time.Minute),
		chasingWin(5, 1, 10, "big_gold", 4*time.Minute),
		chasingWin(6, 1, 4, "normal", 5*time.Minute),
		chasingWin(7, 2, 4, "normal", 10*time.Minute),
	}
	rounds := []model.MatchRound{
		chasingRound(1, 0, 10, 2, "big_gold"),
		chasingRound(2, 10, 10, 1, "big_gold"),
		chasingRound(3, 20, 10, 1, "big_gold"),
		chasingRound(4, 20, 17, 2, "small_gold"),
		chasingRound(5, 30, 17, 1, "big_gold"),
		chasingRound(6, 34, 17, 1, "normal"),
		chasingRound(7, 34, 21, 2, "normal"),
	}
	return MatchReplayInput{Match: chasingMatch(34, 21), Rounds: rounds, Actions: actions, ViewerActor: 1}
}

func TestBuildMatchReplayChasingPointsIsOneWholeMatchFrame(t *testing.T) {
	resp := BuildMatchReplay(chasingReplayFixture())
	if resp.FrameCount != 7 || len(resp.Frames) != 1 || resp.FrameOnly || resp.DetailTruncated || resp.ViewerActor != 1 {
		t.Fatalf("unexpected shape: %+v", resp)
	}
	frame := resp.Frames[0]
	if frame.FrameNo != 1 || frame.MyScore != 34 || frame.OpponentScore != 21 || frame.Winner != 1 || frame.WinType != "" {
		t.Fatalf("whole-match frame: %+v", frame)
	}
	if len(frame.KeyTags) != 0 {
		t.Fatalf("九球追分 has no decider and no comeback tag here: %v", frame.KeyTags)
	}
	if len(frame.Trend) > MaxReplayTrendPoints || frame.Trend[0] != (types.ReplayTrendPoint{Step: 0, Diff: 0}) || frame.Trend[len(frame.Trend)-1].Diff != 13 {
		t.Fatalf("trend must start at 0 and end at the final lead of 13: %+v", frame.Trend)
	}
	if frame.MyMaxRun != 20 || frame.OpponentMaxRun != 10 || frame.MaxDeficit != 10 {
		t.Fatalf("runs and deficit: %+v", frame)
	}
	if frame.MyFoulCount != 0 || frame.OpponentFoulCount != 0 || frame.MyFoulPoints != 0 || frame.OpponentFoulPoints != 0 {
		t.Fatalf("no fouls in this match: %+v", frame)
	}
	if frame.DurationSeconds != 600 {
		t.Fatalf("duration between the first and last valid action: %d", frame.DurationSeconds)
	}

	summary := resp.Summary
	if summary.MyFrameWins != 0 || summary.OpponentFrameWins != 0 || summary.MyLongestStreak != 0 || summary.OpponentLongestStreak != 0 || len(summary.WinTypeStats) != 0 {
		t.Fatalf("九球追分 replay must not report frame wins or streaks: %+v", summary)
	}
	if summary.MyMaxRun != 20 || summary.OpponentMaxRun != 10 {
		t.Fatalf("summary runs: %+v", summary)
	}
	breakdown := summary.ScoreBreakdown
	if breakdown == nil || !breakdown.Available || len(breakdown.Events) != 0 {
		t.Fatalf("summary breakdown without events: %+v", breakdown)
	}
	if breakdown.Player1.Score != 34 || breakdown.Player1.NormalWins != 1 || breakdown.Player1.SmallGolds != 0 || breakdown.Player1.BigGolds != 3 {
		t.Fatalf("player 1 breakdown: %+v", breakdown.Player1)
	}
	if breakdown.Player2.Score != 21 || breakdown.Player2.NormalWins != 1 || breakdown.Player2.SmallGolds != 1 || breakdown.Player2.BigGolds != 1 {
		t.Fatalf("player 2 breakdown: %+v", breakdown.Player2)
	}
}

func TestBuildMatchReplayChasingPointsKeepsRunsThroughOpponentFouls(t *testing.T) {
	input := MatchReplayInput{
		Match: chasingMatch(12, 0),
		Rounds: []model.MatchRound{
			chasingRound(1, 4, 0, 1, "normal"),
			chasingRound(2, 12, 0, 1, "small_gold"),
		},
		Actions: []model.MatchAction{ // 普胜 4 + 对手犯规 1 + 小金 7，中间对手没有得分
			chasingWin(1, 1, 4, "normal", 0),
			chasingFoul(2, 2, 1, time.Minute),
			chasingWin(2, 1, 7, "small_gold", 2*time.Minute),
		},
		ViewerActor: 1,
	}
	resp := BuildMatchReplay(input)
	frame := resp.Frames[0]
	if resp.FrameOnly || frame.MyMaxRun < 12 || frame.OpponentMaxRun != 0 {
		t.Fatalf("opponent fouls do not break my run: %+v", frame)
	}
	if frame.OpponentFoulCount != 1 || frame.OpponentFoulPoints != 1 || frame.MyFoulCount != 0 {
		t.Fatalf("fouls belong to the opponent: %+v", frame)
	}
	if frame.Trend[len(frame.Trend)-1].Diff != 12 {
		t.Fatalf("trend ends at the final 12-point lead: %+v", frame.Trend)
	}
	if resp.Summary.MyMaxRun != frame.MyMaxRun {
		t.Fatalf("summary runs: %+v", resp.Summary)
	}
}

func TestBuildMatchReplayChasingPointsCountsFoulsAfterTheLastWin(t *testing.T) {
	input := MatchReplayInput{
		Match:  chasingMatch(6, 0),
		Rounds: []model.MatchRound{chasingRound(1, 4, 0, 1, "normal")},
		Actions: []model.MatchAction{
			chasingWin(1, 1, 4, "normal", 0),
			chasingFoul(2, 2, 2, 10*time.Minute), // 最后一次胜局之后才犯规，没有对应局记录
		},
		ViewerActor: 1,
	}
	resp := BuildMatchReplay(input)
	frame := resp.Frames[0]
	if resp.FrameOnly || frame.OpponentFoulCount != 1 || frame.OpponentFoulPoints != 2 {
		t.Fatalf("the foul after the last win must count: %+v", frame)
	}
	if frame.MyScore != 6 || frame.Trend[len(frame.Trend)-1].Diff != 6 {
		t.Fatalf("the tail foul adds to the score and the trend: %+v", frame)
	}
	if frame.DurationSeconds != 600 {
		t.Fatalf("duration includes the tail foul: %d", frame.DurationSeconds)
	}
	if b := resp.Summary.ScoreBreakdown; b == nil || !b.Available || b.Player1.Score != 6 || b.Player2.FoulPointsGiven != 2 {
		t.Fatalf("breakdown includes the tail foul: %+v", b)
	}
}

func TestBuildMatchReplayChasingPointsKeepsBreakdownInAbsolutePositions(t *testing.T) {
	in := chasingReplayFixture()
	in.ViewerActor = 2
	resp := BuildMatchReplay(in)
	frame := resp.Frames[0]
	if resp.ViewerActor != 2 || frame.MyScore != 21 || frame.OpponentScore != 34 || frame.Winner != 2 {
		t.Fatalf("viewer 2 sees the match from their side: %+v", frame)
	}
	if frame.MyMaxRun != 10 || frame.OpponentMaxRun != 20 {
		t.Fatalf("runs swap with the viewer: %+v", frame)
	}
	if frame.Trend[len(frame.Trend)-1].Diff != -13 {
		t.Fatalf("trend is signed from viewer 2: %+v", frame.Trend)
	}
	b := resp.Summary.ScoreBreakdown
	if b == nil || !b.Available || b.Player1.Score != 34 || b.Player2.Score != 21 {
		t.Fatalf("the breakdown stays in absolute positions: %+v", b)
	}
}

func TestBuildMatchReplayChasingPointsFallsBackWhenActionsDoNotReconcile(t *testing.T) {
	in := chasingReplayFixture()
	in.Match = chasingMatch(35, 21) // 动作累计 34:21，与最终比分对不上
	resp := BuildMatchReplay(in)
	if !resp.FrameOnly || len(resp.Frames) != 1 {
		t.Fatalf("mismatched actions must degrade to the total score: %+v", resp)
	}
	frame := resp.Frames[0]
	if frame.MyScore != 35 || frame.OpponentScore != 21 || frame.Winner != 1 {
		t.Fatalf("the total score and winner remain: %+v", frame)
	}
	if len(frame.Trend) != 0 || frame.MyMaxRun != 0 || frame.OpponentMaxRun != 0 || frame.DurationSeconds != 0 || len(frame.KeyTags) != 0 {
		t.Fatalf("no detail when the breakdown is unavailable: %+v", frame)
	}
	b := resp.Summary.ScoreBreakdown
	if b == nil || b.Available || b.Player1.Score != 35 || b.Player2.Score != 21 || b.Player1.NormalWins != 0 {
		t.Fatalf("unavailable breakdown keeps only totals: %+v", b)
	}
	if resp.Summary.MyFrameWins != 0 || resp.Summary.OpponentFrameWins != 0 {
		t.Fatalf("no frame wins: %+v", resp.Summary)
	}
}

func TestBuildMatchReplayChasingPointsTruncatedReturnsScoresOnly(t *testing.T) {
	in := chasingReplayFixture()
	in.Actions, in.Truncated = nil, true
	resp := BuildMatchReplay(in)
	if !resp.DetailTruncated || !resp.FrameOnly || len(resp.Frames) != 1 {
		t.Fatalf("truncated replay: %+v", resp)
	}
	frame := resp.Frames[0]
	if frame.MyScore != 34 || frame.OpponentScore != 21 || frame.Winner != 1 || len(frame.Trend) != 0 || frame.MyMaxRun != 0 {
		t.Fatalf("truncated replay keeps only the scores: %+v", frame)
	}
	b := resp.Summary.ScoreBreakdown
	if b == nil || b.Available || b.Player1.Score != 34 || b.Player2.Score != 21 {
		t.Fatalf("truncated breakdown: %+v", b)
	}
}

func TestBuildMatchReplayChasingPointsIgnoresUndoneActions(t *testing.T) {
	undone := chasingWin(2, 1, 7, "small_gold", 2*time.Minute)
	undone.IsUndone = 1
	resp := BuildMatchReplay(MatchReplayInput{
		Match:  chasingMatch(4, 0),
		Rounds: []model.MatchRound{chasingRound(1, 4, 0, 1, "normal")},
		Actions: []model.MatchAction{
			chasingWin(1, 1, 4, "normal", 0),
			undone,
			{RoundNo: 2, ActionType: "undo", Actor: 1, CreatedAt: chasingTestStart.Add(3 * time.Minute)},
		},
		ViewerActor: 1,
	})
	if resp.FrameOnly {
		t.Fatalf("the undone action leaves a reconciled match: %+v", resp)
	}
	frame := resp.Frames[0]
	if frame.MyScore != 4 || frame.MyMaxRun != 4 || frame.Trend[len(frame.Trend)-1].Diff != 4 {
		t.Fatalf("the undone small gold must not count: %+v", frame)
	}
	b := resp.Summary.ScoreBreakdown
	if b == nil || !b.Available || b.Player1.SmallGolds != 0 || b.Player1.NormalWins != 1 || b.Player1.Score != 4 {
		t.Fatalf("breakdown ignores the undone action: %+v", b)
	}
}

func TestBuildMatchReplayChasingPointsLimitsTheWholeMatchTrendTo60Points(t *testing.T) {
	actions := make([]model.MatchAction, 0, 200)
	for i := 0; i < 200; i++ {
		actions = append(actions, chasingScoreAction(1, 1+i%2, 1, time.Duration(i)*time.Second))
	}
	resp := BuildMatchReplay(MatchReplayInput{Match: chasingMatch(100, 100), Actions: actions, ViewerActor: 1})
	if resp.FrameOnly || len(resp.Frames) != 1 {
		t.Fatalf("unexpected shape: %+v", resp)
	}
	frame := resp.Frames[0]
	if frame.Winner != 0 || frame.MyScore != 100 || frame.OpponentScore != 100 {
		t.Fatalf("a tie has no winner: %+v", frame)
	}
	trend := frame.Trend
	if len(trend) > MaxReplayTrendPoints || trend[0] != (types.ReplayTrendPoint{Step: 0, Diff: 0}) || trend[len(trend)-1].Step != 200 || trend[len(trend)-1].Diff != 0 {
		t.Fatalf("trend must be downsampled to at most %d points and keep both endpoints: %+v", MaxReplayTrendPoints, trend)
	}
}

func TestBuildMatchReplayLegacySnookerOnlyReturnsFrameLevelData(t *testing.T) {
	legacy := []model.MatchAction{
		replayLegacyAction(1, "score", 1, 7, 0), replayLegacyAction(1, "score", 1, 7, 5*time.Second), replayLegacyAction(1, "win", 1, 0, 10*time.Second),
	}
	resp := BuildMatchReplay(MatchReplayInput{
		Match:       &model.Match{Id: 4, UserId: 1, GameType: 1, Status: 2, SnookerRulesVersion: model.SnookerRulesVersionLegacy},
		Rounds:      []model.MatchRound{replayRound(1, 14, 3, 1, "normal")},
		Actions:     legacy,
		ViewerActor: 1,
	})
	frame := resp.Frames[0]
	if !resp.FrameOnly || frame.Winner != 1 || frame.MyScore != 14 || frame.OpponentScore != 3 || frame.WinType != "normal" {
		t.Fatalf("legacy snooker keeps frame data: %+v frame=%+v", resp, frame)
	}
	if len(frame.Trend) != 0 || frame.MyMaxRun != 0 || frame.MaxDeficit != 0 || len(frame.KeyTags) != 0 {
		t.Fatalf("legacy snooker must not report breaks, trend or deficits: %+v", frame)
	}
}

func TestBuildMatchReplayTruncatedMatchesOnlyReturnFrameLevelData(t *testing.T) {
	in := snookerReplayFixture()
	in.Actions, in.Truncated = nil, true
	resp := BuildMatchReplay(in)
	if !resp.DetailTruncated || !resp.FrameOnly || len(resp.Frames) != 3 {
		t.Fatalf("truncated replay: %+v", resp)
	}
	for _, frame := range resp.Frames {
		if len(frame.Trend) != 0 || frame.MyMaxRun != 0 || frame.DurationSeconds != 0 {
			t.Fatalf("no event detail when truncated: %+v", frame)
		}
	}
	if resp.Frames[1].MyScore != 68 || resp.Summary.MyFrameWins != 2 {
		t.Fatalf("frame-level data is still complete: %+v", resp.Frames[1])
	}
}

func TestBuildMatchReplayRejectsSnookerV2FramesMixedWithLegacyActions(t *testing.T) {
	in := snookerReplayFixture()
	in.Actions = append(in.Actions, replayLegacyAction(1, "score", 1, 1, time.Minute))
	resp := BuildMatchReplay(in)
	if len(resp.Frames[0].Trend) != 0 {
		t.Fatalf("a frame that mixes legacy actions into a V2 match is not trustworthy: %+v", resp.Frames[0])
	}
	if len(resp.Frames[1].Trend) == 0 {
		t.Fatalf("other frames keep their detail: %+v", resp.Frames[1])
	}
}

func TestBuildMatchReplayWithoutRoundsOrMatch(t *testing.T) {
	if resp := BuildMatchReplay(MatchReplayInput{}); resp.Frames == nil || len(resp.Frames) != 0 || resp.FrameCount != 0 {
		t.Fatalf("no match: %+v", resp)
	}
	resp := BuildMatchReplay(MatchReplayInput{Match: poolReplayMatch(3, 5), ViewerActor: 1})
	if resp.FrameCount != 0 || len(resp.Frames) != 0 || resp.Summary == nil || resp.Summary.MyFrameWins != 0 {
		t.Fatalf("no rounds: %+v", resp)
	}
}

func TestDownsampleReplayTrendKeepsEndpointsAndExtremesWithinTheLimit(t *testing.T) {
	// 500 个点，最大值与最小值都落在不会被等间隔选中的位置。
	diffs := make([]int, 500)
	for i := range diffs {
		diffs[i] = i % 7
	}
	diffs[123], diffs[377] = 90, -90

	points := downsampleReplayTrend(diffs, 60)
	if len(points) != 60 || points[0].Step != 0 || points[len(points)-1].Step != 499 {
		t.Fatalf("expected exactly 60 points including both endpoints: len=%d first=%+v last=%+v", len(points), points[0], points[len(points)-1])
	}
	steps := make([]int, len(points))
	for i, point := range points {
		steps[i] = point.Step
		if point.Diff != diffs[point.Step] {
			t.Fatalf("point %+v must carry the value at its step", point)
		}
	}
	if !slices.IsSorted(steps) || len(slices.Compact(slices.Clone(steps))) != len(steps) {
		t.Fatalf("steps must be strictly increasing: %v", steps)
	}
	if !slices.Contains(steps, 123) || !slices.Contains(steps, 377) {
		t.Fatalf("the maximum and minimum points must be kept: %v", steps)
	}
}

func TestDownsampleReplayTrendLeavesShortSeriesAlone(t *testing.T) {
	short := []int{0, 3, -2, 5}
	points := downsampleReplayTrend(short, 60)
	if len(points) != 4 || points[2] != (types.ReplayTrendPoint{Step: 2, Diff: -2}) {
		t.Fatalf("short series must be returned as is: %+v", points)
	}
	exact := make([]int, 60)
	if got := downsampleReplayTrend(exact, 60); len(got) != 60 {
		t.Fatalf("a series exactly at the limit is kept: %d", len(got))
	}
	if got := downsampleReplayTrend(nil, 60); got == nil || len(got) != 0 {
		t.Fatalf("empty series: %+v", got)
	}
	flat := make([]int, 400)
	if got := downsampleReplayTrend(flat, 60); len(got) != 60 {
		t.Fatalf("a flat series must still honour the limit: %d", len(got))
	}
}

func TestBuildMatchReplayLimitsEveryFrameTrendTo60Points(t *testing.T) {
	var actions []model.MatchAction
	total := 0
	for i := 0; i < 300; i++ {
		actor := 1 + i%2
		actions = append(actions, replayStroke(1, actor, i+1, model.SnookerOutcomePot, 1, time.Duration(i)*time.Second))
		if actor == 1 {
			total++
		}
	}
	resp := BuildMatchReplay(MatchReplayInput{
		Match:       snookerReplayMatch(1),
		Rounds:      []model.MatchRound{replayRound(1, total, 300-total, 1, "clearance")},
		Actions:     actions,
		ViewerActor: 1,
	})
	trend := resp.Frames[0].Trend
	if len(trend) > MaxReplayTrendPoints || trend[0].Step != 0 || trend[len(trend)-1].Step != 300 {
		t.Fatalf("trend must be downsampled but keep both endpoints: len=%d %+v ... %+v", len(trend), trend[0], trend[len(trend)-1])
	}
}
