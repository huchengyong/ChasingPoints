package public

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublicMatchDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取公开对局详情
func NewGetPublicMatchDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublicMatchDetailLogic {
	return &GetPublicMatchDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPublicMatchDetailLogic) GetPublicMatchDetail(req *types.GetPublicMatchDetailReq) (resp *types.GetPublicMatchDetailResp, err error) {
	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.GetPublicMatchDetailResp{Success: false}, nil
	}
	if model.NormalizeMatchVisibility(match.Visibility, match.MatchMode) != model.MatchVisibilityPublic {
		return &types.GetPublicMatchDetailResp{Success: false}, nil
	}

	// 查询创建者(player1)信息
	player1Name := "玩家1"
	player1Avatar := ""
	if player1, _ := l.svcCtx.UserModel.FindById(match.UserId); player1 != nil {
		player1Name = player1.Nickname
		player1Avatar = player1.Avatar
	}

	// 查询对手(player2)信息
	player2Name := match.OpponentName
	player2Avatar := ""
	var player2Id int64 = 0
	if match.OpponentId != nil {
		player2Id = *match.OpponentId
		if player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId); player2 != nil {
			player2Name = player2.Nickname
			player2Avatar = player2.Avatar
		}
	}

	// 计算对局持续时间（秒）
	var durationSeconds int64 = 0
	if match.Status == 1 { // 进行中
		durationSeconds = int64(time.Since(match.CreatedAt).Seconds())
	} else if match.EndTime != nil {
		durationSeconds = int64(match.EndTime.Sub(match.CreatedAt).Seconds())
	}

	// 查询局记录
	var rounds []types.RoundRecord
	if matchRounds, listErr := l.svcCtx.MatchModel.ListCompletedRounds(match.Id); listErr == nil {
		for _, r := range matchRounds {
			winner := 0
			if r.Winner != nil {
				winner = *r.Winner
			}
			rounds = append(rounds, types.RoundRecord{
				RoundNumber:  r.RoundNo,
				Player1Score: r.MyScore,       // MyScore 是 player1 的分数
				Player2Score: r.OpponentScore, // OpponentScore 是 player2 的分数
				Winner:       winner,
			})
		}
	}

	// 获取当前局数和总局数
	roundCount, _ := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	currentRound := int(roundCount) + 1 // 当前进行的是下一局
	totalRounds := int(roundCount)      // 已完成的局数
	if match.Status != 1 && !match.CurrentFrameStarted {
		currentRound = totalRounds
	}

	l.Logger.Infof("公开接口获取对局 %d 详情 (观战模式)", match.Id)

	return &types.GetPublicMatchDetailResp{
		Success: true,
		Match: &types.PublicMatchDetailData{
			Id:                       match.Id,
			GameType:                 match.GameType,
			MatchMode:                model.NormalizeMatchMode(match.MatchMode),
			Visibility:               model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
			FinishState:              model.NormalizeFinishState(match.FinishState),
			Status:                   match.Status,
			ServerRevision:           match.SyncRevision,
			Player1Id:                match.UserId,
			Player1Name:              player1Name,
			Player1Avatar:            player1Avatar,
			Player2Id:                player2Id,
			Player2Name:              player2Name,
			Player2Avatar:            player2Avatar,
			Player1Score:             match.MyScore,
			Player2Score:             match.OpponentScore,
			CurrentFramePlayer1Score: match.CurrentFrameMyScore,
			CurrentFramePlayer2Score: match.CurrentFrameOpponentScore,
			CurrentFrameStarted:      match.CurrentFrameStarted,
			CurrentRound:             currentRound,
			TotalRounds:              totalRounds,
			DurationSeconds:          durationSeconds,
			Rounds:                   rounds,
			CreatedAt:                match.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}
