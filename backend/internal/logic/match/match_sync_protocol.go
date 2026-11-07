package match

import (
	"errors"
	"strings"

	"chasing_points/internal/model"
)

var (
	errMissingClientActionID = errors.New("missing client_action_id")
	errRevisionConflict      = model.ErrMatchRevisionConflict
	errClientActionIDReused  = errors.New("操作幂等键已被其他操作使用")
)

type matchActionMeta struct {
	ClientActionID string
	BaseRevision   int64
}

type actionReplayAck struct {
	Accepted       bool
	ClientActionID string
	ServerRevision int64
}

func validateMatchActionMeta(meta matchActionMeta, currentRevision int64) error {
	if strings.TrimSpace(meta.ClientActionID) == "" {
		return errMissingClientActionID
	}
	if meta.BaseRevision != currentRevision {
		return errRevisionConflict
	}
	return nil
}

// isMatchActionReplay 判断幂等键命中的操作是否就是本次请求的重放；同一键被其他操作占用时不能当作成功。
func isMatchActionReplay(existing *model.MatchAction, actionType string, actor int, baseRevision int64) bool {
	return existing != nil && existing.ActionType == actionType && existing.Actor == actor && existing.BaseRevision == baseRevision
}

// validateNewMatchAction 校验不属于重放的新写入；幂等键已被其他操作占用时直接拒绝。
func validateNewMatchAction(existing *model.MatchAction, meta matchActionMeta, currentRevision int64) error {
	if existing != nil {
		return errClientActionIDReused
	}
	return validateMatchActionMeta(meta, currentRevision)
}

func buildActionReplayAck(clientActionID string, serverRevision int64) actionReplayAck {
	return actionReplayAck{
		Accepted:       true,
		ClientActionID: clientActionID,
		ServerRevision: serverRevision,
	}
}
