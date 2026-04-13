package match

import (
	"context"
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	startMatchActionCreated        = "created"
	startMatchActionResumeExisting = "resume_existing"
	startMatchActionBlocked        = "blocked"

	startMatchBlockReasonSelfOngoing     = "self_ongoing"
	startMatchBlockReasonOpponentOngoing = "opponent_ongoing"
)

type startMatchDecision struct {
	Action      string
	BlockReason string
	Message     string
	Match       *model.Match
}

type StartMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 开始对局
func NewStartMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartMatchLogic {
	return &StartMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartMatchLogic) StartMatch(req *types.StartMatchReq) (resp *types.StartMatchResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.StartMatchResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if message := validateStartMatchReq(userId, req); message != "" {
		return &types.StartMatchResp{Success: false, Message: message}, nil
	}
	opponentUser, err := l.svcCtx.UserModel.FindById(req.OpponentId)
	if err != nil {
		l.Logger.Errorf("查询对手信息失败: opponentId=%d, err=%v", req.OpponentId, err)
		return &types.StartMatchResp{Success: false, Message: "查询对手信息失败"}, nil
	}
	if opponentUser == nil {
		return &types.StartMatchResp{Success: false, Message: "请选择有效的平台对手"}, nil
	}

	var (
		decision     startMatchDecision
		createdMatch *model.Match
	)

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		lockUserIDs := buildStartMatchLockUserIDs(userId, req.OpponentId)
		if err := l.svcCtx.UserModel.LockUsersForUpdate(tx, lockUserIDs); err != nil {
			return err
		}

		existing, findErr := l.svcCtx.MatchModel.FindCurrentByUserIdWithTx(tx, userId)
		if findErr != nil {
			return findErr
		}

		var opponentCurrent *model.Match
		if req.OpponentId > 0 {
			opponentCurrent, findErr = l.svcCtx.MatchModel.FindCurrentByUserIdWithTx(tx, req.OpponentId)
			if findErr != nil {
				return findErr
			}
		}

		decision = evaluateStartMatchDecision(userId, req, existing, opponentCurrent)
		if decision.Action != startMatchActionCreated {
			return nil
		}

		var opponentId *int64
		if req.OpponentId > 0 {
			opponentId = &req.OpponentId
		}

		match := &model.Match{
			UserId:                    userId,
			OpponentId:                opponentId,
			OpponentName:              req.OpponentName,
			GameType:                  req.GameType,
			GameMode:                  req.GameMode,
			CurrentFrameStarted:       true,
			CurrentFrameMyScore:       0,
			CurrentFrameOpponentScore: 0,
			Status:                    1, // 进行中
			MatchTime:                 time.Now(),
		}

		if err := l.svcCtx.MatchModel.CreateWithTx(tx, match); err != nil {
			return err
		}

		if _, err := l.svcCtx.MatchModel.FindOrCreateOpponentWithTx(tx, userId, req.OpponentName, req.OpponentAvatar); err != nil {
			l.Logger.Errorf("创建对手记录失败: %v", err)
		}

		createdMatch = match
		return nil
	})
	if err != nil {
		l.Logger.Errorf("开始对局事务失败: %v", err)
		return &types.StartMatchResp{Success: false, Message: "创建对局失败"}, nil
	}

	switch decision.Action {
	case startMatchActionResumeExisting:
		l.Logger.Infof("用户 %d 继续对局 %d", userId, decision.Match.Id)
		return &types.StartMatchResp{
			Success:      true,
			Action:       decision.Action,
			MatchId:      decision.Match.Id,
			OngoingMatch: buildCurrentMatchInfo(l.svcCtx, userId, decision.Match),
		}, nil
	case startMatchActionBlocked:
		resp := &types.StartMatchResp{
			Success:     true,
			Action:      decision.Action,
			BlockReason: decision.BlockReason,
			Message:     decision.Message,
		}
		if decision.BlockReason == startMatchBlockReasonSelfOngoing && decision.Match != nil {
			resp.MatchId = decision.Match.Id
			resp.OngoingMatch = buildCurrentMatchInfo(l.svcCtx, userId, decision.Match)
		}
		l.Logger.Infof("用户 %d 开始对局被拦截: action=%s blockReason=%s match=%v",
			userId, decision.Action, decision.BlockReason, decision.Match != nil)
		return resp, nil
	}

	if createdMatch == nil {
		return &types.StartMatchResp{Success: false, Message: "创建对局失败"}, nil
	}

	l.Logger.Infof("用户 %d 开始对局 %d，对手: %s", userId, createdMatch.Id, req.OpponentName)

	// 通知对手有新对局（如果对手是注册用户）
	if req.OpponentId > 0 {
		// 获取当前用户信息用于通知对手
		userInfo, _ := l.svcCtx.UserModel.FindById(userId)
		opponentName := ""
		opponentAvatar := ""
		if userInfo != nil {
			opponentName = userInfo.Nickname
			opponentAvatar = userInfo.Avatar
		}

		ws.GlobalHub.SendToUser(req.OpponentId, &ws.Message{
			Type: "match_start",
			Data: map[string]interface{}{
				"match_id":        createdMatch.Id,
				"game_type":       req.GameType,
				"opponent_id":     userId,
				"opponent_name":   opponentName,
				"opponent_avatar": opponentAvatar,
			},
		})
		l.Logger.Infof("已向对手 %d 发送对局开始通知", req.OpponentId)
	}

	return &types.StartMatchResp{
		Success: true,
		Action:  startMatchActionCreated,
		MatchId: createdMatch.Id,
	}, nil
}

func evaluateStartMatchDecision(
	userId int64,
	req *types.StartMatchReq,
	currentMatch *model.Match,
	opponentCurrent *model.Match,
) startMatchDecision {
	if currentMatch != nil {
		if shouldResumeExistingMatch(userId, req, currentMatch) {
			return startMatchDecision{
				Action: startMatchActionResumeExisting,
				Match:  currentMatch,
			}
		}

		return startMatchDecision{
			Action:      startMatchActionBlocked,
			BlockReason: startMatchBlockReasonSelfOngoing,
			Message:     buildSelfOngoingBlockMessage(currentMatch),
			Match:       currentMatch,
		}
	}

	if opponentCurrent != nil {
		return startMatchDecision{
			Action:      startMatchActionBlocked,
			BlockReason: startMatchBlockReasonOpponentOngoing,
			Message:     "对手还有未结束的对局，暂时无法开始新的 PK",
			Match:       opponentCurrent,
		}
	}

	return startMatchDecision{
		Action: startMatchActionCreated,
	}
}

func validateStartMatchReq(userId int64, req *types.StartMatchReq) string {
	if req == nil || req.OpponentId <= 0 {
		return "请选择有效的平台对手"
	}
	if req.OpponentId == userId {
		return "不能和自己发起 PK"
	}
	return ""
}

func shouldResumeExistingMatch(userId int64, req *types.StartMatchReq, match *model.Match) bool {
	if match == nil || req == nil || req.OpponentId <= 0 {
		return false
	}

	return resolveOpponentIDForUser(match, userId) == req.OpponentId && match.GameType == req.GameType
}

func resolveOpponentIDForUser(match *model.Match, userId int64) int64 {
	if match == nil {
		return 0
	}

	if match.UserId == userId {
		if match.OpponentId != nil {
			return *match.OpponentId
		}
		return 0
	}

	if match.OpponentId != nil && *match.OpponentId == userId {
		return match.UserId
	}

	return 0
}

func buildSelfOngoingBlockMessage(match *model.Match) string {
	gameTypeName := "当前"
	if match != nil {
		gameTypeName = GetGameTypeName(match.GameType)
	}

	return "你还有一场" + gameTypeName + "未结束，请先结束之前的对局"
}

func buildStartMatchLockUserIDs(userIDs ...int64) []int64 {
	seen := make(map[int64]struct{}, len(userIDs))
	result := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}
