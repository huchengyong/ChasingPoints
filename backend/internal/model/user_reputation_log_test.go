package model

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserReputationLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&UserReputationLog{}); err != nil {
		t.Fatalf("prepare user reputation log schema: %v", err)
	}
	return db
}

func TestReputationLogTableName(t *testing.T) {
	if got := (UserReputationLog{}).TableName(); got != "user_reputation_logs" {
		t.Fatalf("expected table name user_reputation_logs, got %s", got)
	}
}

func TestReputationLogCreateRejectsDuplicateUserMatchChangeTypeAndReasonCode(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	model := NewUserReputationLogModel(db)

	matchID := int64(18)
	first := &UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -10,
		BeforeScore:     100,
		AfterScore:      90,
		ReasonCode:      ReputationReasonDurationAbnormal,
		ReasonDetail:    "3局比赛总时长只有4分钟",
		OperatorAdminID: 0,
	}
	if err := model.Create(first); err != nil {
		t.Fatalf("create first reputation log: %v", err)
	}

	err := model.Create(&UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -10,
		BeforeScore:     90,
		AfterScore:      80,
		ReasonCode:      ReputationReasonDurationAbnormal,
		ReasonDetail:    "重复写入同一异常原因",
		OperatorAdminID: 0,
	})
	if err == nil {
		t.Fatal("expected duplicate reputation log create to fail")
	}
}

func TestReputationLogCreateAllowsDifferentReasonCodesForSamePenalty(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	model := NewUserReputationLogModel(db)

	matchID := int64(28)
	if err := model.Create(&UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -8,
		BeforeScore:     100,
		AfterScore:      92,
		ReasonCode:      ReputationReasonDurationAbnormal,
		ReasonDetail:    "5局比赛总时长只有6分钟",
		OperatorAdminID: 0,
	}); err != nil {
		t.Fatalf("create first reputation log: %v", err)
	}

	if err := model.Create(&UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -8,
		BeforeScore:     92,
		AfterScore:      84,
		ReasonCode:      ReputationReasonSameOpponentHighFrequency,
		ReasonDetail:    "30分钟内同对手完成第5场",
		OperatorAdminID: 0,
	}); err != nil {
		t.Fatalf("create second reputation log with different reason code: %v", err)
	}
}

func TestReputationLogModelFindByUserMatchChangeTypeAndReasonCode(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	model := NewUserReputationLogModel(db)

	matchID := int64(88)
	if err := model.Create(&UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -8,
		BeforeScore:     100,
		AfterScore:      92,
		ReasonCode:      ReputationReasonDurationAbnormal,
		ReasonDetail:    "5局比赛总时长只有6分钟",
		OperatorAdminID: 0,
	}); err != nil {
		t.Fatalf("create reputation log: %v", err)
	}

	log, err := model.FindByUserMatchChangeTypeAndReasonCode(1001, matchID, ReputationChangeTypePenalty, ReputationReasonDurationAbnormal)
	if err != nil {
		t.Fatalf("find reputation log: %v", err)
	}
	if log == nil || log.AfterScore != 92 {
		t.Fatalf("unexpected reputation log: %+v", log)
	}

	missing, err := model.FindByUserMatchChangeTypeAndReasonCode(1001, matchID, ReputationChangeTypePenalty, ReputationReasonSameOpponentHighFrequency)
	if err != nil {
		t.Fatalf("find missing reputation log: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected missing reputation log, got %+v", missing)
	}
}

func TestReputationLogModelHelperIsMatchScoped(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	model := NewUserReputationLogModel(db)

	matchID := int64(101)
	if err := model.Create(&UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -6,
		BeforeScore:     76,
		AfterScore:      70,
		ReasonCode:      ReputationReasonSameOpponentHighFrequency,
		ReasonDetail:    "30分钟内同对手完成第5场",
		OperatorAdminID: 0,
	}); err != nil {
		t.Fatalf("create match-scoped log: %v", err)
	}

	log, err := model.FindByUserMatchChangeTypeAndReasonCode(1001, matchID, ReputationChangeTypePenalty, ReputationReasonSameOpponentHighFrequency)
	if err != nil {
		t.Fatalf("find match-scoped reputation log: %v", err)
	}
	if log == nil || log.MatchID == nil || *log.MatchID != matchID {
		t.Fatalf("unexpected match-scoped reputation log: %+v", log)
	}
}

func TestReputationLogModelReturnsHelpfulNilErrors(t *testing.T) {
	var logModel *UserReputationLogModel
	if _, err := logModel.FindByUserMatchChangeTypeAndReasonCode(1, 1, ReputationChangeTypePenalty, ReputationReasonDurationAbnormal); !errors.Is(err, ErrUserReputationLogDBNil) {
		t.Fatalf("expected nil db error, got %v", err)
	}
	if err := logModel.CreateWithTx(nil, &UserReputationLog{}); !errors.Is(err, ErrUserReputationLogDBNil) {
		t.Fatalf("expected create nil db error, got %v", err)
	}
}

func TestUserReputationLogModelFindListForAdminPaginatesResults(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seeded := seedUserReputationLogs(t, logModel)

	list, total, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:     2,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("find paginated admin logs: %v", err)
	}
	if total != int64(len(seeded)) {
		t.Fatalf("expected total %d, got %d", len(seeded), total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(list))
	}
	if list[0].ID != seeded[2].ID || list[1].ID != seeded[1].ID {
		t.Fatalf("unexpected page 2 order: got IDs %d, %d", list[0].ID, list[1].ID)
	}
}

func TestUserReputationLogModelFindListForAdminFiltersByUserID(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seedUserReputationLogs(t, logModel)

	list, total, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:     1,
		PageSize: 20,
		UserID:   2001,
	})
	if err != nil {
		t.Fatalf("find admin logs by user_id: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3 for user 2001, got %d", total)
	}
	for _, item := range list {
		if item.UserID != 2001 {
			t.Fatalf("expected only user 2001, got user %d", item.UserID)
		}
	}
}

func TestUserReputationLogModelFindListForAdminFiltersByChangeType(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seedUserReputationLogs(t, logModel)

	list, total, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:       1,
		PageSize:   20,
		ChangeType: ReputationChangeTypeRecovery,
	})
	if err != nil {
		t.Fatalf("find admin logs by change_type: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for recovery logs, got %d", total)
	}
	for _, item := range list {
		if item.ChangeType != ReputationChangeTypeRecovery {
			t.Fatalf("expected only recovery logs, got change_type %s", item.ChangeType)
		}
	}
}

func TestUserReputationLogModelFindListForAdminFiltersByReasonCode(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seedUserReputationLogs(t, logModel)

	list, total, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:       1,
		PageSize:   20,
		ReasonCode: ReputationReasonSameOpponentHighFrequency,
	})
	if err != nil {
		t.Fatalf("find admin logs by reason_code: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 for same_opponent_high_frequency, got %d", total)
	}
	if len(list) != 1 || list[0].ReasonCode != ReputationReasonSameOpponentHighFrequency {
		t.Fatalf("unexpected filtered logs: %+v", list)
	}
}

func TestUserReputationLogModelFindListForUserReturnsOnlyRequestedUsersRecords(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seedUserReputationLogs(t, logModel)

	list, total, err := logModel.FindListForUser(2002, 1, 20)
	if err != nil {
		t.Fatalf("find user reputation logs: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for user 2002, got %d", total)
	}
	for _, item := range list {
		if item.UserID != 2002 {
			t.Fatalf("expected only user 2002, got user %d", item.UserID)
		}
	}
}

func TestUserReputationLogModelFindListForAdminNormalizesInvalidPagination(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)
	seedUserReputationLogs(t, logModel)

	defaultList, defaultTotal, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:     0,
		PageSize: 0,
	})
	if err != nil {
		t.Fatalf("find admin logs with default pagination: %v", err)
	}
	if defaultTotal != 5 {
		t.Fatalf("expected total 5, got %d", defaultTotal)
	}
	if len(defaultList) != 5 {
		t.Fatalf("expected default page size to return all 5 records, got %d", len(defaultList))
	}

	for i := 0; i < 120; i++ {
		entry := UserReputationLog{
			UserID:          int64(3000 + i),
			MatchID:         int64Pointer(int64(5000 + i)),
			ChangeType:      ReputationChangeTypeRecovery,
			ChangeScore:     1,
			BeforeScore:     70,
			AfterScore:      71,
			ReasonCode:      ReputationReasonSystemRecovery,
			ReasonDetail:    "系统自然恢复 1 分",
			OperatorAdminID: 0,
			CreatedAt:       time.Date(2026, 4, 14, 11, 0, i, 0, time.UTC),
		}
		if err := logModel.Create(&entry); err != nil {
			t.Fatalf("seed pagination cap log %d: %v", i, err)
		}
	}

	cappedList, cappedTotal, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:     1,
		PageSize: 101,
	})
	if err != nil {
		t.Fatalf("find admin logs with capped page size: %v", err)
	}
	if cappedTotal != 125 {
		t.Fatalf("expected total 125 after bulk seed, got %d", cappedTotal)
	}
	if len(cappedList) != 100 {
		t.Fatalf("expected capped page size 100, got %d", len(cappedList))
	}
}

func TestUserReputationLogModelFindListForAdminOrdersByCreatedAtThenIDDesc(t *testing.T) {
	db := newUserReputationLogTestDB(t)
	logModel := NewUserReputationLogModel(db)

	sameTime := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	first := UserReputationLog{
		UserID:          4001,
		MatchID:         int64Pointer(801),
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -5,
		BeforeScore:     90,
		AfterScore:      85,
		ReasonCode:      ReputationReasonDurationAbnormal,
		ReasonDetail:    "同时间第一条",
		OperatorAdminID: 0,
		CreatedAt:       sameTime,
	}
	second := UserReputationLog{
		UserID:          4002,
		MatchID:         int64Pointer(802),
		ChangeType:      ReputationChangeTypePenalty,
		ChangeScore:     -6,
		BeforeScore:     88,
		AfterScore:      82,
		ReasonCode:      ReputationReasonSameOpponentHighFrequency,
		ReasonDetail:    "同时间第二条",
		OperatorAdminID: 0,
		CreatedAt:       sameTime,
	}
	if err := logModel.Create(&first); err != nil {
		t.Fatalf("create first same-time log: %v", err)
	}
	if err := logModel.Create(&second); err != nil {
		t.Fatalf("create second same-time log: %v", err)
	}

	list, total, err := logModel.FindListForAdmin(UserReputationLogAdminFilter{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("find admin logs ordered by id desc: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("expected later id first when created_at ties, got %+v", list)
	}
}

func TestUserReputationLogTextHelpersReturnStableDisplayText(t *testing.T) {
	if got := ReputationChangeTypeText(ReputationChangeTypePenalty); got != "扣分" {
		t.Fatalf("expected penalty text 扣分, got %s", got)
	}
	if got := ReputationChangeTypeText("unknown"); got != "信誉变更" {
		t.Fatalf("expected unknown change type text 信誉变更, got %s", got)
	}
	if got := ReputationReasonText(ReputationReasonDurationAbnormal, ReputationChangeTypePenalty); got != "时长异常" {
		t.Fatalf("expected duration reason text 时长异常, got %s", got)
	}
	if got := ReputationReasonText("", ReputationChangeTypeRecovery); got != "系统恢复" {
		t.Fatalf("expected recovery fallback text 系统恢复, got %s", got)
	}
}

func TestUserFacingReputationReasonCodeReturnsControlledEnum(t *testing.T) {
	if got := UserFacingReputationReasonCode(ReputationReasonDurationAbnormal); got != ReputationReasonDurationAbnormal {
		t.Fatalf("expected duration_abnormal to stay unchanged, got %s", got)
	}
	if got := UserFacingReputationReasonCode("manual_review"); got != "other" {
		t.Fatalf("expected unknown reason code to map to other, got %s", got)
	}
}

func seedUserReputationLogs(t *testing.T, logModel *UserReputationLogModel) []UserReputationLog {
	t.Helper()

	base := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	rows := []UserReputationLog{
		{
			UserID:          2001,
			MatchID:         int64Pointer(11),
			ChangeType:      ReputationChangeTypePenalty,
			ChangeScore:     -8,
			BeforeScore:     100,
			AfterScore:      92,
			ReasonCode:      ReputationReasonDurationAbnormal,
			ReasonDetail:    "中八 5 局总时长只有 6 分钟",
			OperatorAdminID: 0,
			CreatedAt:       base.Add(1 * time.Minute),
		},
		{
			UserID:          2001,
			MatchID:         int64Pointer(12),
			ChangeType:      ReputationChangeTypeRecovery,
			ChangeScore:     1,
			BeforeScore:     92,
			AfterScore:      93,
			ReasonCode:      ReputationReasonSystemRecovery,
			ReasonDetail:    "系统自然恢复 1 分",
			OperatorAdminID: 0,
			CreatedAt:       base.Add(2 * time.Minute),
		},
		{
			UserID:          2002,
			MatchID:         int64Pointer(13),
			ChangeType:      ReputationChangeTypePenalty,
			ChangeScore:     -6,
			BeforeScore:     88,
			AfterScore:      82,
			ReasonCode:      ReputationReasonSameOpponentHighFrequency,
			ReasonDetail:    "30 分钟内同对手完成第 5 场",
			OperatorAdminID: 0,
			CreatedAt:       base.Add(3 * time.Minute),
		},
		{
			UserID:          2002,
			MatchID:         nil,
			ChangeType:      ReputationChangeTypeRecovery,
			ChangeScore:     1,
			BeforeScore:     82,
			AfterScore:      83,
			ReasonCode:      ReputationReasonSystemRecovery,
			ReasonDetail:    "系统自然恢复 1 分",
			OperatorAdminID: 0,
			CreatedAt:       base.Add(4 * time.Minute),
		},
		{
			UserID:          2001,
			MatchID:         nil,
			ChangeType:      ReputationChangeTypeManualAdjust,
			ChangeScore:     5,
			BeforeScore:     93,
			AfterScore:      98,
			ReasonCode:      "manual_review",
			ReasonDetail:    "人工申诉恢复 5 分",
			OperatorAdminID: 9001,
			CreatedAt:       base.Add(5 * time.Minute),
		},
	}

	seeded := make([]UserReputationLog, 0, len(rows))
	for _, row := range rows {
		entry := row
		if err := logModel.Create(&entry); err != nil {
			t.Fatalf("seed user reputation log: %v", err)
		}
		seeded = append(seeded, entry)
	}
	return seeded
}

func int64Pointer(v int64) *int64 {
	return &v
}
