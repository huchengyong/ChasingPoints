package geocode

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"billiard_master/internal/model"
)

func TestApihzClientGeocodeSuccess(t *testing.T) {
	client := NewApihzClient("https://example.com/geocode", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.Query().Get("id"); got != "1001" {
			t.Fatalf("expected id=1001, got %s", got)
		}
		if got := r.URL.Query().Get("key"); got != "secret" {
			t.Fatalf("expected key=secret, got %s", got)
		}
		if got := r.URL.Query().Get("address"); got != "上海市浦东新区东明路" {
			t.Fatalf("unexpected address %s", got)
		}

		return jsonResponse(`{"code":200,"lng":"121.496","lat":"31.14238","score":75,"level":"乡镇街道"}`), nil
	})
	account := model.GeocodeAccount{
		ProviderAppID: "1001",
		ProviderKey:   "secret",
	}

	result, err := client.Geocode(context.Background(), account, "上海市浦东新区东明路")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Longitude != 121.496 {
		t.Fatalf("expected longitude 121.496, got %v", result.Longitude)
	}
	if result.Latitude != 31.14238 {
		t.Fatalf("expected latitude 31.14238, got %v", result.Latitude)
	}
	if result.Score != 75 {
		t.Fatalf("expected score 75, got %d", result.Score)
	}
	if result.Level != "乡镇街道" {
		t.Fatalf("expected level 乡镇街道, got %s", result.Level)
	}
}

func TestApihzClientGeocodeAuthFailure(t *testing.T) {
	client := NewApihzClient("https://example.com/geocode", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"code":403,"msg":"key invalid"}`), nil
	})
	account := model.GeocodeAccount{
		ProviderAppID: "1001",
		ProviderKey:   "bad-key",
	}

	_, err := client.Geocode(context.Background(), account, "上海市浦东新区东明路")
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Kind != ErrorKindAuth {
		t.Fatalf("expected auth error kind, got %s", providerErr.Kind)
	}
}

func TestApihzClientGeocodeRejectsInvalidCoordinates(t *testing.T) {
	client := NewApihzClient("https://example.com/geocode", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"code":200,"lng":"","lat":"","score":20,"level":"城市"}`), nil
	})
	account := model.GeocodeAccount{
		ProviderAppID: "1001",
		ProviderKey:   "secret",
	}

	_, err := client.Geocode(context.Background(), account, "上海市浦东新区东明路")
	if err == nil {
		t.Fatal("expected invalid response error, got nil")
	}

	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Kind != ErrorKindInvalidResponse {
		t.Fatalf("expected invalid_response kind, got %s", providerErr.Kind)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
