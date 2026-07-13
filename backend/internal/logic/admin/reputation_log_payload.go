package admin

import (
	"fmt"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildAdminReputationLogListResp(logs []model.UserReputationLog, nicknameByUserID map[int64]string, total int64) *types.AdminReputationLogListResp {
	items := make([]types.AdminReputationLogItem, 0, len(logs))
	for i := range logs {
		nickname := ""
		if nicknameByUserID != nil {
			nickname = nicknameByUserID[logs[i].UserID]
		}
		items = append(items, buildAdminReputationLogItem(&logs[i], nickname))
	}

	return &types.AdminReputationLogListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}
}

func buildAdminReputationLogItem(log *model.UserReputationLog, nickname string) types.AdminReputationLogItem {
	if log == nil {
		return types.AdminReputationLogItem{}
	}

	item := types.AdminReputationLogItem{
		Id:              log.ID,
		UserId:          log.UserID,
		Nickname:        nickname,
		ChangeType:      log.ChangeType,
		ChangeTypeText:  model.ReputationChangeTypeText(log.ChangeType),
		ReasonCode:      log.ReasonCode,
		ReasonText:      model.ReputationReasonText(log.ReasonCode, log.ChangeType),
		ReasonDetail:    log.ReasonDetail,
		ChangeScore:     log.ChangeScore,
		BeforeScore:     log.BeforeScore,
		AfterScore:      log.AfterScore,
		OperatorAdminId: log.OperatorAdminID,
		OperatorText:    reputationOperatorText(log.OperatorAdminID),
		CreatedAt:       logicx.FormatUTC8Time(log.CreatedAt),
	}
	if log.MatchID != nil {
		item.MatchId = *log.MatchID
	}
	return item
}

func reputationOperatorText(operatorAdminID int64) string {
	if operatorAdminID <= 0 {
		return "系统"
	}
	return fmt.Sprintf("管理员 #%d", operatorAdminID)
}
