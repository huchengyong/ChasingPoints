package geocode

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestAmapSignatureVector(t *testing.T) {
	params := map[string]string{
		"a": "23",
		"b": "12",
		"c": "67",
		"d": "48",
		"f": "8",
	}

	if got := amapSignature(params, "bbbbb"); got != "a89e8c2266d888860c46672d77d069f3" {
		t.Fatalf("expected a89e8c2266d888860c46672d77d069f3, got %s", got)
	}
}

func TestAmapSignatureUsesRawChineseAddress(t *testing.T) {
	params := map[string]string{
		"key":     "amap-key-value",
		"address": "上海市浦东新区世纪大道100号",
		"output":  "JSON",
	}

	signedString := "address=上海市浦东新区世纪大道100号&key=amap-key-value&output=JSONamap-secret-value"
	sum := md5.Sum([]byte(signedString))
	expected := hex.EncodeToString(sum[:])

	got := amapSignature(params, "amap-secret-value")
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(got) {
		t.Fatalf("expected 32 lowercase hex chars, got %s", got)
	}
}

func TestAmapClientGeocodeSuccess(t *testing.T) {
	client := NewAmapClient("https://example.com/geocode", "amap-key-value", "amap-secret-value", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		query := r.URL.Query()
		if got := query.Get("key"); got != "amap-key-value" {
			t.Fatalf("expected key=amap-key-value, got %s", got)
		}
		if got := query.Get("address"); got != "上海市浦东新区世纪大道100号" {
			t.Fatalf("unexpected address %s", got)
		}
		if got := query.Get("output"); got != "JSON" {
			t.Fatalf("expected output=JSON, got %s", got)
		}

		params := make(map[string]string, len(query))
		for name := range query {
			if name == "sig" {
				continue
			}
			params[name] = query.Get(name)
		}
		if got, expected := query.Get("sig"), amapSignature(params, "amap-secret-value"); got != expected {
			t.Fatalf("expected sig %s, got %s", expected, got)
		}

		return jsonResponse(`{"status":"1","info":"OK","infocode":"10000","count":"2","geocodes":[{"formatted_address":"上海市浦东新区世纪大道100号","country":"中国","street":[],"number":[],"location":"121.507572,31.234583","level":"门址"}]}`), nil
	})

	result, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Longitude != 121.507572 {
		t.Fatalf("expected longitude 121.507572, got %v", result.Longitude)
	}
	if result.Latitude != 31.234583 {
		t.Fatalf("expected latitude 31.234583, got %v", result.Latitude)
	}
	if result.Level != "门址" {
		t.Fatalf("expected level 门址, got %s", result.Level)
	}
	if !result.Precise {
		t.Fatal("expected 门址 to be precise")
	}
	if result.Source != "amap" {
		t.Fatalf("expected source amap, got %s", result.Source)
	}
}

func TestAmapClientOmitsSignatureWithoutSigningKey(t *testing.T) {
	client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if _, ok := r.URL.Query()["sig"]; ok {
			t.Fatalf("expected no sig without signing key, got %s", r.URL.Query().Get("sig"))
		}
		return jsonResponse(`{"status":"1","info":"OK","infocode":"10000","geocodes":[{"location":"121.507572,31.234583","level":"门址"}]}`), nil
	})

	if _, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestAmapClientLevelPrecision(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		precise bool
	}{
		{name: "区县", level: `"区县"`, precise: false},
		{name: "乡镇", level: `"乡镇"`, precise: false},
		{name: "未知", level: `"未知"`, precise: false},
		{name: "空数组", level: `[]`, precise: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
			client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(fmt.Sprintf(`{"status":"1","info":"OK","infocode":"10000","geocodes":[{"location":"121.507572,31.234583","level":%s}]}`, test.level)), nil
			})

			result, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if result.Precise != test.precise {
				t.Fatalf("expected precise %v, got %v", test.precise, result.Precise)
			}
			if test.level == `[]` && result.Level != "" {
				t.Fatalf("expected empty level for [], got %q", result.Level)
			}
		})
	}
}

func TestAmapClientEmptyGeocodes(t *testing.T) {
	client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"status":"1","info":"OK","infocode":"10000","geocodes":[]}`), nil
	})

	_, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
	assertProviderErrorKind(t, err, ErrorKindInvalidResponse)
}

func TestAmapClientErrorClassification(t *testing.T) {
	tests := []struct {
		infocode string
		kind     string
	}{
		{infocode: "10001", kind: ErrorKindAuth},
		{infocode: "10005", kind: ErrorKindAuth},
		{infocode: "10007", kind: ErrorKindAuth},
		{infocode: "10003", kind: ErrorKindTemporary},
		{infocode: "10004", kind: ErrorKindTemporary},
		{infocode: "10044", kind: ErrorKindTemporary},
		{infocode: "10016", kind: ErrorKindTemporary},
		{infocode: "20000", kind: ErrorKindInvalidResponse},
	}

	for _, test := range tests {
		t.Run(test.infocode, func(t *testing.T) {
			client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
			client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(fmt.Sprintf(`{"status":"0","info":"ERROR","infocode":%q}`, test.infocode)), nil
			})

			_, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
			assertProviderErrorKind(t, err, test.kind)
		})
	}
}

func TestAmapClientHTTPServerError(t *testing.T) {
	client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       http.NoBody,
		}, nil
	})

	_, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
	assertProviderErrorKind(t, err, ErrorKindTemporary)
}

func TestAmapClientTransportErrorOmitsCredentials(t *testing.T) {
	client := NewAmapClient("", "amap-key-value", "amap-secret-value", 2*time.Second)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: i/o timeout")
	})

	_, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
	assertProviderErrorKind(t, err, ErrorKindTemporary)

	message := err.Error()
	for _, forbidden := range []string{"amap-key-value", "amap-secret-value", "sig=", "restapi.amap.com"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("expected error without %q, got %q", forbidden, message)
		}
	}
	if !strings.Contains(message, "i/o timeout") {
		t.Fatalf("expected underlying cause in error, got %q", message)
	}
}

func TestAmapClientRejectsInvalidLocation(t *testing.T) {
	tests := []string{"abc", "0,0", "121.5", "121.5,abc"}

	for _, location := range tests {
		t.Run(location, func(t *testing.T) {
			client := NewAmapClient("https://example.com/geocode", "amap-key-value", "", 2*time.Second)
			client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(fmt.Sprintf(`{"status":"1","info":"OK","infocode":"10000","geocodes":[{"location":%q,"level":"门址"}]}`, location)), nil
			})

			_, err := client.Geocode(context.Background(), model.GeocodeAccount{}, "上海市浦东新区世纪大道100号")
			assertProviderErrorKind(t, err, ErrorKindInvalidResponse)
		})
	}
}

func assertProviderErrorKind(t *testing.T, err error, kind string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", kind)
	}
	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Kind != kind {
		t.Fatalf("expected %s kind, got %s", kind, providerErr.Kind)
	}
}
