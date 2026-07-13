package match

import (
	"context"

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
		svcCtx: svcCtx,
	}
}

func (l *GetMatchListLogic) GetMatchList(req *types.MatchListReq) (resp *types.MatchListResp, err error) {
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

	// 查询对局列表
	matches, total, err := l.svcCtx.MatchModel.ListByUserId(userId, req.GameType, req.Result, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("查询对局列表失败: %v", err)
		return &types.MatchListResp{Success: false}, nil
	}

	// 转换数据
	list := make([]types.MatchListItem, 0, len(matches))
	for _, match := range matches {
		result := 0
		if match.Result != nil {
			result = *match.Result
		}
		// 计算对手ID
		var opponentId int64
		if match.IsAsOpponent {
			// 用户是作为对手参与的，对手就是发起方（UserId）
			opponentId = match.UserId
		} else {
			// 用户是发起方，对手ID在 OpponentId 字段中
			if match.OpponentId != nil {
				opponentId = *match.OpponentId
			}
		}
		list = append(list, types.MatchListItem{
			Id:             match.Id,
			OpponentId:     opponentId,
			GameType:       match.GameType,
			GameTypeName:   GetGameTypeName(match.GameType),
			OpponentName:   match.OpponentName,
			OpponentAvatar: match.OpponentAvatar,
			MyScore:        match.MyScore,
			OpponentScore:  match.OpponentScore,
			Result:         result,
			MatchTime:      match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		})
	}

	return &types.MatchListResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}
