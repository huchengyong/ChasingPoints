package logic

import (
	"fmt"
	"strings"
	"time"
)

func parseTournamentTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05+08:00",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, trimmed)
		if err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid time format")
}
