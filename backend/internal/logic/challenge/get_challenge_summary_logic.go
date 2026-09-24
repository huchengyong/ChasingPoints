package challenge

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChallengeSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 当前约球摘要
func NewGetChallengeSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChallengeSummaryLogic {
	return &GetChallengeSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetChallengeSummaryLogic) GetChallengeSummary() (resp *types.GetChallengeSummaryResp, err error) {
	resp = &types.GetChallengeSummaryResp{ServerTime: logicx.FormatUTC8Time(logicx.NowUTC8())}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return resp, nil
	}
	now := time.Now()
	summary, err := l.svcCtx.ChallengeModel.FindCurrentSummaryByUser(userId, now)
	if err != nil {
		l.Logger.Errorf("查询当前约球失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	if summary != nil {
		matchId, err := l.svcCtx.MatchModel.FindIdByChallengeId(summary.Id)
		if err != nil {
			l.Logger.Errorf("查询约球比赛失败: id=%d err=%v", summary.Id, err)
			return resp, nil
		}
		resp.CurrentChallenge = &types.ChallengeInfo{}
		*resp.CurrentChallenge = buildChallengeInfoFromRecord(summary, now)
		if summary.FromUserId == userId || summary.ToUserId == userId {
			fromProfile, toProfile := loadChallengeProfiles(l.svcCtx, summary.FromUserId, summary.ToUserId)
			resp.CurrentChallenge.FromNickname = fromProfile.nickname
			resp.CurrentChallenge.FromAvatar = fromProfile.avatar
			resp.CurrentChallenge.ToNickname = toProfile.nickname
			resp.CurrentChallenge.ToAvatar = toProfile.avatar
		}
		resp.CurrentChallenge.MatchId = matchId
	}
	count, err := l.svcCtx.ChallengeModel.CountReceivedPending(userId, now)
	if err != nil {
		l.Logger.Errorf("查询收到邀请数失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	resp.ReceivedPendingCount = int(count)
	resp.Success = true
	return resp, nil
}
