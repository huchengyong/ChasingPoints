package wstsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout   = 2 * time.Minute
	defaultRequestRetrys = 3
	pageCoverTimeout     = 12 * time.Second
)

type Client struct {
	seasonsURL     string
	tournamentsURL string
	matchesURL     string
	httpClient     *http.Client
}

var (
	metaTagPattern  = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	metaAttrPattern = regexp.MustCompile(`(?i)([a-zA-Z_:.-]+)\s*=\s*("([^"]*)"|'([^']*)')`)
)

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

func (c *Client) FetchPageCoverImage(ctx context.Context, pageURL string) (string, error) {
	pageCtx, cancel := context.WithTimeout(ctx, pageCoverTimeout)
	defer cancel()

	content, err := c.getTextWithAttempts(pageCtx, strings.TrimSpace(pageURL), 1)
	if err != nil {
		return "", err
	}
	return extractPageCoverImage(pageURL, content), nil
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

func (c *Client) getText(ctx context.Context, endpoint string) (string, error) {
	return c.getTextWithAttempts(ctx, endpoint, defaultRequestRetrys)
}

func (c *Client) getTextWithAttempts(ctx context.Context, endpoint string, maxAttempts int) (string, error) {
	if c == nil {
		return "", fmt.Errorf("wst client is nil")
	}
	if strings.TrimSpace(endpoint) == "" {
		return "", nil
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return "", err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetryRequest(attempt, err) {
				continue
			}
			return "", err
		}

		body, readErr := func() ([]byte, error) {
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return nil, fmt.Errorf("wst request failed: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return body, nil
		}()
		if readErr == nil {
			return string(body), nil
		}

		lastErr = readErr
		if attempt >= maxAttempts || !shouldRetryRequest(attempt, readErr) {
			return "", readErr
		}
	}

	return "", lastErr
}

func extractPageCoverImage(pageURL, html string) string {
	candidates := make([]string, 0, 2)
	for _, metaTag := range metaTagPattern.FindAllString(html, -1) {
		attrs := extractMetaTagAttributes(metaTag)
		property := strings.ToLower(attrs["property"])
		name := strings.ToLower(attrs["name"])
		content := strings.TrimSpace(attrs["content"])
		if content == "" {
			continue
		}
		if property == "og:image" || name == "og:image" {
			return resolveMetaImageURL(pageURL, content)
		}
		if property == "twitter:image" || name == "twitter:image" {
			candidates = append(candidates, resolveMetaImageURL(pageURL, content))
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func extractMetaTagAttributes(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range metaAttrPattern.FindAllStringSubmatch(tag, -1) {
		if len(match) < 5 {
			continue
		}
		value := match[3]
		if value == "" {
			value = match[4]
		}
		attrs[strings.ToLower(match[1])] = value
	}
	return attrs
}

func resolveMetaImageURL(pageURL, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err == nil && parsed.IsAbs() {
		return parsed.String()
	}

	base, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil {
		return value
	}

	ref, err := url.Parse(value)
	if err != nil {
		return value
	}
	return base.ResolveReference(ref).String()
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
