package admin

import (
	"sort"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const adminEventNewsTimeLayout = "2006-01-02 15:04:05"

func parseAdminEventNewsTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(adminEventNewsTimeLayout, trimmed)
	if err != nil {
		return nil, err
	}
	utcTime := parsed.UTC()
	return &utcTime, nil
}

func buildAdminEventNewsInfo(item model.EventNews, stages []model.EventNewsStage) types.EventNewsInfo {
	stageInfos := make([]types.EventNewsStageInfo, 0, len(stages))
	for _, stage := range stages {
		stageInfos = append(stageInfos, types.EventNewsStageInfo{
			Id:         stage.Id,
			EventId:    stage.EventId,
			StageName:  stage.StageName,
			StageOrder: stage.StageOrder,
			Status:     stage.Status,
			ResultText: stage.ResultText,
			CreatedAt:  stage.CreatedAt.Format(adminEventNewsTimeLayout),
			UpdatedAt:  stage.UpdatedAt.Format(adminEventNewsTimeLayout),
		})
		if stage.StartTime != nil {
			stageInfos[len(stageInfos)-1].StartTime = stage.StartTime.Format(adminEventNewsTimeLayout)
		}
		if stage.EndTime != nil {
			stageInfos[len(stageInfos)-1].EndTime = stage.EndTime.Format(adminEventNewsTimeLayout)
		}
		if stage.SortTime != nil {
			stageInfos[len(stageInfos)-1].SortTime = stage.SortTime.Format(adminEventNewsTimeLayout)
		}
	}

	info := types.EventNewsInfo{
		Id:               item.Id,
		Title:            item.Title,
		GameType:         item.GameType,
		SourceType:       item.SourceType,
		SourceName:       item.SourceName,
		SourceUrl:        item.SourceUrl,
		CoverImage:       item.CoverImage,
		Summary:          item.Summary,
		Content:          item.Content,
		Country:          item.Country,
		City:             item.City,
		Venue:            item.Venue,
		Status:           item.Status,
		Featured:         item.Featured,
		Published:        item.Published,
		StageCount:       len(stages),
		Stages:           stageInfos,
	}
	if item.StartTime != nil {
		info.StartTime = item.StartTime.Format(adminEventNewsTimeLayout)
	}
	if item.EndTime != nil {
		info.EndTime = item.EndTime.Format(adminEventNewsTimeLayout)
	}
	if item.SortTime != nil {
		info.SortTime = item.SortTime.Format(adminEventNewsTimeLayout)
	}
	if item.PublishedAt != nil {
		info.PublishedAt = item.PublishedAt.Format(adminEventNewsTimeLayout)
	}
	info.CreatedAt = item.CreatedAt.Format(adminEventNewsTimeLayout)
	info.UpdatedAt = item.UpdatedAt.Format(adminEventNewsTimeLayout)

	summaryStage := pickAdminSummaryStage(stages)
	if summaryStage != nil {
		info.CurrentStageText = summaryStage.StageName
		info.LatestResultText = summaryStage.ResultText
	}

	return info
}

func applyAdminEventNewsUpdate(item *model.EventNews, req *types.AdminEventNewsUpdateReq) error {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return err
	}
	sortTime, err := parseAdminEventNewsTime(req.SortTime)
	if err != nil {
		return err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return err
	}

	item.Title = strings.TrimSpace(req.Title)
	item.GameType = req.GameType
	item.SourceType = strings.TrimSpace(req.SourceType)
	item.SourceName = strings.TrimSpace(req.SourceName)
	item.SourceUrl = strings.TrimSpace(req.SourceUrl)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.Summary = strings.TrimSpace(req.Summary)
	item.Content = req.Content
	item.Country = strings.TrimSpace(req.Country)
	item.City = strings.TrimSpace(req.City)
	item.Venue = strings.TrimSpace(req.Venue)
	item.Status = req.Status
	item.StartTime = startTime
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}
	item.SortTime = sortTime
	item.EndTime = endTime
	item.Featured = req.Featured
	return nil
}

func buildAdminEventNewsStage(req *types.AdminEventNewsStageCreateReq) (*model.EventNewsStage, error) {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	sortTime, err := parseAdminEventNewsTime(req.SortTime)
	if err != nil {
		return nil, err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return nil, err
	}
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}

	return &model.EventNewsStage{
		EventId:    req.EventId,
		StageName:  strings.TrimSpace(req.StageName),
		StageOrder: req.StageOrder,
		StartTime:  startTime,
		EndTime:    endTime,
		Status:     req.Status,
		ResultText: strings.TrimSpace(req.ResultText),
		SortTime:   sortTime,
	}, nil
}

func validateAdminEventNewsStageTimeRange(startTime, endTime *time.Time) string {
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return "结束时间不能早于开始时间"
	}
	return ""
}

func applyAdminEventNewsStageUpdate(item *model.EventNewsStage, req *types.AdminEventNewsStageUpdateReq) error {
	startTime, err := parseAdminEventNewsTime(req.StartTime)
	if err != nil {
		return err
	}
	sortTime, err := parseAdminEventNewsTime(req.SortTime)
	if err != nil {
		return err
	}
	endTime, err := parseAdminEventNewsTime(req.EndTime)
	if err != nil {
		return err
	}
	if sortTime == nil && startTime != nil {
		sortTime = startTime
	}

	item.EventId = req.EventId
	item.StageName = strings.TrimSpace(req.StageName)
	item.StageOrder = req.StageOrder
	item.StartTime = startTime
	item.EndTime = endTime
	item.Status = req.Status
	item.ResultText = strings.TrimSpace(req.ResultText)
	item.SortTime = sortTime
	return nil
}

func validateAdminEventNewsReq(title string, gameType, status int, startTime, sortTime *time.Time) string {
	if title == "" {
		return "请输入标题"
	}
	if gameType <= 0 {
		return "请选择球种"
	}
	if status < model.EventNewsStatusUpcoming || status > model.EventNewsStatusCanceled {
		return "请选择正确的状态"
	}
	if startTime == nil && sortTime == nil {
		return "开始时间和排序时间至少填写一个"
	}
	return ""
}

func validateAdminEventNewsStageReq(eventID int64, stageName string, stageOrder, status int) string {
	if eventID <= 0 {
		return "赛事不存在"
	}
	if stageName == "" {
		return "请输入阶段名称"
	}
	if stageOrder <= 0 {
		return "请输入正确的阶段排序"
	}
	if status < model.EventNewsStatusUpcoming || status > model.EventNewsStatusCanceled {
		return "请选择正确的阶段状态"
	}
	return ""
}

func pickAdminSummaryStage(stages []model.EventNewsStage) *model.EventNewsStage {
	if len(stages) == 0 {
		return nil
	}

	candidates := filterAdminStagesByStatus(stages, model.EventNewsStatusLive)
	if len(candidates) == 0 {
		candidates = filterAdminStagesByStatus(stages, model.EventNewsStatusUpcoming)
	}
	if len(candidates) == 0 {
		candidates = filterAdminStagesByStatus(stages, model.EventNewsStatusFinished)
	}
	if len(candidates) == 0 {
		candidates = append(candidates, stages...)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Status == model.EventNewsStatusUpcoming && candidates[j].Status == model.EventNewsStatusUpcoming {
			if candidates[i].StageOrder != candidates[j].StageOrder {
				return candidates[i].StageOrder < candidates[j].StageOrder
			}
			return candidates[i].Id < candidates[j].Id
		}
		if candidates[i].StageOrder != candidates[j].StageOrder {
			return candidates[i].StageOrder > candidates[j].StageOrder
		}
		return candidates[i].Id > candidates[j].Id
	})

	chosen := candidates[0]
	return &chosen
}

func filterAdminStagesByStatus(stages []model.EventNewsStage, status int) []model.EventNewsStage {
	filtered := make([]model.EventNewsStage, 0, len(stages))
	for _, stage := range stages {
		if stage.Status == status {
			filtered = append(filtered, stage)
		}
	}
	return filtered
}
