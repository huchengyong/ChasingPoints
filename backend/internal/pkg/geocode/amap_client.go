package geocode

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
)

// https://lbs.amap.com/api/webservice/guide/api/georegeo
const defaultAmapEndpoint = "https://restapi.amap.com/v3/geocode/geo"

type AmapClient struct {
	baseURL    string
	key        string
	signingKey string
	httpClient *http.Client
}

type amapResponse struct {
	Status   string        `json:"status"`
	Info     string        `json:"info"`
	Infocode string        `json:"infocode"`
	Geocodes []amapGeocode `json:"geocodes"`
}

type amapGeocode struct {
	Location amapText `json:"location"`
	Level    amapText `json:"level"`
}

// amapText 高德对空字段返回 [] 而不是字符串，这里统一解成空字符串。
type amapText string

func (t *amapText) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		*t = ""
		return nil
	}
	*t = amapText(value)
	return nil
}

func NewAmapClient(baseURL, key, signingKey string, timeout time.Duration) *AmapClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultAmapEndpoint
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &AmapClient{
		baseURL:    baseURL,
		key:        key,
		signingKey: signingKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *AmapClient) Geocode(ctx context.Context, _ model.GeocodeAccount, address string) (*Result, error) {
	// 高德凭据来自环境变量；账号行只承担按分钟限流、冷却和鉴权失败停用。
	params := map[string]string{
		"key":     c.key,
		"address": address,
		"output":  "JSON",
	}
	if c.signingKey != "" {
		params["sig"] = amapSignature(params, c.signingKey)
	}

	queryURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, NewProviderError(ErrorKindTemporary, requestErrorMessage(err))
	}

	query := queryURL.Query()
	for name, value := range params {
		query.Set(name, value)
	}
	queryURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, NewProviderError(ErrorKindTemporary, requestErrorMessage(err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewProviderError(ErrorKindTemporary, requestErrorMessage(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError(ErrorKindTemporary, fmt.Sprintf("amap http status %d", resp.StatusCode))
	}

	var payload amapResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, err.Error())
	}

	if payload.Status != "1" {
		return nil, NewProviderError(classifyAmapError(payload.Infocode), fmt.Sprintf("amap %s %s", payload.Infocode, payload.Info))
	}
	if len(payload.Geocodes) == 0 {
		return nil, NewProviderError(ErrorKindInvalidResponse, "amap no geocode result")
	}

	geocode := payload.Geocodes[0]
	parts := strings.Split(string(geocode.Location), ",")
	if len(parts) != 2 {
		return nil, NewProviderError(ErrorKindInvalidResponse, "invalid amap location")
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, "invalid amap longitude")
	}
	latitude, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nil, NewProviderError(ErrorKindInvalidResponse, "invalid amap latitude")
	}
	if longitude == 0 || latitude == 0 {
		return nil, NewProviderError(ErrorKindInvalidResponse, "empty amap coordinates")
	}

	level := strings.TrimSpace(string(geocode.Level))
	rawCode, err := strconv.Atoi(payload.Infocode)
	if err != nil {
		rawCode = 0
	}

	return &Result{
		Longitude: longitude,
		Latitude:  latitude,
		Level:     level,
		RawCode:   rawCode,
		Source:    "amap",
		Precise:   isPreciseAmapLevel(level),
	}, nil
}

// amapSignature 按高德数字签名规则：除 sig 外的参数按参数名升序，用未编码的原始值拼成 name=value，
// 末尾直接拼接私钥后对整个字符串取 MD5，输出小写十六进制。
func amapSignature(params map[string]string, secret string) string {
	names := make([]string, 0, len(params))
	for name := range params {
		if name == "sig" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+params[name])
	}

	sum := md5.Sum([]byte(strings.Join(parts, "&") + secret))
	return hex.EncodeToString(sum[:])
}

// classifyAmapError 把高德错误码映射到 worker 的处理方式：鉴权失败停用账号，参数问题不重试，
// 其余（配额、频率、服务端繁忙）冷却后重试。
func classifyAmapError(infocode string) string {
	switch infocode {
	case "10001", "10002", "10005", "10006", "10007", "10008", "10009", "10012", "10013", "10026", "10041":
		return ErrorKindAuth
	case "20000", "20001", "20002", "20011", "20012":
		return ErrorKindInvalidResponse
	default:
		return ErrorKindTemporary
	}
}

func isPreciseAmapLevel(level string) bool {
	switch strings.TrimSpace(level) {
	case "门牌号", "门址", "兴趣点", "单元号", "楼层", "房间":
		return true
	default:
		return false
	}
}
