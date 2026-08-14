package model

import (
	"math"
	"testing"
)

func TestNearbyBoundsContainsRadiusCandidateAndExcludesDistantLatitude(t *testing.T) {
	minLatitude, maxLatitude, minLongitude, maxLongitude := nearbyBounds(31.2304, 121.4737, 5000)
	if !(minLatitude < 31.2304 && maxLatitude > 31.2304 && minLongitude < 121.4737 && maxLongitude > 121.4737) {
		t.Fatalf("nearby bounds must contain the query point: lat=[%f,%f] lng=[%f,%f]", minLatitude, maxLatitude, minLongitude, maxLongitude)
	}
	if maxLatitude-minLatitude >= 1 || math.Abs(maxLongitude-minLongitude) >= 1 {
		t.Fatalf("5km candidate prefilter is unexpectedly broad: lat=[%f,%f] lng=[%f,%f]", minLatitude, maxLatitude, minLongitude, maxLongitude)
	}
	if minLatitude <= 30.5 || maxLatitude >= 31.9 {
		t.Fatalf("far latitude must be excluded by the bounding box: lat=[%f,%f]", minLatitude, maxLatitude)
	}
}

func TestNearbyBoundsSupportsAntiMeridianWrap(t *testing.T) {
	_, _, minLongitude, maxLongitude := nearbyBounds(0, 179.99, 5000)
	if minLongitude >= 180 || maxLongitude <= 180 {
		t.Fatalf("expected antimeridian wrapping bounds, got [%f,%f]", minLongitude, maxLongitude)
	}
}
