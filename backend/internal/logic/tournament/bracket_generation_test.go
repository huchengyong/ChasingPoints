package tournament

import (
	"context"
	"errors"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTournamentBracketTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Tournament{}, &model.TournamentParticipant{}, &model.TournamentMatch{}, &model.User{}); err != nil {
		t.Fatalf("prepare tournament bracket schema: %v", err)
	}
	return &svc.ServiceContext{DB: db, TournamentModel: model.NewTournamentModel(db), TournamentParticipantModel: model.NewTournamentParticipantModel(db), TournamentMatchModel: model.NewTournamentMatchModel(db), UserModel: model.NewUserModel(db)}
}

func seedActiveTournamentWithoutBracket(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()
	if err := svcCtx.DB.Create(&model.Tournament{Id: 1, Name: "赛事", Format: 1, Status: 1}).Error; err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.User{{Id: 11, Nickname: "甲"}, {Id: 22, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.TournamentParticipant{{TournamentId: 1, UserId: 11}, {TournamentId: 1, UserId: 22}}).Error; err != nil {
		t.Fatalf("seed participants: %v", err)
	}
}

func TestTournamentBracketReadDoesNotGenerateMissingBracket(t *testing.T) {
	svcCtx := newTournamentBracketTestSvc(t)
	seedActiveTournamentWithoutBracket(t, svcCtx)
	recorder := testsupport.NewSQLWriteRecorder()
	readDB := svcCtx.DB.Session(&gorm.Session{Logger: recorder})
	svcCtx.TournamentModel = model.NewTournamentModel(readDB)
	svcCtx.TournamentMatchModel = model.NewTournamentMatchModel(readDB)
	svcCtx.UserModel = model.NewUserModel(readDB)

	resp, err := NewGetTournamentBracketLogic(context.Background(), svcCtx).GetTournamentBracket(&types.GetTournamentBracketReq{TournamentId: 1})
	if err != nil || !resp.Success || len(resp.Matches) != 0 {
		t.Fatalf("get empty pending bracket: resp=%#v err=%v", resp, err)
	}
	matches, err := svcCtx.TournamentMatchModel.FindByTournament(1)
	if err != nil || len(matches) != 0 {
		t.Fatalf("GET must not persist a bracket: matches=%#v err=%v", matches, err)
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("bracket GET must not issue writes: %q", writes)
	}
}

func TestTournamentBracketWorkerProcessesFixedBatches(t *testing.T) {
	svcCtx := newTournamentBracketTestSvc(t)
	tournaments := make([]model.Tournament, 0, tournamentBracketWorkerBatchSize+1)
	participants := make([]model.TournamentParticipant, 0, (tournamentBracketWorkerBatchSize+1)*2)
	for id := int64(1); id <= tournamentBracketWorkerBatchSize+1; id++ {
		tournaments = append(tournaments, model.Tournament{Id: id, Name: "赛事", Format: 1, Status: 1})
		participants = append(participants,
			model.TournamentParticipant{TournamentId: id, UserId: id*10 + 1},
			model.TournamentParticipant{TournamentId: id, UserId: id*10 + 2},
		)
	}
	if err := svcCtx.DB.Create(&tournaments).Error; err != nil {
		t.Fatalf("seed tournaments: %v", err)
	}
	if err := svcCtx.DB.Create(&participants).Error; err != nil {
		t.Fatalf("seed participants: %v", err)
	}
	worker := NewTournamentBracketGenerationWorker(svcCtx)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("first worker batch: %v", err)
	}
	var count int64
	if err := svcCtx.DB.Model(&model.TournamentMatch{}).Count(&count).Error; err != nil || count != tournamentBracketWorkerBatchSize {
		t.Fatalf("first worker batch match count=%d err=%v", count, err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("second worker batch: %v", err)
	}
	if err := svcCtx.DB.Model(&model.TournamentMatch{}).Count(&count).Error; err != nil || count != tournamentBracketWorkerBatchSize+1 {
		t.Fatalf("second worker batch match count=%d err=%v", count, err)
	}
}

func TestTournamentBracketWorkerRetriesFailedItemsAndHonorsCancellation(t *testing.T) {
	svcCtx := newTournamentBracketTestSvc(t)
	for _, tournament := range []model.Tournament{
		{Id: 1, Name: "无效赛制", Format: 99, Status: 1},
		{Id: 2, Name: "可生成赛制", Format: 1, Status: 1},
	} {
		if err := svcCtx.DB.Create(&tournament).Error; err != nil {
			t.Fatalf("seed tournament: %v", err)
		}
	}
	if err := svcCtx.DB.Create(&[]model.TournamentParticipant{
		{TournamentId: 1, UserId: 11}, {TournamentId: 1, UserId: 12},
		{TournamentId: 2, UserId: 21}, {TournamentId: 2, UserId: 22},
	}).Error; err != nil {
		t.Fatalf("seed participants: %v", err)
	}
	worker := NewTournamentBracketGenerationWorker(svcCtx)
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("unsupported tournament must be reported for retry")
	}
	matches, err := svcCtx.TournamentMatchModel.FindByTournament(2)
	if err != nil || len(matches) != 1 {
		t.Fatalf("one failed tournament must not block later items: matches=%#v err=%v", matches, err)
	}
	if err := svcCtx.DB.Model(&model.Tournament{}).Where("id = ?", 1).Update("format", 1).Error; err != nil {
		t.Fatalf("repair tournament format: %v", err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("retry repaired tournament: %v", err)
	}
	matches, err = svcCtx.TournamentMatchModel.FindByTournament(1)
	if err != nil || len(matches) != 1 {
		t.Fatalf("repaired tournament must be generated on retry: matches=%#v err=%v", matches, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.RunOnce(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker run error=%v, want context.Canceled", err)
	}
}

func TestTournamentBracketWorkerGeneratesOnceAndReadUsesBatchProfiles(t *testing.T) {
	svcCtx := newTournamentBracketTestSvc(t)
	seedActiveTournamentWithoutBracket(t, svcCtx)
	worker := NewTournamentBracketGenerationWorker(svcCtx)
	worker.RunOnce()
	worker.RunOnce()
	matches, err := svcCtx.TournamentMatchModel.FindByTournament(1)
	if err != nil || len(matches) != 1 || matches[0].Player1Id != 11 || matches[0].Player2Id != 22 {
		t.Fatalf("worker must generate one idempotent bracket: matches=%#v err=%v", matches, err)
	}
	resp, err := NewGetTournamentBracketLogic(context.Background(), svcCtx).GetTournamentBracket(&types.GetTournamentBracketReq{TournamentId: 1})
	if err != nil || !resp.Success || len(resp.Matches) != 1 || resp.Matches[0].Player1Name != "甲" || resp.Matches[0].Player2Name != "乙" {
		t.Fatalf("read generated bracket: resp=%#v err=%v", resp, err)
	}
}
