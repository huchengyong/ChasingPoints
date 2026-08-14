package venue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetVenueListUsesAggregateCheckinQuery(t *testing.T) {
	oneQueries := venueListQueryCount(t, 1)
	hundredQueries := venueListQueryCount(t, 100)
	if oneQueries != 2 || hundredQueries != 2 {
		t.Fatalf("venue list must use one COUNT and one page query with checkin aggregation: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func venueListQueryCount(t *testing.T, venueCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", venueCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Venue{}, &model.VenueCheckin{}); err != nil {
		t.Fatalf("prepare venue schema: %v", err)
	}
	venues := make([]model.Venue, 0, venueCount)
	checkins := make([]model.VenueCheckin, 0, venueCount*2)
	for i := 1; i <= venueCount; i++ {
		venueID := int64(i)
		venues = append(venues, model.Venue{Id: venueID, Name: fmt.Sprintf("球馆%d", i), City: "上海", FullAddress: fmt.Sprintf("上海地址%d", i), Status: model.VenueStatusPublished, GeoStatus: model.VenueGeoStatusSuccess, Latitude: 31.2, Longitude: 121.4})
		for checkin := 0; checkin < i%3; checkin++ {
			checkins = append(checkins, model.VenueCheckin{VenueId: venueID, UserId: int64(checkin + 1)})
		}
	}
	if err := db.Create(&venues).Error; err != nil {
		t.Fatalf("seed venues: %v", err)
	}
	if len(checkins) > 0 {
		if err := db.Create(&checkins).Error; err != nil {
			t.Fatalf("seed checkins: %v", err)
		}
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	svcCtx := &svc.ServiceContext{VenueModel: model.NewVenueModel(db.WithContext(ctx))}
	resp, err := NewGetVenueListLogic(ctx, svcCtx).GetVenueList(&types.GetVenueListReq{Page: 1, PageSize: 100, City: "上海"})
	if err != nil || !resp.Success || resp.Total != int64(venueCount) || len(resp.List) != venueCount {
		t.Fatalf("get venues: resp=%#v err=%v", resp, err)
	}
	if venueCount >= 2 && resp.List[venueCount-2].CheckinCount != 2 {
		t.Fatalf("aggregate checkin count missing: %#v", resp.List)
	}
	return int(metrics.Snapshot().SQLCount)
}
