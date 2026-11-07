package member

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberMatchReplayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	now    func() time.Time
}

// 获取会员单场复盘
func NewGetMemberMatchReplayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberMatchReplayLogic {
	return &GetMemberMatchReplayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
		now:    time.Now,
	}
}

// GetMemberMatchReplay 只服务已完成对局的双方参与者。以下情况统一返回 Success=false 且不带任何对局信息，
// 与无权访问私有对局时的拒绝语义一致，不泄露对局是否存在以外的信息：
// 对局不存在、尚未完成（进行中或已取消）、请求者不是双方参与者（包括裁判）。
// 参与者不是有效会员时返回锁定状态和局数，不返回任何复盘数据。
func (l *GetMemberMatchReplayLogic) GetMemberMatchReplay(req *types.GetMemberMatchReplayReq) (resp *types.GetMemberMatchReplayResp, err error) {
	resp = &types.GetMemberMatchReplayResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return resp, nil
	}
	if req == nil || req.MatchId <= 0 {
		return resp, nil
	}
	if l.svcCtx == nil || l.svcCtx.MatchModel == nil || l.svcCtx.UserModel == nil {
		l.Logger.Errorf("单场复盘依赖未就绪: userId=%d", userId)
		return resp, nil
	}

	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil || match.Status != 2 {
		return resp, nil
	}
	viewerActor := 0
	switch {
	case match.UserId == userId:
		viewerActor = 1
	case match.OpponentId != nil && *match.OpponentId == userId:
		viewerActor = 2
	}
	if viewerActor == 0 {
		return resp, nil
	}

	// 会员资格只在服务端按请求时刻判断，不信任客户端。
	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil || user == nil {
		l.Logger.Errorf("查询用户失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	if !logicx.MemberActiveAt(user.MemberExpiresAt, l.now()) {
		roundCount, countErr := l.svcCtx.MatchModel.GetRoundCount(match.Id)
		if countErr != nil {
			l.Logger.Errorf("查询局数失败: matchId=%d err=%v", match.Id, countErr)
			return resp, nil
		}
		resp.Success, resp.Locked = true, true
		resp.MatchId, resp.GameType, resp.FrameCount = match.Id, match.GameType, int(roundCount)
		return resp, nil
	}

	// 只按对局 id 读取这一场的局记录和动作；动作数超过上限时只返回局级数据。
	rounds, err := l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
	if err != nil {
		l.Logger.Errorf("读取复盘局记录失败: matchId=%d err=%v", match.Id, err)
		return resp, nil
	}
	actionCount, err := l.svcCtx.MatchModel.CountActionsWithTx(nil, match.Id)
	if err != nil {
		l.Logger.Errorf("统计复盘动作数失败: matchId=%d err=%v", match.Id, err)
		return resp, nil
	}
	input := logicx.MatchReplayInput{Match: match, Rounds: rounds, ViewerActor: viewerActor}
	if actionCount > logicx.MaxReplayActions {
		input.Truncated = true
	} else if input.Actions, err = l.svcCtx.MatchModel.ListActiveActions(match.Id); err != nil {
		l.Logger.Errorf("读取复盘动作失败: matchId=%d err=%v", match.Id, err)
		return resp, nil
	}

	resp = logicx.BuildMatchReplay(input)
	resp.Success = true
	return resp, nil
}
