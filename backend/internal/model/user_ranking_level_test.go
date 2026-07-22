package model

import (
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRankingModelCalculateLevelUsesSixTierBoundaries(t *testing.T) {
	model := NewRankingModel(nil)
	tests := []struct {
		score int
		level int
	}{
		{score: 1999, level: 4},
		{score: 2000, level: 5},
		{score: 2499, level: 5},
		{score: 2500, level: 6},
	}

	for _, tt := range tests {
		if got := model.CalculateLevel(tt.score); got != tt.level {
			t.Fatalf("score %d: expected level %d, got %d", tt.score, tt.level, got)
		}
	}
}

func TestRankingModelFindOrCreateByGameTypesIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&UserRanking{}); err != nil {
		t.Fatalf("migrate rankings: %v", err)
	}
	rankingModel := NewRankingModel(db)
	const userID int64 = 1005
	var wg sync.WaitGroup
	results := make(chan []UserRanking, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rankings, err := rankingModel.FindOrCreateByGameTypes(userID)
			if err != nil {
				t.Errorf("initialize rankings: %v", err)
				return
			}
			results <- rankings
		}()
	}
	wg.Wait()
	close(results)
	for rankings := range results {
		if len(rankings) != len(SupportedRankingGameTypes()) {
			t.Fatalf("expected complete ranking result, got %#v", rankings)
		}
		for index, ranking := range rankings {
			if ranking.GameType != index+1 {
				t.Fatalf("expected game type %d at index %d, got %#v", index+1, index, rankings)
			}
		}
	}
	var count int64
	if err := db.Model(&UserRanking{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count rankings: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected four rows after concurrent initialization, got %d", count)
	}
}
