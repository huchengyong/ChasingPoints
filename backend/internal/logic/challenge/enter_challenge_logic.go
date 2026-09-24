package challenge

import (
	"context"
	"errors"
	"time"

	logicx "chasing_points/internal/logic"
	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	enterActionWaiting      = "waiting"
	enterActionMatchCreated = "match_created"
	enterActionConfirmEarly = "confirm_early"
	enterActionSelfOngoing  = "self_ongoing"
)

// errChallengeOpponentUnavailable 对手账号已禁用或注销，不能创建比赛。
var errChallengeOpponentUnavailable = errors.New("对方账号暂不可用")

type EnterChallengeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 进入约球对局：首人仅等待；第二人进入原子创建唯一比赛。
func NewEnterChallengeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnterChallengeLogic {
	return &EnterChallengeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *EnterChallengeLogic) EnterChallenge(req *types.EnterChallengeReq) (resp *types.EnterChallengeResp, err error) {
	resp = &types.EnterChallengeResp{}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		resp.Message = "获取用户信息失败"
		return resp, nil
	}
	if req.ChallengeId <= 0 {
		resp.Message = "约球不存在"
		return resp, nil
	}
	// 与扫码开局一致的开局限制：信誉禁赛不能绕过。
	if blocked, message, err := checkStartReputationBlock(l.svcCtx, userId); err != nil {
		l.Logger.Errorf("检查信誉状态失败: userId=%d err=%v", userId, err)
	} else if blocked {
		resp.Message = message
		return resp, nil
	}

	main, err := l.svcCtx.ChallengeModel.FindMainById(req.ChallengeId)
	if err != nil {
		l.Logger.Errorf("查询约球失败: id=%d err=%v", req.ChallengeId, err)
		resp.Message = "查询约球失败"
		return resp, nil
	}
	if main == nil {
		resp.Message = "约球不存在"
		return resp, nil
	}
	resp.ChallengeId = main.Id
	var (
		created   *model.Match
		otherUser int64
	)
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定双方用户行，与扫码开局共用同一冲突边界。
		otherIdForLock := main.FromUserId
		if otherIdForLock == userId {
			otherIdForLock = main.ToUserId
		}
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, buildLockUserIds(userId, otherIdForLock)); err != nil {
			return err
		}
		challenge, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, main.Id)
		if err != nil {
			return err
		}
		// 用户与约球行都已锁定；排队期间跨过失效时间的请求不能进入。
		now := time.Now()
		if challenge == nil {
			resp.Message = "约球不存在"
			return nil
		}
		if challenge.FromUserId != userId && challenge.ToUserId != userId {
			// 非参与者不得探测约球或其关联比赛。
			resp.Message = "只能处理自己的约球"
			return nil
		}
		resp.ChallengeId = challenge.Id
		// 已开局：幂等恢复该约球绑定的比赛。
		if challenge.Status == model.ChallengeStatusStarted {
			matchId, err := l.svcCtx.MatchModel.FindIdByChallengeIdWithTx(tx, challenge.Id)
			if err != nil {
				return err
			}
			if matchId > 0 {
				created = &model.Match{Id: matchId}
				resp.Success = true
				resp.Action = enterActionMatchCreated
				return nil
			}
			resp.Message = "约球已开局但比赛不存在"
			return nil
		}
		if challenge.Status != model.ChallengeStatusAccepted {
			resp.Message = "约球尚未接受或已结束"
			return nil
		}
		if challenge.ScheduledDate == nil || challenge.StartHour == nil {
			resp.Message = "约球信息不完整"
			return nil
		}
		if !challenge.ExpiresAt.After(now) {
			resp.Message = "约球已失效"
			return nil
		}

		otherUser = challenge.FromUserId
		if otherUser == userId {
			otherUser = challenge.ToUserId
		}

		if challenge.WaitingUserId != nil && *challenge.WaitingUserId == userId {
			// 已在等待：返回等待态，不重复提交。
			resp.Success = true
			resp.Action = enterActionWaiting
			return nil
		}

		if challenge.WaitingUserId != nil && *challenge.WaitingUserId == otherUser {
			// 第二人进入：双方用户行已锁，检查双方无冲突正式比赛。
			selfCurrent, err := l.svcCtx.MatchModel.FindCurrentByUserIdWithTx(tx, userId)
			if err != nil {
				return err
			}
			if selfCurrent != nil {
				resp.Success = true
				resp.Action = enterActionSelfOngoing
				resp.MatchId = selfCurrent.Id
				resp.Message = "你还有未结束的对局"
				return nil
			}
			otherCurrent, err := l.svcCtx.MatchModel.FindCurrentByUserIdWithTx(tx, otherUser)
			if err != nil {
				return err
			}
			if otherCurrent != nil {
				resp.Message = "对方还有未结束的对局，请稍后再试"
				return nil
			}
			match, err := createChallengeMatchWithTx(tx, l.svcCtx, userId, challenge, otherUser)
			if err != nil {
				return err
			}
			ok, err := l.svcCtx.ChallengeModel.UpdateStatusWithTx(tx, challenge.Id, []int{model.ChallengeStatusAccepted}, model.ChallengeStatusStarted, nil)
			if err != nil {
				return err
			}
			if !ok {
				resp.Message = "约球状态已变化，请稍后重试"
				return nil
			}
			created = match
			resp.Success = true
			resp.Action = enterActionMatchCreated
			return nil
		}

		// 无人等待：本人已有正式比赛时不得再写入等待，返回继续当前比赛入口。
		selfCurrent, err := l.svcCtx.MatchModel.FindCurrentByUserIdWithTx(tx, userId)
		if err != nil {
			return err
		}
		if selfCurrent != nil {
			resp.Success = true
			resp.Action = enterActionSelfOngoing
			resp.MatchId = selfCurrent.Id
			resp.Message = "你还有未结束的对局"
			return nil
		}

		// 无人等待：提前且未确认则要求一次确认，不写等待。
		scheduledStart := ChallengeScheduledStartTime(*challenge.ScheduledDate, *challenge.StartHour)
		if scheduledStart.After(now) && !req.ConfirmEarly {
			resp.Action = enterActionConfirmEarly
			resp.Message = "尚未到预约开始时间，确认提前进入？"
			return nil
		}
		ok, err := l.svcCtx.ChallengeModel.SetWaitingWithTx(tx, challenge.Id, userId, now)
		if err != nil {
			return err
		}
		if !ok {
			// 极小概率竞态：重查。
			fresh, err := l.svcCtx.ChallengeModel.FindByIdForUpdateWithTx(tx, challenge.Id)
			if err != nil {
				return err
			}
			if fresh != nil && fresh.WaitingUserId != nil && *fresh.WaitingUserId == userId {
				resp.Success = true
				resp.Action = enterActionWaiting
				return nil
			}
			resp.Message = "约球状态已变化，请稍后重试"
			return nil
		}
		resp.Success = true
		resp.Action = enterActionWaiting
		return nil
	})
	if err != nil {
		l.Logger.Errorf("进入约球失败: id=%d userId=%d err=%v", req.ChallengeId, userId, err)
		// 事务已回滚：不能把事务内写入的成功结果返回给客户端。
		resp.Success = false
		resp.Action = ""
		resp.MatchId = 0
		created = nil
		if errors.Is(err, errChallengeOpponentUnavailable) {
			resp.Message = "对方账号暂不可用，无法开局"
			return resp, nil
		}
		if resp.Message == "" {
			resp.Message = "操作失败"
		}
		return resp, nil
	}

	if resp.Action == enterActionSelfOngoing {
		info, infoErr := matchlogic.NewGetCurrentMatchLogic(l.ctx, l.svcCtx).GetCurrentMatch()
		if infoErr == nil && info != nil {
			resp.OngoingMatch = info.Match
		}
		return resp, nil
	}
	if created != nil && resp.Action == enterActionMatchCreated {
		l.Logger.Infof("用户 %d 通过约球 %d 获取对局 %d", userId, resp.ChallengeId, created.Id)
		if resp.MatchId <= 0 {
			resp.MatchId = created.Id
		}
		info, infoErr := matchlogic.NewGetCurrentMatchLogic(l.ctx, l.svcCtx).GetCurrentMatch()
		if infoErr == nil && info != nil {
			resp.OngoingMatch = info.Match
		}
		// 仅真正创建比赛的这次进入需要通知对手；恢复路径 otherUser 为 0。
		if otherUser > 0 && created.UserId == userId {
			logicx.SendUserDataUpdated(otherUser, logicx.UserDataUpdatedEvent{Scopes: []string{"challenge"}})
			if ws.GlobalHub != nil {
				ws.GlobalHub.SendToUser(otherUser, &ws.Message{
					Type: "match_start",
					Data: map[string]interface{}{
						"match_id":    created.Id,
						"game_type":   created.GameType,
						"opponent_id": userId,
					},
				})
			}
		}
		return resp, nil
	}
	if resp.Success && resp.Action == enterActionWaiting {
		l.Logger.Infof("用户 %d 进入约球 %d 等待", userId, resp.ChallengeId)
		if otherUser > 0 {
			logicx.SendUserDataUpdated(otherUser, logicx.UserDataUpdatedEvent{Scopes: []string{"challenge"}})
		}
	}
	return resp, nil
}

// buildLockUserIds 按 ID 升序去重，避免交叉锁顺序。
func buildLockUserIds(ids ...int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	ordered := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && ordered[j] < ordered[j-1]; j-- {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
		}
	}
	return ordered
}

// checkStartReputationBlock 与扫码开局共用同一信誉禁赛限制。
func checkStartReputationBlock(svcCtx *svc.ServiceContext, userId int64) (bool, string, error) {
	if svcCtx.UserReputationProfileModel == nil || svcCtx.ReputationConfigModel == nil {
		return false, "", nil
	}
	profile, err := logicx.NewReputationService(svcCtx, logicx.NowUTC8).GetOrCreateProfile(userId)
	if err != nil {
		return false, "", err
	}
	if profile == nil || profile.BanUntil == nil {
		return false, "", nil
	}
	if logicx.InUTC8(*profile.BanUntil).After(logicx.InUTC8(logicx.NowUTC8())) {
		return true, "当前处于禁赛状态，暂时无法开始对局", nil
	}
	return false, "", nil
}

// createChallengeMatchWithTx 用邀请保存的选项创建比赛；排位无需双方确认。
func createChallengeMatchWithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, userId int64, challenge *model.Challenge, opponentId int64) (*model.Match, error) {
	opponent, err := svcCtx.UserModel.FindByIdWithTx(tx, opponentId)
	if err != nil {
		return nil, err
	}
	if opponent == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if opponent.Status != 1 {
		return nil, errChallengeOpponentUnavailable
	}
	opponentIdPtr := opponentId
	match := &model.Match{
		UserId:              userId,
		OpponentId:          &opponentIdPtr,
		OpponentName:        opponent.Nickname,
		GameType:            challenge.GameType,
		SnookerRulesVersion: challenge.SnookerRulesVersion,
		BestOfFrames:        challenge.BestOfFrames,
		SnookerFormat:       challenge.SnookerFormat,
		SnookerTargetWins:   challenge.SnookerTargetWins,
		MatchFormat:         challenge.MatchFormat,
		TargetWins:          challenge.TargetWins,
		StartingActor:       challenge.StartingActor,
		MatchMode:           challenge.MatchMode,
		Visibility:          challenge.Visibility,
		// 约球比赛：任一参赛方单方结束，无需对方确认。
		FinishConfirmationRequired: false,
		FinishState:                model.FinishStateNone,
		CurrentFrameStarted:        true,
		Status:                     1,
		MatchTime:                  time.Now(),
		ChallengeId:                &challenge.Id,
	}
	if match.SnookerRulesVersion == 0 {
		match.SnookerRulesVersion = model.SnookerRulesVersionLegacy
	}
	if match.SnookerFormat == "" {
		match.SnookerFormat = model.SnookerFormatLegacy
	}
	if match.MatchFormat == "" {
		match.MatchFormat = "legacy"
	}
	if err := svcCtx.MatchModel.CreateWithTx(tx, match); err != nil {
		return nil, err
	}
	if _, err := svcCtx.MatchModel.FindOrCreateOpponentWithTx(tx, userId, opponent.Nickname, opponent.Avatar); err != nil {
		// 对手记录仅用于展示，失败不阻塞开局。
		logx.WithContext(context.Background()).Errorf("创建对手记录失败: %v", err)
	}
	return match, nil
}
