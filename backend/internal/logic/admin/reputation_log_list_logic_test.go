package admin

import (
	"context"
	"reflect"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAdminReputationLogTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserReputationLog{}); err != nil {
		t.Fatalf("prepare admin reputation log schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                     db,
		UserModel:              model.NewUserModel(db),
		UserReputationLogModel: model.NewUserReputationLogModel(db),
	}
}

func mustSeedAdminReputationUser(t *testing.T, svcCtx *svc.ServiceContext, user *model.User) {
	t.Helper()
	if err := svcCtx.UserModel.Create(user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func mustSeedAdminReputationLog(t *testing.T, svcCtx *svc.ServiceContext, log *model.UserReputationLog) {
	t.Helper()
	if err := svcCtx.UserReputationLogModel.Create(log); err != nil {
		t.Fatalf("seed reputation log: %v", err)
	}
}

func TestAdminGetReputationLogListSupportsPaginationAndFilters(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)
	now := time.Date(2026, 4, 14, 18, 30, 0, 0, time.UTC)
	matchID := int64(9001)

	mustSeedAdminReputationUser(t, svcCtx, &model.User{Id: 1001, Nickname: "Alice", Status: 1})
	mustSeedAdminReputationUser(t, svcCtx, &model.User{Id: 1002, Nickname: "Bob", Status: 1})

	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          1001,
		MatchID:         &matchID,
		ChangeType:      model.ReputationChangeTypePenalty,
		ChangeScore:     -10,
		BeforeScore:     90,
		AfterScore:      80,
		ReasonCode:      model.ReputationReasonDurationAbnormal,
		ReasonDetail:    "中式八球 10 局总时长 5 分钟",
		OperatorAdminID: 0,
		CreatedAt:       now,
	})
	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          1001,
		ChangeType:      model.ReputationChangeTypeRecovery,
		ChangeScore:     1,
		BeforeScore:     80,
		AfterScore:      81,
		ReasonCode:      model.ReputationReasonSystemRecovery,
		ReasonDetail:    "自然恢复 1 分",
		OperatorAdminID: 0,
		CreatedAt:       now.Add(time.Minute),
	})
	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          1002,
		ChangeType:      model.ReputationChangeTypePenalty,
		ChangeScore:     -8,
		BeforeScore:     88,
		AfterScore:      80,
		ReasonCode:      model.ReputationReasonSameOpponentHighFrequency,
		ReasonDetail:    "30 分钟内与同一对手完成 6 场",
		OperatorAdminID: 42,
		CreatedAt:       now.Add(2 * time.Minute),
	})

	req := &types.AdminReputationLogListReq{
		Page:       1,
		PageSize:   1,
		UserId:     1001,
		ChangeType: model.ReputationChangeTypePenalty,
		ReasonCode: model.ReputationReasonDurationAbnormal,
	}

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(req)
	if err != nil {
		t.Fatalf("get admin reputation logs: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected filtered pagination result, got %#v", resp)
	}

	item := resp.List[0]
	if item.UserId != 1001 || item.Nickname != "Alice" {
		t.Fatalf("expected user filter to keep Alice only, got %#v", item)
	}
	if item.ChangeType != model.ReputationChangeTypePenalty || item.ChangeTypeText != "扣分" {
		t.Fatalf("expected penalty change type display, got %#v", item)
	}
	if item.ReasonCode != model.ReputationReasonDurationAbnormal || item.ReasonText != "时长异常" {
		t.Fatalf("expected duration abnormal display text, got %#v", item)
	}
	if item.OperatorText != "系统" {
		t.Fatalf("expected system operator label, got %#v", item)
	}
	if item.ReasonDetail == "" {
		t.Fatalf("expected admin log item to retain reason detail, got %#v", item)
	}
}

func TestAdminGetReputationLogListExposesOperatorTextForManualAdjustments(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)

	mustSeedAdminReputationUser(t, svcCtx, &model.User{Id: 1003, Nickname: "Carol", Status: 1})
	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:          1003,
		ChangeType:      model.ReputationChangeTypeManualAdjust,
		ChangeScore:     -5,
		BeforeScore:     70,
		AfterScore:      65,
		ReasonCode:      "manual_review",
		ReasonDetail:    "管理员手动调整",
		OperatorAdminID: 42,
	})

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(&types.AdminReputationLogListReq{
		Page:       1,
		PageSize:   20,
		UserId:     1003,
		ChangeType: model.ReputationChangeTypeManualAdjust,
		ReasonCode: "manual_review",
	})
	if err != nil {
		t.Fatalf("get admin reputation logs: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected one manual adjustment log, got %#v", resp)
	}
	if resp.List[0].OperatorText != "管理员 #42" {
		t.Fatalf("expected admin operator label, got %#v", resp.List[0])
	}
}

func TestAdminReputationLogListContractStructs(t *testing.T) {
	if _, ok := reflect.TypeOf(types.AdminReputationLogItem{}).FieldByName("ReasonDetail"); !ok {
		t.Fatalf("expected admin reputation log item to expose ReasonDetail")
	}
	if _, ok := reflect.TypeOf(types.AdminReputationLogItem{}).FieldByName("OperatorText"); !ok {
		t.Fatalf("expected admin reputation log item to expose OperatorText")
	}
}

func TestAdminGetReputationLogListReturnsFailureWhenDependencyMissing(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)
	svcCtx.UserReputationLogModel = nil

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(&types.AdminReputationLogListReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Success || resp.Code != 500 || resp.Message != "获取信誉日志失败" {
		t.Fatalf("expected standard failure response, got %#v", resp)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list on dependency failure, got %#v", resp)
	}
}

func TestAdminGetReputationLogListReturnsFailureWhenQueryFails(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)
	if err := svcCtx.DB.Exec("DROP TABLE user_reputation_logs").Error; err != nil {
		t.Fatalf("drop reputation log table: %v", err)
	}

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(&types.AdminReputationLogListReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Success || resp.Code != 500 || resp.Message != "获取信誉日志失败" {
		t.Fatalf("expected standard failure response on query failure, got %#v", resp)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list on query failure, got %#v", resp)
	}
}

func TestAdminGetReputationLogListKeepsLogsWhenUserIsMissing(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)

	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:       404,
		ChangeType:   model.ReputationChangeTypePenalty,
		ChangeScore:  -10,
		BeforeScore:  90,
		AfterScore:   80,
		ReasonCode:   model.ReputationReasonDurationAbnormal,
		ReasonDetail: "用户记录缺失时仍应展示日志",
	})

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(&types.AdminReputationLogListReq{
		Page:     1,
		PageSize: 20,
		UserId:   404,
	})
	if err != nil {
		t.Fatalf("get admin reputation logs: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("expected log list success when user is missing, got %#v", resp)
	}
	if resp.List[0].Nickname != "" {
		t.Fatalf("expected empty nickname for missing user, got %#v", resp.List[0])
	}
}

func TestAdminGetReputationLogListKeepsLogsWhenUserLookupFails(t *testing.T) {
	svcCtx := newAdminReputationLogTestSvc(t)

	mustSeedAdminReputationLog(t, svcCtx, &model.UserReputationLog{
		UserID:       505,
		ChangeType:   model.ReputationChangeTypePenalty,
		ChangeScore:  -10,
		BeforeScore:  90,
		AfterScore:   80,
		ReasonCode:   model.ReputationReasonDurationAbnormal,
		ReasonDetail: "用户表异常时仍应展示日志",
	})
	if err := svcCtx.DB.Exec("DROP TABLE users").Error; err != nil {
		t.Fatalf("drop users table: %v", err)
	}

	resp, err := NewAdminGetReputationLogListLogic(context.Background(), svcCtx).AdminGetReputationLogList(&types.AdminReputationLogListReq{
		Page:     1,
		PageSize: 20,
		UserId:   505,
	})
	if err != nil {
		t.Fatalf("get admin reputation logs: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("expected log list success when user lookup fails, got %#v", resp)
	}
	if resp.List[0].Nickname != "" {
		t.Fatalf("expected empty nickname when user lookup fails, got %#v", resp.List[0])
	}
}
