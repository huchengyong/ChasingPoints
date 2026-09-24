package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// challengeStatusText 状态文案由前端负责，这里仅透传数值。

func dayOffsetOf(scheduledDate *time.Time, now time.Time) int {
	if scheduledDate == nil {
		return 0
	}
	date := logicx.InUTC8(*scheduledDate)
	today := logicx.InUTC8(now)
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, logicx.UTC8Location)
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, logicx.UTC8Location)
	offset := int(dateStart.Sub(todayStart).Hours() / 24)
	if offset < 0 {
		return 0
	}
	return offset
}

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
		Id:                  row.Id,
		MatchId:             matchID,
		FromUserId:          row.FromUserId,
		FromNickname:        row.FromNickname,
		FromAvatar:          row.FromAvatar,
		ToUserId:            row.ToUserId,
		ToNickname:          row.ToNickname,
		ToAvatar:            row.ToAvatar,
		GameType:            row.GameType,
		MatchMode:           row.MatchMode,
		Visibility:          row.Visibility,
		MatchFormat:         row.MatchFormat,
		TargetWins:          row.TargetWins,
		SnookerRulesVersion: row.SnookerRulesVersion,
		BestOfFrames:        row.BestOfFrames,
		SnookerFormat:       row.SnookerFormat,
		SnookerTargetWins:   row.SnookerTargetWins,
		StartingActor:       row.StartingActor,
		Message:             row.Message,
		Status:              row.Status,
		CloseReason:         row.CloseReason,
		CreatedAt:           formatTimePtr(row.CreatedAt),
		ExpiresAt:           formatTimePtr(row.ExpiresAt),
	}
	if row.WaitingUserId != nil {
		info.WaitingUserId = *row.WaitingUserId
	}
	if row.ScheduledDate != nil {
		info.ScheduledDate = formatHourDate(*row.ScheduledDate)
		info.DayOffset = dayOffsetOf(row.ScheduledDate, now)
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
		Id:                  challenge.Id,
		FromUserId:          challenge.FromUserId,
		ToUserId:            challenge.ToUserId,
		GameType:            challenge.GameType,
		MatchMode:           challenge.MatchMode,
		Visibility:          challenge.Visibility,
		MatchFormat:         challenge.MatchFormat,
		TargetWins:          challenge.TargetWins,
		SnookerRulesVersion: challenge.SnookerRulesVersion,
		BestOfFrames:        challenge.BestOfFrames,
		SnookerFormat:       challenge.SnookerFormat,
		SnookerTargetWins:   challenge.SnookerTargetWins,
		StartingActor:       challenge.StartingActor,
		Message:             challenge.Message,
		Status:              challenge.Status,
		CloseReason:         challenge.CloseReason,
		ExpiresAt:           formatTimePtr(challenge.ExpiresAt),
	}
	if challenge.WaitingUserId != nil {
		info.WaitingUserId = *challenge.WaitingUserId
	}
	if challenge.ScheduledDate != nil {
		info.ScheduledDate = formatHourDate(*challenge.ScheduledDate)
		info.DayOffset = dayOffsetOf(challenge.ScheduledDate, now)
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

// validateChallengeMatchOptions 服务端权威校验球种与赛制，返回规范化后的选项。
func validateChallengeMatchOptions(gameType int, matchMode, visibility, matchFormat string, targetWins int,
	snookerRulesVersion, bestOfFrames, snookerTargetWins, startingActor int,
	snookerFormat string,
) (mode, vis, format string, wins int, snookerVersion, frames, snookerWins, actor int, snookerFmt string, err error) {
	if matchMode == "" {
		matchMode = model.MatchModeRanked
	}
	if matchMode != model.MatchModePractice && matchMode != model.MatchModeRanked {
		return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择有效的比赛类型")
	}
	if visibility == "" {
		if matchMode == model.MatchModePractice {
			visibility = model.MatchVisibilityPrivate
		} else {
			visibility = model.MatchVisibilityPublic
		}
	}
	if visibility != model.MatchVisibilityPrivate && visibility != model.MatchVisibilityPublic {
		return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择有效的公开范围")
	}
	if matchMode == model.MatchModeRanked && visibility != model.MatchVisibilityPublic {
		return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("排位赛必须公开展示")
	}
	if gameType < 1 || gameType > 4 {
		return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择有效的球种")
	}
	if gameType == 1 {
		version := snookerRulesVersion
		if version == 0 {
			version = model.SnookerRulesVersionLegacy
		}
		if version != model.SnookerRulesVersionLegacy && version != model.SnookerRulesVersionWPBSA {
			return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("不支持的斯诺克规则版本")
		}
		if version == model.SnookerRulesVersionWPBSA {
			if snookerFormat == "" {
				if bestOfFrames <= 0 || bestOfFrames%2 == 0 {
					return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("斯诺克总局数必须为正奇数")
				}
				if startingActor != 1 && startingActor != 2 {
					return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择首局开球方")
				}
			}
			formatValue, targetValue, valid := model.NormalizeSnookerFormat(snookerFormat, snookerTargetWins, bestOfFrames)
			if !valid {
				return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择有效的斯诺克赛制")
			}
			snookerFormat = formatValue
			snookerTargetWins = targetValue
			if startingActor != 1 && startingActor != 2 {
				startingActor = 1
			}
		} else {
			bestOfFrames = 0
			snookerFormat = model.SnookerFormatLegacy
			snookerTargetWins = 0
			startingActor = 0
		}
		matchFormat = ""
		targetWins = 0
	} else if gameType == 2 {
		// 九球追分保持既有赛制，不适用灵活赛制字段。
		matchFormat = ""
		targetWins = 0
		snookerFormat = ""
		snookerTargetWins = 0
		bestOfFrames = 0
		startingActor = 0
		snookerRulesVersion = model.SnookerRulesVersionLegacy
	} else {
		formatValue, targetValue, valid := model.NormalizePoolMatchFormat(gameType, matchFormat, targetWins)
		if !valid {
			return "", "", "", 0, 0, 0, 0, 0, "", fmt.Errorf("请选择有效的赛制")
		}
		matchFormat = formatValue
		targetWins = targetValue
		bestOfFrames = 0
		startingActor = 0
		snookerFormat = ""
		snookerTargetWins = 0
		snookerRulesVersion = 0
	}
	return matchMode, visibility, matchFormat, targetWins, snookerRulesVersion, bestOfFrames, snookerTargetWins, startingActor, snookerFormat, nil
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
