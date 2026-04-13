package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModelConstructorsDoNotAutoMigrate(t *testing.T) {
	testCases := []struct {
		name      string
		tableName string
		construct func(db *gorm.DB)
	}{
		{
			name:      "area model",
			tableName: "dou_area",
			construct: func(db *gorm.DB) { NewAreaModel(db) },
		},
		{
			name:      "user model",
			tableName: "users",
			construct: func(db *gorm.DB) { NewUserModel(db) },
		},
		{
			name:      "venue model",
			tableName: "venues",
			construct: func(db *gorm.DB) { NewVenueModel(db) },
		},
		{
			name:      "venue checkin model",
			tableName: "venue_checkins",
			construct: func(db *gorm.DB) { NewVenueCheckinModel(db) },
		},
		{
			name:      "venue geocode task model",
			tableName: "venue_geocode_tasks",
			construct: func(db *gorm.DB) { NewVenueGeocodeTaskModel(db) },
		},
		{
			name:      "geocode account model",
			tableName: "geocode_accounts",
			construct: func(db *gorm.DB) { NewGeocodeAccountModel(db) },
		},
		{
			name:      "tournament model",
			tableName: "tournaments",
			construct: func(db *gorm.DB) { NewTournamentModel(db) },
		},
		{
			name:      "tournament participant model",
			tableName: "tournament_participants",
			construct: func(db *gorm.DB) { NewTournamentParticipantModel(db) },
		},
		{
			name:      "tournament match model",
			tableName: "tournament_matches",
			construct: func(db *gorm.DB) { NewTournamentMatchModel(db) },
		},
		{
			name:      "season model",
			tableName: "seasons",
			construct: func(db *gorm.DB) { NewSeasonModel(db) },
		},
		{
			name:      "challenge model",
			tableName: "challenges",
			construct: func(db *gorm.DB) { NewChallengeModel(db) },
		},
		{
			name:      "favorite venue reward config model",
			tableName: "favorite_venue_reward_configs",
			construct: func(db *gorm.DB) { NewFavoriteVenueRewardConfigModel(db) },
		},
		{
			name:      "favorite venue reward record model",
			tableName: "favorite_venue_reward_records",
			construct: func(db *gorm.DB) { NewFavoriteVenueRewardRecordModel(db) },
		},
		{
			name:      "member subscription order model",
			tableName: "member_subscription_orders",
			construct: func(db *gorm.DB) { NewMemberSubscriptionOrderModel(db) },
		},
		{
			name:      "member growth profile model",
			tableName: "member_growth_profiles",
			construct: func(db *gorm.DB) { NewMemberGrowthProfileModel(db) },
		},
		{
			name:      "member growth log model",
			tableName: "member_growth_logs",
			construct: func(db *gorm.DB) { NewMemberGrowthLogModel(db) },
		},
		{
			name:      "member rights config model",
			tableName: "member_rights_configs",
			construct: func(db *gorm.DB) { NewMemberRightsConfigModel(db) },
		},
		{
			name:      "feedback ticket model",
			tableName: "feedback_tickets",
			construct: func(db *gorm.DB) { NewFeedbackTicketModel(db) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite db: %v", err)
			}

			if db.Migrator().HasTable(tc.tableName) {
				t.Fatalf("table %s should not exist before constructor", tc.tableName)
			}

			tc.construct(db)

			if db.Migrator().HasTable(tc.tableName) {
				t.Fatalf("constructor unexpectedly created table %s", tc.tableName)
			}
		})
	}
}
