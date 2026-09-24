package match

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CancelMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消对局
func NewCancelMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelMatchLogic {
	return &CancelMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CancelMatchLogic) CancelMatch(req *types.CancelMatchReq) (resp *types.CommonResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.CommonResp{Success: false, Message: "用户未登录"}, nil
	}

	// 事务内锁定比赛并重检状态，避免与结束结算竞争时用旧对象覆盖赛果。
	var cancelled bool
	var cancelledMatch *model.Match
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		match, err := l.svcCtx.MatchModel.FindByIdForUpdateWithTx(tx, req.MatchId)
		if err != nil {
			return err
		}
		if match == nil {
			resp = &types.CommonResp{Success: false, Message: "对局不存在"}
			return nil
		}
		if match.UserId != userId {
			resp = &types.CommonResp{Success: false, Message: "无权操作"}
			return nil
		}
		if match.Status != 1 {
			resp = &types.CommonResp{Success: false, Message: "对局已结束"}
			return nil
		}
		now := time.Now()
		match.Status = 3
		match.EndTime = &now
		match.CompletedByUserId = nil
		match.CompletionSource = model.CompletionSourceUnknown
		// 通过 revision 条件更新落库：已读旧快照的结束请求会因 revision 冲突失败，不会覆盖取消终态。
		if _, err := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match); err != nil {
			return err
		}
		// 关联约球同步结束（不当作赛果）。
		if match.ChallengeId != nil && *match.ChallengeId > 0 {
			reason := model.ChallengeCloseReasonMatchGone
			if match.Remark != "" {
				reason = match.Remark
			}
			if _, err := l.svcCtx.ChallengeModel.MarkMatchCancelledWithTx(tx, *match.ChallengeId, reason); err != nil {
				return err
			}
		}
		cancelled = true
		cancelledMatch = match
		return nil
	})
	if err != nil {
		l.Logger.Errorf("取消对局失败: matchId=%d err=%v", req.MatchId, err)
		return &types.CommonResp{Success: false, Message: "取消失败"}, nil
	}
	if !cancelled {
		if resp == nil {
			resp = &types.CommonResp{Success: false, Message: "取消失败"}
		}
		return resp, nil
	}

	l.Logger.Infof("用户 %d 取消对局 %d", userId, req.MatchId)
	// 取消已提交：失效双方的战绩读取；关联约球已一并结束，主 Tab 活动卡也要刷新。
	scopes := []string{"history", "h2h", "opponents"}
	if cancelledMatch.ChallengeId != nil && *cancelledMatch.ChallengeId > 0 {
		scopes = append(scopes, "challenge")
	}
	targets := []int64{cancelledMatch.UserId}
	if cancelledMatch.OpponentId != nil && *cancelledMatch.OpponentId > 0 && *cancelledMatch.OpponentId != cancelledMatch.UserId {
		targets = append(targets, *cancelledMatch.OpponentId)
	}
	for _, targetUserID := range targets {
		logicx.SendUserDataUpdated(targetUserID, logicx.UserDataUpdatedEvent{Scopes: scopes})
	}
	return &types.CommonResp{
		Success: true,
		Message: "对局已取消",
	}, nil
}
