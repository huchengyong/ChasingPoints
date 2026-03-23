package logic

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOngoingMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取正在进行的对局列表
func NewGetOngoingMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOngoingMatchesLogic {
	return &GetOngoingMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOngoingMatchesLogic) GetOngoingMatches(req *types.GetOngoingMatchesReq) (resp *types.GetOngoingMatchesResp, err error) {
	matchModel := model.NewMatchModel(l.svcCtx.DB)

	offset := (req.Page - 1) * req.PageSize
	matches, total, err := matchModel.ListOngoingMatches(offset, req.PageSize)
	if err != nil {
		l.Errorf("获取正在进行的对局列表失败: %v", err)
		return &types.GetOngoingMatchesResp{
			Success: false,
			Total:   0,
			List:    []types.OngoingMatchItem{},
		}, nil
	}

	// 构建响应列表
	list := make([]types.OngoingMatchItem, 0, len(matches))
	for _, m := range matches {
		// 获取当前局数
		roundCount, _ := matchModel.GetRoundCount(m.Id)

		// 获取对手ID
		var player2Id int64
		if m.OpponentId != nil {
			player2Id = *m.OpponentId
		}

		// 计算对局已持续的秒数（服务端计算，避免客户端时间不同步问题）
		durationSeconds := int64(time.Since(m.MatchTime).Seconds())
		if durationSeconds < 0 {
			durationSeconds = 0
		}

		list = append(list, types.OngoingMatchItem{
			Id:              m.Id,
			GameType:        m.GameType,
			GameTypeName:    GetGameTypeName(m.GameType),
			Player1Id:       m.UserId,
			Player1Name:     m.Player1Name,
			Player1Avatar:   m.Player1Avatar,
			Player2Id:       player2Id,
			Player2Name:     m.OpponentName,
			Player2Avatar:   m.Player2Avatar,
			Player1Score:    m.MyScore,
			Player2Score:    m.OpponentScore,
			CurrentRound:    int(roundCount) + 1,
			MatchTime:       m.MatchTime.Format("2006-01-02T15:04:05+08:00"),
			DurationSeconds: durationSeconds,
		})
	}

	return &types.GetOngoingMatchesResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}
