package stats

func calculateWinRatePercent(wins int, matches int) float64 {
	if matches <= 0 {
		return 0
	}
	return float64(wins) / float64(matches) * 100
}
