package logic

import (
	"errors"
	"strings"

	"billiard_master/internal/model"
)

var (
	errMissingClientActionID = errors.New("missing client_action_id")
	errRevisionConflict      = model.ErrMatchRevisionConflict
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

func buildActionReplayAck(clientActionID string, serverRevision int64) actionReplayAck {
	return actionReplayAck{
		Accepted:       true,
		ClientActionID: clientActionID,
		ServerRevision: serverRevision,
	}
}
