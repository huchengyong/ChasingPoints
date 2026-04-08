package match

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	matchRefereeJoinTokenTTL = 5 * time.Minute
	matchRefereePayloadType  = "match_referee"
)

var (
	errMatchRefereeCodeInvalid = errors.New("裁判二维码已失效")
	errMatchRefereeAlreadyBound = errors.New("本场已绑定裁判")
	errMatchRefereeJoinUnavailable = errors.New("当前对局无法加入裁判")
)

type matchRefereeQRCodePayload struct {
	Type      string `json:"type"`
	MatchId   int64  `json:"match_id"`
	JoinToken string `json:"join_token"`
	Ts        int64  `json:"ts"`
}

func buildMatchRefereeJoinTokenKey(matchId int64) string {
	return fmt.Sprintf("match:referee:join:%d", matchId)
}

func newMatchRefereeJoinToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
