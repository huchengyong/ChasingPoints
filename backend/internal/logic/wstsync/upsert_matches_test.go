package wstsync

import (
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestMapOfficialMatchStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		source     string
		wantStatus int
	}{
		{name: "scheduled", source: "Scheduled", wantStatus: model.EventNewsStatusUpcoming},
		{name: "live", source: "Live", wantStatus: model.EventNewsStatusLive},
		{name: "suspended", source: "Suspended", wantStatus: model.EventNewsStatusLive},
		{name: "completed", source: "Completed", wantStatus: model.EventNewsStatusFinished},
		{name: "finished", source: "Finished", wantStatus: model.EventNewsStatusFinished},
		{name: "canceled", source: "Canceled", wantStatus: model.EventNewsStatusCanceled},
		{name: "cancelled", source: "Cancelled", wantStatus: model.EventNewsStatusCanceled},
		{name: "walkover", source: "Walkover", wantStatus: model.EventNewsStatusCanceled},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := mapOfficialMatchStatus(tc.source); got != tc.wantStatus {
				t.Fatalf("mapOfficialMatchStatus(%q) = %d, want %d", tc.source, got, tc.wantStatus)
			}
		})
	}
}

func TestUpsertMatchesInsertsMissingMatchAndBindsTournamentAndPlayers(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	tournamentModel := model.NewTournamentModel(db)
	playerModel := model.NewPlayerModel(db)

	tournament := &model.Tournament{
		Id:                 21,
		Name:               "Sportsbet.io Tour Championship 2026",
		CoverImage:         "https://example.com/cover.png",
		GameType:           1,
		Country:            "England",
		City:               "Manchester",
		VenueName:          "Manchester Central",
		SourceType:         "official",
		SourceTournamentId: "tour-1",
		SourceSeasonId:     "2025",
		InformationPage:    "https://www.wst.tv/tourchampionship/",
		TicketingLink:      "https://tickets.example.com",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}

	players := []model.Player{
		{
			Id:             31,
			SourceType:     "official",
			SourcePlayerId: "player-home",
			FirstName:      "Neil",
			LastName:       "Robertson",
			DisplayName:    "Neil Robertson",
			Avatar:         "https://example.com/neil.png",
			CountryCode:    "au",
			FlagEmoji:      "🇦🇺",
		},
		{
			Id:             32,
			SourceType:     "official",
			SourcePlayerId: "player-away",
			FirstName:      "Barry",
			LastName:       "Hawkins",
			DisplayName:    "Barry Hawkins",
			Avatar:         "https://example.com/barry.png",
			CountryCode:    "gb-eng",
			FlagEmoji:      "🏴",
		},
	}
	for i := range players {
		if err := playerModel.Create(&players[i]); err != nil {
			t.Fatalf("seed player %d: %v", i, err)
		}
	}

	startTime := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if err := UpsertMatches(db, now, []MatchUpsertRecord{
		{
			SourceType:         "official",
			SourceMatchId:      "match-1",
			SourceTournamentId: "tour-1",
			RoundName:          "Quarter Finals",
			RoundOrder:         10,
			MatchOrder:         1,
			StartTime:          &startTime,
			BestOf:             19,
			HomePlayerSourceId: "player-home",
			HomePlayerName:     "Neil Robertson",
			AwayPlayerSourceId: "player-away",
			AwayPlayerName:     "Barry Hawkins",
			HomeScore:          10,
			AwayScore:          8,
			SourceStatus:       "Completed",
			PlayersAllocated:   true,
		},
	}); err != nil {
		t.Fatalf("upsert matches: %v", err)
	}

	matchMap, err := model.NewTournamentMatchModel(db).FindBySourceMatchIds("official", []string{"match-1"})
	if err != nil {
		t.Fatalf("find match by source id: %v", err)
	}
	got, ok := matchMap["match-1"]
	if !ok {
		t.Fatal("expected inserted match")
	}
	if got.Id == 0 {
		t.Fatal("expected inserted match to have local id")
	}
	if got.TournamentId != tournament.Id {
		t.Fatalf("expected local tournament id %d, got %d", tournament.Id, got.TournamentId)
	}
	if got.HomePlayerId != players[0].Id || got.AwayPlayerId != players[1].Id {
		t.Fatalf("expected home/away player ids to bind local ids, got %#v", got)
	}
	if got.Player1Id != players[0].Id || got.Player2Id != players[1].Id {
		t.Fatalf("expected player1/player2 ids to mirror home/away local ids, got %#v", got)
	}
	if got.WinnerSide != 1 || got.WinnerId != players[0].Id {
		t.Fatalf("expected winner to map from completed score, got %#v", got)
	}
	if got.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected finished status, got %#v", got)
	}
	if got.StartTime == nil || !got.StartTime.Equal(startTime) {
		t.Fatalf("expected start time %v, got %#v", startTime, got.StartTime)
	}
	if got.BestOf != 19 || got.RoundName != "Quarter Finals" || got.RoundOrder != 10 || got.MatchOrder != 1 {
		t.Fatalf("unexpected round/best-of mapping: %#v", got)
	}
	if got.HomeScore != 10 || got.AwayScore != 8 {
		t.Fatalf("unexpected score mapping: %#v", got)
	}
	if got.IsPlaceholder {
		t.Fatalf("expected non-placeholder match, got %#v", got)
	}
}

func TestUpsertMatchesUpdatesExistingMatchWithoutChangingID(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	tournamentModel := model.NewTournamentModel(db)
	playerModel := model.NewPlayerModel(db)
	matchModel := model.NewTournamentMatchModel(db)

	tournament := &model.Tournament{
		Id:                 41,
		Name:               "Sportsbet.io Tour Championship 2026",
		GameType:           1,
		Country:            "England",
		City:               "Manchester",
		VenueName:          "Manchester Central",
		SourceType:         "official",
		SourceTournamentId: "tour-1",
		SourceSeasonId:     "2025",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}

	players := []model.Player{
		{
			Id:             51,
			SourceType:     "official",
			SourcePlayerId: "player-home",
			FirstName:      "Neil",
			LastName:       "Robertson",
			DisplayName:    "Neil Robertson",
			Avatar:         "https://example.com/neil.png",
			CountryCode:    "au",
			FlagEmoji:      "🇦🇺",
		},
		{
			Id:             52,
			SourceType:     "official",
			SourcePlayerId: "player-away",
			FirstName:      "John",
			LastName:       "Higgins",
			DisplayName:    "John Higgins",
			Avatar:         "https://example.com/john.png",
			CountryCode:    "gb-sct",
			FlagEmoji:      "🏴",
		},
	}
	for i := range players {
		if err := playerModel.Create(&players[i]); err != nil {
			t.Fatalf("seed player %d: %v", i, err)
		}
	}

	existingStart := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	seed := &model.TournamentMatch{
		Id:              88,
		TournamentId:    tournament.Id,
		SourceType:      "official",
		SourceMatchId:   "match-1",
		RoundName:       "Quarter Finals",
		RoundOrder:      10,
		MatchOrder:      1,
		StartTime:       &existingStart,
		BestOf:          17,
		HomePlayerId:    players[0].Id,
		HomePlayerName:  players[0].DisplayName,
		AwayPlayerId:    players[1].Id,
		AwayPlayerName:  players[1].DisplayName,
		HomeScore:       5,
		AwayScore:       3,
		WinnerSide:      1,
		IsPlaceholder:   false,
		Player1Id:       players[0].Id,
		Player2Id:       players[1].Id,
		WinnerId:        players[0].Id,
		MatchId:         19,
		BracketPosition: "upper-left",
		Status:          model.EventNewsStatusLive,
	}
	if err := matchModel.Create(seed); err != nil {
		t.Fatalf("seed match: %v", err)
	}

	newStart := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if err := UpsertMatches(db, now, []MatchUpsertRecord{
		{
			SourceType:         "official",
			SourceMatchId:      "match-1",
			SourceTournamentId: "tour-1",
			RoundName:          "Quarter Finals",
			RoundOrder:         10,
			MatchOrder:         1,
			StartTime:          &newStart,
			BestOf:             19,
			HomePlayerSourceId: "player-home",
			HomePlayerName:     "Neil Robertson",
			AwayPlayerSourceId: "player-away",
			AwayPlayerName:     "John Higgins",
			HomeScore:          8,
			AwayScore:          10,
			SourceStatus:       "Completed",
			PlayersAllocated:   true,
		},
	}); err != nil {
		t.Fatalf("upsert matches: %v", err)
	}

	matchMap, err := matchModel.FindBySourceMatchIds("official", []string{"match-1"})
	if err != nil {
		t.Fatalf("find match by source id: %v", err)
	}
	got, ok := matchMap["match-1"]
	if !ok {
		t.Fatal("expected updated match")
	}
	if got.Id != 88 {
		t.Fatalf("expected local id to stay 88, got %d", got.Id)
	}
	if got.TournamentId != tournament.Id {
		t.Fatalf("expected tournament id %d, got %d", tournament.Id, got.TournamentId)
	}
	if got.HomePlayerId != players[0].Id || got.AwayPlayerId != players[1].Id {
		t.Fatalf("expected player ids to remain bound, got %#v", got)
	}
	if got.Player1Id != players[0].Id || got.Player2Id != players[1].Id {
		t.Fatalf("expected player1/player2 ids to remain bound, got %#v", got)
	}
	if got.WinnerSide != 2 || got.WinnerId != players[1].Id {
		t.Fatalf("expected updated winner to map from away score, got %#v", got)
	}
	if got.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected finished status after rerun, got %#v", got)
	}
	if got.StartTime == nil || !got.StartTime.Equal(newStart) {
		t.Fatalf("expected updated start time %v, got %#v", newStart, got.StartTime)
	}
	if got.BestOf != 19 || got.HomeScore != 8 || got.AwayScore != 10 {
		t.Fatalf("expected updated score/best-of mapping, got %#v", got)
	}
	if got.BracketPosition != "upper-left" || got.MatchId != 19 {
		t.Fatalf("expected unrelated fields to be preserved, got %#v", got)
	}
}

func TestUpsertMatchesMarksPlaceholderMatchesWithoutPlayerBindings(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	tournamentModel := model.NewTournamentModel(db)

	tournament := &model.Tournament{
		Id:                 61,
		Name:               "Sportsbet.io Tour Championship 2026",
		GameType:           1,
		SourceType:         "official",
		SourceTournamentId: "tour-1",
		SourceSeasonId:     "2025",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}

	if err := UpsertMatches(db, now, []MatchUpsertRecord{
		{
			SourceType:         "official",
			SourceMatchId:      "match-placeholder",
			SourceTournamentId: "tour-1",
			RoundName:          "Final",
			RoundOrder:         30,
			MatchOrder:         1,
			BestOf:             19,
			HomePlayerName:     "TBD",
			AwayPlayerName:     "待定",
			SourceStatus:       "Scheduled",
			PlayersAllocated:   false,
		},
	}); err != nil {
		t.Fatalf("upsert placeholder match: %v", err)
	}

	matchMap, err := model.NewTournamentMatchModel(db).FindBySourceMatchIds("official", []string{"match-placeholder"})
	if err != nil {
		t.Fatalf("find match by source id: %v", err)
	}
	got, ok := matchMap["match-placeholder"]
	if !ok {
		t.Fatal("expected placeholder match")
	}
	if !got.IsPlaceholder {
		t.Fatalf("expected placeholder flag to be set, got %#v", got)
	}
	if got.HomePlayerId != 0 || got.AwayPlayerId != 0 || got.Player1Id != 0 || got.Player2Id != 0 || got.WinnerId != 0 {
		t.Fatalf("expected placeholder match to stay unbound, got %#v", got)
	}
	if got.Status != model.EventNewsStatusUpcoming {
		t.Fatalf("expected scheduled placeholder to stay upcoming, got %#v", got)
	}
}

func TestUpsertMatchesConvertsStaleScheduledMatchToFinished(t *testing.T) {
	db := newWSTSyncTestDB(t)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	tournamentModel := model.NewTournamentModel(db)
	playerModel := model.NewPlayerModel(db)

	tournament := &model.Tournament{
		Id:                 71,
		Name:               "Sportsbet.io Tour Championship 2026",
		GameType:           1,
		SourceType:         "official",
		SourceTournamentId: "tour-1",
		SourceSeasonId:     "2025",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}

	players := []model.Player{
		{Id: 81, SourceType: "official", SourcePlayerId: "player-home", DisplayName: "John Higgins"},
		{Id: 82, SourceType: "official", SourcePlayerId: "player-away", DisplayName: "Zhao Xintong"},
	}
	for i := range players {
		if err := playerModel.Create(&players[i]); err != nil {
			t.Fatalf("seed player %d: %v", i, err)
		}
	}

	startTime := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	if err := UpsertMatches(db, now, []MatchUpsertRecord{
		{
			SourceType:         "official",
			SourceMatchId:      "match-stale-scheduled",
			SourceTournamentId: "tour-1",
			RoundName:          "Semi Finals",
			RoundOrder:         60,
			MatchOrder:         1,
			StartTime:          &startTime,
			BestOf:             19,
			HomePlayerSourceId: "player-home",
			HomePlayerName:     "John Higgins",
			AwayPlayerSourceId: "player-away",
			AwayPlayerName:     "Zhao Xintong",
			HomeScore:          10,
			AwayScore:          4,
			SourceStatus:       "Scheduled",
			PlayersAllocated:   true,
		},
	}); err != nil {
		t.Fatalf("upsert stale scheduled match: %v", err)
	}

	matchMap, err := model.NewTournamentMatchModel(db).FindBySourceMatchIds("official", []string{"match-stale-scheduled"})
	if err != nil {
		t.Fatalf("find match by source id: %v", err)
	}
	got := matchMap["match-stale-scheduled"]
	if got.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected stale scheduled match to become finished, got %#v", got)
	}
	if got.WinnerSide != 1 || got.WinnerId != players[0].Id {
		t.Fatalf("expected winner to be derived after stale status correction, got %#v", got)
	}
}
