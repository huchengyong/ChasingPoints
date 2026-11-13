package venue

import (
	"math"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestValidateCreateVenueReqRejectsMissingBaseFields(t *testing.T) {
	req := &types.CreateVenueReq{
		Name:    " ",
		City:    "上海市",
		Address: "浦东新区东明路",
	}

	if err := validateCreateVenueReq(req); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestBuildVenueAndTaskFromReqCreatesAsyncGeocodePayload(t *testing.T) {
	req := &types.CreateVenueReq{
		Name:     "UK台球",
		City:     "上海市",
		District: "浦东新区",
		Address:  "东明路街道新达汇",
	}

	venue, task, err := buildVenueAndTaskFromReq(1001, req, 5, true, time.Now())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if venue.FullAddress != "上海市浦东新区东明路街道新达汇" {
		t.Fatalf("unexpected full address %s", venue.FullAddress)
	}
	if venue.Status != model.VenueStatusPending {
		t.Fatalf("expected pending status, got %d", venue.Status)
	}
	if venue.GeoStatus != model.VenueGeoStatusPending {
		t.Fatalf("expected pending geocode status, got %d", venue.GeoStatus)
	}
	if venue.Latitude != 0 || venue.Longitude != 0 {
		t.Fatalf("expected zero coordinates before geocode, got %v,%v", venue.Latitude, venue.Longitude)
	}
	if venue.Phone != "" {
		t.Fatalf("expected empty phone, got %q", venue.Phone)
	}
	if venue.BusinessHours != "" {
		t.Fatalf("expected empty business hours, got %q", venue.BusinessHours)
	}
	if venue.TableCount != 0 {
		t.Fatalf("expected zero table count, got %d", venue.TableCount)
	}
	if venue.PriceRange != "" {
		t.Fatalf("expected empty price range, got %q", venue.PriceRange)
	}
	if venue.Description != "" {
		t.Fatalf("expected empty description, got %q", venue.Description)
	}
	if venue.Images != "[]" {
		t.Fatalf("expected empty images json, got %s", venue.Images)
	}
	if task.Status != model.VenueGeocodeTaskStatusPending {
		t.Fatalf("expected pending task status, got %d", task.Status)
	}
	if task.MaxAttempts != 5 {
		t.Fatalf("expected max attempts 5, got %d", task.MaxAttempts)
	}
}

func TestBuildVenueAndTaskFromReqUsesPickedLocationAndSkipsGeocodeTask(t *testing.T) {
	req := &types.CreateVenueReq{
		Name:      "UK台球",
		City:      "上海市",
		District:  "浦东新区",
		Address:   "东明路街道新达汇",
		Latitude:  31.234583,
		Longitude: 121.507572,
	}

	now := time.Now()
	venue, task, err := buildVenueAndTaskFromReq(1001, req, 5, true, now)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if task != nil {
		t.Fatalf("expected no geocode task for picked location, got %#v", task)
	}
	if venue.Latitude != req.Latitude || venue.Longitude != req.Longitude {
		t.Fatalf("unexpected coordinates %v,%v", venue.Latitude, venue.Longitude)
	}
	if venue.Status != model.VenueStatusPublished {
		t.Fatalf("expected published status, got %d", venue.Status)
	}
	if venue.GeoStatus != model.VenueGeoStatusSuccess {
		t.Fatalf("expected success geocode status, got %d", venue.GeoStatus)
	}
	if venue.GeoSource != "user_pick" {
		t.Fatalf("unexpected geo source %q", venue.GeoSource)
	}
	if venue.GeoLevel != "用户选点" {
		t.Fatalf("unexpected geo level %q", venue.GeoLevel)
	}
	if venue.GeoUpdatedAt == nil {
		t.Fatal("expected geo updated at to be set")
	}
}

func TestBuildVenueAndTaskFromReqKeepsPickedLocationPendingWithoutAutoPublish(t *testing.T) {
	req := &types.CreateVenueReq{
		Name:      "UK台球",
		City:      "上海市",
		District:  "浦东新区",
		Address:   "东明路街道新达汇",
		Latitude:  31.234583,
		Longitude: 121.507572,
	}

	venue, task, err := buildVenueAndTaskFromReq(1001, req, 5, false, time.Now())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if task != nil {
		t.Fatalf("expected no geocode task for picked location, got %#v", task)
	}
	if venue.Status != model.VenueStatusPending {
		t.Fatalf("expected pending status, got %d", venue.Status)
	}
}

func TestValidateCreateVenueLocation(t *testing.T) {
	cases := []struct {
		name      string
		latitude  float64
		longitude float64
		wantPick  bool
		wantErr   bool
	}{
		{name: "未选点", latitude: 0, longitude: 0},
		{name: "缺少经度", latitude: 31.23, longitude: 0, wantErr: true},
		{name: "缺少纬度", latitude: 0, longitude: 121.5, wantErr: true},
		{name: "纬度超范围", latitude: 91, longitude: 121.5, wantErr: true},
		{name: "经度超范围", latitude: 31.23, longitude: 200, wantErr: true},
		{name: "境外坐标", latitude: -33.8, longitude: 151.2, wantErr: true},
		{name: "NaN", latitude: math.NaN(), longitude: 121.5, wantErr: true},
		{name: "有效选点", latitude: 31.234583, longitude: 121.507572, wantPick: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			picked, err := validateCreateVenueLocation(tc.latitude, tc.longitude)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if picked != tc.wantPick {
				t.Fatalf("expected picked=%v, got %v", tc.wantPick, picked)
			}
		})
	}
}

func TestValidateCreateVenueReqRejectsIncompleteLocation(t *testing.T) {
	req := &types.CreateVenueReq{
		Name:      "UK台球",
		City:      "上海市",
		Address:   "东明路街道新达汇",
		Latitude:  31.23,
		Longitude: 0,
	}

	if err := validateCreateVenueReq(req); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestBuildCreateVenueRespUsesAsyncMessage(t *testing.T) {
	resp := buildCreateVenueResp(88)

	if !resp.Success {
		t.Fatal("expected success response")
	}
	if resp.VenueId != 88 {
		t.Fatalf("expected venue id 88, got %d", resp.VenueId)
	}
	if resp.Message == "" {
		t.Fatal("expected async message, got empty string")
	}
	if resp.Message != "已提交，审核通过后会展示给附近球友" {
		t.Fatalf("unexpected success message %q", resp.Message)
	}
}
