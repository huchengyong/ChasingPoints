package logic

import (
	"fmt"
	"strings"
	"unicode"
)

// OpponentIdentity 是读模型使用的稳定对手键。注册用户永远按用户 ID 归并；
// 历史匿名对手没有独立 ID 时只能按标准化昵称归并，同名匿名访客因此是一个
// 明确、可审计的歧义边界。空昵称不与其他空昵称合并，而是隔离到本场比赛。
type OpponentIdentity struct {
	UserId  int64
	NameKey string
	Guest   bool
}

func NormalizeOpponentIdentity(userId int64, name string, matchId int64) OpponentIdentity {
	if userId > 0 {
		return OpponentIdentity{
			UserId:  userId,
			NameKey: fmt.Sprintf("user:%d", userId),
		}
	}

	nameKey := normalizeGuestOpponentName(name)
	if nameKey == "" {
		return OpponentIdentity{
			NameKey: fmt.Sprintf("guest:match:%d", matchId),
			Guest:   true,
		}
	}
	return OpponentIdentity{
		NameKey: "guest:name:" + nameKey,
		Guest:   true,
	}
}

func normalizeGuestOpponentName(name string) string {
	return strings.ToLower(strings.Join(strings.FieldsFunc(strings.TrimSpace(name), unicode.IsSpace), " "))
}
