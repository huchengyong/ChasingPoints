package model

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 装备类别。目录型号四类都有；用户装备只有 cue 与 break_jump。
const (
	GearCategoryCue       = "cue"
	GearCategoryBreakJump = "break_jump"
	GearCategoryTip       = "tip"
	GearCategoryChalk     = "chalk"
)

// 球杆杆型与冲跳身份。
const (
	GearCueStyleSnooker = "snooker"
	GearCueStylePool    = "pool"

	GearRoleBreak = "break"
	GearRoleJump  = "jump"
)

// 用户装备状态。已删除的装备只做软删除，对局用杆记录仍要能解析它。
const (
	GearItemStatusActive  = 1
	GearItemStatusRetired = 2
	GearItemStatusDeleted = 3
)

// 主力设置槽位与对局用杆记录来源。
const (
	GearSlotCue   = "cue"
	GearSlotBreak = "break"

	GearSnapshotSourceAuto   = "auto"
	GearSnapshotSourceManual = "manual"
)

const (
	// GearMaxItemsPerUser 是每位用户未删除装备（在用与退役）的上限。
	GearMaxItemsPerUser = 30
	// GearCatalogMaxPageSize 是目录型号列表的单页上限。
	GearCatalogMaxPageSize = 50
	// GearPendingLinkBatchSize 是后台一次关联未收录登记的最大条数。
	GearPendingLinkBatchSize = 500
	// GearSnapshotEditWindow 是对局完成后选手可以修改或补记本场用杆的期限。
	GearSnapshotEditWindow = 7 * 24 * time.Hour
)

var (
	ErrGearDBNil     = errors.New("gear db is nil")
	ErrGearDuplicate = errors.New("gear catalog entry already exists")
)

// GearBrand 是平台维护的装备品牌。
type GearBrand struct {
	Id        int64     `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"size:64;not null;uniqueIndex:uniq_gear_brands_name" json:"name"`
	Aliases   string    `gorm:"size:255;not null" json:"aliases"`
	Sort      int       `gorm:"not null" json:"sort"`
	Status    int       `gorm:"not null" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (GearBrand) TableName() string {
	return "gear_brands"
}

// GearProduct 是平台维护的装备型号。规格为 0 表示品牌未提供；BrandId 为 0 表示无品牌（球房公用条目）。
type GearProduct struct {
	Id            int64     `gorm:"primarykey" json:"id"`
	Category      string    `gorm:"size:16;not null;uniqueIndex:uniq_gear_products_identity,priority:1;index:idx_gear_products_list,priority:1" json:"category"`
	BrandId       int64     `gorm:"not null;uniqueIndex:uniq_gear_products_identity,priority:2;index:idx_gear_products_list,priority:3" json:"brand_id"`
	Name          string    `gorm:"size:128;not null;uniqueIndex:uniq_gear_products_identity,priority:3" json:"name"`
	House         bool      `gorm:"not null" json:"house"`
	CueStyle      string    `gorm:"size:16;not null" json:"cue_style"`
	Roles         string    `gorm:"size:32;not null" json:"roles"`
	LengthMm      int       `gorm:"not null" json:"length_mm"`
	WeightG       int       `gorm:"not null" json:"weight_g"`
	TipDiameterMm float64   `gorm:"type:decimal(5,2);not null" json:"tip_diameter_mm"`
	ImagePath     string    `gorm:"size:255;not null" json:"image_path"`
	Sort          int       `gorm:"not null;index:idx_gear_products_list,priority:4" json:"sort"`
	Status        int       `gorm:"not null;index:idx_gear_products_list,priority:2" json:"status"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (GearProduct) TableName() string {
	return "gear_products"
}

// UserGearItem 是用户登记的一件装备（球杆或冲跳装备）。ProductId 为 0 表示未收录型号，CustomXxx 保存自填内容。
type UserGearItem struct {
	Id              int64     `gorm:"primarykey" json:"id"`
	UserId          int64     `gorm:"not null;index:idx_user_gear_items_user_status,priority:1" json:"user_id"`
	Category        string    `gorm:"size:16;not null" json:"category"`
	ProductId       int64     `gorm:"not null;index:idx_user_gear_items_pending,priority:1" json:"product_id"`
	CustomBrandId   int64     `gorm:"not null" json:"custom_brand_id"`
	CustomBrandName string    `gorm:"size:32;not null" json:"custom_brand_name"`
	CustomModelName string    `gorm:"size:64;not null" json:"custom_model_name"`
	CustomKey       string    `gorm:"size:255;not null;index:idx_user_gear_items_pending,priority:2" json:"custom_key"`
	CueStyle        string    `gorm:"size:16;not null" json:"cue_style"`
	Roles           string    `gorm:"size:32;not null" json:"roles"`
	TipProductId    int64     `gorm:"not null" json:"tip_product_id"`
	ChalkProductId  int64     `gorm:"not null" json:"chalk_product_id"`
	Status          int       `gorm:"not null;index:idx_user_gear_items_user_status,priority:2" json:"status"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserGearItem) TableName() string {
	return "user_gear_items"
}

// UserGearDefault 是用户在某个球种的主力装备设置。
type UserGearDefault struct {
	Id        int64     `gorm:"primarykey" json:"id"`
	UserId    int64     `gorm:"not null;uniqueIndex:uniq_user_gear_defaults,priority:1" json:"user_id"`
	GameType  int       `gorm:"not null;uniqueIndex:uniq_user_gear_defaults,priority:2" json:"game_type"`
	Slot      string    `gorm:"size:8;not null;uniqueIndex:uniq_user_gear_defaults,priority:3" json:"slot"`
	ItemId    int64     `gorm:"not null" json:"item_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserGearDefault) TableName() string {
	return "user_gear_defaults"
}

// MatchGearSnapshot 是一位选手在一场对局里的用杆记录：球杆与冲杆存装备 ID，皮头与巧克存记录当时的型号 ID。
type MatchGearSnapshot struct {
	Id             int64     `gorm:"primarykey" json:"id"`
	MatchId        int64     `gorm:"not null;uniqueIndex:uniq_match_gear_snapshots,priority:1;index:idx_match_gear_snapshots_user,priority:2" json:"match_id"`
	UserId         int64     `gorm:"not null;uniqueIndex:uniq_match_gear_snapshots,priority:2;index:idx_match_gear_snapshots_user,priority:1" json:"user_id"`
	CueItemId      int64     `gorm:"not null" json:"cue_item_id"`
	TipProductId   int64     `gorm:"not null" json:"tip_product_id"`
	ChalkProductId int64     `gorm:"not null" json:"chalk_product_id"`
	BreakItemId    int64     `gorm:"not null" json:"break_item_id"`
	Source         string    `gorm:"size:16;not null" json:"source"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MatchGearSnapshot) TableName() string {
	return "match_gear_snapshots"
}

// IsEmpty 表示这条记录没有任何装备信息。
func (s MatchGearSnapshot) IsEmpty() bool {
	return s.CueItemId == 0 && s.TipProductId == 0 && s.ChalkProductId == 0 && s.BreakItemId == 0
}

// IsGearCategory 判断是否是目录支持的类别。
func IsGearCategory(category string) bool {
	switch category {
	case GearCategoryCue, GearCategoryBreakJump, GearCategoryTip, GearCategoryChalk:
		return true
	}
	return false
}

// IsGearGameType 判断是否是装备主力设置支持的球种：1=斯诺克 2=九球追分 3=中式八球 4=美式九球。
func IsGearGameType(gameType int) bool {
	return gameType >= 1 && gameType <= 4
}

// GearGameTypeRecordsBreak 判断该球种是否记录冲杆：九球追分、中式八球、美式九球记录，斯诺克没有开球冲杆。
func GearGameTypeRecordsBreak(gameType int) bool {
	return gameType >= 2 && gameType <= 4
}

// IsGearCueStyle 判断是否是合法杆型。
func IsGearCueStyle(style string) bool {
	return style == GearCueStyleSnooker || style == GearCueStylePool
}

// NormalizeGearRoles 去重、校验并按固定顺序（break、jump）拼成存储形式；出现未知身份时 ok 为 false。
func NormalizeGearRoles(roles []string) (joined string, ok bool) {
	hasBreak, hasJump := false, false
	for _, role := range roles {
		switch strings.TrimSpace(role) {
		case GearRoleBreak:
			hasBreak = true
		case GearRoleJump:
			hasJump = true
		case "":
		default:
			return "", false
		}
	}
	parts := make([]string, 0, 2)
	if hasBreak {
		parts = append(parts, GearRoleBreak)
	}
	if hasJump {
		parts = append(parts, GearRoleJump)
	}
	return strings.Join(parts, ","), true
}

// ParseGearRoles 把存储形式拆成身份列表，永远不返回 nil。
func ParseGearRoles(raw string) []string {
	roles := make([]string, 0, 2)
	for role := range strings.SplitSeq(raw, ",") {
		switch role = strings.TrimSpace(role); role {
		case GearRoleBreak, GearRoleJump:
			roles = append(roles, role)
		}
	}
	return roles
}

// GearRolesContain 判断存储形式的身份里是否包含指定身份。
func GearRolesContain(raw, role string) bool {
	return slices.Contains(ParseGearRoles(raw), role)
}

// NormalizeGearText 把自填文本规范化：全角 ASCII 与全角空格转半角，转小写，去首尾空白并合并连续空白。
func NormalizeGearText(text string) string {
	converted := strings.Map(func(r rune) rune {
		switch {
		case r == 0x3000:
			return ' '
		case r >= 0xFF01 && r <= 0xFF5E:
			r -= 0xFEE0
		}
		return unicode.ToLower(r)
	}, text)
	return strings.Join(strings.Fields(converted), " ")
}

// BuildGearCustomKey 生成未收录登记的归并键：类别 | 品牌（目录品牌用 ID，自填用规范化名称） | 规范化型号名。
func BuildGearCustomKey(category string, brandId int64, brandName, modelName string) string {
	brandPart := "b" + strconv.FormatInt(brandId, 10)
	if brandId <= 0 {
		brandPart = "n" + NormalizeGearText(brandName)
	}
	return category + "|" + brandPart + "|" + NormalizeGearText(modelName)
}

// GearCustomKeyCategory 取归并键里的类别。
func GearCustomKeyCategory(customKey string) string {
	category, _, _ := strings.Cut(customKey, "|")
	return category
}

// escapeGearLike 转义 LIKE 通配符，转义字符固定为 !，与 ESCAPE '!' 配合，MySQL 与 SQLite 行为一致。
func escapeGearLike(keyword string) string {
	replacer := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
	return replacer.Replace(keyword)
}

// GearProductFilter 是型号分页查询条件。Status 为 -1 表示不限状态；OnlyEnabledBrand 用于用户端，排除已停用品牌下的型号。
type GearProductFilter struct {
	Category         string
	BrandId          int64
	Keyword          string
	HouseOnly        bool
	Status           int
	OnlyEnabledBrand bool
	Page             int
	PageSize         int
}

// GearBrandRow 是后台品牌列表行，附带型号数。
type GearBrandRow struct {
	GearBrand
	ProductCount int64 `gorm:"column:product_count"`
}

// GearPendingGroup 是待收录分组：同一归并键下尚未关联目录型号的登记。
type GearPendingGroup struct {
	CustomKey string `gorm:"column:custom_key"`
	Category  string `gorm:"column:category"`
	Count     int64  `gorm:"column:cnt"`
	LatestId  int64  `gorm:"column:latest_id"`
}

type GearModel struct {
	db *gorm.DB
}

func NewGearModel(db *gorm.DB) *GearModel {
	return &GearModel{db: db}
}

// DB 返回底层连接，供调用方开启事务。
func (m *GearModel) DB() *gorm.DB {
	if m == nil {
		return nil
	}
	return m.db
}

func (m *GearModel) conn(tx *gorm.DB) (*gorm.DB, error) {
	if m == nil || m.db == nil {
		return nil, ErrGearDBNil
	}
	if tx != nil {
		return tx, nil
	}
	return m.db, nil
}

// ---------- 品牌与型号 ----------

// ListBrandsWithProducts 返回启用、且在该类别下有启用型号的品牌，一条 SQL。
func (m *GearModel) ListBrandsWithProducts(category string) ([]GearBrand, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	sub := db.Table("gear_products AS p").Select("1").
		Where("p.brand_id = gear_brands.id AND p.category = ? AND p.status = 1", category)
	var brands []GearBrand
	err = db.Where("gear_brands.status = 1").Where("EXISTS (?)", sub).
		Order("gear_brands.sort ASC, gear_brands.id ASC").Find(&brands).Error
	return brands, err
}

func (m *GearModel) FindBrandsByIds(ids []int64) (map[int64]GearBrand, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]GearBrand, len(ids))
	ids = uniquePositiveInt64s(ids)
	if len(ids) == 0 {
		return result, nil
	}
	var brands []GearBrand
	if err := db.Where("id IN ?", ids).Find(&brands).Error; err != nil {
		return nil, err
	}
	for _, brand := range brands {
		result[brand.Id] = brand
	}
	return result, nil
}

func (m *GearModel) FindProductsByIds(ids []int64) (map[int64]GearProduct, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]GearProduct, len(ids))
	ids = uniquePositiveInt64s(ids)
	if len(ids) == 0 {
		return result, nil
	}
	var products []GearProduct
	if err := db.Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	for _, product := range products {
		result[product.Id] = product
	}
	return result, nil
}

// FindProduct 不存在时返回 (nil, nil)。
func (m *GearModel) FindProduct(id int64) (*GearProduct, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var product GearProduct
	err = db.Where("id = ?", id).First(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindBrand 不存在时返回 (nil, nil)。
func (m *GearModel) FindBrand(id int64) (*GearBrand, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var brand GearBrand
	err = db.Where("id = ?", id).First(&brand).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func (m *GearModel) productQuery(db *gorm.DB, filter GearProductFilter) *gorm.DB {
	q := db.Table("gear_products AS p").Joins("LEFT JOIN gear_brands AS b ON b.id = p.brand_id")
	if filter.Category != "" {
		q = q.Where("p.category = ?", filter.Category)
	}
	if filter.BrandId > 0 {
		q = q.Where("p.brand_id = ?", filter.BrandId)
	}
	if filter.HouseOnly {
		q = q.Where("p.house = ?", true)
	}
	if filter.Status >= 0 {
		q = q.Where("p.status = ?", filter.Status)
	}
	if filter.OnlyEnabledBrand {
		q = q.Where("(p.brand_id = 0 OR b.status = 1)")
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + escapeGearLike(keyword) + "%"
		q = q.Where("(p.name LIKE ? ESCAPE '!' OR b.name LIKE ? ESCAPE '!' OR b.aliases LIKE ? ESCAPE '!')", like, like, like)
	}
	return q
}

// ListProductsPage 在数据库内分页，按排序值与 ID 稳定排序，两条 SQL（计数与分页）。
func (m *GearModel) ListProductsPage(filter GearProductFilter) ([]GearProduct, int64, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := m.productQuery(db, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []GearProduct{}, 0, nil
	}
	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var products []GearProduct
	err = m.productQuery(db, filter).Select("p.*").Order("p.sort ASC, p.id ASC").
		Limit(pageSize).Offset((page - 1) * pageSize).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// ListBrandsPage 是后台品牌列表，附带每个品牌的型号数。
func (m *GearModel) ListBrandsPage(keyword string, page, pageSize int) ([]GearBrandRow, int64, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, 0, err
	}
	build := func() *gorm.DB {
		q := db.Table("gear_brands AS b")
		if keyword = strings.TrimSpace(keyword); keyword != "" {
			like := "%" + escapeGearLike(keyword) + "%"
			q = q.Where("(b.name LIKE ? ESCAPE '!' OR b.aliases LIKE ? ESCAPE '!')", like, like)
		}
		return q
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var rows []GearBrandRow
	err = build().
		Select("b.*, (SELECT COUNT(*) FROM gear_products AS p WHERE p.brand_id = b.id) AS product_count").
		Order("b.sort ASC, b.id ASC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SaveBrand 新增（Id 为 0）或整行更新品牌；名称重复时返回 ErrGearDuplicate。
func (m *GearModel) SaveBrand(brand *GearBrand) error {
	db, err := m.conn(nil)
	if err != nil {
		return err
	}
	var count int64
	if err := db.Model(&GearBrand{}).Where("name = ? AND id <> ?", brand.Name, brand.Id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrGearDuplicate
	}
	if brand.Id == 0 {
		return db.Create(brand).Error
	}
	return db.Save(brand).Error
}

// SaveProduct 新增（Id 为 0）或整行更新型号；同类别、品牌下名称重复时返回 ErrGearDuplicate。
func (m *GearModel) SaveProduct(product *GearProduct) error {
	db, err := m.conn(nil)
	if err != nil {
		return err
	}
	var count int64
	err = db.Model(&GearProduct{}).
		Where("category = ? AND brand_id = ? AND name = ? AND id <> ?", product.Category, product.BrandId, product.Name, product.Id).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrGearDuplicate
	}
	if product.Id == 0 {
		return db.Create(product).Error
	}
	return db.Save(product).Error
}

// ---------- 用户装备 ----------

// ListUserItems 按 ID 升序返回用户指定状态的装备，上限为数量上限的两倍作为兜底。
func (m *GearModel) ListUserItems(userId int64, statuses []int) ([]UserGearItem, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var items []UserGearItem
	err = db.Where("user_id = ? AND status IN ?", userId, statuses).Order("id ASC").
		Limit(GearMaxItemsPerUser * 2).Find(&items).Error
	return items, err
}

// FindItemsByIds 按 ID 批量读取装备，包含已删除的，供历史对局解析。
func (m *GearModel) FindItemsByIds(ids []int64) (map[int64]UserGearItem, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]UserGearItem, len(ids))
	ids = uniquePositiveInt64s(ids)
	if len(ids) == 0 {
		return result, nil
	}
	var items []UserGearItem
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.Id] = item
	}
	return result, nil
}

// FindItemWithTx 不存在时返回 (nil, nil)；tx 为 nil 时使用默认连接。
func (m *GearModel) FindItemWithTx(tx *gorm.DB, id int64) (*UserGearItem, error) {
	db, err := m.conn(tx)
	if err != nil {
		return nil, err
	}
	var item UserGearItem
	err = db.Where("id = ?", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CountUserItemsWithTx 统计未删除装备数量。
func (m *GearModel) CountUserItemsWithTx(tx *gorm.DB, userId int64) (int64, error) {
	db, err := m.conn(tx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Model(&UserGearItem{}).Where("user_id = ? AND status < ?", userId, GearItemStatusDeleted).Count(&count).Error
	return count, err
}

func (m *GearModel) CreateItemWithTx(tx *gorm.DB, item *UserGearItem) error {
	db, err := m.conn(tx)
	if err != nil {
		return err
	}
	return db.Create(item).Error
}

func (m *GearModel) SaveItemWithTx(tx *gorm.DB, item *UserGearItem) error {
	db, err := m.conn(tx)
	if err != nil {
		return err
	}
	return db.Save(item).Error
}

// ---------- 主力设置 ----------

func (m *GearModel) ListDefaults(userId int64) ([]UserGearDefault, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var defaults []UserGearDefault
	err = db.Where("user_id = ?", userId).Order("game_type ASC, slot ASC").Find(&defaults).Error
	return defaults, err
}

// ListDefaultsForUsers 读取多位用户在同一球种的主力设置，一条 SQL。
func (m *GearModel) ListDefaultsForUsers(userIds []int64, gameType int) ([]UserGearDefault, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	userIds = uniquePositiveInt64s(userIds)
	if len(userIds) == 0 {
		return nil, nil
	}
	var defaults []UserGearDefault
	err = db.Where("user_id IN ? AND game_type = ?", userIds, gameType).Find(&defaults).Error
	return defaults, err
}

// UpsertDefaultWithTx 设置主力装备；同一用户、球种、槽位已有设置时更新装备。
func (m *GearModel) UpsertDefaultWithTx(tx *gorm.DB, d *UserGearDefault) error {
	db, err := m.conn(tx)
	if err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "game_type"}, {Name: "slot"}},
		DoUpdates: clause.AssignmentColumns([]string{"item_id", "updated_at"}),
	}).Create(d).Error
}

func (m *GearModel) DeleteDefaultWithTx(tx *gorm.DB, userId int64, gameType int, slot string) error {
	db, err := m.conn(tx)
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND game_type = ? AND slot = ?", userId, gameType, slot).Delete(&UserGearDefault{}).Error
}

// DeleteDefaultsByItemWithTx 清除指向某件装备的全部主力设置。
func (m *GearModel) DeleteDefaultsByItemWithTx(tx *gorm.DB, userId, itemId int64) error {
	db, err := m.conn(tx)
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND item_id = ?", userId, itemId).Delete(&UserGearDefault{}).Error
}

// ---------- 对局用杆记录 ----------

// InsertSnapshotsIgnore 批量写入对局用杆记录；同一对局同一选手已有记录时什么都不做，不覆盖。
func (m *GearModel) InsertSnapshotsIgnore(rows []MatchGearSnapshot) error {
	db, err := m.conn(nil)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (m *GearModel) FindSnapshotsByMatch(matchId int64) ([]MatchGearSnapshot, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var rows []MatchGearSnapshot
	err = db.Where("match_id = ?", matchId).Order("user_id ASC").Find(&rows).Error
	return rows, err
}

// UpsertManualSnapshot 修改或补记一位选手的本场记录，来源记为手动。
func (m *GearModel) UpsertManualSnapshot(row *MatchGearSnapshot) error {
	db, err := m.conn(nil)
	if err != nil {
		return err
	}
	row.Source = GearSnapshotSourceManual
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "match_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cue_item_id", "tip_product_id", "chalk_product_id", "break_item_id", "source", "updated_at",
		}),
	}).Create(row).Error
}

func (m *GearModel) DeleteSnapshot(matchId, userId int64) error {
	db, err := m.conn(nil)
	if err != nil {
		return err
	}
	return db.Where("match_id = ? AND user_id = ?", matchId, userId).Delete(&MatchGearSnapshot{}).Error
}

// ---------- 未收录型号归并 ----------

func (m *GearModel) pendingBase(db *gorm.DB, category string) *gorm.DB {
	q := db.Table("user_gear_items").Where("product_id = 0 AND status < ?", GearItemStatusDeleted)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	return q
}

// ListPendingGroups 按登记数倒序分页返回待收录分组：分组数一条 SQL，分页一条 SQL。
func (m *GearModel) ListPendingGroups(category string, page, pageSize int) ([]GearPendingGroup, int64, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	groups := m.pendingBase(db, category).Select("custom_key").Group("custom_key")
	if err := db.Table("(?) AS g", groups).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []GearPendingGroup{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var rows []GearPendingGroup
	err = m.pendingBase(db, category).
		Select("custom_key, MIN(category) AS category, COUNT(*) AS cnt, MAX(id) AS latest_id").
		Group("custom_key").Order("cnt DESC, latest_id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindFirstByCustomKey 取一条仍未关联的登记，不存在时返回 (nil, nil)。
func (m *GearModel) FindFirstByCustomKey(customKey string) (*UserGearItem, error) {
	db, err := m.conn(nil)
	if err != nil {
		return nil, err
	}
	var item UserGearItem
	err = db.Where("product_id = 0 AND custom_key = ?", customKey).Order("id DESC").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// LinkPending 把同一归并键下最多 limit 条未关联的登记关联到目录型号，返回本次更新数与剩余数。
// 已删除的登记也一起关联，使引用它的历史对局同样显示为正式型号。保留自填字段与归并键。
func (m *GearModel) LinkPending(customKey string, productId int64, limit int) (updated int64, remaining int64, err error) {
	db, err := m.conn(nil)
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > GearPendingLinkBatchSize {
		limit = GearPendingLinkBatchSize
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var ids []int64
		if err := tx.Model(&UserGearItem{}).Where("product_id = 0 AND custom_key = ?", customKey).
			Order("id ASC").Limit(limit).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) > 0 {
			result := tx.Model(&UserGearItem{}).Where("id IN ? AND product_id = 0", ids).Update("product_id", productId)
			if result.Error != nil {
				return result.Error
			}
			updated = result.RowsAffected
		}
		return tx.Model(&UserGearItem{}).Where("product_id = 0 AND custom_key = ?", customKey).Count(&remaining).Error
	})
	return updated, remaining, err
}

func uniquePositiveInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
