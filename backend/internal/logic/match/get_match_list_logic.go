package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局列表
func NewGetMatchListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchListLogic {
	return &GetMatchListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMatchListLogic) GetMatchList(req *types.MatchListReq) (resp *types.MatchListResp, err error) {
	if req == nil {
		req = &types.MatchListReq{}
	}
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.MatchListResp{Success: false}, nil
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

	if l.svcCtx == nil || l.svcCtx.MatchModel == nil {
		return &types.MatchListResp{Success: false}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		return l.getLegacyMatchList(userId, req, offset, pageSize)
	}
	if l.svcCtx.DB == nil || l.svcCtx.CompetitiveReadModel == nil {
		return &types.MatchListResp{Success: false}, nil
	}
	matches, total, err := l.svcCtx.CompetitiveReadModel.ListParticipantMatchPageWithTx(l.svcCtx.DB.WithContext(l.ctx), userId, req.GameType, req.Result, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("查询对局列表失败: %v", err)
		return &types.MatchListResp{Success: false}, nil
	}

	list := make([]types.MatchListItem, 0, len(matches))
	for _, match := range matches {
		list = append(list, types.MatchListItem{
			Id:             match.MatchId,
			OpponentId:     match.OpponentUserId,
			GameType:       match.GameType,
			GameTypeName:   GetGameTypeName(match.GameType),
			MatchMode:      model.NormalizeMatchMode(match.MatchMode),
			Visibility:     model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
			FinishState:    model.NormalizeFinishState(match.FinishState),
			OpponentName:   match.OpponentName,
			OpponentAvatar: match.OpponentAvatar,
			MyScore:        match.MyScore,
			OpponentScore:  match.OpponentScore,
			Result:         match.Result,
			MatchTime:      match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		})
	}

	return &types.MatchListResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}

func (l *GetMatchListLogic) getLegacyMatchList(userID int64, req *types.MatchListReq, offset, pageSize int) (*types.MatchListResp, error) {
	matches, total, err := l.svcCtx.MatchModel.ListByUserId(userID, req.GameType, req.Result, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("查询历史对局列表失败: %v", err)
		return &types.MatchListResp{Success: false}, nil
	}
	list := make([]types.MatchListItem, 0, len(matches))
	for _, match := range matches {
		result := 0
		if match.Result != nil {
			result = *match.Result
		}
		opponentID := int64(0)
		if match.IsAsOpponent {
			opponentID = match.UserId
		} else if match.OpponentId != nil {
			opponentID = *match.OpponentId
		}
		list = append(list, types.MatchListItem{
			Id:             match.Id,
			OpponentId:     opponentID,
			GameType:       match.GameType,
			GameTypeName:   GetGameTypeName(match.GameType),
			MatchMode:      model.NormalizeMatchMode(match.MatchMode),
			Visibility:     model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
			FinishState:    model.NormalizeFinishState(match.FinishState),
			OpponentName:   match.OpponentName,
			OpponentAvatar: match.OpponentAvatar,
			MyScore:        match.MyScore,
			OpponentScore:  match.OpponentScore,
			Result:         result,
			MatchTime:      match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		})
	}
	return &types.MatchListResp{Success: true, Total: total, List: list}, nil
}
