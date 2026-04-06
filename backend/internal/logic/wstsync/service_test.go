package wstsync

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeDataClient struct {
	seasons             []SeasonResource
	tournamentsBySeason map[int][]TournamentResource
	matchesPages        []MatchListResponse
	pageCoverImages     map[string]string
}

func (f *fakeDataClient) FetchSeasons(ctx context.Context) ([]SeasonResource, error) {
	return f.seasons, nil
}

func (f *fakeDataClient) FetchTournamentsBySeason(ctx context.Context, season int) ([]TournamentResource, error) {
	return f.tournamentsBySeason[season], nil
}

func (f *fakeDataClient) FetchMatchesPage(ctx context.Context, pageNumber, pageSize int) (MatchListResponse, error) {
	index := pageNumber - 1
	if index < 0 || index >= len(f.matchesPages) {
		return MatchListResponse{}, nil
	}
	return f.matchesPages[index], nil
}

func (f *fakeDataClient) FetchPageCoverImage(ctx context.Context, pageURL string) (string, error) {
	if f.pageCoverImages == nil {
		return "", nil
	}
	return f.pageCoverImages[pageURL], nil
}

func newWSTSyncServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return db
}

func newWSTSyncServiceForTest(t *testing.T, db *gorm.DB, client DataClient) *Service {
	t.Helper()

	svcCtx := &svc.ServiceContext{
		DB:                   db,
		PlayerModel:          model.NewPlayerModel(db),
		TournamentModel:      model.NewTournamentModel(db),
		TournamentMatchModel: model.NewTournamentMatchModel(db),
		EventNewsModel:       model.NewEventNewsModel(db),
	}
	return NewService(svcCtx, client)
}

func TestServiceDryRunBuildsSummaryWithoutWritingDatabase(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	client := &fakeDataClient{
		seasons: []SeasonResource{
			{ID: "2024", Attributes: SeasonAttributes{Name: "2024/25"}},
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
		tournamentsBySeason: map[int][]TournamentResource{
			2024: {
				{
					ID: "world-open-2025",
					Attributes: TournamentAttributes{
						Name:            "World Open 2025",
						StartDate:       "2025-03-16",
						EndDate:         "2025-03-22",
						City:            "Yushan",
						Country:         "China",
						InformationPage: "https://www.wst.tv/worldopen/",
						Season:          TournamentSeason{ID: "2024"},
					},
				},
				{
					ID: "world-open-qualifiers-2025",
					Attributes: TournamentAttributes{
						Name:            "World Open Qualifiers 2025",
						StartDate:       "2025-01-01",
						EndDate:         "2025-01-03",
						City:            "Leicester",
						Country:         "England",
						InformationPage: "https://www.wst.tv/worldopenqualifiers/",
						Season:          TournamentSeason{ID: "2024"},
					},
				},
			},
			2025: {
				{
					ID: "british-open-2025",
					Attributes: TournamentAttributes{
						Name:            "British Open 2025",
						StartDate:       "2025-08-25",
						EndDate:         "2025-08-31",
						City:            "Cheltenham",
						Country:         "England",
						InformationPage: "https://www.wst.tv/britishopen/",
						Season:          TournamentSeason{ID: "2025"},
					},
				},
			},
		},
		matchesPages: []MatchListResponse{
			{
				Data: []MatchResource{
					{
						ID: "match-1",
						Attributes: MatchAttributes{
							TournamentID:    "world-open-2025",
							HomePlayerID:    "player-a",
							AwayPlayerID:    "player-b",
							HomePlayerScore: 5,
							AwayPlayerScore: 2,
							StartDateTime:   "2025-03-16 09:00:00",
							Round:           "Round 1",
							Status:          "Completed",
							NumberOfFrames:  9,
							FixtureNumber:   1,
							PlayersAllocated:true,
							HomePlayer:      WstPlayer{PlayerID: "player-a", FirstName: "Player", Surname: "A", CountryCode: "cn"},
							AwayPlayer:      WstPlayer{PlayerID: "player-b", FirstName: "Player", Surname: "B", CountryCode: "gb-eng"},
						},
					},
					{
						ID: "match-2",
						Attributes: MatchAttributes{
							TournamentID:    "british-open-2025",
							HomePlayerID:    "player-c",
							AwayPlayerID:    "player-d",
							HomePlayerScore: 0,
							AwayPlayerScore: 0,
							StartDateTime:   "2025-08-25 09:00:00",
							Round:           "Round 1",
							Status:          "Scheduled",
							NumberOfFrames:  9,
							FixtureNumber:   1,
							PlayersAllocated:true,
							HomePlayer:      WstPlayer{PlayerID: "player-c", FirstName: "Player", Surname: "C", CountryCode: "au"},
							AwayPlayer:      WstPlayer{PlayerID: "player-d", FirstName: "Player", Surname: "D", CountryCode: "gb-sct"},
						},
					},
					{
						ID: "match-qualifier",
						Attributes: MatchAttributes{
							TournamentID:    "world-open-qualifiers-2025",
							HomePlayerID:    "player-e",
							AwayPlayerID:    "player-f",
							StartDateTime:   "2025-01-01 09:00:00",
							Round:           "Round 1",
							Status:          "Scheduled",
							NumberOfFrames:  9,
							FixtureNumber:   1,
							PlayersAllocated:true,
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	summary, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeYear,
		Year:              2025,
		From:              &from,
		To:                &to,
		Publish:           true,
		DryRun:            true,
		IncludeQualifiers: false,
		GameType:          1,
	})
	if err != nil {
		t.Fatalf("service dry run: %v", err)
	}

	if summary.SeasonsFetched != 2 || summary.CandidateSeasons != 2 {
		t.Fatalf("unexpected season summary: %#v", summary)
	}
	if summary.TournamentsFetched != 3 || summary.TournamentsSelected != 2 {
		t.Fatalf("unexpected tournament summary: %#v", summary)
	}
	if summary.MatchesScanned != 3 || summary.MatchesSelected != 2 {
		t.Fatalf("unexpected match summary: %#v", summary)
	}
	if summary.PlayersPrepared != 4 || summary.TournamentsPrepared != 2 || summary.MatchesPrepared != 2 || summary.EventNewsProjected != 2 {
		t.Fatalf("unexpected prepared summary: %#v", summary)
	}

	tournamentMap, err := model.NewTournamentModel(db).FindBySourceTournamentIds("official", []string{"world-open-2025", "british-open-2025"})
	if err != nil {
		t.Fatalf("find tournaments after dry run: %v", err)
	}
	if len(tournamentMap) != 0 {
		t.Fatalf("expected dry run to avoid DB writes, got %#v", tournamentMap)
	}
}

func TestServiceSyncWritesFactsAndProjectsEventNews(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	client := &fakeDataClient{
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {
				{
					ID: "tour-championship-2026",
					Attributes: TournamentAttributes{
						Name:            "Sportsbet.io Tour Championship 2026",
						StartDate:       "2026-03-30",
						EndDate:         "2026-04-05",
						City:            "Manchester",
						Country:         "England",
						InformationPage: "https://www.wst.tv/tourchampionship/",
						TicketingLink:   "https://tickets.example.com",
						Season:          TournamentSeason{ID: "2025"},
					},
				},
			},
		},
		matchesPages: []MatchListResponse{
			{
				Data: []MatchResource{
					{
						ID: "match-1",
						Attributes: MatchAttributes{
							TournamentID:    "tour-championship-2026",
							HomePlayerID:    "player-home",
							AwayPlayerID:    "player-away",
							HomePlayerScore: 10,
							AwayPlayerScore: 8,
							StartDateTime:   "2026-03-30 12:00:00",
							Round:           "Quarter Finals",
							Status:          "Completed",
							NumberOfFrames:  19,
							FixtureNumber:   1,
							PlayersAllocated:true,
							HomePlayer: WstPlayer{
								PlayerID:    "player-home",
								FirstName:   "Judd",
								Surname:     "Trump",
								CountryCode: "gb-eng",
								Media:       WstPlayerMedia{Profile: "judd.png"},
							},
							AwayPlayer: WstPlayer{
								PlayerID:    "player-away",
								FirstName:   "Mark",
								Surname:     "Allen",
								CountryCode: "gb-nir",
								Media:       WstPlayerMedia{Profile: "mark.png"},
							},
						},
					},
				},
			},
		},
		pageCoverImages: map[string]string{
			"https://www.wst.tv/tourchampionship/": "https://images.gc.wstservices.co.uk/fit-in/1000x1000/tour-cover.jpg",
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	summary, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeSeason,
		Season:            2025,
		Publish:           true,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	})
	if err != nil {
		t.Fatalf("service sync: %v", err)
	}

	if summary.TournamentsPrepared != 1 || summary.MatchesPrepared != 1 || summary.EventNewsProjected != 1 {
		t.Fatalf("unexpected sync summary: %#v", summary)
	}

	playerMap, err := model.NewPlayerModel(db).FindBySourcePlayerIds("official", []string{"player-home", "player-away"})
	if err != nil {
		t.Fatalf("find synced players: %v", err)
	}
	if len(playerMap) != 2 {
		t.Fatalf("expected 2 synced players, got %#v", playerMap)
	}

	tournamentMap, err := model.NewTournamentModel(db).FindBySourceTournamentIds("official", []string{"tour-championship-2026"})
	if err != nil {
		t.Fatalf("find synced tournament: %v", err)
	}
	tournament, ok := tournamentMap["tour-championship-2026"]
	if !ok {
		t.Fatalf("expected synced tournament, got %#v", tournamentMap)
	}
	if tournament.Status != model.EventNewsStatusFinished {
		t.Fatalf("expected finished tournament status, got %#v", tournament)
	}
	if tournament.CoverImage != "https://images.gc.wstservices.co.uk/fit-in/1000x1000/tour-cover.jpg" {
		t.Fatalf("expected fetched cover image, got %#v", tournament)
	}

	matchMap, err := model.NewTournamentMatchModel(db).FindBySourceMatchIds("official", []string{"match-1"})
	if err != nil {
		t.Fatalf("find synced match: %v", err)
	}
	match, ok := matchMap["match-1"]
	if !ok {
		t.Fatalf("expected synced match, got %#v", matchMap)
	}
	if match.TournamentId != tournament.Id || match.WinnerSide != 1 || match.Status != model.EventNewsStatusFinished {
		t.Fatalf("unexpected synced match: %#v", match)
	}

	eventItem, err := model.NewEventNewsModel(db).FindByTournamentAndSourceType(tournament.Id, "official")
	if err != nil {
		t.Fatalf("find projected event news: %v", err)
	}
	if eventItem == nil || !eventItem.Published {
		t.Fatalf("expected published projected event news, got %#v", eventItem)
	}
	if eventItem.Title != "Sportsbet.io Tour Championship 2026" || eventItem.Summary != "共 1 场比赛，当前轮次 Quarter Finals" {
		t.Fatalf("unexpected projected event news: %#v", eventItem)
	}
}

func TestServiceSyncRemovesStaleOfficialMatchesForTouchedTournament(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	tournamentModel := model.NewTournamentModel(db)
	matchModel := model.NewTournamentMatchModel(db)

	tournament := &model.Tournament{
		Id:                 99,
		Name:               "Sportsbet.io Tour Championship 2026",
		GameType:           1,
		SourceType:         "official",
		SourceTournamentId: "tour-championship-2026",
		SourceSeasonId:     "2025",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	staleMatch := &model.TournamentMatch{
		Id:             199,
		TournamentId:   tournament.Id,
		SourceType:     "official",
		SourceMatchId:  "stale-match",
		RoundName:      "Quarter Finals",
		RoundOrder:     50,
		MatchOrder:     1,
		HomePlayerName: "Stale A",
		AwayPlayerName: "Stale B",
		Status:         model.EventNewsStatusUpcoming,
	}
	if err := matchModel.Create(staleMatch); err != nil {
		t.Fatalf("seed stale match: %v", err)
	}

	client := &fakeDataClient{
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {
				{
					ID: "tour-championship-2026",
					Attributes: TournamentAttributes{
						Name:            "Sportsbet.io Tour Championship 2026",
						StartDate:       "2026-03-30",
						EndDate:         "2026-04-05",
						City:            "Manchester",
						Country:         "England",
						InformationPage: "https://www.wst.tv/tourchampionship/",
						Season:          TournamentSeason{ID: "2025"},
					},
				},
			},
		},
		matchesPages: []MatchListResponse{
			{
				Data: []MatchResource{
					{
						ID: "fresh-match",
						Attributes: MatchAttributes{
							TournamentID:    "tour-championship-2026",
							HomePlayerID:    "player-home",
							AwayPlayerID:    "player-away",
							StartDateTime:   "2026-03-30 12:00:00",
							Round:           "Quarter Finals",
							Status:          "Completed",
							HomePlayerScore: 10,
							AwayPlayerScore: 8,
							NumberOfFrames:  19,
							FixtureNumber:   1,
							PlayersAllocated:true,
							HomePlayer:      WstPlayer{PlayerID: "player-home", FirstName: "Judd", Surname: "Trump", CountryCode: "gb-eng"},
							AwayPlayer:      WstPlayer{PlayerID: "player-away", FirstName: "Mark", Surname: "Allen", CountryCode: "gb-nir"},
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	if _, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeSeason,
		Season:            2025,
		Publish:           true,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	}); err != nil {
		t.Fatalf("service sync: %v", err)
	}

	matchMap, err := matchModel.FindBySourceMatchIds("official", []string{"stale-match", "fresh-match"})
	if err != nil {
		t.Fatalf("find matches after sync: %v", err)
	}
	if _, ok := matchMap["stale-match"]; ok {
		t.Fatalf("expected stale match to be deleted, got %#v", matchMap["stale-match"])
	}
	if _, ok := matchMap["fresh-match"]; !ok {
		t.Fatalf("expected fresh match to remain, got %#v", matchMap)
	}
}

func TestServiceResolveTournamentCoverImageNormalizesRelativeWSTPaths(t *testing.T) {
	service := &Service{
		client: &fakeDataClient{
			pageCoverImages: map[string]string{
				"https://www.wst.tv/themasters": "https://images.gc.wstservices.co.uk/fit-in/1000x1000/masters-cover.png",
			},
		},
	}

	coverImage := service.resolveTournamentCoverImage(context.Background(), TournamentResource{
		ID: "masters-2025",
		Attributes: TournamentAttributes{
			InformationPage: "/themasters",
		},
	})

	if coverImage != "https://images.gc.wstservices.co.uk/fit-in/1000x1000/masters-cover.png" {
		t.Fatalf("expected normalized relative path to resolve cover image, got %q", coverImage)
	}
}
