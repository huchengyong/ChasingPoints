package logic

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTournamentMatchPlayerBackfillTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareEventNewsSchema(db); err != nil {
		t.Fatalf("prepare event news schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                   db,
		PlayerModel:          model.NewPlayerModel(db),
		TournamentMatchModel: model.NewTournamentMatchModel(db),
	}
}

func TestTournamentMatchPlayerBackfillServiceBackfillsPlayerLinks(t *testing.T) {
	svcCtx := newTournamentMatchPlayerBackfillTestSvc(t)

	players := []*model.Player{
		{SourceType: "official", SourcePlayerId: "p1", FirstName: "Judd", LastName: "Trump", DisplayName: "Judd Trump"},
		{SourceType: "official", SourcePlayerId: "p2", FirstName: "Mark", LastName: "Allen", DisplayName: "Mark Allen"},
	}
	for _, item := range players {
		if err := svcCtx.PlayerModel.Create(item); err != nil {
			t.Fatalf("create player: %v", err)
		}
	}

	match := &model.TournamentMatch{
		TournamentId:   1,
		RoundName:      "Round 1",
		HomePlayerName: "  judd   trump  ",
		AwayPlayerName: "Mark Allen",
		WinnerSide:     1,
	}
	if err := svcCtx.TournamentMatchModel.Create(match); err != nil {
		t.Fatalf("create match: %v", err)
	}
	placeholder := &model.TournamentMatch{
		TournamentId:   1,
		RoundName:      "Final",
		HomePlayerName: "待定",
		AwayPlayerName: "待定",
	}
	if err := svcCtx.TournamentMatchModel.Create(placeholder); err != nil {
		t.Fatalf("create placeholder match: %v", err)
	}

	service := NewTournamentMatchPlayerBackfillService(svcCtx)

	dryRun, err := service.DryRun(context.Background(), 0)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if dryRun.MatchesScanned != 2 {
		t.Fatalf("expected 2 scanned matches, got %+v", dryRun)
	}
	if dryRun.HomeLinked != 1 || dryRun.AwayLinked != 1 {
		t.Fatalf("expected one home/away link in dry run, got %+v", dryRun)
	}
	if dryRun.Player1Linked != 1 || dryRun.Player2Linked != 1 || dryRun.WinnerLinked != 1 {
		t.Fatalf("expected bracket links in dry run, got %+v", dryRun)
	}
	if dryRun.PlaceholderSkipped != 1 {
		t.Fatalf("expected one placeholder skipped, got %+v", dryRun)
	}

	summary, err := service.Rebuild(context.Background(), 0)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if summary.HomeLinked != 1 || summary.AwayLinked != 1 {
		t.Fatalf("expected one home/away link in rebuild, got %+v", summary)
	}
	if summary.Player1Linked != 1 || summary.Player2Linked != 1 || summary.WinnerLinked != 1 {
		t.Fatalf("expected bracket links in rebuild, got %+v", summary)
	}

	refreshed, err := svcCtx.TournamentMatchModel.FindById(match.Id)
	if err != nil {
		t.Fatalf("refresh match: %v", err)
	}
	if refreshed.HomePlayerId != players[0].Id || refreshed.AwayPlayerId != players[1].Id {
		t.Fatalf("expected player ids backfilled, got %+v", refreshed)
	}
	if refreshed.Player1Id != players[0].Id || refreshed.Player2Id != players[1].Id {
		t.Fatalf("expected bracket player ids backfilled, got %+v", refreshed)
	}
	if refreshed.WinnerId != players[0].Id {
		t.Fatalf("expected winner id backfilled, got %+v", refreshed)
	}

	refreshedPlaceholder, err := svcCtx.TournamentMatchModel.FindById(placeholder.Id)
	if err != nil {
		t.Fatalf("refresh placeholder match: %v", err)
	}
	if refreshedPlaceholder.HomePlayerId != 0 || refreshedPlaceholder.AwayPlayerId != 0 {
		t.Fatalf("expected placeholder match to remain unlinked, got %+v", refreshedPlaceholder)
	}
}
