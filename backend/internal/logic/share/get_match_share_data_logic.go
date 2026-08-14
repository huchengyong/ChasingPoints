package share

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
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
		svcCtx: svcCtx.WithContext(ctx),
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

	var completedCore *logicx.CompletedMatchCoreSummary
	if match.Status == 2 {
		completedCore, _ = logicx.BuildCompletedMatchCoreSummary(l.ctx, l.svcCtx, match)
	}
	player1Name := "玩家1"
	player1Avatar := ""
	if completedCore != nil && completedCore.Player1 != nil {
		player1Name = completedCore.Player1.Nickname
		player1Avatar = completedCore.Player1.Avatar
	} else if player1, _ := l.svcCtx.UserModel.FindById(match.UserId); player1 != nil {
		player1Name = player1.Nickname
		player1Avatar = player1.Avatar
	}

	player2Name := match.OpponentName
	player2Avatar := ""
	if match.OpponentId != nil {
		var player2NameOverride, player2AvatarOverride string
		if completedCore != nil && completedCore.Player2 != nil {
			player2NameOverride = completedCore.Player2.Nickname
			player2AvatarOverride = completedCore.Player2.Avatar
		} else if player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId); player2 != nil {
			player2NameOverride = player2.Nickname
			player2AvatarOverride = player2.Avatar
		}
		if player2NameOverride != "" {
			player2Name = player2NameOverride
			player2Avatar = player2AvatarOverride
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
	if completedCore != nil {
		viewer := completedCore.ViewerPerspective(userId)
		myName, myAvatar = viewer.MyName, viewer.MyAvatar
		opponentName, opponentAvatar = viewer.OpponentName, viewer.OpponentAvatar
		myScore, opponentScore = viewer.MyScore, viewer.OpponentScore
	}

	result := "战平"
	rankChange := 0
	if myScore > opponentScore {
		result = "胜利"
	} else if myScore < opponentScore {
		result = "失利"
	}

	if match.Status == 2 {
		if completedCore != nil {
			for _, rankLog := range completedCore.RankChanges {
				if rankLog.UserId == userId {
					rankChange = rankLog.FinalChange
					break
				}
			}
		} else if myRankLog, logErr := l.svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(match.Id, userId, match.GameType); logErr != nil {
			l.Logger.Errorf("获取分享段位明细失败: matchId=%d userId=%d err=%v", match.Id, userId, logErr)
		} else if myRankLog != nil {
			rankChange = myRankLog.FinalChange
		}
	}

	achievements := types.MatchAchievement{}
	if completedCore != nil {
		achievements = completedCore.Achievements
	} else if achList, achErr := l.svcCtx.MatchModel.GetAchievements(match.Id); achErr == nil {
		achievements = buildMatchAchievementPayload(achList)
	}

	player1Profile, _ := logicx.LoadCurrentCompetitiveProfile(l.svcCtx, match.UserId, match.GameType)
	player2Profile := logicx.CurrentCompetitiveProfile{}
	if match.OpponentId != nil {
		player2Profile, _ = logicx.LoadCurrentCompetitiveProfile(l.svcCtx, *match.OpponentId, match.GameType)
	}
	player1WinRate, player1MaxScore := player1Profile.WinRate, player1Profile.MaxScore
	player2WinRate, player2MaxScore := player2Profile.WinRate, player2Profile.MaxScore

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

	var roundCount int64
	if completedCore != nil {
		roundCount = int64(len(completedCore.Rounds))
	} else {
		roundCount, _ = l.svcCtx.MatchModel.GetRoundCount(match.Id)
	}
	currentRound := int(roundCount) + 1
	redBallCount := 0
	if match.GameType == 1 {
		if count, countErr := l.svcCtx.MatchModel.CountScoreActions(match.Id, currentRound, 1); countErr == nil {
			redBallCount = int(count)
		}
	}

	var completedRounds []model.MatchRound
	var actions []model.MatchAction
	if completedCore != nil {
		completedRounds = completedCore.Rounds
		actions = completedCore.Actions
	} else {
		completedRounds, _ = l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
		actions, _ = l.svcCtx.MatchModel.ListActiveActions(match.Id)
	}
	actionsForSummary := actions
	if match.GameType == 1 {
		if !match.CurrentFrameStarted {
			actionsForSummary = filterSnookerActionsToCompletedRounds(actions, completedRounds)
		}
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
