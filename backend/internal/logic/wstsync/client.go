package wstsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout   = 2 * time.Minute
	defaultRequestRetrys = 3
	pageCoverTimeout     = 12 * time.Second
	maxBackoffDelay      = 30 * time.Second
	maxTotalRetryWait    = 60 * time.Second
	maxPaginationPages   = 500
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
	var all []SeasonResource
	seen := make(map[string]SeasonResource)
	var expectedTotal *int
	for page := 1; page <= maxPaginationPages; page++ {
		query := url.Values{}
		query.Set("page.number", strconv.Itoa(page))
		query.Set("page.size", "200")
		var resp SeasonListResponse
		if _, err := c.getJSONWithRetry(ctx, c.seasonsURL+"/v2", query, &resp); err != nil {
			return nil, fmt.Errorf("fetch seasons page %d: %w", page, err)
		}
		if resp.Data == nil {
			return nil, fmt.Errorf("wst seasons page %d response missing 'data' field", page)
		}
		if err := validatePaginationMeta("seasons", page, len(resp.Data), resp.Meta, &expectedTotal); err != nil {
			return nil, err
		}
		if len(resp.Data) == 0 {
			if resp.Links != nil && resp.Links.Next != nil {
				return nil, fmt.Errorf("wst seasons page %d is empty but links.next is present", page)
			}
			if expectedTotal != nil && len(seen) != *expectedTotal {
				return nil, fmt.Errorf("wst seasons ended with %d unique records, expected %d", len(seen), *expectedTotal)
			}
			return all, nil
		}
		newIDs := 0
		for _, item := range resp.Data {
			if strings.TrimSpace(item.ID) == "" {
				return nil, fmt.Errorf("wst seasons page %d contains an empty id", page)
			}
			if current, ok := seen[item.ID]; ok {
				if !reflect.DeepEqual(current, item) {
					return nil, fmt.Errorf("wst seasons duplicate id %s has conflicting data", item.ID)
				}
				continue
			}
			seen[item.ID] = item
			all = append(all, item)
			newIDs++
		}
		if newIDs == 0 {
			return nil, fmt.Errorf("wst seasons pagination stalled at page %d (no new IDs)", page)
		}
		if expectedTotal != nil {
			if len(seen) > *expectedTotal {
				return nil, fmt.Errorf("wst seasons returned %d unique records, expected %d", len(seen), *expectedTotal)
			}
			if len(seen) == *expectedTotal {
				if resp.Links != nil && resp.Links.Next != nil {
					return nil, fmt.Errorf("wst seasons reached expected total %d at page %d but links.next is present", *expectedTotal, page)
				}
				return all, nil
			}
		}
		if resp.Links != nil && resp.Links.Next == nil {
			if expectedTotal != nil {
				return nil, fmt.Errorf("wst seasons pagination ended at page %d before expected total", page)
			}
			return all, nil
		}
	}
	return nil, fmt.Errorf("wst seasons pagination exceeded page limit %d", maxPaginationPages)
}

func (c *Client) FetchTournamentsBySeason(ctx context.Context, season int) ([]TournamentResource, error) {
	var all []TournamentResource
	seen := make(map[string]TournamentResource)
	var expectedTotal *int

	for page := 1; page <= maxPaginationPages; page++ {
		query := url.Values{}
		query.Set("season", fmt.Sprintf("%d", season))
		query.Set("page.number", fmt.Sprintf("%d", page))
		query.Set("page.size", "200")

		var resp TournamentListResponse
		_, err := c.getJSONWithRetry(ctx, c.tournamentsURL+"/v2", query, &resp)
		if err != nil {
			return nil, fmt.Errorf("fetch tournaments season %d page %d: %w", season, page, err)
		}
		if resp.Data == nil {
			return nil, fmt.Errorf("wst tournaments season %d page %d response missing 'data' field", season, page)
		}
		if err := validatePaginationMeta(fmt.Sprintf("tournaments season %d", season), page, len(resp.Data), resp.Meta, &expectedTotal); err != nil {
			return nil, err
		}
		if len(resp.Data) == 0 {
			if resp.Links != nil && resp.Links.Next != nil {
				return nil, fmt.Errorf("wst tournaments season %d page %d is empty but links.next is present", season, page)
			}
			if expectedTotal != nil && len(seen) != *expectedTotal {
				return nil, fmt.Errorf("wst tournaments season %d ended with %d unique records, expected %d", season, len(seen), *expectedTotal)
			}
			return all, nil
		}

		newIDs := 0
		for _, item := range resp.Data {
			if strings.TrimSpace(item.ID) == "" {
				return nil, fmt.Errorf("wst tournaments season %d page %d contains an empty id", season, page)
			}
			if item.Attributes.Season.ID == "" {
				item.Attributes.Season.ID = strconv.Itoa(season)
			}
			if current, ok := seen[item.ID]; ok {
				if !reflect.DeepEqual(current, item) {
					return nil, fmt.Errorf("wst tournaments season %d duplicate id %s has conflicting data", season, item.ID)
				}
				continue
			}
			seen[item.ID] = item
			all = append(all, item)
			newIDs++
		}
		if newIDs == 0 {
			return nil, fmt.Errorf("wst tournaments season %d pagination stalled at page %d (no new IDs)", season, page)
		}
		if expectedTotal != nil {
			if len(seen) > *expectedTotal {
				return nil, fmt.Errorf("wst tournaments season %d returned %d unique records, expected %d", season, len(seen), *expectedTotal)
			}
			if len(seen) == *expectedTotal {
				if resp.Links != nil && resp.Links.Next != nil {
					return nil, fmt.Errorf("wst tournaments season %d reached expected total %d at page %d but links.next is present", season, *expectedTotal, page)
				}
				return all, nil
			}
		}
		if resp.Links != nil && resp.Links.Next == nil {
			if expectedTotal != nil {
				return nil, fmt.Errorf("wst tournaments season %d pagination ended at page %d before expected total", season, page)
			}
			return all, nil
		}
	}

	return nil, fmt.Errorf("wst tournaments season %d pagination exceeded page limit %d", season, maxPaginationPages)
}

func (c *Client) FetchMatchesPage(ctx context.Context, pageNumber, pageSize int) (MatchListResponse, error) {
	query := url.Values{}
	query.Set("page.number", fmt.Sprintf("%d", pageNumber))
	query.Set("page.size", fmt.Sprintf("%d", pageSize))

	var resp MatchListResponse
	if _, err := c.getJSONWithRetry(ctx, c.matchesURL+"/v2", query, &resp); err != nil {
		return MatchListResponse{}, err
	}
	if resp.Data == nil {
		return MatchListResponse{}, fmt.Errorf("wst matches page %d response missing 'data' field", pageNumber)
	}
	if resp.Meta != nil && resp.Meta.Count != nil && *resp.Meta.Count != len(resp.Data) {
		return MatchListResponse{}, fmt.Errorf("wst matches page %d meta count %d does not match data length %d", pageNumber, *resp.Meta.Count, len(resp.Data))
	}
	return resp, nil
}

func validatePaginationMeta(label string, page, dataCount int, meta *PaginationMeta, expectedTotal **int) error {
	if meta == nil {
		return nil
	}
	if meta.Count != nil && *meta.Count != dataCount {
		return fmt.Errorf("wst %s page %d meta count %d does not match data length %d", label, page, *meta.Count, dataCount)
	}
	if meta.TotalCount == nil {
		return nil
	}
	if *meta.TotalCount < 0 {
		return fmt.Errorf("wst %s page %d has invalid total count %d", label, page, *meta.TotalCount)
	}
	if *expectedTotal == nil {
		total := *meta.TotalCount
		*expectedTotal = &total
		return nil
	}
	if **expectedTotal != *meta.TotalCount {
		return fmt.Errorf("wst %s total count changed from %d to %d at page %d", label, **expectedTotal, *meta.TotalCount, page)
	}
	return nil
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

// getJSONWithRetry performs a GET request with retry/backoff for 429 and 5xx.
// It returns the total retry wait time alongside any error.
func (c *Client) getJSONWithRetry(ctx context.Context, endpoint string, query url.Values, target any) (time.Duration, error) {
	if c == nil {
		return 0, fmt.Errorf("wst client is nil")
	}
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var totalWait time.Duration
	var lastErr error

	for attempt := 1; attempt <= defaultRequestRetrys; attempt++ {
		if err := ctx.Err(); err != nil {
			return totalWait, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return totalWait, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetryRequest(attempt, err) {
				wait := backoffDelay(attempt, totalWait)
				totalWait += wait
				if sleepErr := sleepContext(ctx, wait); sleepErr != nil {
					return totalWait, sleepErr
				}
				continue
			}
			return totalWait, err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := validateDataField(body); err != nil {
				return totalWait, err
			}
			if err := json.Unmarshal(body, target); err != nil {
				return totalWait, err
			}
			return totalWait, nil
		}

		if readErr != nil {
			lastErr = readErr
		} else {
			lastErr = fmt.Errorf("wst request failed: %s", resp.Status)
		}
		retryable := readErr != nil && shouldRetryRequest(attempt, readErr) || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if !retryable || attempt == defaultRequestRetrys {
			return totalWait, lastErr
		}
		wait := backoffDelay(attempt, totalWait)
		if resp.StatusCode == http.StatusTooManyRequests {
			wait = retryAfterDelay(resp, attempt, totalWait)
		}
		totalWait += wait
		if sleepErr := sleepContext(ctx, wait); sleepErr != nil {
			return totalWait, sleepErr
		}
	}

	return totalWait, lastErr
}

// validateDataField checks that the JSON response body contains a non-null 'data' field.
func validateDataField(body []byte) error {
	var raw struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	if raw.Data == nil {
		return fmt.Errorf("response missing 'data' field")
	}
	return nil
}

// retryAfterDelay parses the Retry-After header or computes a backoff delay for 429/5xx.
func retryAfterDelay(resp *http.Response, attempt int, totalWait time.Duration) time.Duration {
	if resp.StatusCode == 429 {
		if secs, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); err == nil && secs > 0 {
			wait := time.Duration(secs) * time.Second
			if totalWait+wait > maxTotalRetryWait {
				return maxTotalRetryWait - totalWait
			}
			return wait
		}
	}
	return backoffDelay(attempt, totalWait)
}

// backoffDelay computes exponential backoff: 1s, 2s, 4s, ... capped at maxBackoffDelay.
// It respects the maxTotalRetryWait budget.
func backoffDelay(attempt int, totalWait time.Duration) time.Duration {
	base := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
	if base > maxBackoffDelay {
		base = maxBackoffDelay
	}
	if totalWait+base > maxTotalRetryWait {
		remaining := maxTotalRetryWait - totalWait
		if remaining <= 0 {
			return 0
		}
		return remaining
	}
	return base
}

// sleepContext sleeps for d or until ctx is cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	_, err := c.getJSONWithRetry(ctx, endpoint, query, target)
	return err
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
