package logic

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMySeasonRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的赛季记录
func NewGetMySeasonRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMySeasonRecordLogic {
	return &GetMySeasonRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMySeasonRecordLogic) GetMySeasonRecord(req *types.GetMySeasonRecordReq) (resp *types.GetMySeasonRecordResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMySeasonRecordResp{Success: false}, nil
	}

	seasonId := req.SeasonId
	var season *model.Season

	if seasonId == 0 {
		season, err = l.svcCtx.SeasonModel.FindCurrent()
		if err != nil {
			l.Logger.Errorf("查询当前赛季失败: err=%v", err)
			return &types.GetMySeasonRecordResp{Success: false}, nil
		}
	} else {
		season, err = l.svcCtx.SeasonModel.FindById(seasonId)
		if err != nil {
			l.Logger.Errorf("查询赛季失败: seasonId=%d err=%v", seasonId, err)
			return &types.GetMySeasonRecordResp{Success: false}, nil
		}
	}

	if season == nil {
		return &types.GetMySeasonRecordResp{Success: true, Record: nil}, nil
	}

	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	record, err := l.svcCtx.SeasonRecordModel.FindBySeasonAndUserAndGameType(season.Id, userIdInt, gameType)
	if err != nil {
		l.Logger.Errorf("查询赛季记录失败: seasonId=%d userId=%d err=%v", season.Id, userIdInt, err)
		return &types.GetMySeasonRecordResp{Success: false}, nil
	}
	if record == nil {
		return &types.GetMySeasonRecordResp{Success: true, Record: nil}, nil
	}

	return &types.GetMySeasonRecordResp{
		Success: true,
		Record:  buildSeasonRecordInfo(record, season.Name),
	}, nil
}

func buildSeasonRecordInfo(record *model.SeasonRecord, seasonName string) *types.SeasonRecordInfo {
	if record == nil {
		return nil
	}

	return &types.SeasonRecordInfo{
		SeasonId:       record.SeasonId,
		SeasonName:     seasonName,
		StartRankScore: record.StartRankScore,
		EndRankScore:   record.EndRankScore,
		PeakRankScore:  record.PeakRankScore,
		MatchesPlayed:  record.MatchesPlayed,
		Wins:           record.Wins,
		FinalRank:      record.FinalRank,
	}
}
