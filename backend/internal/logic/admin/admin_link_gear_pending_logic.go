package admin

import (
	"context"
	"fmt"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminLinkGearPendingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 把未收录型号关联到目录型号（管理员）
func NewAdminLinkGearPendingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminLinkGearPendingLogic {
	return &AdminLinkGearPendingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminLinkGearPendingLogic) AdminLinkGearPending(req *types.AdminGearPendingLinkReq) (resp *types.AdminGearPendingLinkResp, err error) {
	fail := func(code int, message string) *types.AdminGearPendingLinkResp {
		return &types.AdminGearPendingLinkResp{Code: code, Success: false, Message: message}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	customKey := ""
	if req != nil {
		customKey = strings.TrimSpace(req.CustomKey)
	}
	if customKey == "" || req.ProductId <= 0 {
		return fail(400, "请求参数错误"), nil
	}
	category := model.GearCustomKeyCategory(customKey)
	if !isPendingCategory(category) {
		return fail(400, "归并键无效"), nil
	}

	product, err := l.svcCtx.GearModel.FindProduct(req.ProductId)
	if err != nil {
		l.Logger.Errorf("读取装备型号失败: id=%d err=%v", req.ProductId, err)
		return fail(500, "关联失败"), nil
	}
	if product == nil {
		return fail(404, "目标型号不存在"), nil
	}
	if product.Category != category {
		return fail(400, "目标型号的类别与待收录登记不一致"), nil
	}
	if product.Status != 1 {
		return fail(400, "目标型号已停用，请先启用或选择其他型号"), nil
	}

	pending, err := l.svcCtx.GearModel.FindFirstByCustomKey(customKey)
	if err != nil {
		l.Logger.Errorf("读取待收录登记失败: err=%v", err)
		return fail(500, "关联失败"), nil
	}
	if pending == nil {
		return fail(404, "该分组不存在或已全部关联"), nil
	}

	updated, remaining, err := l.svcCtx.GearModel.LinkPending(customKey, product.Id, model.GearPendingLinkBatchSize)
	if err != nil {
		l.Logger.Errorf("关联待收录登记失败: productId=%d err=%v", product.Id, err)
		return fail(500, "关联失败"), nil
	}

	message := fmt.Sprintf("已关联 %d 条登记", updated)
	if remaining > 0 {
		message = fmt.Sprintf("已关联 %d 条登记，还剩 %d 条，可以再次执行", updated, remaining)
	}
	return &types.AdminGearPendingLinkResp{Code: 0, Success: true, Message: message, Updated: updated, Remaining: remaining}, nil
}
