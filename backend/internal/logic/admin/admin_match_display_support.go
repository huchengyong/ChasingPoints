package admin

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

func buildAdminPlayerNameMap(svcCtx *svc.ServiceContext, matches []model.Match) map[int64]string {
	playerNameMap := make(map[int64]string)
	if svcCtx == nil || svcCtx.DB == nil || len(matches) == 0 {
		return playerNameMap
	}

	playerIds := make([]int64, 0, len(matches))
	seen := make(map[int64]bool)
	for _, match := range matches {
		if match.UserId <= 0 || seen[match.UserId] {
			continue
		}
		seen[match.UserId] = true
		playerIds = append(playerIds, match.UserId)
	}
	if len(playerIds) == 0 {
		return playerNameMap
	}

	var users []struct {
		Id       int64
		Nickname string
	}
	if err := svcCtx.DB.Table("users").Select("id, nickname").Where("id IN ?", playerIds).Scan(&users).Error; err != nil {
		return playerNameMap
	}
	for _, user := range users {
		if user.Nickname != "" {
			playerNameMap[user.Id] = user.Nickname
		}
	}
	return playerNameMap
}

func resolveAdminPlayerNames(match model.Match, playerNameMap map[int64]string) (string, string) {
	player1Name := playerNameMap[match.UserId]
	if player1Name == "" {
		player1Name = "玩家"
	}

	player2Name := match.OpponentName
	if player2Name == "" {
		player2Name = "玩家"
	}

	return player1Name, player2Name
}

func resolveAdminWinnerName(result *int, player1Name, player2Name string) string {
	if result == nil {
		return "-"
	}

	switch *result {
	case 1:
		return player1Name
	case 2:
		return player2Name
	case 3:
		return "平局"
	default:
		return "-"
	}
}
