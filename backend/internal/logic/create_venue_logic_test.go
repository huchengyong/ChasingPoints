package logic

import (
	"testing"

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

	venue, task, err := buildVenueAndTaskFromReq(1001, req, 5)
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
	if resp.Message != "已提交，审核通过后会员将自动到账" {
		t.Fatalf("unexpected success message %q", resp.Message)
	}
}
