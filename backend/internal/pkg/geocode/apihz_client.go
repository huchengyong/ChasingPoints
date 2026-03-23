package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"billiard_master/internal/model"
)

const defaultApihzEndpoint = "https://cn.apihz.cn/api/other/jwjuhe.php"

type ApihzClient struct {
	baseURL    string
	httpClient *http.Client
}

type apihzResponse struct {
	Code    int    `json:"code"`
	Lng     string `json:"lng"`
	Lat     string `json:"lat"`
	Score   int    `json:"score"`
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
}

func NewApihzClient(baseURL string, timeout time.Duration) *ApihzClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultApihzEndpoint
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &ApihzClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *ApihzClient) Geocode(ctx context.Context, account model.GeocodeAccount, address string) (*Result, error) {
	queryURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, err.Error())
	}

	query := queryURL.Query()
	query.Set("id", account.ProviderAppID)
	query.Set("key", account.ProviderKey)
	query.Set("address", address)
	queryURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, NewProviderError(ErrorKindTemporary, err.Error())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewProviderError(ErrorKindTemporary, err.Error())
	}
	defer resp.Body.Close()

	var payload apihzResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, err.Error())
	}

	if payload.Code != http.StatusOK {
		return nil, NewProviderError(classifyApihzError(resp.StatusCode, payload.Code, payload.errorMessage()), payload.errorMessage())
	}

	longitude, err := strconv.ParseFloat(strings.TrimSpace(payload.Lng), 64)
	if err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, fmt.Sprintf("invalid longitude: %v", err))
	}
	latitude, err := strconv.ParseFloat(strings.TrimSpace(payload.Lat), 64)
	if err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, fmt.Sprintf("invalid latitude: %v", err))
	}
	if longitude == 0 || latitude == 0 {
		return nil, NewProviderError(ErrorKindInvalidResponse, "empty coordinates")
	}

	return &Result{
		Longitude: longitude,
		Latitude:  latitude,
		Score:     payload.Score,
		Level:     payload.Level,
		RawCode:   payload.Code,
		Source:    "apihz",
	}, nil
}

func classifyApihzError(httpStatus, responseCode int, message string) string {
	lowerMessage := strings.ToLower(message)
	if httpStatus == http.StatusUnauthorized || httpStatus == http.StatusForbidden || responseCode == http.StatusUnauthorized || responseCode == http.StatusForbidden {
		return ErrorKindAuth
	}
	if strings.Contains(lowerMessage, "key") || strings.Contains(lowerMessage, "id") || strings.Contains(lowerMessage, "auth") {
		return ErrorKindAuth
	}
	if httpStatus >= http.StatusInternalServerError {
		return ErrorKindTemporary
	}
	return ErrorKindTemporary
}

func (r apihzResponse) errorMessage() string {
	if strings.TrimSpace(r.Msg) != "" {
		return strings.TrimSpace(r.Msg)
	}
	if strings.TrimSpace(r.Message) != "" {
		return strings.TrimSpace(r.Message)
	}
	return "upstream geocode request failed"
}
