package match

import (
	"fmt"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
)

type h2hReadTarget struct {
	subjectUserID   int64
	opponentUserID  int64
	opponentName    string
	opponentAvatar  string
	opponentNameKey string
}

func resolveH2HReadTarget(svcCtx *svc.ServiceContext, viewerUserID, targetUserID, opponentUserID int64, opponentName string) (h2hReadTarget, error) {
	target := h2hReadTarget{subjectUserID: viewerUserID, opponentUserID: opponentUserID, opponentName: opponentName}
	if targetUserID > 0 && targetUserID != viewerUserID {
		if svcCtx == nil || svcCtx.FriendModel == nil {
			return target, fmt.Errorf("无法验证好友关系")
		}
		areFriends, err := svcCtx.FriendModel.AreFriends(viewerUserID, targetUserID)
		if err != nil {
			return target, err
		}
		if !areFriends {
			return target, fmt.Errorf("仅可查看好友的对方战绩")
		}
		target.subjectUserID = targetUserID
	}
	if opponentUserID > 0 {
		if svcCtx == nil || svcCtx.UserModel == nil {
			return target, fmt.Errorf("无法查询对手资料")
		}
		user, err := svcCtx.UserModel.FindById(opponentUserID)
		if err != nil {
			return target, err
		}
		if user != nil {
			target.opponentUserID = user.Id
			if target.opponentName == "" {
				target.opponentName = user.Nickname
			}
			target.opponentAvatar = user.Avatar
		}
	}
	if target.opponentName == "" {
		return target, fmt.Errorf("缺少对手信息")
	}
	identity := logicx.NormalizeOpponentIdentity(target.opponentUserID, target.opponentName, 0)
	target.opponentNameKey = identity.NameKey
	return target, nil
}
