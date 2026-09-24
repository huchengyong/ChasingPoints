package wstsync

import (
	"context"
	"errors"
	"strings"
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

type fakeWSTImageMirror struct {
	managedPrefix string
	results       map[string]string
	errors        map[string]error
	calls         []string
	onMirror      func()
}

func (f *fakeWSTImageMirror) MirrorWSTImage(ctx context.Context, category, sourceURL string) (string, error) {
	f.calls = append(f.calls, category+"|"+sourceURL)
	if f.onMirror != nil {
		f.onMirror()
	}
	if err := f.errors[sourceURL]; err != nil {
		return "", err
	}
	return f.results[sourceURL], nil
}

func (f *fakeWSTImageMirror) IsManagedPublicURL(rawURL string) bool {
	return f.managedPrefix != "" && strings.HasPrefix(rawURL, f.managedPrefix)
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
							TournamentID:     "world-open-2025",
							HomePlayerID:     "player-a",
							AwayPlayerID:     "player-b",
							HomePlayerScore:  5,
							AwayPlayerScore:  2,
							StartDateTime:    "2025-03-16 09:00:00",
							Round:            "Round 1",
							Status:           "Completed",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "player-a", FirstName: "Player", Surname: "A", CountryCode: "cn"},
							AwayPlayer:       WstPlayer{PlayerID: "player-b", FirstName: "Player", Surname: "B", CountryCode: "gb-eng"},
						},
					},
					{
						ID: "match-2",
						Attributes: MatchAttributes{
							TournamentID:     "british-open-2025",
							HomePlayerID:     "player-c",
							AwayPlayerID:     "player-d",
							HomePlayerScore:  0,
							AwayPlayerScore:  0,
							StartDateTime:    "2025-08-25 09:00:00",
							Round:            "Round 1",
							Status:           "Scheduled",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "player-c", FirstName: "Player", Surname: "C", CountryCode: "au"},
							AwayPlayer:       WstPlayer{PlayerID: "player-d", FirstName: "Player", Surname: "D", CountryCode: "gb-sct"},
						},
					},
					{
						ID: "match-qualifier",
						Attributes: MatchAttributes{
							TournamentID:     "world-open-qualifiers-2025",
							HomePlayerID:     "player-e",
							AwayPlayerID:     "player-f",
							StartDateTime:    "2025-01-01 09:00:00",
							Round:            "Round 1",
							Status:           "Scheduled",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	mirror := &fakeWSTImageMirror{managedPrefix: "https://cdn.example.com/"}
	service.imageMirror = mirror
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
	if len(mirror.calls) != 0 {
		t.Fatalf("expected dry run to avoid image mirror calls, got %#v", mirror.calls)
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
							TournamentID:     "tour-championship-2026",
							HomePlayerID:     "player-home",
							AwayPlayerID:     "player-away",
							HomePlayerScore:  10,
							AwayPlayerScore:  8,
							StartDateTime:    "2026-03-30 12:00:00",
							Round:            "Quarter Finals",
							Status:           "Completed",
							NumberOfFrames:   19,
							FixtureNumber:    1,
							PlayersAllocated: true,
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
	mirror := &fakeWSTImageMirror{
		managedPrefix: "https://cdn.example.com/",
		results: map[string]string{
			"https://images.gc.wstservices.co.uk/fit-in/400x600/judd.png":         "https://cdn.example.com/wst/players/judd.png",
			"https://images.gc.wstservices.co.uk/fit-in/400x600/mark.png":         "https://cdn.example.com/wst/players/mark.png",
			"https://images.gc.wstservices.co.uk/fit-in/1000x1000/tour-cover.jpg": "https://cdn.example.com/wst/tournaments/tour-cover.jpg",
		},
	}
	mirror.onMirror = func() {
		var count int64
		if err := db.Model(&model.Player{}).Count(&count).Error; err != nil {
			t.Fatalf("count players during mirror: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected image mirror before database writes, got %d players", count)
		}
	}
	service.imageMirror = mirror
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
	if playerMap["player-home"].Avatar != "https://cdn.example.com/wst/players/judd.png" ||
		playerMap["player-away"].Avatar != "https://cdn.example.com/wst/players/mark.png" {
		t.Fatalf("expected mirrored player avatars, got %#v", playerMap)
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
	if tournament.CoverImage != "https://cdn.example.com/wst/tournaments/tour-cover.jpg" {
		t.Fatalf("expected mirrored cover image, got %#v", tournament)
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
	if eventItem.CoverImage != "https://cdn.example.com/wst/tournaments/tour-cover.jpg" {
		t.Fatalf("expected projected mirrored cover, got %#v", eventItem)
	}
	if len(mirror.calls) != 3 {
		t.Fatalf("expected two player mirrors and one tournament mirror, got %#v", mirror.calls)
	}
}

func TestMirrorPreparedImagesFallsBackToManagedURLs(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	playerModel := model.NewPlayerModel(db)
	tournamentModel := model.NewTournamentModel(db)

	if err := playerModel.Create(&model.Player{
		SourceType:     wstSourceType,
		SourcePlayerId: "managed-player",
		DisplayName:    "Managed Player",
		Avatar:         "https://cdn.example.com/wst/players/old.png",
	}); err != nil {
		t.Fatalf("seed managed player: %v", err)
	}
	if err := playerModel.Create(&model.Player{
		SourceType:     wstSourceType,
		SourcePlayerId: "unmanaged-player",
		DisplayName:    "Unmanaged Player",
		Avatar:         "https://images.gc.wstservices.co.uk/old.png",
	}); err != nil {
		t.Fatalf("seed unmanaged player: %v", err)
	}
	if err := tournamentModel.Create(&model.Tournament{
		Name:               "Managed Tournament",
		GameType:           1,
		SourceType:         wstSourceType,
		SourceTournamentId: "managed-tournament",
		CoverImage:         "https://cdn.example.com/wst/tournaments/old.png",
	}); err != nil {
		t.Fatalf("seed managed tournament: %v", err)
	}

	players := []PlayerUpsertRecord{
		{SourceType: wstSourceType, SourcePlayerId: "managed-player", Avatar: "https://images.gc.wstservices.co.uk/new-managed.png"},
		{SourceType: wstSourceType, SourcePlayerId: "unmanaged-player", Avatar: "https://images.gc.wstservices.co.uk/new-unmanaged.png"},
		{SourceType: wstSourceType, SourcePlayerId: "new-player", Avatar: "https://images.gc.wstservices.co.uk/new-player.png"},
	}
	tournaments := []TournamentUpsertRecord{
		{SourceType: wstSourceType, SourceTournamentId: "managed-tournament", CoverImage: "https://images.gc.wstservices.co.uk/new-cover.png"},
		{SourceType: wstSourceType, SourceTournamentId: "new-tournament", CoverImage: "https://images.gc.wstservices.co.uk/new-tournament.png"},
	}
	mirror := &fakeWSTImageMirror{
		managedPrefix: "https://cdn.example.com/",
		results:       map[string]string{},
		errors: map[string]error{
			"https://images.gc.wstservices.co.uk/new-managed.png":    errors.New("player mirror failed"),
			"https://images.gc.wstservices.co.uk/new-unmanaged.png":  errors.New("player mirror failed"),
			"https://images.gc.wstservices.co.uk/new-player.png":     errors.New("player mirror failed"),
			"https://images.gc.wstservices.co.uk/new-cover.png":      errors.New("cover mirror failed"),
			"https://images.gc.wstservices.co.uk/new-tournament.png": errors.New("cover mirror failed"),
		},
	}
	service := newWSTSyncServiceForTest(t, db, &fakeDataClient{})
	service.imageMirror = mirror

	if err := service.mirrorPreparedImages(context.Background(), players, tournaments); err != nil {
		t.Fatalf("mirror prepared images: %v", err)
	}
	if players[0].Avatar != "https://cdn.example.com/wst/players/old.png" {
		t.Fatalf("expected managed player fallback, got %#v", players[0])
	}
	if players[1].Avatar != "" || players[2].Avatar != "" {
		t.Fatalf("expected unmanaged player fallbacks to clear, got %#v", players)
	}
	if tournaments[0].CoverImage != "https://cdn.example.com/wst/tournaments/old.png" {
		t.Fatalf("expected managed tournament fallback, got %#v", tournaments[0])
	}
	if tournaments[1].CoverImage != "" {
		t.Fatalf("expected missing tournament fallback to clear, got %#v", tournaments[1])
	}
	if len(mirror.calls) != 5 {
		t.Fatalf("expected all source images to be attempted, got %#v", mirror.calls)
	}
}

func TestMirrorPreparedImagesWithoutConfiguredMirrorClearsRemoteSources(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	service := newWSTSyncServiceForTest(t, db, &fakeDataClient{})
	service.imageMirror = nil
	players := []PlayerUpsertRecord{{SourceType: wstSourceType, SourcePlayerId: "player", Avatar: "https://images.gc.wstservices.co.uk/player.png"}}
	tournaments := []TournamentUpsertRecord{{SourceType: wstSourceType, SourceTournamentId: "tournament", CoverImage: "https://images.gc.wstservices.co.uk/cover.png"}}

	if err := service.mirrorPreparedImages(context.Background(), players, tournaments); err != nil {
		t.Fatalf("mirror prepared images: %v", err)
	}
	if players[0].Avatar != "" || tournaments[0].CoverImage != "" {
		t.Fatalf("expected unavailable mirror to clear remote sources: players=%#v tournaments=%#v", players, tournaments)
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
							TournamentID:     "tour-championship-2026",
							HomePlayerID:     "player-home",
							AwayPlayerID:     "player-away",
							StartDateTime:    "2026-03-30 12:00:00",
							Round:            "Quarter Finals",
							Status:           "Completed",
							HomePlayerScore:  10,
							AwayPlayerScore:  8,
							NumberOfFrames:   19,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "player-home", FirstName: "Judd", Surname: "Trump", CountryCode: "gb-eng"},
							AwayPlayer:       WstPlayer{PlayerID: "player-away", FirstName: "Mark", Surname: "Allen", CountryCode: "gb-nir"},
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

func TestServiceResolveTournamentCoverImageIgnoresWSTDefault(t *testing.T) {
	service := &Service{
		client: &fakeDataClient{
			pageCoverImages: map[string]string{
				"https://www.wst.tv/themasters": defaultTournamentCoverImage,
			},
		},
	}

	coverImage := service.resolveTournamentCoverImage(context.Background(), TournamentResource{
		ID: "masters-2025",
		Attributes: TournamentAttributes{
			InformationPage: "/themasters",
		},
	})
	if coverImage != "" {
		t.Fatalf("expected WST default cover to be treated as missing, got %q", coverImage)
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
