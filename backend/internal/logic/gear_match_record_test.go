package logic

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newGearRecordDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := testsupport.PrepareGearSchema(db); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	return db
}

type gearRecordEnv struct {
	t       *testing.T
	db      *gorm.DB
	gears   *model.GearModel
	svcCtx  *svc.ServiceContext
	metrics *observability.RequestMetrics
}

func newGearRecordEnv(t *testing.T) *gearRecordEnv {
	t.Helper()
	db := newGearRecordDB(t)
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	gears := model.NewGearModel(db.WithContext(ctx))
	return &gearRecordEnv{t: t, db: db, gears: gears, svcCtx: &svc.ServiceContext{GearModel: gears}, metrics: metrics}
}

func (e *gearRecordEnv) item(userId int64, category string, status int, tip, chalk int64) int64 {
	e.t.Helper()
	item := model.UserGearItem{UserId: userId, Category: category, ProductId: 1, TipProductId: tip, ChalkProductId: chalk, Status: status}
	if err := e.gears.CreateItemWithTx(nil, &item); err != nil {
		e.t.Fatalf("seed item: %v", err)
	}
	return item.Id
}

func (e *gearRecordEnv) setDefault(userId int64, gameType int, slot string, itemId int64) {
	e.t.Helper()
	if err := e.gears.UpsertDefaultWithTx(nil, &model.UserGearDefault{UserId: userId, GameType: gameType, Slot: slot, ItemId: itemId}); err != nil {
		e.t.Fatalf("seed default: %v", err)
	}
}

func (e *gearRecordEnv) snapshots(matchId int64) map[int64]model.MatchGearSnapshot {
	e.t.Helper()
	rows, err := e.gears.FindSnapshotsByMatch(matchId)
	if err != nil {
		e.t.Fatalf("find snapshots: %v", err)
	}
	byUser := map[int64]model.MatchGearSnapshot{}
	for _, row := range rows {
		byUser[row.UserId] = row
	}
	return byUser
}

func gearMatch(id int64, gameType int, player1, player2 int64) *model.Match {
	match := &model.Match{Id: id, UserId: player1, GameType: gameType, Status: 2}
	if player2 > 0 {
		match.OpponentId = &player2
	}
	return match
}

func TestRecordMatchGearSnapshotsWritesBothPlayersWithTheCurrentCombination(t *testing.T) {
	env := newGearRecordEnv(t)
	cue1 := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 11, 21)
	break1 := env.item(1, model.GearCategoryBreakJump, model.GearItemStatusActive, 0, 0)
	cue2 := env.item(2, model.GearCategoryCue, model.GearItemStatusActive, 12, 0)
	env.setDefault(1, 3, model.GearSlotCue, cue1)
	env.setDefault(1, 3, model.GearSlotBreak, break1)
	env.setDefault(2, 3, model.GearSlotCue, cue2)
	env.setDefault(1, 1, model.GearSlotCue, cue1) // 另一个球种的设置不应被带入

	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(100, 3, 1, 2)); err != nil {
		t.Fatalf("record: %v", err)
	}
	rows := env.snapshots(100)
	if len(rows) != 2 {
		t.Fatalf("expected one row per registered player, got %+v", rows)
	}
	p1, p2 := rows[1], rows[2]
	if p1.CueItemId != cue1 || p1.TipProductId != 11 || p1.ChalkProductId != 21 || p1.BreakItemId != break1 || p1.Source != model.GearSnapshotSourceAuto {
		t.Fatalf("player 1 should record cue, its current tip and chalk, and the break cue: %+v", p1)
	}
	if p2.CueItemId != cue2 || p2.TipProductId != 12 || p2.ChalkProductId != 0 || p2.BreakItemId != 0 {
		t.Fatalf("player 2 has no chalk and no break cue set: %+v", p2)
	}
}

func TestRecordMatchGearSnapshotsIsASnapshotNotALiveReference(t *testing.T) {
	env := newGearRecordEnv(t)
	cue := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 11, 21)
	env.setDefault(1, 4, model.GearSlotCue, cue)
	match := gearMatch(101, 4, 1, 0)
	if err := RecordMatchGearSnapshots(env.svcCtx, match); err != nil {
		t.Fatalf("record: %v", err)
	}

	// 对局完成后换皮头：历史记录保持当时的皮头
	if err := env.db.Model(&model.UserGearItem{}).Where("id = ?", cue).Update("tip_product_id", 99).Error; err != nil {
		t.Fatalf("change tip: %v", err)
	}
	// 重复触发后处理：不重复、不覆盖
	if err := RecordMatchGearSnapshots(env.svcCtx, match); err != nil {
		t.Fatalf("record again: %v", err)
	}
	rows := env.snapshots(101)
	if len(rows) != 1 || rows[1].TipProductId != 11 {
		t.Fatalf("a repeated trigger must leave a single, unchanged row: %+v", rows)
	}
}

func TestRecordMatchGearSnapshotsDoesNotOverwriteManualRecords(t *testing.T) {
	env := newGearRecordEnv(t)
	cue := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 11, 0)
	env.setDefault(1, 3, model.GearSlotCue, cue)
	if err := env.gears.UpsertManualSnapshot(&model.MatchGearSnapshot{MatchId: 102, UserId: 1, CueItemId: 777, ChalkProductId: 5}); err != nil {
		t.Fatalf("seed manual record: %v", err)
	}
	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(102, 3, 1, 0)); err != nil {
		t.Fatalf("record: %v", err)
	}
	row := env.snapshots(102)[1]
	if row.CueItemId != 777 || row.Source != model.GearSnapshotSourceManual {
		t.Fatalf("the automatic write must not replace a manual record: %+v", row)
	}
}

func TestRecordMatchGearSnapshotsSkipsPlayersWithoutUsableGear(t *testing.T) {
	env := newGearRecordEnv(t)
	retired := env.item(1, model.GearCategoryCue, model.GearItemStatusRetired, 0, 0)
	deleted := env.item(1, model.GearCategoryCue, model.GearItemStatusDeleted, 0, 0)
	othersCue := env.item(9, model.GearCategoryCue, model.GearItemStatusActive, 0, 0)
	breakAsCue := env.item(2, model.GearCategoryBreakJump, model.GearItemStatusActive, 0, 0)
	env.setDefault(1, 3, model.GearSlotCue, retired)
	env.setDefault(1, 3, model.GearSlotBreak, deleted)
	env.setDefault(2, 3, model.GearSlotCue, othersCue)    // 别人的装备
	env.setDefault(2, 3, model.GearSlotBreak, breakAsCue) // 有效冲杆，但球杆槽位被无效装备占用

	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(103, 3, 1, 2)); err != nil {
		t.Fatalf("record: %v", err)
	}
	rows := env.snapshots(103)
	if len(rows) != 1 {
		t.Fatalf("only player 2 has a usable break cue, got %+v", rows)
	}
	if row := rows[2]; row.CueItemId != 0 || row.BreakItemId != breakAsCue {
		t.Fatalf("the invalid cue slot must stay empty while the valid break cue is recorded: %+v", row)
	}
}

func TestRecordMatchGearSnapshotsIgnoresBreakCueInSnooker(t *testing.T) {
	env := newGearRecordEnv(t)
	cue := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 0, 0)
	breakCue := env.item(1, model.GearCategoryBreakJump, model.GearItemStatusActive, 0, 0)
	env.setDefault(1, 1, model.GearSlotCue, cue)
	env.setDefault(1, 1, model.GearSlotBreak, breakCue) // 理论上设置不出来，写入时仍要防御

	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(104, 1, 1, 0)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if row := env.snapshots(104)[1]; row.CueItemId != cue || row.BreakItemId != 0 {
		t.Fatalf("snooker matches never record a break cue: %+v", row)
	}
}

func TestRecordMatchGearSnapshotsSkipsUnregisteredOpponentAndEmptyDefaults(t *testing.T) {
	env := newGearRecordEnv(t)
	cue := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 0, 0)
	env.setDefault(1, 3, model.GearSlotCue, cue)

	// 对手不是注册用户（OpponentId 为空）：只写本人
	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(105, 3, 1, 0)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if rows := env.snapshots(105); len(rows) != 1 {
		t.Fatalf("only the registered player gets a row: %+v", rows)
	}

	// 双方都没有主力设置：不写任何记录，只有一条查询
	before := env.metrics.Snapshot().SQLCount
	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(106, 2, 5, 6)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if rows := env.snapshots(106); len(rows) != 0 {
		t.Fatalf("no defaults means no rows: %+v", rows)
	}
	// 这里的计数包含上面 snapshots 的 1 条读取，因此写入逻辑本身只花了 1 条
	if used := env.metrics.Snapshot().SQLCount - before; used != 2 {
		t.Fatalf("with no defaults the recorder must stop after one read (plus the one read in this test), used %d", used)
	}
}

func TestRecordMatchGearSnapshotsUsesAtMostThreeQueries(t *testing.T) {
	env := newGearRecordEnv(t)
	cue1 := env.item(1, model.GearCategoryCue, model.GearItemStatusActive, 11, 21)
	cue2 := env.item(2, model.GearCategoryCue, model.GearItemStatusActive, 12, 22)
	break1 := env.item(1, model.GearCategoryBreakJump, model.GearItemStatusActive, 0, 0)
	break2 := env.item(2, model.GearCategoryBreakJump, model.GearItemStatusActive, 0, 0)
	for _, d := range []struct {
		user int64
		slot string
		item int64
	}{{1, model.GearSlotCue, cue1}, {2, model.GearSlotCue, cue2}, {1, model.GearSlotBreak, break1}, {2, model.GearSlotBreak, break2}} {
		env.setDefault(d.user, 3, d.slot, d.item)
	}
	before := env.metrics.Snapshot().SQLCount
	if err := RecordMatchGearSnapshots(env.svcCtx, gearMatch(107, 3, 1, 2)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if used := env.metrics.Snapshot().SQLCount - before; used != 3 {
		t.Fatalf("expected defaults, items and one batch insert, used %d", used)
	}
}

func TestRecordMatchGearSnapshotsIsNilSafe(t *testing.T) {
	if err := RecordMatchGearSnapshots(nil, gearMatch(1, 3, 1, 2)); err != nil {
		t.Fatalf("nil service context: %v", err)
	}
	if err := RecordMatchGearSnapshots(&svc.ServiceContext{}, gearMatch(1, 3, 1, 2)); err != nil {
		t.Fatalf("service context without a gear model: %v", err)
	}
	env := newGearRecordEnv(t)
	if err := RecordMatchGearSnapshots(env.svcCtx, nil); err != nil {
		t.Fatalf("nil match: %v", err)
	}
}

// statementRecorder 记录 Gorm 实际执行的全部 SQL（读与写），用来证明某段流程没有碰指定的表。
type statementRecorder struct {
	mu         sync.Mutex
	statements []string
}

func (r *statementRecorder) LogMode(logger.LogLevel) logger.Interface    { return r }
func (*statementRecorder) Info(context.Context, string, ...interface{})  {}
func (*statementRecorder) Warn(context.Context, string, ...interface{})  {}
func (*statementRecorder) Error(context.Context, string, ...interface{}) {}

func (r *statementRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	r.mu.Lock()
	r.statements = append(r.statements, statement)
	r.mu.Unlock()
}

func (r *statementRecorder) touching(fragment string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var hits []string
	for _, statement := range r.statements {
		if strings.Contains(strings.ToLower(statement), fragment) {
			hits = append(hits, statement)
		}
	}
	return hits
}

// 竞技读模型重建只依赖对局事实，不得读取或改写对局用杆记录：这些记录是完成当时的快照，不可由事实重建。
func TestCompetitiveReadRebuildNeverTouchesGearRecords(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.AutoMigrate(&model.Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	if err := testsupport.PrepareGearSchema(db); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	draw := 3
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	matches := []model.Match{
		{Id: 1, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &draw, MatchTime: base, EndTime: &base},
		{Id: 2, UserId: 1, OpponentId: ptrInt64(2), OpponentName: "two", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &draw, MatchTime: base.Add(time.Hour), EndTime: ptrTime(base.Add(time.Hour))},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	gears := model.NewGearModel(db)
	seeded := []model.MatchGearSnapshot{
		{MatchId: 1, UserId: 1, CueItemId: 11, TipProductId: 21, ChalkProductId: 31, Source: model.GearSnapshotSourceAuto},
		{MatchId: 1, UserId: 2, CueItemId: 12, Source: model.GearSnapshotSourceManual},
		{MatchId: 2, UserId: 1, CueItemId: 13, BreakItemId: 41, Source: model.GearSnapshotSourceAuto},
	}
	if err := gears.InsertSnapshotsIgnore(seeded); err != nil {
		t.Fatalf("seed snapshots: %v", err)
	}
	before, err := gears.FindSnapshotsByMatch(1)
	if err != nil {
		t.Fatalf("read snapshots before: %v", err)
	}

	recorder := &statementRecorder{}
	db.Config.Logger = recorder
	summary, err := NewCompetitiveReadModelRebuildService(ctx).Rebuild(context.Background(), CompetitiveReadModelRebuildOptions{BatchSize: 100})
	if err != nil || !summary.CompletedAtBackfillComplete || summary.ParticipantRows == 0 {
		t.Fatalf("the rebuild must actually run to completion for this check to mean anything: %+v err=%v", summary, err)
	}
	if len(recorder.touching("match_participant_results")) == 0 {
		t.Fatal("the recorder must have seen the rebuild's own statements, otherwise the check below proves nothing")
	}
	if hits := recorder.touching("gear"); len(hits) != 0 {
		t.Fatalf("the rebuild must not read or write gear tables, but ran: %v", hits)
	}

	db.Config.Logger = logger.Default.LogMode(logger.Silent)
	after, err := gears.FindSnapshotsByMatch(1)
	if err != nil || len(after) != len(before) {
		t.Fatalf("snapshots after the rebuild: %+v err=%v", after, err)
	}
	for i := range before {
		if after[i].CueItemId != before[i].CueItemId || after[i].TipProductId != before[i].TipProductId || after[i].Source != before[i].Source || !after[i].UpdatedAt.Equal(before[i].UpdatedAt) {
			t.Fatalf("a gear record changed during the rebuild: before=%+v after=%+v", before[i], after[i])
		}
	}
}
