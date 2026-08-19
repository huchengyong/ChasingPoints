package model

const (
	MatchFormatLegacy = "legacy"
	MatchFormatFree   = "free"
	MatchFormatRaceTo = "race_to"

	PoolMatchMaxTargetWins      = 65
	PoolMatchMaxCompletedRounds = 129
)

func IsPoolMatchFormatGameType(gameType int) bool {
	return gameType == 3 || gameType == 4
}

func NormalizePoolMatchFormat(gameType int, format string, targetWins int) (string, int, bool) {
	if !IsPoolMatchFormatGameType(gameType) {
		if format == "" || format == MatchFormatLegacy {
			return MatchFormatLegacy, 0, targetWins == 0
		}
		return "", 0, false
	}

	switch format {
	case MatchFormatFree:
		return MatchFormatFree, 0, targetWins == 0
	case MatchFormatRaceTo:
		return MatchFormatRaceTo, targetWins, targetWins >= 1 && targetWins <= PoolMatchMaxTargetWins
	case "", MatchFormatLegacy:
		return MatchFormatLegacy, 0, targetWins == 0
	default:
		return "", 0, false
	}
}

func IsFlexiblePoolMatch(match *Match) bool {
	if match == nil || !IsPoolMatchFormatGameType(match.GameType) {
		return false
	}
	format, _, ok := NormalizePoolMatchFormat(match.GameType, match.MatchFormat, match.TargetWins)
	return ok && (format == MatchFormatFree || format == MatchFormatRaceTo)
}

func PoolMatchTargetReached(match *Match) bool {
	if !IsFlexiblePoolMatch(match) {
		return false
	}
	format, targetWins, ok := NormalizePoolMatchFormat(match.GameType, match.MatchFormat, match.TargetWins)
	return ok && format == MatchFormatRaceTo && targetWins > 0 &&
		(match.MyScore >= targetWins || match.OpponentScore >= targetWins)
}
