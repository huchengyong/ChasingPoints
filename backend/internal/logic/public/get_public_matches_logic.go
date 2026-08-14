package public

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublicMatchesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取公开观赛对局列表
func NewGetPublicMatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublicMatchesLogic {
	return &GetPublicMatchesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetPublicMatchesLogic) GetPublicMatches(req *types.PublicMatchListReq) (resp *types.PublicMatchListResp, err error) {
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope != "friends" {
		scope = "hall"
	}

	viewerUserId := int64(0)
	if scope == "friends" {
		viewerUserId, _ = utils.GetOptionalUserIDFromCtx(l.ctx)
		if viewerUserId <= 0 {
			return &types.PublicMatchListResp{
				Success: false,
				Message: "请先登录后查看好友对局",
				Total:   0,
				List:    []types.PublicMatchListItem{},
			}, nil
		}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	matchModel := l.svcCtx.MatchModel
	if matchModel == nil {
		matchModel = model.NewMatchModel(l.svcCtx.DB)
	}
	rows, total, listErr := matchModel.ListPublicMatches(model.PublicMatchListOptions{
		Scope:        scope,
		ViewerUserId: viewerUserId,
		Status:       req.Status,
		GameType:     req.GameType,
		Offset:       (page - 1) * pageSize,
		Limit:        pageSize,
	})
	if listErr != nil {
		l.Errorf("获取公开观赛对局列表失败: %v", listErr)
		return &types.PublicMatchListResp{
			Success: false,
			Message: "获取对局列表失败",
			Total:   0,
			List:    []types.PublicMatchListItem{},
		}, nil
	}

	list := make([]types.PublicMatchListItem, 0, len(rows))
	for _, row := range rows {
		var player2Id int64
		if row.OpponentId != nil {
			player2Id = *row.OpponentId
		}

		result := 0
		if row.Result != nil {
			result = *row.Result
		}

		endTime := ""
		if row.EndTime != nil {
			endTime = row.EndTime.Format("2006-01-02T15:04:05+08:00")
		}

		list = append(list, types.PublicMatchListItem{
			Id:              row.Id,
			GameType:        row.GameType,
			GameTypeName:    GetGameTypeName(row.GameType),
			MatchMode:       model.NormalizeMatchMode(row.MatchMode),
			Visibility:      model.NormalizeMatchVisibility(row.Visibility, row.MatchMode),
			FinishState:     model.NormalizeFinishState(row.FinishState),
			Status:          row.Status,
			StatusText:      getPublicMatchStatusText(row.Status),
			Player1Id:       row.UserId,
			Player1Name:     row.Player1Name,
			Player1Avatar:   row.Player1Avatar,
			Player2Id:       player2Id,
			Player2Name:     row.Player2Name,
			Player2Avatar:   row.Player2Avatar,
			Player1Score:    row.MyScore,
			Player2Score:    row.OpponentScore,
			CurrentRound:    resolvePublicMatchCurrentRound(row.Status, row.RoundCount),
			Result:          result,
			MatchTime:       row.MatchTime.Format("2006-01-02T15:04:05+08:00"),
			EndTime:         endTime,
			DurationSeconds: resolvePublicMatchDurationSeconds(row.MatchTime, row.EndTime),
		})
	}

	return &types.PublicMatchListResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}

func getPublicMatchStatusText(status int) string {
	if status == 2 {
		return "已结束"
	}
	return "进行中"
}

func resolvePublicMatchCurrentRound(status int, roundCount int64) int {
	if status == 2 {
		return int(roundCount)
	}
	return int(roundCount) + 1
}

func resolvePublicMatchDurationSeconds(matchTime time.Time, endTime *time.Time) int64 {
	end := time.Now()
	if endTime != nil {
		end = *endTime
	}

	durationSeconds := int64(end.Sub(matchTime).Seconds())
	if durationSeconds < 0 {
		return 0
	}
	return durationSeconds
}
