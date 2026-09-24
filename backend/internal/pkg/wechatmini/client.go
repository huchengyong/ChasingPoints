package wechatmini

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginEndpoint       = "https://api.weixin.qq.com/sns/jscode2session"
	defaultAccessTokenEndpoint = "https://api.weixin.qq.com/cgi-bin/token"
	defaultPhoneEndpoint       = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"
	defaultTimeout             = 5 * time.Second
	tokenRefreshSkew           = time.Minute
)

type Client interface {
	ExchangeLoginCode(ctx context.Context, code string) (*Identity, error)
	GetPhoneNumber(ctx context.Context, code string) (string, error)
}

type Identity struct {
	OpenID  string
	UnionID string
}

type ErrorKind string

const (
	ErrorKindUnavailable     ErrorKind = "unavailable"
	ErrorKindRejected        ErrorKind = "rejected"
	ErrorKindTemporary       ErrorKind = "temporary"
	ErrorKindInvalidResponse ErrorKind = "invalid_response"
)

type Error struct {
	Kind ErrorKind
}

func (e *Error) Error() string {
	switch e.Kind {
	case ErrorKindUnavailable:
		return "微信小程序登录暂不可用"
	case ErrorKindRejected:
		return "微信授权已失效，请重试"
	case ErrorKindInvalidResponse:
		return "微信服务响应异常，请稍后重试"
	default:
		return "微信服务暂时不可用，请稍后重试"
	}
}

func IsErrorKind(err error, kind ErrorKind) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == kind
}

type HTTPClient struct {
	appID               string
	appSecret           string
	httpClient          *http.Client
	loginEndpoint       string
	accessTokenEndpoint string
	phoneEndpoint       string

	mu                   sync.Mutex
	accessToken          string
	accessTokenExpiresAt time.Time
}

func NewHTTPClient(appID, appSecret string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &HTTPClient{
		appID:               strings.TrimSpace(appID),
		appSecret:           strings.TrimSpace(appSecret),
		httpClient:          &http.Client{Timeout: timeout},
		loginEndpoint:       defaultLoginEndpoint,
		accessTokenEndpoint: defaultAccessTokenEndpoint,
		phoneEndpoint:       defaultPhoneEndpoint,
	}
}

type loginResponse struct {
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid"`
	ErrCode int    `json:"errcode"`
}

func (c *HTTPClient) ExchangeLoginCode(ctx context.Context, code string) (*Identity, error) {
	if !c.isConfigured() {
		return nil, &Error{Kind: ErrorKindUnavailable}
	}
	if strings.TrimSpace(code) == "" {
		return nil, &Error{Kind: ErrorKindRejected}
	}

	endpoint, err := withQuery(c.loginEndpoint, url.Values{
		"appid":      {c.appID},
		"secret":     {c.appSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	})
	if err != nil {
		return nil, &Error{Kind: ErrorKindTemporary}
	}

	var payload loginResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	if payload.ErrCode != 0 {
		return nil, &Error{Kind: classifyWechatError(payload.ErrCode)}
	}
	if strings.TrimSpace(payload.OpenID) == "" {
		return nil, &Error{Kind: ErrorKindInvalidResponse}
	}

	return &Identity{
		OpenID:  payload.OpenID,
		UnionID: payload.UnionID,
	}, nil
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
}

func (c *HTTPClient) GetPhoneNumber(ctx context.Context, code string) (string, error) {
	if !c.isConfigured() {
		return "", &Error{Kind: ErrorKindUnavailable}
	}
	if strings.TrimSpace(code) == "" {
		return "", &Error{Kind: ErrorKindRejected}
	}

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	endpoint, err := withQuery(c.phoneEndpoint, url.Values{"access_token": {accessToken}})
	if err != nil {
		return "", &Error{Kind: ErrorKindTemporary}
	}
	body, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return "", &Error{Kind: ErrorKindInvalidResponse}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", &Error{Kind: ErrorKindTemporary}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &Error{Kind: ErrorKindTemporary}
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", &Error{Kind: ErrorKindTemporary}
	}

	var payload phoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", &Error{Kind: ErrorKindInvalidResponse}
	}
	if payload.ErrCode != 0 {
		return "", &Error{Kind: classifyWechatError(payload.ErrCode)}
	}
	if strings.TrimSpace(payload.PhoneInfo.PhoneNumber) == "" {
		return "", &Error{Kind: ErrorKindInvalidResponse}
	}

	return payload.PhoneInfo.PhoneNumber, nil
}

type phoneResponse struct {
	ErrCode   int `json:"errcode"`
	PhoneInfo struct {
		PhoneNumber string `json:"phoneNumber"`
	} `json:"phone_info"`
}

func (c *HTTPClient) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Add(tokenRefreshSkew).Before(c.accessTokenExpiresAt) {
		return c.accessToken, nil
	}

	endpoint, err := withQuery(c.accessTokenEndpoint, url.Values{
		"grant_type": {"client_credential"},
		"appid":      {c.appID},
		"secret":     {c.appSecret},
	})
	if err != nil {
		return "", &Error{Kind: ErrorKindTemporary}
	}

	var payload accessTokenResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return "", err
	}
	if payload.ErrCode != 0 {
		return "", &Error{Kind: classifyWechatError(payload.ErrCode)}
	}
	if strings.TrimSpace(payload.AccessToken) == "" || payload.ExpiresIn <= 0 {
		return "", &Error{Kind: ErrorKindInvalidResponse}
	}

	c.accessToken = payload.AccessToken
	c.accessTokenExpiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func (c *HTTPClient) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return &Error{Kind: ErrorKindTemporary}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{Kind: ErrorKindTemporary}
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &Error{Kind: ErrorKindTemporary}
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return &Error{Kind: ErrorKindInvalidResponse}
	}
	return nil
}

func (c *HTTPClient) isConfigured() bool {
	return strings.TrimSpace(c.appID) != "" && strings.TrimSpace(c.appSecret) != ""
}

func withQuery(endpoint string, values url.Values) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, value := range values {
		query[key] = value
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func classifyWechatError(code int) ErrorKind {
	switch code {
	case 40029, 40163, 41008:
		return ErrorKindRejected
	case 40013, 40125:
		return ErrorKindUnavailable
	default:
		return ErrorKindTemporary
	}
}
