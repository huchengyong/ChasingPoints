package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type honorWallAchievementSchema struct {
	Id              int64  `gorm:"primarykey"`
	Key             string `gorm:"uniqueIndex:uk_key;size:64;not null"`
	Name            string
	Description     string
	Icon            string
	Category        string
	GameType        int
	MetricKey       string
	ProgressMode    string
	RewardTitleKey  string
	RewardTitleName string
	Sort            int
	Status          int
	Threshold       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (honorWallAchievementSchema) TableName() string { return "achievements" }

type honorWallUserAchievementSchema struct {
	Id                 int64 `gorm:"primarykey"`
	UserId             int64 `gorm:"uniqueIndex:uk_user_achievement,priority:1"`
	AchievementId      int64 `gorm:"uniqueIndex:uk_user_achievement,priority:2"`
	Progress           int
	Unlocked           int
	RewardGranted      int
	UnlockedAt         *time.Time
	UnlockedSourceType string
	UnlockedSourceId   int64
	RewardGrantedAt    *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (honorWallUserAchievementSchema) TableName() string { return "user_achievements" }

type honorWallUserTitleSchema struct {
	Id                     int64  `gorm:"primarykey"`
	UserId                 int64  `gorm:"uniqueIndex:uk_user_title_source,priority:1"`
	TitleKey               string `gorm:"uniqueIndex:uk_user_title_source,priority:2"`
	TitleName              string
	Source                 string
	SourceType             string `gorm:"uniqueIndex:uk_user_title_source,priority:3"`
	SourceRefId            int64  `gorm:"uniqueIndex:uk_user_title_source,priority:4"`
	SourceRefName          string
	GrantedByAchievementId *int64
	Equipped               int
	EquippedAt             *time.Time
	GrantedAt              *time.Time
	CreatedAt              time.Time
}

func (honorWallUserTitleSchema) TableName() string { return "user_titles" }

type honorWallProgressEventSchema struct {
	Id          int64  `gorm:"primarykey"`
	UserId      int64  `gorm:"uniqueIndex:uk_user_source_metric,priority:1"`
	SourceType  string `gorm:"uniqueIndex:uk_user_source_metric,priority:2"`
	SourceId    int64  `gorm:"uniqueIndex:uk_user_source_metric,priority:3"`
	GameType    int
	MetricKey   string `gorm:"uniqueIndex:uk_user_source_metric,priority:4"`
	MetricValue int
	OccurredAt  time.Time
	CreatedAt   time.Time
}

func (honorWallProgressEventSchema) TableName() string { return "achievement_progress_events" }

type honorWallSeasonSchema struct {
	Id             int64 `gorm:"primarykey"`
	Name           string
	StartDate      time.Time
	EndDate        time.Time
	Status         int
	RankResetRatio float64
	CreatedAt      time.Time
}

func (honorWallSeasonSchema) TableName() string { return "seasons" }

type honorWallSeasonRecordSchema struct {
	Id             int64 `gorm:"primarykey"`
	SeasonId       int64 `gorm:"uniqueIndex:uk_season_user_game,priority:1"`
	UserId         int64 `gorm:"uniqueIndex:uk_season_user_game,priority:2"`
	GameType       int   `gorm:"uniqueIndex:uk_season_user_game,priority:3"`
	StartRankScore int
	EndRankScore   int
	PeakRankScore  int
	MatchesPlayed  int
	Wins           int
	FinalRank      int
	Rewards        string
	CreatedAt      time.Time
}

func (honorWallSeasonRecordSchema) TableName() string { return "season_records" }

type honorWallSeasonChallengeSnapshotSchema struct {
	Id            int64  `gorm:"primarykey"`
	SeasonId      int64  `gorm:"uniqueIndex:uk_season_challenge_snapshot,priority:1"`
	UserId        int64  `gorm:"uniqueIndex:uk_season_challenge_snapshot,priority:2"`
	GameType      int    `gorm:"uniqueIndex:uk_season_challenge_snapshot,priority:3"`
	ChallengeKey  string `gorm:"uniqueIndex:uk_season_challenge_snapshot,priority:4"`
	ChallengeName string
	Threshold     int
	Progress      int
	Completed     int
	ArchivedAt    time.Time
	CreatedAt     time.Time
}

func (honorWallSeasonChallengeSnapshotSchema) TableName() string {
	return "season_challenge_snapshots"
}

type honorWallSeasonSettlementSchema struct {
	Id           int64 `gorm:"primarykey"`
	SeasonId     int64 `gorm:"uniqueIndex:uk_season_settlements_season"`
	NextSeasonId *int64
	Status       string
	Attempts     int
	StartedAt    *time.Time
	CompletedAt  *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (honorWallSeasonSettlementSchema) TableName() string { return "season_settlements" }

type honorWallMatchSchema struct {
	Id                  int64 `gorm:"primarykey"`
	UserId              int64
	OpponentId          *int64
	OpponentName        string
	GameType            int
	MatchMode           string
	Status              int
	Result              *int
	MatchTime           time.Time
	EndTime             *time.Time
	AchievementSyncedAt *time.Time
	DeletedAt           gorm.DeletedAt
}

func (honorWallMatchSchema) TableName() string { return "matches" }

func PrepareHonorWallSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&honorWallAchievementSchema{},
		&honorWallUserAchievementSchema{},
		&honorWallUserTitleSchema{},
		&honorWallProgressEventSchema{},
		&honorWallSeasonSchema{},
		&honorWallSeasonRecordSchema{},
		&honorWallSeasonChallengeSnapshotSchema{},
		&honorWallSeasonSettlementSchema{},
		&honorWallMatchSchema{},
	)
}
