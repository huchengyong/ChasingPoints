package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"chasing_points/internal/logic/wstsync"
)

func TestPrintBackfillSummaryPrintsCommittedCounts(t *testing.T) {
	tests := []struct {
		name    string
		summary *wstsync.SyncSummary
		want    []string
	}{
		{
			name: "known",
			summary: &wstsync.SyncSummary{
				CommittedCountsKnown: true,
				PlayersCommitted:     2,
				TournamentsCommitted: 160,
				MatchesCommitted:     8864,
				EventNewsCommitted:   160,
			},
			want: []string{
				"players_committed:    2",
				"tournaments_committed: 160",
				"matches_committed:    8864",
				"event_news_committed: 160",
			},
		},
		{
			name: "unknown",
			summary: &wstsync.SyncSummary{
				PlayersCommitted:     2,
				TournamentsCommitted: 160,
				MatchesCommitted:     8864,
				EventNewsCommitted:   160,
			},
			want: []string{
				"players_committed:    unknown",
				"tournaments_committed: unknown",
				"matches_committed:    unknown",
				"event_news_committed: unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() { printBackfillSummary(tt.summary) })
			for _, want := range tt.want {
				if !strings.Contains(output, want+"\n") {
					t.Fatalf("summary missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return string(output)
}
