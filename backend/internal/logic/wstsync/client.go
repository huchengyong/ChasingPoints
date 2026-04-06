package wstsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout   = 2 * time.Minute
	defaultRequestRetrys = 3
)

type Client struct {
	seasonsURL     string
	tournamentsURL string
	matchesURL     string
	httpClient     *http.Client
}

func NewClient(seasonsURL, tournamentsURL, matchesURL string) *Client {
	return &Client{
		seasonsURL:     strings.TrimRight(seasonsURL, "/"),
		tournamentsURL: strings.TrimRight(tournamentsURL, "/"),
		matchesURL:     strings.TrimRight(matchesURL, "/"),
		httpClient:     &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (c *Client) FetchSeasons(ctx context.Context) ([]SeasonResource, error) {
	var resp SeasonListResponse
	if err := c.getJSON(ctx, c.seasonsURL+"/v2", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) FetchTournamentsBySeason(ctx context.Context, season int) ([]TournamentResource, error) {
	query := url.Values{}
	query.Set("season", fmt.Sprintf("%d", season))
	query.Set("page.number", "1")
	query.Set("page.size", "200")

	var resp TournamentListResponse
	if err := c.getJSON(ctx, c.tournamentsURL+"/v2", query, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) FetchMatchesPage(ctx context.Context, pageNumber, pageSize int) (MatchListResponse, error) {
	query := url.Values{}
	query.Set("page.number", fmt.Sprintf("%d", pageNumber))
	query.Set("page.size", fmt.Sprintf("%d", pageSize))

	var resp MatchListResponse
	if err := c.getJSON(ctx, c.matchesURL+"/v2", query, &resp); err != nil {
		return MatchListResponse{}, err
	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	if c == nil {
		return fmt.Errorf("wst client is nil")
	}
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 1; attempt <= defaultRequestRetrys; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetryRequest(attempt, err) {
				continue
			}
			return err
		}

		readErr := func() error {
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("wst request failed: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(body, target); err != nil {
				return err
			}
			return nil
		}()
		if readErr == nil {
			return nil
		}

		lastErr = readErr
		if !shouldRetryRequest(attempt, readErr) {
			return readErr
		}
	}

	return lastErr
}

func shouldRetryRequest(attempt int, err error) bool {
	return attempt < defaultRequestRetrys && isRetryableRequestError(err)
}

func isRetryableRequestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errText := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(errText, "timeout") || strings.Contains(errText, "deadline exceeded") || strings.Contains(errText, "eof") {
		return true
	}
	return false
}
