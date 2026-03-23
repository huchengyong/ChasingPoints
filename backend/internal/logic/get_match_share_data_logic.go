package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchShareDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局分享数据
func NewGetMatchShareDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchShareDataLogic {
	return &GetMatchShareDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchShareDataLogic) GetMatchShareData(req *types.GetMatchShareDataReq) (resp *types.GetMatchShareDataResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchShareDataResp{Success: false}, nil
	}

	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.GetMatchShareDataResp{Success: false}, nil
	}

	isPlayer1 := match.UserId == userId
	isPlayer2 := match.OpponentId != nil && *match.OpponentId == userId
	if !isPlayer1 && !isPlayer2 {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, currentUserId=%d", match.UserId, match.OpponentId, userId)
		return &types.GetMatchShareDataResp{Success: false}, nil
	}

	player1, _ := l.svcCtx.UserModel.FindById(match.UserId)
	player1Name := "玩家1"
	player1Avatar := ""
	if player1 != nil {
		player1Name = player1.Nickname
		player1Avatar = player1.Avatar
	}

	player2Name := match.OpponentName
	player2Avatar := ""
	if match.OpponentId != nil {
		player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId)
		if player2 != nil {
			player2Name = player2.Nickname
			player2Avatar = player2.Avatar
		}
	}

	var myName, myAvatar, opponentName, opponentAvatar string
	var myScore, opponentScore int
	if isPlayer1 {
		myName = player1Name
		myAvatar = player1Avatar
		opponentName = player2Name
		opponentAvatar = player2Avatar
		myScore = match.MyScore
		opponentScore = match.OpponentScore
	} else {
		myName = player2Name
		myAvatar = player2Avatar
		opponentName = player1Name
		opponentAvatar = player1Avatar
		myScore = match.OpponentScore
		opponentScore = match.MyScore
	}

	result := "战平"
	rankChange := 0
	if myScore > opponentScore {
		result = "胜利"
	} else if myScore < opponentScore {
		result = "失利"
	}

	if match.Status == 2 {
		if myRankLog, logErr := l.svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(match.Id, userId, match.GameType); logErr != nil {
			l.Logger.Errorf("获取分享段位明细失败: matchId=%d userId=%d err=%v", match.Id, userId, logErr)
		} else if myRankLog != nil {
			rankChange = myRankLog.FinalChange
		}
	}

	achievements := types.MatchAchievement{}
	if achList, achErr := l.svcCtx.MatchModel.GetAchievements(match.Id); achErr == nil {
		achievements = buildMatchAchievementPayload(achList)
	}

	var player1WinRate, player2WinRate float64
	var player1MaxScore, player2MaxScore int
	if stats1, statsErr := l.svcCtx.MatchModel.GetUserStats(match.UserId); statsErr == nil && stats1 != nil && stats1.TotalMatches > 0 {
		player1WinRate = float64(stats1.Wins) / float64(stats1.TotalMatches) * 100
	}
	if maxScore1, maxErr := loadUserMaxSingleScore(l.svcCtx, match.UserId, match.GameType); maxErr == nil {
		player1MaxScore = maxScore1
	}
	if match.OpponentId != nil {
		if stats2, statsErr := l.svcCtx.MatchModel.GetUserStats(*match.OpponentId); statsErr == nil && stats2 != nil && stats2.TotalMatches > 0 {
			player2WinRate = float64(stats2.Wins) / float64(stats2.TotalMatches) * 100
		}
		if maxScore2, maxErr := loadUserMaxSingleScore(l.svcCtx, *match.OpponentId, match.GameType); maxErr == nil {
			player2MaxScore = maxScore2
		}
	}

	myWinRate := player1WinRate
	opponentWinRate := player2WinRate
	myMaxScore := player1MaxScore
	opponentMaxScore := player2MaxScore
	myActor := 1
	if !isPlayer1 {
		myWinRate = player2WinRate
		opponentWinRate = player1WinRate
		myMaxScore = player2MaxScore
		opponentMaxScore = player1MaxScore
		myActor = 2
	}

	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	currentRound := int(roundCount) + 1
	redBallCount := 0
	if match.GameType == 1 {
		if count, countErr := l.svcCtx.MatchModel.CountScoreActions(match.Id, currentRound, 1); countErr == nil {
			redBallCount = int(count)
		}
	}

	completedRounds, _ := l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
	actions, _ := l.svcCtx.MatchModel.ListActiveActions(match.Id)
	actionsForSummary := actions
	if match.GameType == 1 {
		if !match.CurrentFrameStarted {
			actionsForSummary = filterSnookerActionsToCompletedRounds(actions, completedRounds)
		}
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

	return &types.GetMatchShareDataResp{
		Success: true,
		Data: &types.MatchShareData{
			MatchId:           match.Id,
			GameType:          match.GameType,
			GameTypeName:      GetGameTypeName(match.GameType),
			MyName:            myName,
			MyAvatar:          myAvatar,
			MyScore:           myScore,
			OpponentName:      opponentName,
			OpponentAvatar:    opponentAvatar,
			OpponentScore:     opponentScore,
			Result:            result,
			MatchTime:         match.CreatedAt.Format("2006-01-02 15:04:05"),
			RankChange:        rankChange,
			SummaryHighlights: summaryHighlights,
			SummaryStats:      summaryStats,
		},
	}, nil
}
