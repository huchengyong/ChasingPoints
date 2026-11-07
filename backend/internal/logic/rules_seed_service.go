package logic

import (
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

func EnsureDefaultRulesContent(svcCtx *svc.ServiceContext) error {
	if svcCtx == nil || svcCtx.RulesContentModel == nil {
		return fmt.Errorf("rules content model is unavailable")
	}
	return svcCtx.RulesContentModel.SeedData(model.GetDefaultRulesContent())
}
