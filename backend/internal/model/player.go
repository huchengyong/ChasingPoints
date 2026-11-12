package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type Player struct {
	Id             int64          `gorm:"primarykey"`
	SourceType     string         `gorm:"size:32;not null;default:''"`
	SourcePlayerId string         `gorm:"size:128;not null;default:'';index"`
	FirstName      string         `gorm:"size:64;not null;default:''"`
	LastName       string         `gorm:"size:64;not null;default:''"`
	DisplayName    string         `gorm:"size:128;not null;default:''"`
	NameZh         string         `gorm:"column:name_zh;size:64;not null;default:''"`
	Avatar         string         `gorm:"size:512;not null;default:''"`
	CountryCode    string         `gorm:"size:32;not null;default:''"`
	FlagEmoji      string         `gorm:"size:32;not null;default:''"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (Player) TableName() string {
	return "players"
}

type PlayerModel struct {
	db *gorm.DB
}

func NewPlayerModel(db *gorm.DB) *PlayerModel {
	return &PlayerModel{db: db}
}

func (m *PlayerModel) Create(player *Player) error {
	return m.db.Create(player).Error
}

func (m *PlayerModel) FindByIds(ids []int64) (map[int64]Player, error) {
	result := make(map[int64]Player, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var list []Player
	if err := m.db.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.Id] = item
	}
	return result, nil
}

func (m *PlayerModel) FindBySourcePlayerIds(sourceType string, sourcePlayerIds []string) (map[string]Player, error) {
	result := make(map[string]Player, len(sourcePlayerIds))
	if len(sourcePlayerIds) == 0 {
		return result, nil
	}

	var list []Player
	if err := m.db.
		Where("source_type = ? AND source_player_id IN ?", sourceType, sourcePlayerIds).
		Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		result[item.SourcePlayerId] = item
	}
	return result, nil
}

// PlayerAdminFilter 是管理后台维护球员中文名时的分页条件。
type PlayerAdminFilter struct {
	Page      int
	PageSize  int
	Keyword   string
	MissingZh bool
}

func (m *PlayerModel) adminListQuery(filter PlayerAdminFilter) *gorm.DB {
	query := m.db.Model(&Player{})
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + escapeGearLike(keyword) + "%"
		query = query.Where(
			"(display_name LIKE ? ESCAPE '!' OR first_name LIKE ? ESCAPE '!' OR last_name LIKE ? ESCAPE '!' OR name_zh LIKE ? ESCAPE '!')",
			like, like, like, like,
		)
	}
	if filter.MissingZh {
		query = query.Where("name_zh = ''")
	}
	return query
}

// FindAdminList 在数据库内分页（计数与分页两条 SQL），按最近同步时间倒序、ID 稳定排序；
// 关键字同时匹配英文名与中文名，并转义 LIKE 通配符。
func (m *PlayerModel) FindAdminList(filter PlayerAdminFilter) ([]Player, int64, error) {
	var total int64
	if err := m.adminListQuery(filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []Player{}, 0, nil
	}

	var list []Player
	err := m.adminListQuery(filter).
		Order("updated_at DESC").
		Order("id DESC").
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (m *PlayerModel) FindById(id int64) (*Player, error) {
	var player Player
	if err := m.db.Where("id = ?", id).Limit(1).Find(&player).Error; err != nil {
		return nil, err
	}
	if player.Id == 0 {
		return nil, nil
	}
	return &player, nil
}

// UpdateNameZh 只改中文名，不触碰 WST 同步维护的其他字段；不更新 updated_at，避免打乱按同步时间的排序。
func (m *PlayerModel) UpdateNameZh(id int64, nameZh string) error {
	return m.db.Model(&Player{}).Where("id = ?", id).UpdateColumn("name_zh", nameZh).Error
}
