package model

import (
	"errors"
	"math"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type Venue struct {
	Id                 int64      `gorm:"primarykey" json:"id"`
	Name               string     `gorm:"size:128;not null" json:"name"`
	Address            string     `gorm:"size:256;not null" json:"address"`
	City               string     `gorm:"size:64;not null;index" json:"city"`
	District           string     `gorm:"size:64;not null;default:''" json:"district"`
	FullAddress        string     `gorm:"size:512;not null;default:'';uniqueIndex:uniq_full_address" json:"full_address"`
	Latitude           float64    `gorm:"not null;default:0" json:"latitude"`
	Longitude          float64    `gorm:"not null;default:0" json:"longitude"`
	Phone              string     `gorm:"size:32;not null;default:''" json:"phone"`
	Images             string     `gorm:"type:text" json:"images"`
	BusinessHours      string     `gorm:"size:128;not null;default:''" json:"business_hours"`
	TableCount         int        `gorm:"not null;default:0" json:"table_count"`
	PriceRange         string     `gorm:"size:64;not null;default:''" json:"price_range"`
	Description        string     `gorm:"type:text" json:"description"`
	OwnerUserId        int64      `gorm:"default:null" json:"owner_user_id"`
	Status             int        `gorm:"not null;default:0;index" json:"status"`
	GeoStatus          int        `gorm:"not null;default:0;index:idx_status_geo_status,priority:2" json:"geo_status"`
	GeoSource          string     `gorm:"size:32;not null;default:''" json:"geo_source"`
	GeoScore           int        `gorm:"not null;default:0" json:"geo_score"`
	GeoLevel           string     `gorm:"size:64;not null;default:''" json:"geo_level"`
	GeoAttempts        int        `gorm:"not null;default:0" json:"geo_attempts"`
	GeoError           string     `gorm:"size:255;not null;default:''" json:"geo_error"`
	GeoUpdatedAt       *time.Time `json:"geo_updated_at"`
	DuplicateOfVenueId *int64     `gorm:"default:null" json:"duplicate_of_venue_id"`
	RejectReason       string     `gorm:"size:255;not null;default:'';comment:审核拒绝原因" json:"reject_reason"`
	CheckinCount       int64      `gorm:"->;-:migration" json:"-"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (Venue) TableName() string {
	return "venues"
}

type VenueCheckin struct {
	Id        int64     `gorm:"primarykey" json:"id"`
	VenueId   int64     `gorm:"not null;index" json:"venue_id"`
	UserId    int64     `gorm:"not null;index" json:"user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (VenueCheckin) TableName() string {
	return "venue_checkins"
}

type VenueGeocodeTask struct {
	Id          int64      `gorm:"primarykey" json:"id"`
	VenueId     int64      `gorm:"uniqueIndex;not null" json:"venue_id"`
	Status      int        `gorm:"not null;default:0;index:idx_status_next_retry_at,priority:1" json:"status"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int        `gorm:"not null;default:5" json:"max_attempts"`
	NextRetryAt *time.Time `gorm:"index:idx_status_next_retry_at,priority:2" json:"next_retry_at"`
	LockedAt    *time.Time `json:"locked_at"`
	LockedBy    string     `gorm:"size:64;not null;default:''" json:"locked_by"`
	LastError   string     `gorm:"size:255;not null;default:''" json:"last_error"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (VenueGeocodeTask) TableName() string {
	return "venue_geocode_tasks"
}

type GeocodeAccount struct {
	Id             int64      `gorm:"primarykey" json:"id"`
	Provider       string     `gorm:"size:32;not null;default:'apihz';index:idx_provider_status,priority:1" json:"provider"`
	ProviderAppID  string     `gorm:"size:64;not null;default:''" json:"provider_app_id"`
	ProviderKey    string     `gorm:"size:255;not null;default:''" json:"provider_key"`
	PerMinuteLimit int        `gorm:"not null;default:10" json:"per_minute_limit"`
	Status         int        `gorm:"not null;default:1;index:idx_provider_status,priority:2" json:"status"`
	Priority       int        `gorm:"not null;default:100" json:"priority"`
	CoolDownUntil  *time.Time `json:"cool_down_until"`
	FailStreak     int        `gorm:"not null;default:0" json:"fail_streak"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	LastErrorAt    *time.Time `json:"last_error_at"`
	Remark         string     `gorm:"size:255;not null;default:''" json:"remark"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (GeocodeAccount) TableName() string {
	return "geocode_accounts"
}

const (
	VenueStatusHidden    = 0 // 隐藏/草稿
	VenueStatusPublished = 1 // 已发布（审核通过）
	VenueStatusPending   = 2 // 待审核
	VenueStatusRejected  = 3 // 审核拒绝
)

const (
	VenueGeoStatusPending  = 0
	VenueGeoStatusSuccess  = 1
	VenueGeoStatusRetrying = 2
	VenueGeoStatusFailed   = 3
)

const (
	VenueGeocodeTaskStatusPending    = 0
	VenueGeocodeTaskStatusProcessing = 1
	VenueGeocodeTaskStatusSuccess    = 2
	VenueGeocodeTaskStatusRetry      = 3
	VenueGeocodeTaskStatusDead       = 4
)

const (
	GeocodeAccountStatusDisabled = 0
	GeocodeAccountStatusEnabled  = 1
)

type VenueModel struct {
	db *gorm.DB
}

func NewVenueModel(db *gorm.DB) *VenueModel {
	return &VenueModel{db: db}
}

func BuildVenueFullAddress(city, district, address string) string {
	parts := []string{
		strings.TrimSpace(city),
		strings.TrimSpace(district),
		strings.TrimSpace(address),
	}

	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(part)
	}
	return builder.String()
}

func IsDuplicateVenueFullAddressError(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	if mysqlErr.Number != 1062 {
		return false
	}

	message := strings.ToLower(mysqlErr.Message)
	return strings.Contains(message, "uniq_full_address") || strings.Contains(message, "full_address")
}

func (m *VenueModel) Create(venue *Venue) error {
	return m.db.Create(venue).Error
}

func (m *VenueModel) CreateWithTx(tx *gorm.DB, venue *Venue) error {
	if venue == nil {
		return errors.New("venue is nil")
	}
	if tx != nil {
		return tx.Create(venue).Error
	}
	return m.db.Create(venue).Error
}

func (m *VenueModel) FindById(id int64) (*Venue, error) {
	var venue Venue
	err := m.db.First(&venue, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &venue, err
}

func (m *VenueModel) FindList(page, pageSize int, city string) ([]Venue, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&Venue{}).
		Where("status = ? AND geo_status = ?", VenueStatusPublished, VenueGeoStatusSuccess).
		Where("duplicate_of_venue_id IS NULL")
	if city != "" {
		query = query.Where("city = ?", city)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Venue
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *VenueModel) FindListWithCheckinCount(page, pageSize int, city string) ([]Venue, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	countQuery := m.db.Model(&Venue{}).
		Where("status = ? AND geo_status = ?", VenueStatusPublished, VenueGeoStatusSuccess).
		Where("duplicate_of_venue_id IS NULL")
	if city != "" {
		countQuery = countQuery.Where("city = ?", city)
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Venue
	listQuery := m.db.Table("venues").
		Select("venues.*, COALESCE(checkins.checkin_count, 0) AS checkin_count").
		Joins("LEFT JOIN (SELECT venue_id, COUNT(*) AS checkin_count FROM venue_checkins GROUP BY venue_id) AS checkins ON checkins.venue_id = venues.id").
		Where("venues.status = ? AND venues.geo_status = ?", VenueStatusPublished, VenueGeoStatusSuccess).
		Where("venues.duplicate_of_venue_id IS NULL")
	if city != "" {
		listQuery = listQuery.Where("venues.city = ?", city)
	}
	err := listQuery.Order("venues.id DESC").Offset(offset).Limit(pageSize).Scan(&list).Error
	return list, total, err
}

// FindListForAdmin 管理员查询球馆列表（不受状态限制）
func (m *VenueModel) FindListForAdmin(page, pageSize int, status int, city string) ([]Venue, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&Venue{}).
		Where("duplicate_of_venue_id IS NULL")

	// 状态筛选: -1 表示全部
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if city != "" {
		query = query.Where("city = ?", city)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []Venue
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// UpdateStatus 更新球馆状态
func (m *VenueModel) UpdateStatus(venueId int64, status int) error {
	return m.db.Model(&Venue{}).Where("id = ?", venueId).Update("status", status).Error
}

func (m *VenueModel) UpdateReviewWithTx(tx *gorm.DB, venueId int64, status int, rejectReason string) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return errors.New("venue db is nil")
	}
	return db.Model(&Venue{}).Where("id = ?", venueId).Updates(map[string]interface{}{
		"status":        status,
		"reject_reason": strings.TrimSpace(rejectReason),
	}).Error
}

func (m *VenueModel) FindLatestByOwnerUserId(userId int64) (*Venue, error) {
	if userId <= 0 {
		return nil, nil
	}

	var venue Venue
	err := m.db.Where("owner_user_id = ?", userId).Order("id DESC").First(&venue).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &venue, err
}

func (m *VenueModel) FindByFullAddress(fullAddress string) (*Venue, error) {
	fullAddress = strings.TrimSpace(fullAddress)
	if fullAddress == "" {
		return nil, nil
	}

	var venue Venue
	err := m.db.Where("full_address = ?", fullAddress).First(&venue).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &venue, err
}

func (m *VenueModel) FindNearby(latitude, longitude float64, radiusMeters, limit int) ([]Venue, error) {
	return m.findNearbyWithCheckinCount(latitude, longitude, radiusMeters, limit, false)
}

func (m *VenueModel) FindNearbyWithCheckinCount(latitude, longitude float64, radiusMeters, limit int) ([]Venue, error) {
	return m.findNearbyWithCheckinCount(latitude, longitude, radiusMeters, limit, true)
}

func (m *VenueModel) findNearbyWithCheckinCount(latitude, longitude float64, radiusMeters, limit int, includeCheckinCount bool) ([]Venue, error) {
	if radiusMeters <= 0 {
		radiusMeters = 5000
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	minLatitude, maxLatitude, minLongitude, maxLongitude := nearbyBounds(latitude, longitude, float64(radiusMeters))
	distanceSQL := "(6371000 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude))))"
	table := "venues"
	if m.db.Dialector.Name() == "mysql" {
		table = "venues FORCE INDEX (idx_venues_nearby_candidates)"
	}
	query := m.db.Table(table).
		Select("venues.*, "+distanceSQL+" AS distance", latitude, longitude, latitude).
		Where("venues.status = ? AND venues.geo_status = ?", VenueStatusPublished, VenueGeoStatusSuccess).
		Where("venues.duplicate_of_venue_id IS NULL").
		Where("venues.latitude <> 0 AND venues.longitude <> 0").
		Where("venues.latitude BETWEEN ? AND ?", minLatitude, maxLatitude)
	if minLongitude < -180 {
		query = query.Where("(venues.longitude >= ? OR venues.longitude <= ?)", minLongitude+360, maxLongitude)
	} else if maxLongitude > 180 {
		query = query.Where("(venues.longitude >= ? OR venues.longitude <= ?)", minLongitude, maxLongitude-360)
	} else {
		query = query.Where("venues.longitude BETWEEN ? AND ?", minLongitude, maxLongitude)
	}
	if includeCheckinCount {
		query = query.Select("venues.*, COALESCE(checkins.checkin_count, 0) AS checkin_count, "+distanceSQL+" AS distance", latitude, longitude, latitude).
			Joins("LEFT JOIN (SELECT venue_id, COUNT(*) AS checkin_count FROM venue_checkins GROUP BY venue_id) AS checkins ON checkins.venue_id = venues.id")
	}
	var list []Venue
	err := query.Having("distance < ?", radiusMeters).Order("distance ASC").Limit(limit).Scan(&list).Error
	return list, err
}

func nearbyBounds(latitude, longitude, radiusMeters float64) (float64, float64, float64, float64) {
	const metersPerDegree = 111320.0
	latitudeDelta := radiusMeters / metersPerDegree
	longitudeScale := math.Cos(latitude * math.Pi / 180)
	if math.Abs(longitudeScale) < 0.01 {
		longitudeScale = 0.01
	}
	longitudeDelta := radiusMeters / (metersPerDegree * math.Abs(longitudeScale))
	minLatitude := math.Max(-90, latitude-latitudeDelta)
	maxLatitude := math.Min(90, latitude+latitudeDelta)
	return minLatitude, maxLatitude, longitude - longitudeDelta, longitude + longitudeDelta
}

func (m *VenueModel) GetCheckinCount(venueId int64) (int64, error) {
	var count int64
	err := m.db.Model(&VenueCheckin{}).Where("venue_id = ?", venueId).Count(&count).Error
	return count, err
}

func (m *VenueModel) MarkGeocodeFailureWithTx(tx *gorm.DB, venueId int64, geoStatus int, attempts int, message string) error {
	updates := map[string]interface{}{
		"geo_status":   geoStatus,
		"geo_attempts": attempts,
		"geo_error":    strings.TrimSpace(message),
	}
	if tx != nil {
		return tx.Model(&Venue{}).Where("id = ?", venueId).Updates(updates).Error
	}
	return m.db.Model(&Venue{}).Where("id = ?", venueId).Updates(updates).Error
}

func (m *VenueModel) ApplyGeocodeResultWithTx(tx *gorm.DB, venueId int64, longitude, latitude float64, score int, level, source string, attempts int, autoPublish bool) error {
	now := time.Now()
	status := VenueStatusHidden
	if autoPublish {
		status = VenueStatusPublished
	}

	updates := map[string]interface{}{
		"longitude":      longitude,
		"latitude":       latitude,
		"geo_status":     VenueGeoStatusSuccess,
		"geo_source":     source,
		"geo_score":      score,
		"geo_level":      level,
		"geo_attempts":   attempts,
		"geo_error":      "",
		"geo_updated_at": &now,
		"status":         status,
	}
	if tx != nil {
		return tx.Model(&Venue{}).Where("id = ?", venueId).Updates(updates).Error
	}
	return m.db.Model(&Venue{}).Where("id = ?", venueId).Updates(updates).Error
}

type VenueCheckinModel struct {
	db *gorm.DB
}

func NewVenueCheckinModel(db *gorm.DB) *VenueCheckinModel {
	return &VenueCheckinModel{db: db}
}

func (m *VenueCheckinModel) Create(checkin *VenueCheckin) error {
	return m.db.Create(checkin).Error
}

func (m *VenueCheckinModel) HasCheckedInToday(venueId, userId int64) (bool, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var count int64
	err := m.db.Model(&VenueCheckin{}).
		Where("venue_id = ? AND user_id = ? AND created_at >= ? AND created_at < ?", venueId, userId, startOfDay, endOfDay).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *VenueCheckinModel) FindByUser(userId int64, page, pageSize int) ([]VenueCheckin, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	query := m.db.Model(&VenueCheckin{}).Where("user_id = ?", userId)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []VenueCheckin
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

type VenueGeocodeTaskModel struct {
	db *gorm.DB
}

func NewVenueGeocodeTaskModel(db *gorm.DB) *VenueGeocodeTaskModel {
	return &VenueGeocodeTaskModel{db: db}
}

func (m *VenueGeocodeTaskModel) CreateWithTx(tx *gorm.DB, task *VenueGeocodeTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if tx != nil {
		return tx.Create(task).Error
	}
	return m.db.Create(task).Error
}

func (m *VenueGeocodeTaskModel) ListDueTasks(limit int, now time.Time) ([]VenueGeocodeTask, error) {
	if limit <= 0 {
		limit = 10
	}

	staleBefore := now.Add(-2 * time.Minute)
	var tasks []VenueGeocodeTask
	err := m.db.
		Where(
			"(status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND locked_at IS NOT NULL AND locked_at <= ?)",
			[]int{VenueGeocodeTaskStatusPending, VenueGeocodeTaskStatusRetry},
			now,
			VenueGeocodeTaskStatusProcessing,
			staleBefore,
		).
		Order("id ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func (m *VenueGeocodeTaskModel) ClaimTask(taskId int64, locker string, now time.Time) (bool, error) {
	staleBefore := now.Add(-2 * time.Minute)
	result := m.db.Model(&VenueGeocodeTask{}).
		Where("id = ?", taskId).
		Where(
			"(status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND locked_at IS NOT NULL AND locked_at <= ?)",
			[]int{VenueGeocodeTaskStatusPending, VenueGeocodeTaskStatusRetry},
			now,
			VenueGeocodeTaskStatusProcessing,
			staleBefore,
		).
		Updates(map[string]interface{}{
			"status":    VenueGeocodeTaskStatusProcessing,
			"locked_at": &now,
			"locked_by": locker,
		})
	return result.RowsAffected > 0, result.Error
}

func (m *VenueGeocodeTaskModel) ReleaseClaim(taskId int64, nextRetryAt time.Time, message string) error {
	return m.db.Model(&VenueGeocodeTask{}).Where("id = ?", taskId).Updates(map[string]interface{}{
		"status":        VenueGeocodeTaskStatusRetry,
		"next_retry_at": &nextRetryAt,
		"last_error":    strings.TrimSpace(message),
		"locked_at":     nil,
		"locked_by":     "",
	}).Error
}

func (m *VenueGeocodeTaskModel) MarkSuccess(taskId int64) error {
	return m.db.Model(&VenueGeocodeTask{}).Where("id = ?", taskId).Updates(map[string]interface{}{
		"status":        VenueGeocodeTaskStatusSuccess,
		"last_error":    "",
		"locked_at":     nil,
		"locked_by":     "",
		"updated_at":    time.Now(),
		"next_retry_at": nil,
	}).Error
}

func (m *VenueGeocodeTaskModel) MarkRetry(taskId int64, attempts int, nextRetryAt time.Time, message string) error {
	return m.db.Model(&VenueGeocodeTask{}).Where("id = ?", taskId).Updates(map[string]interface{}{
		"status":        VenueGeocodeTaskStatusRetry,
		"attempts":      attempts,
		"next_retry_at": &nextRetryAt,
		"last_error":    strings.TrimSpace(message),
		"locked_at":     nil,
		"locked_by":     "",
	}).Error
}

func (m *VenueGeocodeTaskModel) MarkDead(taskId int64, attempts int, message string) error {
	return m.db.Model(&VenueGeocodeTask{}).Where("id = ?", taskId).Updates(map[string]interface{}{
		"status":     VenueGeocodeTaskStatusDead,
		"attempts":   attempts,
		"last_error": strings.TrimSpace(message),
		"locked_at":  nil,
		"locked_by":  "",
	}).Error
}

type GeocodeAccountModel struct {
	db *gorm.DB
}

func NewGeocodeAccountModel(db *gorm.DB) *GeocodeAccountModel {
	return &GeocodeAccountModel{db: db}
}

func (m *GeocodeAccountModel) ListEnabled(provider string, now time.Time) ([]GeocodeAccount, error) {
	if provider == "" {
		provider = "apihz"
	}

	var accounts []GeocodeAccount
	err := m.db.Where("provider = ? AND status = ?", provider, GeocodeAccountStatusEnabled).
		Where("cool_down_until IS NULL OR cool_down_until <= ?", now).
		Order("priority ASC, id ASC").
		Find(&accounts).Error
	return accounts, err
}

func (m *GeocodeAccountModel) UpdateCooldown(accountId int64, coolDownUntil time.Time) error {
	now := time.Now()
	return m.db.Model(&GeocodeAccount{}).Where("id = ?", accountId).Updates(map[string]interface{}{
		"cool_down_until": &coolDownUntil,
		"last_error_at":   &now,
		"fail_streak":     gorm.Expr("fail_streak + 1"),
	}).Error
}

func (m *GeocodeAccountModel) TouchSuccess(accountId int64) error {
	now := time.Now()
	return m.db.Model(&GeocodeAccount{}).Where("id = ?", accountId).Updates(map[string]interface{}{
		"cool_down_until": nil,
		"last_success_at": &now,
		"fail_streak":     0,
	}).Error
}

func (m *GeocodeAccountModel) MarkDisabled(accountId int64) error {
	now := time.Now()
	return m.db.Model(&GeocodeAccount{}).Where("id = ?", accountId).Updates(map[string]interface{}{
		"status":          GeocodeAccountStatusDisabled,
		"last_error_at":   &now,
		"cool_down_until": nil,
	}).Error
}
