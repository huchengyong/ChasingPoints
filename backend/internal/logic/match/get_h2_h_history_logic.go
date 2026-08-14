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

const h2hHistoryDateLayout = "2006-01-02"

type GetH2HHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取交锋历史
func NewGetH2HHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetH2HHistoryLogic {
	return &GetH2HHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetH2HHistoryLogic) GetH2HHistory(req *types.H2HHistoryReq) (resp *types.H2HHistoryResp, err error) {
	if req == nil {
		req = &types.H2HHistoryReq{}
	}
	// 获取用户ID
	viewerUserId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.H2HHistoryResp{Success: false, Message: "获取交锋历史失败"}, nil
	}
	if l.svcCtx == nil {
		return &types.H2HHistoryResp{Success: false, Message: "竞技读模型不可用"}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if l.svcCtx.MatchModel == nil {
			return &types.H2HHistoryResp{Success: false, Message: "竞技读模型不可用"}, nil
		}
		return l.getLegacyH2HHistory(viewerUserId, req)
	}

	target, targetErr := resolveH2HReadTarget(l.svcCtx, viewerUserId, req.TargetUserId, req.OpponentId, req.OpponentName)
	if targetErr != nil {
		return &types.H2HHistoryResp{Success: false, Message: targetErr.Error()}, nil
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.H2HHistoryResp{Success: false, Message: "竞技读模型不可用"}, nil
	}

	// 分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	startTime, endTime, dateErr := parseH2HHistoryDateRange(req.StartDate, req.EndDate)
	if dateErr != nil {
		return &types.H2HHistoryResp{Success: false, Message: "日期格式错误"}, nil
	}

	matches, total, queryErr := l.svcCtx.CompetitiveReadModel.ListParticipantH2HPage(
		target.subjectUserID,
		target.opponentUserID,
		target.opponentNameKey,
		0,
		req.Result,
		startTime,
		endTime,
		offset,
		pageSize,
	)
	if queryErr != nil {
		l.Logger.Errorf("查询交锋历史失败: %v", queryErr)
		return &types.H2HHistoryResp{Success: false, Message: "获取交锋历史失败"}, nil
	}

	list := h2hHistoryItems(matches)

	return &types.H2HHistoryResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}

func (l *GetH2HHistoryLogic) getLegacyH2HHistory(viewerUserID int64, req *types.H2HHistoryReq) (*types.H2HHistoryResp, error) {
	subjectUserID := viewerUserID
	if req.TargetUserId > 0 && req.TargetUserId != viewerUserID {
		areFriends, err := l.svcCtx.FriendModel.AreFriends(viewerUserID, req.TargetUserId)
		if err != nil {
			return &types.H2HHistoryResp{Success: false, Message: "获取对方战绩失败"}, nil
		}
		if !areFriends {
			return &types.H2HHistoryResp{Success: false, Message: "仅可查看好友的对方战绩"}, nil
		}
		subjectUserID = req.TargetUserId
	}
	if req.OpponentId <= 0 && req.OpponentName == "" {
		return &types.H2HHistoryResp{Success: false, Message: "缺少对手信息"}, nil
	}
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	start, end, err := parseH2HHistoryDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return &types.H2HHistoryResp{Success: false, Message: "日期格式错误"}, nil
	}
	offset := (page - 1) * pageSize
	var rows []model.H2HMatchRecord
	var total int64
	if req.OpponentId > 0 {
		rows, total, err = l.svcCtx.MatchModel.ListByOpponentId(subjectUserID, req.OpponentId, req.Result, offset, pageSize, start, end)
	} else {
		rows, total, err = l.svcCtx.MatchModel.ListByOpponentName(subjectUserID, req.OpponentName, req.Result, offset, pageSize, start, end)
	}
	if err != nil {
		l.Logger.Errorf("查询历史交锋失败: %v", err)
		return &types.H2HHistoryResp{Success: false, Message: "获取交锋历史失败"}, nil
	}
	list := make([]types.MatchListItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.MatchListItem{
			Id:            row.Id,
			GameType:      row.GameType,
			GameTypeName:  GetGameTypeName(row.GameType),
			OpponentName:  row.OpponentName,
			MyScore:       row.MyScore,
			OpponentScore: row.OpponentScore,
			Result:        row.Result,
			MatchTime:     row.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		})
	}
	return &types.H2HHistoryResp{Success: true, Total: total, List: list}, nil
}

func parseH2HHistoryDateRange(startDate, endDate string) (*time.Time, *time.Time, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}

	var startTime *time.Time
	if startDate != "" {
		parsed, parseErr := time.ParseInLocation(h2hHistoryDateLayout, startDate, location)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		startTime = &parsed
	}

	var endTime *time.Time
	if endDate != "" {
		parsed, parseErr := time.ParseInLocation(h2hHistoryDateLayout, endDate, location)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		exclusiveEnd := parsed.AddDate(0, 0, 1)
		endTime = &exclusiveEnd
	}

	return startTime, endTime, nil
}
