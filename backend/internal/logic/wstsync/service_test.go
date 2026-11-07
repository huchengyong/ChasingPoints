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
	matchPageRequests   []int
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
	f.matchPageRequests = append(f.matchPageRequests, pageNumber)
	index := pageNumber - 1
	if index < 0 || index >= len(f.matchesPages) {
		return MatchListResponse{}, nil
	}
	return f.matchesPages[index], nil
}

func TestScanMatchesHonorsConsistentPaginationMetadata(t *testing.T) {
	client := &fakeDataClient{matchesPages: []MatchListResponse{{
		Data:  []MatchResource{{ID: "match-1", Attributes: MatchAttributes{TournamentID: "tournament-1"}}},
		Meta:  &PaginationMeta{TotalCount: intPtr(1), Count: intPtr(1)},
		Links: &PaginationLinks{Next: nil},
	}}}
	service := &Service{client: client}
	matches, scanned, pages, err := service.scanMatchesForTournaments(context.Background(), map[string]struct{}{"tournament-1": {}})
	if err != nil || len(matches) != 1 || scanned != 1 || pages != 1 || len(client.matchPageRequests) != 1 {
		t.Fatalf("unexpected completed scan: matches=%d scanned=%d pages=%d requests=%v err=%v", len(matches), scanned, pages, client.matchPageRequests, err)
	}

	next := "page-2"
	client = &fakeDataClient{matchesPages: []MatchListResponse{{
		Data:  []MatchResource{{ID: "match-1", Attributes: MatchAttributes{TournamentID: "tournament-1"}}},
		Meta:  &PaginationMeta{TotalCount: intPtr(1), Count: intPtr(1)},
		Links: &PaginationLinks{Next: &next},
	}}}
	service.client = client
	_, _, _, err = service.scanMatchesForTournaments(context.Background(), map[string]struct{}{"tournament-1": {}})
	if err == nil || !strings.Contains(err.Error(), "links.next is present") {
		t.Fatalf("expected contradictory next link to fail, got %v", err)
	}
}

func TestBackfillPreCheckDetectsSoftDeletedEventNews(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	tournament := &model.Tournament{Name: "Soft Event Parent", GameType: 1, SourceType: wstSourceType, SourceTournamentId: "soft-event-tournament"}
	if err := db.Create(tournament).Error; err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	event := &model.EventNews{Title: "Soft Event", TournamentId: tournament.Id, GameType: 1, SourceType: wstSourceType}
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("seed event news: %v", err)
	}
	if err := db.Delete(event).Error; err != nil {
		t.Fatalf("soft-delete event news: %v", err)
	}

	service := newWSTSyncServiceForTest(t, db, nil)
	err := service.preCheckBackfill(context.Background(), []TournamentUpsertRecord{{SourceTournamentId: tournament.SourceTournamentId}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "soft-deleted event news") {
		t.Fatalf("expected soft-deleted event news conflict, got %v", err)
	}
}

func TestResolveSeasonIDsForWindowIncludesUnknownNames(t *testing.T) {
	window := DateWindow{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)}
	ids, unknown := resolveSeasonIDsForWindow([]SeasonResource{
		{ID: "2024", Attributes: SeasonAttributes{Name: "2024/25"}},
		{ID: "2025", Attributes: SeasonAttributes{Name: "current"}},
		{ID: "2027", Attributes: SeasonAttributes{Name: "2027/28"}},
	}, window)
	if len(ids) != 2 || ids[0] != 2024 || ids[1] != 2025 || len(unknown) != 1 || unknown[0].ID != "2025" {
		t.Fatalf("unexpected season resolution: ids=%v unknown=%+v", ids, unknown)
	}
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
							HomePlayerScore:  intPtr(5),
							AwayPlayerScore:  intPtr(2),
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
							HomePlayerScore:  intPtr(0),
							AwayPlayerScore:  intPtr(0),
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
		seasons: []SeasonResource{
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
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
							HomePlayerScore:  intPtr(10),
							AwayPlayerScore:  intPtr(8),
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
		seasons: []SeasonResource{
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
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
							HomePlayerScore:  intPtr(10),
							AwayPlayerScore:  intPtr(8),
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

func TestBackfillModeSkipsDeleteMissingMatches(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	tournamentModel := model.NewTournamentModel(db)
	matchModel := model.NewTournamentMatchModel(db)

	tournament := &model.Tournament{
		Id:                 99,
		Name:               "Backfill Tournament",
		GameType:           1,
		SourceType:         "official",
		SourceTournamentId: "backfill-tournament",
		SourceSeasonId:     "2025",
	}
	if err := tournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	oldMatch := &model.TournamentMatch{
		Id:             199,
		TournamentId:   tournament.Id,
		SourceType:     "official",
		SourceMatchId:  "old-match",
		RoundName:      "Round 1",
		RoundOrder:     10,
		MatchOrder:     1,
		HomePlayerName: "Old A",
		AwayPlayerName: "Old B",
		Status:         model.EventNewsStatusFinished,
	}
	if err := matchModel.Create(oldMatch); err != nil {
		t.Fatalf("seed old match: %v", err)
	}

	client := &fakeDataClient{
		seasons: []SeasonResource{
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {
				{
					ID: "backfill-tournament",
					Attributes: TournamentAttributes{
						Name:      "Backfill Tournament",
						StartDate: "2025-06-01",
						EndDate:   "2025-06-07",
						City:      "Test",
						Country:   "Test",
						Season:    TournamentSeason{ID: "2025"},
					},
				},
			},
		},
		matchesPages: []MatchListResponse{
			{
				Data: []MatchResource{
					{
						ID: "new-match",
						Attributes: MatchAttributes{
							TournamentID:     "backfill-tournament",
							HomePlayerID:     "p1",
							AwayPlayerID:     "p2",
							HomePlayerScore:  intPtr(5),
							AwayPlayerScore:  intPtr(3),
							StartDateTime:    "2025-06-01 12:00:00",
							Round:            "Round 1",
							Status:           "Completed",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "p1", FirstName: "New", Surname: "A", CountryCode: "cn"},
							AwayPlayer:       WstPlayer{PlayerID: "p2", FirstName: "New", Surname: "B", CountryCode: "cn"},
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	summary, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeBackfill,
		Backfill:          true,
		From:              timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:                timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
		Publish:           true,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	})
	if err != nil {
		t.Fatalf("backfill sync: %v", err)
	}
	if summary.MatchesRetained != 1 {
		t.Fatalf("expected 1 retained match, got %d", summary.MatchesRetained)
	}

	// Old match should still exist (backfill skips delete).
	matchMap, err := matchModel.FindBySourceMatchIds("official", []string{"old-match", "new-match"})
	if err != nil {
		t.Fatalf("find matches after backfill: %v", err)
	}
	if _, ok := matchMap["old-match"]; !ok {
		t.Fatal("expected old match to be retained in backfill mode")
	}
	if _, ok := matchMap["new-match"]; !ok {
		t.Fatal("expected new match to be upserted")
	}
}

func TestBackfillModeSkipsMatchesWithoutScores(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	client := &fakeDataClient{
		seasons: []SeasonResource{
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {
				{
					ID: "backfill-tournament",
					Attributes: TournamentAttributes{
						Name:      "Backfill Tournament",
						StartDate: "2025-06-01",
						EndDate:   "2025-06-07",
						City:      "Test",
						Country:   "Test",
						Season:    TournamentSeason{ID: "2025"},
					},
				},
			},
		},
		matchesPages: []MatchListResponse{
			{
				Data: []MatchResource{
					{
						ID: "match-with-scores",
						Attributes: MatchAttributes{
							TournamentID:     "backfill-tournament",
							HomePlayerID:     "p1",
							AwayPlayerID:     "p2",
							HomePlayerScore:  intPtr(5),
							AwayPlayerScore:  intPtr(3),
							StartDateTime:    "2025-06-01 12:00:00",
							Round:            "Round 1",
							Status:           "Completed",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "p1", FirstName: "A", Surname: "B", CountryCode: "cn"},
							AwayPlayer:       WstPlayer{PlayerID: "p2", FirstName: "C", Surname: "D", CountryCode: "cn"},
						},
					},
					{
						ID: "match-without-scores",
						Attributes: MatchAttributes{
							TournamentID:     "backfill-tournament",
							HomePlayerID:     "p3",
							AwayPlayerID:     "p4",
							HomePlayerScore:  nil,
							AwayPlayerScore:  nil,
							StartDateTime:    "2025-06-02 12:00:00",
							Round:            "Round 2",
							Status:           "Completed",
							NumberOfFrames:   9,
							FixtureNumber:    2,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "p3", FirstName: "E", Surname: "F", CountryCode: "cn"},
							AwayPlayer:       WstPlayer{PlayerID: "p4", FirstName: "G", Surname: "H", CountryCode: "cn"},
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	summary, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeBackfill,
		Backfill:          true,
		From:              timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:                timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
		Publish:           true,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	})
	if err != nil {
		t.Fatalf("backfill sync: %v", err)
	}

	if summary.MatchesSkipped != 1 {
		t.Fatalf("expected 1 match skipped, got %d", summary.MatchesSkipped)
	}
	if summary.MatchesPrepared != 1 {
		t.Fatalf("expected 1 match prepared, got %d", summary.MatchesPrepared)
	}

	// Only the match with scores should be in the database.
	matchMap, err := model.NewTournamentMatchModel(db).FindBySourceMatchIds("official", []string{"match-with-scores", "match-without-scores"})
	if err != nil {
		t.Fatalf("find matches: %v", err)
	}
	if _, ok := matchMap["match-with-scores"]; !ok {
		t.Fatal("expected match with scores to be upserted")
	}
	if _, ok := matchMap["match-without-scores"]; ok {
		t.Fatal("expected match without scores to be skipped")
	}
}

func TestBackfillPreCheckDetectsSoftDeletedRecord(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	playerModel := model.NewPlayerModel(db)

	// Seed a soft-deleted player.
	softDeletedPlayer := &model.Player{
		SourceType:     "official",
		SourcePlayerId: "soft-deleted-player",
		DisplayName:    "Soft Deleted",
		CountryCode:    "cn",
	}
	if err := playerModel.Create(softDeletedPlayer); err != nil {
		t.Fatalf("seed player: %v", err)
	}
	if err := db.Delete(softDeletedPlayer).Error; err != nil {
		t.Fatalf("soft delete player: %v", err)
	}

	client := &fakeDataClient{
		seasons: []SeasonResource{
			{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}},
		},
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {
				{
					ID: "backfill-tournament",
					Attributes: TournamentAttributes{
						Name:      "Backfill Tournament",
						StartDate: "2025-06-01",
						EndDate:   "2025-06-07",
						City:      "Test",
						Country:   "Test",
						Season:    TournamentSeason{ID: "2025"},
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
							TournamentID:     "backfill-tournament",
							HomePlayerID:     "soft-deleted-player",
							AwayPlayerID:     "p2",
							HomePlayerScore:  intPtr(5),
							AwayPlayerScore:  intPtr(3),
							StartDateTime:    "2025-06-01 12:00:00",
							Round:            "Round 1",
							Status:           "Completed",
							NumberOfFrames:   9,
							FixtureNumber:    1,
							PlayersAllocated: true,
							HomePlayer:       WstPlayer{PlayerID: "soft-deleted-player", FirstName: "A", Surname: "B", CountryCode: "cn"},
							AwayPlayer:       WstPlayer{PlayerID: "p2", FirstName: "C", Surname: "D", CountryCode: "cn"},
						},
					},
				},
			},
		},
	}

	service := newWSTSyncServiceForTest(t, db, client)
	now := time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	summary, err := service.Sync(context.Background(), SyncParams{
		Mode:              SyncModeBackfill,
		Backfill:          true,
		From:              timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:                timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
		Publish:           true,
		DryRun:            false,
		IncludeQualifiers: true,
		GameType:          1,
	})
	if err != nil {
		t.Fatalf("backfill sync: %v", err)
	}

	if summary.Status != BackfillStatusFailed {
		t.Fatalf("expected failed status due to soft-deleted player, got %s", summary.Status)
	}
	if summary.ExitCode != ExitCodeFailed {
		t.Fatalf("expected exit code %d, got %d", ExitCodeFailed, summary.ExitCode)
	}
}

func TestBackfillDryRunChecksSoftDeletedRecord(t *testing.T) {
	db := newWSTSyncServiceTestDB(t)
	player := &model.Player{SourceType: "official", SourcePlayerId: "p1", DisplayName: "Deleted"}
	if err := db.Create(player).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(player).Error; err != nil {
		t.Fatal(err)
	}
	client := &fakeDataClient{
		seasons: []SeasonResource{{ID: "2025", Attributes: SeasonAttributes{Name: "2025/26"}}},
		tournamentsBySeason: map[int][]TournamentResource{
			2025: {{ID: "t1", Attributes: TournamentAttributes{Name: "Test", StartDate: "2025-01-01", EndDate: "2025-01-02"}}},
		},
		matchesPages: []MatchListResponse{{
			Data: []MatchResource{{
				ID: "m1",
				Attributes: MatchAttributes{
					TournamentID: "t1", HomePlayerID: "p1", AwayPlayerID: "p2", HomePlayerScore: intPtr(1), AwayPlayerScore: intPtr(0), Status: "Completed", PlayersAllocated: true,
					HomePlayer: WstPlayer{PlayerID: "p1"}, AwayPlayer: WstPlayer{PlayerID: "p2"},
				},
			}},
		}},
	}
	summary, err := newWSTSyncServiceForTest(t, db, client).Sync(context.Background(), SyncParams{Mode: SyncModeBackfill, Backfill: true, DryRun: true, GameType: 1, IncludeQualifiers: true, From: timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)), To: timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))})
	if err != nil || summary.Status != BackfillStatusFailed {
		t.Fatalf("expected dry-run pre-check failure: status=%s err=%v", summary.Status, err)
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestBuildFlagEmoji(t *testing.T) {
	if got := buildFlagEmoji("au"); got != "\U0001F1E6\U0001F1FA" {
		t.Fatalf("expected AU flag, got %q", got)
	}
	if got := buildFlagEmoji("CN"); got != "\U0001F1E8\U0001F1F3" {
		t.Fatalf("expected CN flag, got %q", got)
	}
	if got := buildFlagEmoji(""); got != "" {
		t.Fatalf("expected empty for blank code, got %q", got)
	}

	// England, Scotland, Wales subdivision flags
	eng := buildFlagEmoji("gb-eng")
	if !strings.Contains(eng, "\U0001F3F4") {
		t.Fatalf("expected England flag to contain black flag base, got %q", eng)
	}
	if len([]rune(eng)) != 7 {
		t.Fatalf("expected England flag to have 7 runes (base + g+b+e+n+g + cancel), got %d: %q", len([]rune(eng)), eng)
	}

	sco := buildFlagEmoji("gb-sct")
	if !strings.Contains(sco, "\U0001F3F4") {
		t.Fatalf("expected Scotland flag to contain black flag base, got %q", sco)
	}
	if len([]rune(sco)) != 7 {
		t.Fatalf("expected Scotland flag to have 7 runes, got %d: %q", len([]rune(sco)), sco)
	}

	wls := buildFlagEmoji("gb-wls")
	if !strings.Contains(wls, "\U0001F3F4") {
		t.Fatalf("expected Wales flag to contain black flag base, got %q", wls)
	}
	if len([]rune(wls)) != 7 {
		t.Fatalf("expected Wales flag to have 7 runes, got %d: %q", len([]rune(wls)), wls)
	}

	// Northern Ireland and unknown UK subdivisions should return empty
	if got := buildFlagEmoji("gb-nir"); got != "" {
		t.Fatalf("expected empty for Northern Ireland, got %q", got)
	}
	if got := buildFlagEmoji("gb-xxx"); got != "" {
		t.Fatalf("expected empty for unknown subdivision, got %q", got)
	}
	if got := buildFlagEmoji("gb-"); got != "" {
		t.Fatalf("expected empty for bare gb-, got %q", got)
	}

	// Non-UK gb-* codes (like gb-aus) are not subdivision flags, return empty
	if got := buildFlagEmoji("gb-aus"); got != "" {
		t.Fatalf("expected empty for non-UK gb-aus, got %q", got)
	}
}
