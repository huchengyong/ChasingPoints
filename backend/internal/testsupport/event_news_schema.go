package testsupport

import (
	"time"

	"gorm.io/gorm"
)

type eventNewsEventSchema struct {
	Id           int64          `gorm:"primarykey"`
	Title        string         `gorm:"size:128;not null"`
	TournamentId int64          `gorm:"not null;default:0;index"`
	GameType     int            `gorm:"not null;index"`
	SourceType   string         `gorm:"size:32;not null;default:''"`
	SourceName   string         `gorm:"size:64;not null;default:''"`
	SourceUrl    string         `gorm:"size:512;not null;default:''"`
	CoverImage   string         `gorm:"size:512;not null;default:''"`
	Summary      string         `gorm:"size:512;not null;default:''"`
	Content      string         `gorm:"type:text"`
	Country      string         `gorm:"size:64;not null;default:''"`
	City         string         `gorm:"size:64;not null;default:'';index"`
	Venue        string         `gorm:"size:128;not null;default:''"`
	StartDate    *time.Time     `gorm:"type:date;default:null;index"`
	EndDate      *time.Time     `gorm:"type:date;default:null"`
	StartTime    *time.Time     `gorm:"default:null;index"`
	EndTime      *time.Time     `gorm:"default:null"`
	Status       int            `gorm:"not null;default:0;index"`
	SortTime     *time.Time     `gorm:"default:null;index"`
	Published    bool           `gorm:"not null;default:false;index"`
	PublishedAt  *time.Time     `gorm:"default:null"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (eventNewsEventSchema) TableName() string {
	return "event_news_events"
}

type tournamentSchema struct {
	Id                 int64      `gorm:"primarykey"`
	CreatorId          int64      `gorm:"not null;index"`
	Name               string     `gorm:"size:128;not null"`
	Description        string     `gorm:"type:text"`
	CoverImage         string     `gorm:"size:512;not null;default:''"`
	GameType           int        `gorm:"not null;index"`
	Format             int        `gorm:"not null;default:1"`
	MaxPlayers         int        `gorm:"not null;default:16"`
	CurrentPlayers     int        `gorm:"not null;default:0"`
	Status             int        `gorm:"not null;default:0;index"`
	Country            string     `gorm:"size:64;not null;default:''"`
	City               string     `gorm:"size:64;not null;default:'';index"`
	VenueName          string     `gorm:"size:128;not null;default:''"`
	StartDate          *time.Time `gorm:"type:date;default:null;index"`
	EndDate            *time.Time `gorm:"type:date;default:null"`
	StartTime          *time.Time `gorm:"default:null;index"`
	EndTime            *time.Time `gorm:"default:null"`
	SourceType         string     `gorm:"size:32;not null;default:''"`
	SourceTournamentId string     `gorm:"size:128;not null;default:'';index"`
	SourceSeasonId     string     `gorm:"size:64;not null;default:'';index"`
	InformationPage    string     `gorm:"size:512;not null;default:''"`
	TicketingLink      string     `gorm:"size:512;not null;default:''"`
	LastSyncedAt       *time.Time `gorm:"default:null;index"`
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
}

func (tournamentSchema) TableName() string {
	return "tournaments"
}

type tournamentMatchSchema struct {
	Id              int64          `gorm:"primarykey"`
	TournamentId    int64          `gorm:"not null;index"`
	SourceType      string         `gorm:"size:32;not null;default:''"`
	SourceMatchId   string         `gorm:"size:128;not null;default:'';index"`
	RoundName       string         `gorm:"size:128;not null;default:''"`
	RoundNumber     int            `gorm:"not null;default:0"`
	RoundOrder      int            `gorm:"not null;default:0;index"`
	MatchOrder      int            `gorm:"not null;default:0"`
	StartTime       *time.Time     `gorm:"default:null;index"`
	BestOf          int            `gorm:"not null;default:0"`
	HomePlayerId    int64          `gorm:"not null;default:0"`
	HomePlayerName  string         `gorm:"size:128;not null;default:''"`
	AwayPlayerId    int64          `gorm:"not null;default:0"`
	AwayPlayerName  string         `gorm:"size:128;not null;default:''"`
	HomeScore       int            `gorm:"not null;default:0"`
	AwayScore       int            `gorm:"not null;default:0"`
	WinnerSide      int            `gorm:"not null;default:0"`
	IsPlaceholder   bool           `gorm:"not null;default:false"`
	Player1Id       int64          `gorm:"not null;default:0"`
	Player2Id       int64          `gorm:"not null;default:0"`
	WinnerId        int64          `gorm:"not null;default:0"`
	MatchId         int64          `gorm:"not null;default:0"`
	BracketPosition string         `gorm:"size:32;not null;default:''"`
	Status          int            `gorm:"not null;default:0;index"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (tournamentMatchSchema) TableName() string {
	return "tournament_matches"
}

type playerSchema struct {
	Id             int64          `gorm:"primarykey"`
	SourceType     string         `gorm:"size:32;not null;default:''"`
	SourcePlayerId string         `gorm:"size:128;not null;default:'';index"`
	FirstName      string         `gorm:"size:64;not null;default:''"`
	LastName       string         `gorm:"size:64;not null;default:''"`
	DisplayName    string         `gorm:"size:128;not null;default:''"`
	Avatar         string         `gorm:"size:512;not null;default:''"`
	CountryCode    string         `gorm:"size:32;not null;default:''"`
	FlagEmoji      string         `gorm:"size:16;not null;default:''"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (playerSchema) TableName() string {
	return "players"
}

// PrepareEventNewsSchema bootstraps the event_news, tournament and tournament_match tables for sqlite-backed tests.
func PrepareEventNewsSchema(db *gorm.DB) error {
	return db.AutoMigrate(&eventNewsEventSchema{}, &tournamentSchema{}, &tournamentMatchSchema{}, &playerSchema{})
}
