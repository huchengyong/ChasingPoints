package challenge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	logicx "chasing_points/internal/logic"
	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// challengeStatusText 状态文案由前端负责，这里仅透传数值。

func formatHourDate(t time.Time) string {
	return logicx.InUTC8(t).Format("2006-01-02")
}

func formatTimePtr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return logicx.InUTC8(t).Format("2006-01-02 15:04:05")
}

// buildChallengeInfo 组装 API 信息；matchIds 由调用方批量查询。
func buildChallengeInfo(row model.ChallengeListRow, matchID int64, now time.Time) types.ChallengeInfo {
	if row.LinkedMatchId > 0 {
		matchID = row.LinkedMatchId
	}
	info := types.ChallengeInfo{
		Id:           row.Id,
		MatchId:      matchID,
		FromUserId:   row.FromUserId,
		FromNickname: row.FromNickname,
		FromAvatar:   row.FromAvatar,
		ToUserId:     row.ToUserId,
		ToNickname:   row.ToNickname,
		ToAvatar:     row.ToAvatar,
		GameType:     row.GameType,
		MatchMode:    row.MatchMode,
		Message:      row.Message,
		Status:       row.Status,
		CloseReason:  row.CloseReason,
		CreatedAt:    formatTimePtr(row.CreatedAt),
		ExpiresAt:    formatTimePtr(row.ExpiresAt),
	}
	if row.WaitingUserId != nil {
		info.WaitingUserId = *row.WaitingUserId
	}
	if row.ScheduledDate != nil {
		info.ScheduledDate = formatHourDate(*row.ScheduledDate)
	}
	if row.StartHour != nil {
		info.StartHour = *row.StartHour
	}
	if row.EndHour != nil {
		info.EndHour = *row.EndHour
	}
	applyEffectiveChallengeState(&info, row.ExpiresAt, now)
	return info
}

// applyEffectiveChallengeState 只派生读结果，不在 GET 中写库或修改已开局比赛。
func applyEffectiveChallengeState(info *types.ChallengeInfo, expiresAt, now time.Time) {
	if (info.Status == model.ChallengeStatusPending || info.Status == model.ChallengeStatusAccepted) && !expiresAt.After(now) {
		info.Status = model.ChallengeStatusExpired
		info.CloseReason = model.ChallengeCloseReasonExpired
		info.WaitingUserId = 0
	}
}

// buildChallengeInfoFromRecord 由 Challenge 记录组装（发送/接受/摘要等写路径返回）。
func buildChallengeInfoFromRecord(challenge *model.Challenge, now time.Time) types.ChallengeInfo {
	if challenge == nil {
		return types.ChallengeInfo{}
	}
	info := types.ChallengeInfo{
		Id:          challenge.Id,
		FromUserId:  challenge.FromUserId,
		ToUserId:    challenge.ToUserId,
		GameType:    challenge.GameType,
		MatchMode:   challenge.MatchMode,
		Message:     challenge.Message,
		Status:      challenge.Status,
		CloseReason: challenge.CloseReason,
		ExpiresAt:   formatTimePtr(challenge.ExpiresAt),
	}
	if challenge.WaitingUserId != nil {
		info.WaitingUserId = *challenge.WaitingUserId
	}
	if challenge.ScheduledDate != nil {
		info.ScheduledDate = formatHourDate(*challenge.ScheduledDate)
	}
	if challenge.StartHour != nil {
		info.StartHour = *challenge.StartHour
	}
	if challenge.EndHour != nil {
		info.EndHour = *challenge.EndHour
	}
	applyEffectiveChallengeState(&info, challenge.ExpiresAt, now)
	return info
}

// challengeStartReq 约球只保存球种与比赛类型：赛制固定为发起页展示的自由局数（九球追分沿用其既有赛制），
// 公开范围随类型缺省。发送与第二人进入都走扫码开局的同一套规范化校验，邀请可开局且开局规则与扫码一致。
func challengeStartReq(userId, opponentId int64, gameType int, matchMode string) (*types.StartMatchReq, error) {
	if gameType < 1 || gameType > 4 {
		return nil, fmt.Errorf("请选择有效的球种")
	}
	req := &types.StartMatchReq{OpponentId: opponentId, GameType: gameType, MatchMode: matchMode}
	if gameType == 1 {
		req.SnookerRulesVersion = model.SnookerRulesVersionWPBSA
		req.SnookerFormat = model.SnookerFormatFree
	} else if model.IsPoolMatchFormatGameType(gameType) {
		req.MatchFormat = model.MatchFormatFree
	}
	if message := matchlogic.NormalizeStartMatchOptions(userId, req); message != "" {
		return nil, errors.New(message)
	}
	return req, nil
}

// canSendChallengeTo 服务端校验发送资格：好友或曾有真实比赛的平台对手；再按接收方好友限制判断。
// db 传入事务句柄时全部读取走同一连接。
func canSendChallengeTo(db *gorm.DB, svcCtx *svc.ServiceContext, senderId, recipientId int64) (bool, string, error) {
	recipient, err := svcCtx.UserModel.FindByIdWithTx(db, recipientId)
	if err != nil {
		return false, "", err
	}
	if recipient == nil || recipient.Status != 1 {
		return false, "请选择有效的平台对手", nil
	}
	areFriends, err := svcCtx.FriendModel.AreFriendsWithTx(db, senderId, recipientId)
	if err != nil {
		return false, "", err
	}
	if !areFriends {
		exists, err := svcCtx.MatchModel.ExistsBetweenUsersWithTx(db, senderId, recipientId)
		if err != nil {
			return false, "", err
		}
		if !exists {
			return false, "只能约好友或曾一起打过球的对手", nil
		}
	}
	if recipient.FriendsOnlyChallenges && !areFriends {
		return false, "对方仅允许好友约球", nil
	}
	return true, "", nil
}

type challengeProfileInfo struct {
	nickname string
	avatar   string
}

func loadChallengeProfiles(svcCtx *svc.ServiceContext, userIds ...int64) (from challengeProfileInfo, to challengeProfileInfo) {
	if len(userIds) == 0 {
		return
	}
	fromID, toID := userIds[0], userIds[0]
	if len(userIds) > 1 {
		toID = userIds[1]
	}
	users, err := svcCtx.UserModel.FindByIds([]int64{fromID, toID})
	if err != nil {
		return
	}
	if user, ok := users[fromID]; ok {
		from = challengeProfileInfo{nickname: user.Nickname, avatar: user.Avatar}
	}
	if toID != fromID {
		if user, ok := users[toID]; ok {
			to = challengeProfileInfo{nickname: user.Nickname, avatar: user.Avatar}
		}
	}
	return
}

// buildChallengeInfoList 批量组装列表（联表已含比赛 ID）。
func buildChallengeInfoList(rows []model.ChallengeListRow, now time.Time) []types.ChallengeInfo {
	list := make([]types.ChallengeInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, buildChallengeInfo(row, 0, now))
	}
	return list
}

// BuildChallengeInfoForBootstrap 供 Bootstrap 组装当前约球摘要。
func BuildChallengeInfoForBootstrap(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, challenge *model.Challenge, now time.Time) (*types.ChallengeInfo, error) {
	matchId, err := svcCtx.MatchModel.FindIdByChallengeId(challenge.Id)
	if err != nil {
		return nil, err
	}
	record := challenge
	if record.MergedIntoId != nil && *record.MergedIntoId > 0 {
		main, err := svcCtx.ChallengeModel.FindMainById(challenge.Id)
		if err != nil {
			return nil, err
		}
		if main != nil {
			record = main
		}
	}
	row, err := svcCtx.ChallengeModel.FindRowById(record.Id)
	if err != nil {
		return nil, err
	}
	info := buildChallengeInfoFromRecord(record, now)
	info.MatchId = matchId
	if row != nil {
		filled := buildChallengeInfo(*row, matchId, now)
		info.FromNickname = filled.FromNickname
		info.FromAvatar = filled.FromAvatar
		info.ToNickname = filled.ToNickname
		info.ToAvatar = filled.ToAvatar
	}
	fillChallengeVipLevels(ctx, svcCtx, now, &info)
	return &info, nil
}

// ServerTimeNow 服务端当前北京时间，App 用于计算预约相对日期文案。
func ServerTimeNow() string {
	return logicx.FormatUTC8Time(logicx.NowUTC8())
}

func dispatchChallengeNotification(svcCtx *svc.ServiceContext, userId int64, title, content string, data map[string]interface{}, log logx.Logger) {
	var dataJSON *string
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			dataJSON = stringPointer(string(raw))
		}
	}
	if err := logicx.DispatchNotification(svcCtx, logicx.NotificationDispatchInput{
		UserId:      userId,
		Type:        "challenge",
		Title:       title,
		Content:     content,
		PushTitle:   title,
		PushContent: content,
		WSCategory:  "challenge",
		Data:        dataJSON,
	}); err != nil {
		log.Errorf("分发约球通知失败: userId=%d err=%v", userId, err)
	}
	// 通知偏好只控制提醒，不影响约球活动状态在另一端的失效刷新。
	logicx.SendUserDataUpdated(userId, logicx.UserDataUpdatedEvent{Scopes: []string{"challenge"}})
}

func stringPointer(value string) *string {
	copy := value
	return &copy
}
