package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

// BuildRoundRecords 把局记录转换为固定选手1/选手2视角的局记录，不随查看者交换。
func BuildRoundRecords(rounds []model.MatchRound) []types.RoundRecord {
	if len(rounds) == 0 {
		return nil
	}
	records := make([]types.RoundRecord, 0, len(rounds))
	for _, r := range rounds {
		winner := 0
		if r.Winner != nil {
			winner = *r.Winner
		}
		records = append(records, types.RoundRecord{
			RoundNumber:  r.RoundNo,
			Player1Score: r.MyScore,
			Player2Score: r.OpponentScore,
			Winner:       winner,
			WinType:      r.WinType,
		})
	}
	return records
}
