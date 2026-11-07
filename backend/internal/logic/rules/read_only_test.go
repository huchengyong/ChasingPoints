package rules

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRuleReadEndpointsDoNotSeedContent(t *testing.T) {
	recorder := testsupport.NewSQLWriteRecorder()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: recorder})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.RulesContent{}); err != nil {
		t.Fatalf("prepare rules schema: %v", err)
	}
	recorder.Reset()
	svcCtx := &svc.ServiceContext{RulesContentModel: model.NewRulesContentModel(db)}
	ctx := context.Background()
	if resp, err := NewGetRuleContentLogic(ctx, svcCtx).GetRuleContent(&types.GetRuleContentReq{Category: "snooker", ContentType: "rule"}); err != nil || !resp.Success {
		t.Fatalf("read rule content: resp=%#v err=%v", resp, err)
	}
	if resp, err := NewGetGlossaryLogic(ctx, svcCtx).GetGlossary(&types.GetGlossaryReq{}); err != nil || !resp.Success {
		t.Fatalf("read glossary: resp=%#v err=%v", resp, err)
	}
	if resp, err := NewSearchRulesLogic(ctx, svcCtx).SearchRules(&types.SearchRulesReq{Keyword: "规则"}); err != nil || !resp.Success {
		t.Fatalf("search rules: resp=%#v err=%v", resp, err)
	}
	if resp, err := NewGetRuleCategoriesLogic(ctx, svcCtx).GetRuleCategories(); err != nil || !resp.Success {
		t.Fatalf("read categories: resp=%#v err=%v", resp, err)
	}
	var total int64
	if err := db.Model(&model.RulesContent{}).Count(&total).Error; err != nil || total != 0 {
		t.Fatalf("rule GET endpoints must not seed data: total=%d err=%v", total, err)
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("rule GET endpoints must not issue writes: %q", writes)
	}
}
