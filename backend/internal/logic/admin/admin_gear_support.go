package admin

import (
	"context"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"chasing_points/internal/model"
	"chasing_points/internal/utils"
)

const (
	adminGearPageSizeMax = 100

	adminGearBrandNameMaxRunes   = 64
	adminGearAliasesMaxRunes     = 255
	adminGearProductNameMaxRunes = 128
	adminGearMaxLengthMm         = 3000
	adminGearMaxWeightG          = 2000
	adminGearMaxTipDiameterMm    = 30.0
)

// 目录图只接受自有 CDN 装备图目录下的 png 或 webp 文件名，不允许子目录和路径穿越。
var adminGearImagePathPattern = regexp.MustCompile(`^/static/images/gear/[A-Za-z0-9][A-Za-z0-9._-]*\.(png|webp)$`)

// requireGearAdmin 校验当前请求带的是管理员令牌：普通用户令牌（正数 ID）与匿名请求都会被拒绝。
func requireGearAdmin(ctx context.Context) bool {
	_, err := utils.GetAdminIDFromCtx(ctx)
	return err == nil
}

func normalizeAdminGearPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > adminGearPageSizeMax {
		pageSize = adminGearPageSizeMax
	}
	return page, pageSize
}

// normalizeGearAliases 统一别名分隔符，去掉空项与首尾空白。
func normalizeGearAliases(raw string) string {
	raw = strings.ReplaceAll(raw, "，", ",")
	parts := make([]string, 0, 4)
	for part := range strings.SplitSeq(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ",")
}

func isValidGearImagePath(path string) bool {
	return path == "" || adminGearImagePathPattern.MatchString(path)
}

func runeLen(text string) int { return utf8.RuneCountInString(text) }

func roundTo2(v float64) float64 { return math.Round(v*100) / 100 }

// isPendingCategory 待收录登记只可能来自球杆与冲跳装备。
func isPendingCategory(category string) bool {
	return category == model.GearCategoryCue || category == model.GearCategoryBreakJump
}
