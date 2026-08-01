package qiniu

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	qiniustorage "github.com/qiniu/go-sdk/v7/storage"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestUploadService() *UploadService {
	return NewUploadService(Config{
		AccessKey:    "access-key",
		SecretKey:    "secret-key",
		Bucket:       "bucket",
		UploadURL:    "https://up-z2.qiniup.com",
		PublicDomain: "https://cdn.example.com/assets/",
	})
}

func imageResponse(req *http.Request, status int, contentType string, data []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Header:        http.Header{"Content-Type": []string{contentType}},
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request:       req,
	}
}

func TestBuildWSTImageObjectKeyIsDeterministic(t *testing.T) {
	left, err := BuildWSTImageObjectKey(
		WSTImageCategoryPlayer,
		"https://IMAGES.GC.WSTSERVICES.CO.UK:443/fit-in/400x600/player.jpeg?b=2&a=1#ignored",
	)
	if err != nil {
		t.Fatalf("build left key: %v", err)
	}
	right, err := BuildWSTImageObjectKey(
		WSTImageCategoryPlayer,
		"https://images.gc.wstservices.co.uk/fit-in/400x600/player.jpeg?a=1&b=2",
	)
	if err != nil {
		t.Fatalf("build right key: %v", err)
	}
	if left != right {
		t.Fatalf("keys differ: %q != %q", left, right)
	}
	if !strings.HasPrefix(left, "wst/players/") || !strings.HasSuffix(left, ".jpg") {
		t.Fatalf("unexpected key: %q", left)
	}

	tournament, err := BuildWSTImageObjectKey(
		WSTImageCategoryTournament,
		"https://images.gc.wstservices.co.uk/fit-in/1000x1000/cover.png",
	)
	if err != nil {
		t.Fatalf("build tournament key: %v", err)
	}
	if !strings.HasPrefix(tournament, "wst/tournaments/") || !strings.HasSuffix(tournament, ".png") {
		t.Fatalf("unexpected tournament key: %q", tournament)
	}
	if tournament == left {
		t.Fatal("different categories must not share a key")
	}
}

func TestBuildWSTImageObjectKeyRejectsUnsupportedSources(t *testing.T) {
	tests := []struct {
		name     string
		category string
		url      string
	}{
		{name: "category", category: "other", url: "https://images.gc.wstservices.co.uk/a.png"},
		{name: "http", category: WSTImageCategoryPlayer, url: "http://images.gc.wstservices.co.uk/a.png"},
		{name: "host", category: WSTImageCategoryPlayer, url: "https://example.com/a.png"},
		{name: "host suffix", category: WSTImageCategoryPlayer, url: "https://images.gc.wstservices.co.uk.evil.example/a.png"},
		{name: "userinfo", category: WSTImageCategoryPlayer, url: "https://user@images.gc.wstservices.co.uk/a.png"},
		{name: "port", category: WSTImageCategoryPlayer, url: "https://images.gc.wstservices.co.uk:8443/a.png"},
		{name: "extension", category: WSTImageCategoryPlayer, url: "https://images.gc.wstservices.co.uk/a.svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildWSTImageObjectKey(tt.category, tt.url); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestUploadServiceRecognizesManagedPublicURL(t *testing.T) {
	service := newTestUploadService()
	if !service.IsManagedPublicURL("https://cdn.example.com/assets/wst/players/a.png") {
		t.Fatal("expected managed URL")
	}
	for _, value := range []string{
		"https://cdn.example.com/other/a.png",
		"https://cdn.example.com.evil/assets/wst/players/a.png",
		"https://images.gc.wstservices.co.uk/a.png",
		"",
	} {
		if service.IsManagedPublicURL(value) {
			t.Fatalf("unexpected managed URL: %q", value)
		}
	}
}

func TestDownloadWSTImageAcceptsAllowedTypesWithoutReferer(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		contentType string
		data        []byte
	}{
		{name: "png", url: "https://images.gc.wstservices.co.uk/a.png", contentType: "image/png", data: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0}},
		{name: "jpeg", url: "https://images.gc.wstservices.co.uk/a.jpg", contentType: "image/jpeg", data: []byte{0xff, 0xd8, 0xff, 0xdb}},
		{name: "webp", url: "https://images.gc.wstservices.co.uk/a.webp", contentType: "image/webp", data: []byte("RIFF1234WEBP")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Referer") != "" {
					t.Fatalf("unexpected referer: %q", req.Header.Get("Referer"))
				}
				return imageResponse(req, http.StatusOK, tt.contentType+"; charset=binary", tt.data), nil
			})}

			data, contentType, err := downloadWSTImage(context.Background(), client, tt.url)
			if err != nil {
				t.Fatalf("download image: %v", err)
			}
			if contentType != tt.contentType || !bytes.Equal(data, tt.data) {
				t.Fatalf("unexpected result: type=%q data=%v", contentType, data)
			}
		})
	}
}

func TestDownloadWSTImageRejectsInvalidResponses(t *testing.T) {
	validPNG := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0}
	tests := []struct {
		name        string
		status      int
		contentType string
		data        []byte
		length      int64
	}{
		{name: "status", status: http.StatusForbidden, contentType: "image/png", data: validPNG},
		{name: "content type", status: http.StatusOK, contentType: "text/html", data: []byte("blocked")},
		{name: "extension mismatch", status: http.StatusOK, contentType: "image/jpeg", data: []byte{0xff, 0xd8, 0xff}},
		{name: "signature", status: http.StatusOK, contentType: "image/png", data: []byte("not-png")},
		{name: "empty", status: http.StatusOK, contentType: "image/png", data: nil},
		{name: "content length", status: http.StatusOK, contentType: "image/png", data: validPNG, length: mirrorDownloadLimit + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				resp := imageResponse(req, tt.status, tt.contentType, tt.data)
				if tt.length > 0 {
					resp.ContentLength = tt.length
				}
				return resp, nil
			})}
			if _, _, err := downloadWSTImage(context.Background(), client, "https://images.gc.wstservices.co.uk/a.png"); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestDownloadWSTImageRejectsRedirectOutsideAllowedHost(t *testing.T) {
	service := newTestUploadService()
	client := service.imageMirror.httpClient
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://example.com/image.png"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})

	if _, _, err := downloadWSTImage(context.Background(), client, "https://images.gc.wstservices.co.uk/a.png"); err == nil {
		t.Fatal("expected redirect rejection")
	}
}

func TestDownloadWSTImageRejectsBodyOverLimit(t *testing.T) {
	data := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, mirrorDownloadLimit)...)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := imageResponse(req, http.StatusOK, "image/png", data)
		resp.ContentLength = -1
		return resp, nil
	})}
	if _, _, err := downloadWSTImage(context.Background(), client, "https://images.gc.wstservices.co.uk/a.png"); err == nil {
		t.Fatal("expected size rejection")
	}
}

func TestMirrorWSTImageUploadsOnlyWhenObjectIsMissing(t *testing.T) {
	service := newTestUploadService()
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0}
	var checkedKey string
	var uploadedKey string
	var uploadedType string
	var uploadedData []byte
	service.imageMirror.objectExists = func(key string) (bool, error) {
		checkedKey = key
		return false, nil
	}
	service.imageMirror.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return imageResponse(req, http.StatusOK, "image/png", png), nil
	})
	service.imageMirror.uploadObject = func(ctx context.Context, key, contentType string, data []byte) error {
		uploadedKey = key
		uploadedType = contentType
		uploadedData = append([]byte(nil), data...)
		return nil
	}

	publicURL, err := service.MirrorWSTImage(
		context.Background(),
		WSTImageCategoryPlayer,
		"https://images.gc.wstservices.co.uk/player.png",
	)
	if err != nil {
		t.Fatalf("mirror image: %v", err)
	}
	if checkedKey == "" || checkedKey != uploadedKey {
		t.Fatalf("unexpected keys: checked=%q uploaded=%q", checkedKey, uploadedKey)
	}
	if uploadedType != "image/png" || !bytes.Equal(uploadedData, png) {
		t.Fatalf("unexpected upload: type=%q data=%v", uploadedType, uploadedData)
	}
	if publicURL != "https://cdn.example.com/assets/"+checkedKey {
		t.Fatalf("unexpected public URL: %q", publicURL)
	}
}

func TestMirrorWSTImageReusesExistingObject(t *testing.T) {
	service := newTestUploadService()
	var key string
	service.imageMirror.objectExists = func(value string) (bool, error) {
		key = value
		return true, nil
	}
	service.imageMirror.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("existing object should not be downloaded")
		return nil, nil
	})
	service.imageMirror.uploadObject = func(ctx context.Context, key, contentType string, data []byte) error {
		t.Fatal("existing object should not be uploaded")
		return nil
	}

	publicURL, err := service.MirrorWSTImage(
		context.Background(),
		WSTImageCategoryTournament,
		"https://images.gc.wstservices.co.uk/cover.jpg",
	)
	if err != nil {
		t.Fatalf("mirror image: %v", err)
	}
	if publicURL != "https://cdn.example.com/assets/"+key {
		t.Fatalf("unexpected public URL: %q", publicURL)
	}
}

func TestMirrorWSTImageReturnsStorageErrors(t *testing.T) {
	service := newTestUploadService()
	service.imageMirror.objectExists = func(key string) (bool, error) {
		return false, errors.New("stat failed")
	}
	if _, err := service.MirrorWSTImage(
		context.Background(),
		WSTImageCategoryPlayer,
		"https://images.gc.wstservices.co.uk/player.png",
	); err == nil || !strings.Contains(err.Error(), "stat failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQiniuErrorCodes(t *testing.T) {
	if !isQiniuErrorCode(&qiniustorage.ErrorInfo{Code: 612}, 612) {
		t.Fatal("expected matching error code")
	}
	if isQiniuErrorCode(&qiniustorage.ErrorInfo{Code: 614}, 612) {
		t.Fatal("unexpected matching error code")
	}
}

func TestIssueAvatarUploadStillReturnsCredential(t *testing.T) {
	service := newTestUploadService()
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	credential, err := service.IssueAvatarUpload(42, "png", now)
	if err != nil {
		t.Fatalf("issue upload: %v", err)
	}
	if credential.UploadToken == "" || credential.Key != "avatars/42/20260731/1785499200000000000.png" {
		t.Fatalf("unexpected credential: %+v", credential)
	}
	if credential.UploadURL != "https://up-z2.qiniup.com" || credential.Domain != "https://cdn.example.com/assets/" {
		t.Fatalf("unexpected endpoints: %+v", credential)
	}
}
