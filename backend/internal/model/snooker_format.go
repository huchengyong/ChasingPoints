package model

const (
	SnookerFormatLegacy = "legacy"
	SnookerFormatFree   = "free"
	SnookerFormatRaceTo = "race_to"

	SnookerMaxCompletedFrames = 49
	SnookerMaxTargetWins      = 25
)

func NormalizeSnookerFormat(format string, targetWins, bestOfFrames int) (string, int, bool) {
	switch format {
	case SnookerFormatFree:
		return SnookerFormatFree, 0, targetWins == 0
	case SnookerFormatRaceTo:
		return SnookerFormatRaceTo, targetWins, targetWins >= 1 && targetWins <= SnookerMaxTargetWins
	case "":
		if bestOfFrames > 0 && bestOfFrames%2 == 1 {
			return SnookerFormatRaceTo, bestOfFrames/2 + 1, true
		}
		return SnookerFormatLegacy, 0, true
	case SnookerFormatLegacy:
		if bestOfFrames > 0 && bestOfFrames%2 == 1 {
			return SnookerFormatRaceTo, bestOfFrames/2 + 1, true
		}
		return SnookerFormatLegacy, 0, true
	default:
		return "", 0, false
	}
}

func SnookerTargetReached(match *Match) bool {
	if match == nil || match.GameType != 1 {
		return false
	}
	format, targetWins, ok := NormalizeSnookerFormat(match.SnookerFormat, match.SnookerTargetWins, match.BestOfFrames)
	if !ok || format != SnookerFormatRaceTo || targetWins <= 0 {
		return false
	}
	return match.MyScore >= targetWins || match.OpponentScore >= targetWins
}
