package match

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRefereeHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRefereeHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRefereeHistoryLogic {
	return &GetRefereeHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetRefereeHistoryLogic) GetRefereeHistory(req *types.RefereeHistoryReq) (resp *types.RefereeHistoryResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.RefereeHistoryResp{Success: false, Total: 0, List: []types.RefereeHistoryItem{}}, nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	matches, total, err := l.svcCtx.MatchModel.ListByRefereeUserIdWithProfiles(userId, offset, pageSize)
	if err != nil {
		l.Errorf("查询裁判历史失败: userId=%d err=%v", userId, err)
		return &types.RefereeHistoryResp{Success: false, Total: 0, List: []types.RefereeHistoryItem{}}, nil
	}

	list := make([]types.RefereeHistoryItem, 0, len(matches))
	for _, row := range matches {
		m := row.Match
		player1Name := row.Player1Name
		if player1Name == "" {
			player1Name = "玩家1"
		}
		player1Avatar := row.Player1Avatar

		player2Id := int64(0)
		player2Name := "玩家2"
		player2Avatar := ""
		if m.OpponentId != nil {
			player2Id = *m.OpponentId
			if row.Player2Name != "" {
				player2Name = row.Player2Name
			}
			player2Avatar = row.Player2Avatar
		}

		refereeDuration := calculateRefereeDurationSeconds(m.RefereeJoinedAt, m.EndTime)

		refereeJoinedAt := ""
		if m.RefereeJoinedAt != nil {
			refereeJoinedAt = m.RefereeJoinedAt.Format("2006-01-02T15:04:05+08:00")
		}
		endTime := ""
		if m.EndTime != nil {
			endTime = m.EndTime.Format("2006-01-02T15:04:05+08:00")
		} else if m.Status == 3 {
			endTime = m.UpdatedAt.Format("2006-01-02T15:04:05+08:00")
		}

		statusText := "已完成"
		if m.Status == 3 {
			statusText = "已取消"
		}

		completedByUserId := resolveCompletedByUserId(&m)

		list = append(list, types.RefereeHistoryItem{
			Id:                     m.Id,
			Player1Id:              m.UserId,
			Player1Name:            player1Name,
			Player1Avatar:          player1Avatar,
			Player2Id:              player2Id,
			Player2Name:            player2Name,
			Player2Avatar:          player2Avatar,
			Player1Score:           m.MyScore,
			Player2Score:           m.OpponentScore,
			GameType:               m.GameType,
			GameTypeName:           GetGameTypeName(m.GameType),
			MatchMode:              model.NormalizeMatchMode(m.MatchMode),
			Visibility:             model.NormalizeMatchVisibility(m.Visibility, m.MatchMode),
			Status:                 m.Status,
			StatusText:             statusText,
			RefereeJoinedAt:        refereeJoinedAt,
			EndTime:                endTime,
			RefereeDurationSeconds: refereeDuration,
			CompletedByUserId:      completedByUserId,
			CompletionSource:       resolveCompletionSource(&m),
		})
	}

	return &types.RefereeHistoryResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}

func calculateRefereeDurationSeconds(joinedAt, endTime *time.Time) int64 {
	if joinedAt == nil {
		return 0
	}
	end := time.Now()
	if endTime != nil {
		end = *endTime
	}
	duration := int64(end.Sub(*joinedAt).Seconds())
	if duration < 0 {
		return 0
	}
	return duration
}
