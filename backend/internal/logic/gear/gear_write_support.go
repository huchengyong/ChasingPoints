package gear

import (
	"errors"
	"unicode/utf8"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

const (
	brandNameMaxRunes = 32
	modelNameMaxRunes = 64
)

// userError 是可以直接展示给用户的业务错误；其他错误只记日志，对外统一为通用提示。
type userError struct{ message string }

func (e *userError) Error() string { return e.message }

func newUserError(message string) error { return &userError{message: message} }

// userMessage 取业务错误的提示，非业务错误返回 fallback。
func userMessage(err error, fallback string) string {
	var ue *userError
	if errors.As(err, &ue) {
		return ue.message
	}
	return fallback
}

// withUserLock 在事务里先锁住用户行，使同一用户的装备写操作串行执行，
// 这样数量上限与主力设置的校验不会被并发请求绕过。
func withUserLock(svcCtx *svc.ServiceContext, userId int64, fn func(tx *gorm.DB) error) error {
	return svcCtx.GearModel.DB().Transaction(func(tx *gorm.DB) error {
		if _, err := svcCtx.UserModel.FindByIdForUpdateWithTx(tx, userId); err != nil {
			return err
		}
		return fn(tx)
	})
}

// checkSubProduct 校验皮头或巧克型号；id 为 0 表示留空。
// requireEnabled 为 true 时型号及其品牌必须仍在启用状态（设置新值时），为 false 时允许已停用的型号（对局记录保留原值）。
func checkSubProduct(gears *model.GearModel, id int64, category string, requireEnabled bool) error {
	if id == 0 {
		return nil
	}
	label := "皮头"
	if category == model.GearCategoryChalk {
		label = "巧克"
	}
	product, err := gears.FindProduct(id)
	if err != nil {
		return err
	}
	if product == nil || product.Category != category {
		return newUserError(label + "型号不存在")
	}
	if requireEnabled {
		return checkProductAvailable(gears, product, label)
	}
	return nil
}

// checkProductAvailable 校验型号和它的品牌都在启用状态。
func checkProductAvailable(gears *model.GearModel, product *model.GearProduct, label string) error {
	if product.Status != 1 {
		return newUserError(label + "型号已下架")
	}
	if product.BrandId > 0 {
		brand, err := gears.FindBrand(product.BrandId)
		if err != nil {
			return err
		}
		if brand == nil || brand.Status != 1 {
			return newUserError(label + "型号已下架")
		}
	}
	return nil
}

// itemRoles 返回装备实际的冲跳身份：关联目录型号的以目录为准，未收录的用登记时选择的身份。
func itemRoles(gears *model.GearModel, item *model.UserGearItem) (string, error) {
	if item.ProductId == 0 {
		return item.Roles, nil
	}
	product, err := gears.FindProduct(item.ProductId)
	if err != nil {
		return "", err
	}
	if product == nil {
		return item.Roles, nil
	}
	return product.Roles, nil
}

// isOwnedAvailableItem 判断装备属于该用户且未删除。
func isOwnedAvailableItem(item *model.UserGearItem, userId int64) bool {
	return item != nil && item.UserId == userId && item.Status != model.GearItemStatusDeleted
}

func runeLength(text string) int { return utf8.RuneCountInString(text) }
