package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局详情
func NewGetMatchDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchDetailLogic {
	return &GetMatchDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchDetailLogic) GetMatchDetail(req *types.GetMatchDetailReq) (resp *types.GetMatchDetailResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchDetailResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.GetMatchDetailResp{Success: false}, nil
	}

	// 验证用户权限（对局双方和裁判都可以查看）
	capabilities := resolveMatchViewerCapabilities(match, userId)
	if capabilities.ViewerRole == matchViewerRoleUnknown {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, refereeUserId=%v, currentUserId=%d",
			match.UserId, match.OpponentId, match.RefereeUserId, userId)
		return &types.GetMatchDetailResp{Success: false}, nil
	}
	isPlayer1 := capabilities.ViewerRole != matchViewerRolePlayer2

	// 根据视角获取分数和玩家信息
	var myScore, opponentScore int
	var currentFrameMyScore, currentFrameOpponentScore int
	var myName, opponentName, myAvatar, opponentAvatar string

	// 查询创建者(player1)信息
	player1, _ := l.svcCtx.UserModel.FindById(match.UserId)
	player1Name := "玩家1"
	player1Avatar := ""
	if player1 != nil {
		player1Name = player1.Nickname
		player1Avatar = player1.Avatar
	}

	// 查询对手(player2)信息
	player2Name := match.OpponentName
	player2Avatar := ""
	if match.OpponentId != nil {
		player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId)
		if player2 != nil {
			player2Name = player2.Nickname
			player2Avatar = player2.Avatar
		}
	}
	refereeName := ""
	if capabilities.RefereeBound && capabilities.RefereeUserId > 0 {
		if referee, err := l.svcCtx.UserModel.FindById(capabilities.RefereeUserId); err == nil && referee != nil {
			refereeName = referee.Nickname
		}
	}

	if capabilities.ViewerRole == matchViewerRoleReferee || isPlayer1 {
		// 当前用户是创建者，使用原始视角
		myScore = match.MyScore
		opponentScore = match.OpponentScore
		currentFrameMyScore = match.CurrentFrameMyScore
		currentFrameOpponentScore = match.CurrentFrameOpponentScore
		myName = player1Name
		myAvatar = player1Avatar
		opponentName = player2Name
		opponentAvatar = player2Avatar
	} else {
		// 当前用户是对手，交换视角
		myScore = match.OpponentScore
		opponentScore = match.MyScore
		currentFrameMyScore = match.CurrentFrameOpponentScore
		currentFrameOpponentScore = match.CurrentFrameMyScore
		myName = player2Name
		myAvatar = player2Avatar
		opponentName = player1Name
		opponentAvatar = player1Avatar
	}

	// 查询成绩数据
	achievements := types.MatchAchievement{}
	if achList, err := l.svcCtx.MatchModel.GetAchievements(match.Id); err == nil {
		achievements = buildMatchAchievementPayload(achList)
	}

	// 获取双方的胜率和单杆最高分
	var player1WinRate, player2WinRate float64
	var player1MaxScore, player2MaxScore int

	// 获取 player1(创建者) 的统计数据
	if stats1, err := l.svcCtx.MatchModel.GetUserStats(match.UserId); err == nil && stats1 != nil {
		if stats1.TotalMatches > 0 {
			player1WinRate = float64(stats1.Wins) / float64(stats1.TotalMatches) * 100
		}
	}
	if maxScore1, err := loadUserMaxSingleScore(l.svcCtx, match.UserId, match.GameType); err == nil {
		player1MaxScore = maxScore1
	}

	// 获取 player2(对手) 的统计数据（需要对手是注册用户）
	if match.OpponentId != nil {
		if stats2, err := l.svcCtx.MatchModel.GetUserStats(*match.OpponentId); err == nil && stats2 != nil {
			if stats2.TotalMatches > 0 {
				player2WinRate = float64(stats2.Wins) / float64(stats2.TotalMatches) * 100
			}
		}
		if maxScore2, err := loadUserMaxSingleScore(l.svcCtx, *match.OpponentId, match.GameType); err == nil {
			player2MaxScore = maxScore2
		}
	}

	// 根据视角设置我方和对手的统计数据
	var myWinRate, opponentWinRate float64
	var myMaxScore, opponentMaxScore int
	if isPlayer1 {
		myWinRate = player1WinRate
		myMaxScore = player1MaxScore
		opponentWinRate = player2WinRate
		opponentMaxScore = player2MaxScore
	} else {
		myWinRate = player2WinRate
		myMaxScore = player2MaxScore
		opponentWinRate = player1WinRate
		opponentMaxScore = player1MaxScore
	}

	myRankChange := 0
	opponentRankChange := 0
	myRankDetails := []types.RankDetail{}

	if match.Status == 2 {
		myRankLog, logErr := l.svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(match.Id, userId, match.GameType)
		if logErr != nil {
			l.Logger.Errorf("获取我的段位明细失败: matchId=%d userId=%d err=%v", match.Id, userId, logErr)
		} else if myRankLog != nil {
			myRankChange = myRankLog.FinalChange
			myRankDetails = rankLogToDetails(myRankLog)
		}

		allLogs, logsErr := l.svcCtx.RankingModel.ListRankChangesByMatchAndGameType(match.Id, match.GameType)
		if logsErr != nil {
			l.Logger.Errorf("获取对局段位明细失败: matchId=%d err=%v", match.Id, logsErr)
		} else if opponentLog := findOpponentRankLog(allLogs, userId); opponentLog != nil {
			opponentRankChange = opponentLog.FinalChange
		}
	}

	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	currentRound := int(roundCount) + 1
	redBallCount := 0
	snookerClearanceStarted := false
	snookerClearedColors := make([]int, 0, 6)
	snookerExpectedClearanceScore := 0
	snookerClearanceCompleted := false
	completedRounds, _ := l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
	actions, _ := l.svcCtx.MatchModel.ListActiveActions(match.Id)
	actionsForSummary := actions
	if match.GameType == 1 {
		state := model.BuildSnookerRoundState(actions, currentRound)
		redBallCount = state.RedBallCount
		snookerClearanceStarted = state.ClearanceStarted
		snookerClearedColors = state.ClearedColors
		snookerExpectedClearanceScore = state.ExpectedClearanceScore
		snookerClearanceCompleted = state.ClearanceCompleted
		if !match.CurrentFrameStarted {
			redBallCount = 0
			snookerClearanceStarted = false
			snookerClearedColors = make([]int, 0, 6)
			snookerExpectedClearanceScore = 0
			snookerClearanceCompleted = false
			actionsForSummary = filterSnookerActionsToCompletedRounds(actions, completedRounds)
		}
	}

	myActor := 1
	if !isPlayer1 {
		myActor = 2
	}
	if match.GameType == 1 {
		myMaxScore, opponentMaxScore = calculateSnookerHighestBreaks(actionsForSummary, myActor)
	}
	summaryHighlights, summaryStats := buildMatchSummary(
		match.GameType,
		myActor,
		myScore,
		opponentScore,
		myWinRate,
		opponentWinRate,
		myMaxScore,
		opponentMaxScore,
		redBallCount,
		match.CreatedAt.Format("2006-01-02 15:04:05"),
		completedRounds,
		actionsForSummary,
		achievements,
	)

	l.Logger.Infof("用户 %d 获取对局 %d 详情, isPlayer1=%v", userId, match.Id, isPlayer1)

	return &types.GetMatchDetailResp{
		Success: true,
		Match: types.MatchDetailData{
			Id:                            match.Id,
			GameType:                      match.GameType,
			Status:                        match.Status,
			ServerRevision:                match.SyncRevision,
			IsPlayer1:                     isPlayer1,
			ViewerRole:                    capabilities.ViewerRole,
			RefereeBound:                  capabilities.RefereeBound,
			RefereeUserId:                 capabilities.RefereeUserId,
			RefereeName:                   refereeName,
			CanScore:                      capabilities.CanScore,
			CanUndo:                       capabilities.CanUndo,
			CanFinish:                     capabilities.CanFinish,
			MyScore:                       myScore,
			OpponentScore:                 opponentScore,
			MyName:                        myName,
			OpponentName:                  opponentName,
			MyAvatar:                      myAvatar,
			OpponentAvatar:                opponentAvatar,
			MyWinRate:                     myWinRate,
			OpponentWinRate:               opponentWinRate,
			MyMaxScore:                    myMaxScore,
			OpponentMaxScore:              opponentMaxScore,
			MyRankChange:                  myRankChange,
			OpponentRankChange:            opponentRankChange,
			MyRankDetails:                 myRankDetails,
			CurrentFrameStarted:           match.CurrentFrameStarted,
			CurrentFrameMyScore:           currentFrameMyScore,
			CurrentFrameOpponentScore:     currentFrameOpponentScore,
			RedBallCount:                  redBallCount,
			SnookerClearanceStarted:       snookerClearanceStarted,
			SnookerClearedColors:          snookerClearedColors,
			SnookerExpectedClearanceScore: snookerExpectedClearanceScore,
			SnookerClearanceCompleted:     snookerClearanceCompleted,
			SummaryHighlights:             summaryHighlights,
			SummaryStats:                  summaryStats,
			Achievements:                  achievements,
			CreatedAt:                     match.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}
