package model

import (
	"time"

	"gorm.io/gorm"
)

type WSTSyncJobState struct {
	Id                   int64      `gorm:"primarykey"`
	JobName              string     `gorm:"size:64;not null;uniqueIndex"`
	LastSuccessfulSyncAt *time.Time `gorm:"default:null"`
	CreatedAt            time.Time  `gorm:"autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime"`
}

func (WSTSyncJobState) TableName() string {
	return "wst_sync_job_states"
}

type WSTSyncJobStateModel struct {
	db *gorm.DB
}

func NewWSTSyncJobStateModel(db *gorm.DB) *WSTSyncJobStateModel {
	return &WSTSyncJobStateModel{db: db}
}

func (m *WSTSyncJobStateModel) FindByJobName(jobName string) (*WSTSyncJobState, error) {
	var state WSTSyncJobState
	err := m.db.Where("job_name = ?", jobName).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *WSTSyncJobStateModel) UpsertLastSuccessfulSyncAt(jobName string, syncedAt time.Time) error {
	state, err := m.FindByJobName(jobName)
	if err != nil {
		return err
	}

	syncedAt = syncedAt.UTC()
	if state == nil {
		state = &WSTSyncJobState{
			JobName:              jobName,
			LastSuccessfulSyncAt: &syncedAt,
		}
		return m.db.Create(state).Error
	}

	state.LastSuccessfulSyncAt = &syncedAt
	return m.db.Save(state).Error
}
