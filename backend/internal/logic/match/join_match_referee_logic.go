package match

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type JoinMatchRefereeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 扫码加入并担任本场裁判
func NewJoinMatchRefereeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMatchRefereeLogic {
	return &JoinMatchRefereeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JoinMatchRefereeLogic) JoinMatchReferee(req *types.JoinMatchRefereeReq) (resp *types.JoinMatchRefereeResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.JoinMatchRefereeResp{Success: false}, nil
	}
	if req.MatchId <= 0 || req.JoinToken == "" {
		return &types.JoinMatchRefereeResp{Success: false, Message: "无效的裁判二维码"}, nil
	}

	var match *model.Match
	now := time.Now()
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		lockedMatch, findErr := l.svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if findErr != nil {
			return findErr
		}
		if lockedMatch == nil || lockedMatch.Status != 1 {
			return errMatchRefereeJoinUnavailable
		}
		if model.NormalizeFinishState(lockedMatch.FinishState) == model.FinishStatePendingConfirmation {
			return errMatchRefereeFinishPending
		}
		if lockedMatch.RefereeUserId != nil && *lockedMatch.RefereeUserId > 0 {
			return errMatchRefereeAlreadyBound
		}
		if lockedMatch.UserId == userId || (lockedMatch.OpponentId != nil && *lockedMatch.OpponentId == userId) {
			return errMatchRefereeJoinUnavailable
		}

		token, redisErr := l.svcCtx.Redis.Get(l.ctx, buildMatchRefereeJoinTokenKey(req.MatchId)).Result()
		if redisErr != nil || token != req.JoinToken {
			return errMatchRefereeCodeInvalid
		}

		lockedMatch.RefereeUserId = &userId
		lockedMatch.RefereeJoinedAt = &now
		if err := l.svcCtx.MatchModel.UpdateWithTx(tx, lockedMatch); err != nil {
			return err
		}
		if err := l.svcCtx.Redis.Del(l.ctx, buildMatchRefereeJoinTokenKey(req.MatchId)).Err(); err != nil {
			return err
		}
		match = lockedMatch
		return nil
	})
	if err != nil {
		switch err {
		case errMatchRefereeAlreadyBound:
			return &types.JoinMatchRefereeResp{Success: false, Message: "本场已绑定裁判"}, nil
		case errMatchRefereeJoinUnavailable:
			return &types.JoinMatchRefereeResp{Success: false, Message: "当前对局无法加入裁判"}, nil
		case errMatchRefereeFinishPending:
			return &types.JoinMatchRefereeResp{Success: false, Message: "请先处理当前结束请求，再加入裁判"}, nil
		case errMatchRefereeCodeInvalid:
			return &types.JoinMatchRefereeResp{Success: false, Message: "裁判二维码已失效"}, nil
		default:
			l.Logger.Errorf("绑定裁判失败: matchId=%d userId=%d err=%v", req.MatchId, userId, err)
			return &types.JoinMatchRefereeResp{Success: false, Message: "加入裁判失败"}, nil
		}
	}

	if ws.GlobalHub != nil && match != nil {
		ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
			Type: "match_role_changed",
			Data: map[string]interface{}{
				"match_id": match.Id,
			},
		})
	}

	return &types.JoinMatchRefereeResp{
		Success: true,
		Message: "已成为本场裁判",
		MatchId: req.MatchId,
		Match:   buildCurrentMatchInfo(l.svcCtx, userId, match),
	}, nil
}
