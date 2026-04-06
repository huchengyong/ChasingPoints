package wstsync

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientFetchSeasonsDecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v2" {
			t.Fatalf("unexpected path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id":"2025","attributes":{"name":"2025/26"}},
				{"id":"2024","attributes":{"name":"2024/25"}}
			]
		}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, srv.URL)
	items, err := client.FetchSeasons(context.Background())
	if err != nil {
		t.Fatalf("fetch seasons: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(items))
	}
	if items[0].ID != "2025" || items[0].Attributes.Name != "2025/26" {
		t.Fatalf("unexpected first season: %+v", items[0])
	}
}

func TestClientFetchTournamentsDecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("season"); got != "2025" {
			t.Fatalf("unexpected season query: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id":"t-1",
					"attributes":{
						"name":"Example Open 2025",
						"startDate":"2025-01-01",
						"endDate":"2025-01-07",
						"city":"Sheffield",
						"country":"England",
						"informationPage":"https://example.com",
						"season":{"id":"2025"}
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	client := NewClient("https://example.com/seasons", srv.URL, "https://example.com/matches")
	items, err := client.FetchTournamentsBySeason(context.Background(), 2025)
	if err != nil {
		t.Fatalf("fetch tournaments: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 tournament, got %d", len(items))
	}
	if items[0].ID != "t-1" || items[0].Attributes.Name != "Example Open 2025" {
		t.Fatalf("unexpected tournament: %+v", items[0])
	}
	if items[0].Attributes.Season.ID != "2025" {
		t.Fatalf("unexpected season id: %+v", items[0].Attributes.Season)
	}
}

func TestClientFetchMatchesPageDecodesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page.number"); got != "1" {
			t.Fatalf("unexpected page.number: %s", got)
		}
		if got := r.URL.Query().Get("page.size"); got != "50" {
			t.Fatalf("unexpected page.size: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id":"m-1",
					"attributes":{
						"name":"Player A vs Player B",
						"homePlayerID":"p-1",
						"homePlayerScore":3,
						"awayPlayerID":"p-2",
						"awayPlayerScore":1,
						"tournamentID":"t-1",
						"startDateTime":"2025-01-01 12:00:00",
						"round":"Round 1",
						"status":"Scheduled",
						"numberOfFrames":7,
						"fixtureNumber":1,
						"playersAllocated":true,
						"published":true,
						"homePlayer":{"playerID":"p-1","firstName":"Player","surname":"A","countryCode":"gb-eng","media":{"profile":"a.png"}},
						"awayPlayer":{"playerID":"p-2","firstName":"Player","surname":"B","countryCode":"gb-eng","media":{"profile":"b.png"}},
						"tournament":{"tournamentID":"t-1","name":"Example Open 2025","startDate":"2025-01-01","endDate":"2025-01-07","city":"Sheffield","country":"England","ticketingLink":"https://tickets.example.com"}
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	client := NewClient("https://example.com/seasons", "https://example.com/tournaments", srv.URL)
	page, err := client.FetchMatchesPage(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("fetch matches: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 match, got %d", len(page.Data))
	}
	if page.Data[0].Attributes.TournamentID != "t-1" {
		t.Fatalf("unexpected tournament id: %+v", page.Data[0].Attributes)
	}
	if page.Data[0].Attributes.HomePlayer.FirstName != "Player" {
		t.Fatalf("unexpected home player: %+v", page.Data[0].Attributes.HomePlayer)
	}
	if page.Data[0].Attributes.HomePlayerScore != 3 || page.Data[0].Attributes.AwayPlayerScore != 1 {
		t.Fatalf("unexpected score mapping: %+v", page.Data[0].Attributes)
	}
	if page.Data[0].Attributes.Tournament.TicketingLink != "https://tickets.example.com" {
		t.Fatalf("unexpected ticketing link: %+v", page.Data[0].Attributes.Tournament)
	}
}

func TestClientFetchMatchesPageRetriesTransientEOF(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("response writer does not support hijacking")
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m-1","attributes":{"tournamentID":"t-1","round":"Round 1","status":"Scheduled"}}]}`))
	}))
	defer srv.Close()

	client := NewClient("https://example.com/seasons", "https://example.com/tournaments", srv.URL)
	page, err := client.FetchMatchesPage(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("expected retry to recover from transient EOF, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "m-1" {
		t.Fatalf("unexpected page data: %+v", page.Data)
	}
}

func TestClientFetchMatchesPageRetriesTimeoutOnce(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			time.Sleep(80 * time.Millisecond)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	client := NewClient("https://example.com/seasons", "https://example.com/tournaments", srv.URL)
	client.httpClient.Timeout = 20 * time.Millisecond

	_, err := client.FetchMatchesPage(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("expected retry to recover from timeout, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts after timeout, got %d", attempts)
	}
}

func TestIsRetryableRequestError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "eof", err: net.ErrClosed, want: false},
		{name: "custom eof", err: &net.OpError{Err: net.UnknownNetworkError("EOF")}, want: true},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
	}

	for _, tc := range testCases {
		if got := isRetryableRequestError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
