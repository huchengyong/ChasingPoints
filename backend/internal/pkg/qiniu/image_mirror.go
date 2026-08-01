package qiniu

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	qbox "github.com/qiniu/go-sdk/v7/auth/qbox"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"
)

const (
	WSTImageCategoryPlayer     = "players"
	WSTImageCategoryTournament = "tournaments"

	wstImageHost         = "images.gc.wstservices.co.uk"
	mirrorDownloadLimit  = 8 * 1024 * 1024
	mirrorRequestTimeout = 15 * time.Second
)

type imageMirrorClient struct {
	httpClient   *http.Client
	objectExists func(key string) (bool, error)
	uploadObject func(ctx context.Context, key, contentType string, data []byte) error
}

func newImageMirrorClient(service *UploadService) *imageMirrorClient {
	client := &http.Client{
		Timeout: mirrorRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many image redirects")
			}
			_, _, err := normalizeWSTImageURL(req.URL.String())
			return err
		},
	}
	return &imageMirrorClient{
		httpClient:   client,
		objectExists: service.qiniuObjectExists,
		uploadObject: service.qiniuUploadObject,
	}
}

func BuildWSTImageObjectKey(category, sourceURL string) (string, error) {
	if category != WSTImageCategoryPlayer && category != WSTImageCategoryTournament {
		return "", fmt.Errorf("unsupported WST image category: %s", category)
	}

	normalizedURL, ext, err := normalizeWSTImageURL(sourceURL)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(normalizedURL))
	return fmt.Sprintf("wst/%s/%x%s", category, digest, ext), nil
}

func (s *UploadService) MirrorWSTImage(ctx context.Context, category, sourceURL string) (string, error) {
	if !s.Enabled() || s.imageMirror == nil {
		return "", fmt.Errorf("qiniu upload service is not configured")
	}

	key, err := BuildWSTImageObjectKey(category, sourceURL)
	if err != nil {
		return "", err
	}

	exists, err := s.imageMirror.objectExists(key)
	if err != nil {
		return "", fmt.Errorf("check mirrored image: %w", err)
	}
	if exists {
		return JoinPublicURL(s.publicDomain, key), nil
	}

	data, contentType, err := downloadWSTImage(ctx, s.imageMirror.httpClient, sourceURL)
	if err != nil {
		return "", err
	}
	if err := s.imageMirror.uploadObject(ctx, key, contentType, data); err != nil {
		return "", fmt.Errorf("upload mirrored image: %w", err)
	}
	return JoinPublicURL(s.publicDomain, key), nil
}

func (s *UploadService) IsManagedPublicURL(rawURL string) bool {
	base := strings.TrimRight(strings.TrimSpace(s.publicDomain), "/")
	value := strings.TrimSpace(rawURL)
	return base != "" && strings.HasPrefix(value, base+"/")
}

func normalizeWSTImageURL(rawURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", fmt.Errorf("parse WST image URL: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", "", fmt.Errorf("WST image URL must use HTTPS")
	}
	if parsed.User != nil || !strings.EqualFold(parsed.Hostname(), wstImageHost) {
		return "", "", fmt.Errorf("unsupported WST image host")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return "", "", fmt.Errorf("unsupported WST image port")
	}

	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".jpeg":
		ext = ".jpg"
	case ".png", ".jpg", ".webp":
	default:
		return "", "", fmt.Errorf("unsupported WST image extension")
	}

	parsed.Scheme = "https"
	parsed.Host = wstImageHost
	parsed.Fragment = ""
	parsed.RawFragment = ""
	parsed.RawQuery = parsed.Query().Encode()
	return parsed.String(), ext, nil
}

func downloadWSTImage(ctx context.Context, client *http.Client, sourceURL string) ([]byte, string, error) {
	normalizedURL, ext, err := normalizeWSTImageURL(sourceURL)
	if err != nil {
		return nil, "", err
	}
	if client == nil {
		return nil, "", fmt.Errorf("image HTTP client is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizedURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create image request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download WST image: %w", err)
	}
	defer resp.Body.Close()

	if resp.Request != nil {
		if _, _, err := normalizeWSTImageURL(resp.Request.URL.String()); err != nil {
			return nil, "", fmt.Errorf("invalid final image URL: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download WST image: unexpected status %d", resp.StatusCode)
	}
	if resp.ContentLength > mirrorDownloadLimit {
		return nil, "", fmt.Errorf("WST image exceeds size limit")
	}

	declaredContentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", fmt.Errorf("unsupported WST image content type")
	}
	declaredContentType = normalizeImageContentType(declaredContentType)
	if !isAllowedImageContentType(declaredContentType) {
		return nil, "", fmt.Errorf("unsupported WST image content type")
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, mirrorDownloadLimit+1))
	if err != nil {
		return nil, "", fmt.Errorf("read WST image: %w", err)
	}
	if len(data) == 0 || len(data) > mirrorDownloadLimit {
		return nil, "", fmt.Errorf("WST image has invalid size")
	}

	contentType := detectImageContentType(data)
	if contentType == "" {
		return nil, "", fmt.Errorf("WST image content is invalid")
	}
	if !extensionMatchesContentType(ext, contentType) {
		return nil, "", fmt.Errorf("WST image extension does not match content type")
	}
	return data, contentType, nil
}

func normalizeImageContentType(contentType string) string {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	if normalized == "image/jpg" {
		return "image/jpeg"
	}
	return normalized
}

func isAllowedImageContentType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

func extensionMatchesContentType(ext, contentType string) bool {
	switch ext {
	case ".png":
		return contentType == "image/png"
	case ".jpg":
		return contentType == "image/jpeg"
	case ".webp":
		return contentType == "image/webp"
	default:
		return false
	}
}

func detectImageContentType(data []byte) string {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return "image/png"
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	default:
		return ""
	}
}

func (s *UploadService) qiniuObjectExists(key string) (bool, error) {
	mac := qbox.NewMac(s.accessKey, s.secretKey)
	manager := qiniustorage.NewBucketManager(mac, &qiniustorage.Config{UseHTTPS: true})
	_, err := manager.Stat(s.bucket, key)
	if err == nil {
		return true, nil
	}
	if isQiniuErrorCode(err, 612) {
		return false, nil
	}
	return false, err
}

func (s *UploadService) qiniuUploadObject(ctx context.Context, key, contentType string, data []byte) error {
	policy := qiniustorage.PutPolicy{
		Scope:      fmt.Sprintf("%s:%s", s.bucket, key),
		Expires:    uint64(defaultUploadExpiry.Seconds()),
		InsertOnly: 1,
	}
	token := policy.UploadToken(qbox.NewMac(s.accessKey, s.secretKey))
	uploader := qiniustorage.NewFormUploader(&qiniustorage.Config{
		UseHTTPS: true,
		UpHost:   s.uploadURL,
	})
	var result qiniustorage.PutRet
	err := uploader.Put(ctx, &result, token, key, bytes.NewReader(data), int64(len(data)), &qiniustorage.PutExtra{
		MimeType: contentType,
		UpHost:   s.uploadURL,
	})
	if isQiniuErrorCode(err, 614) {
		return nil
	}
	return err
}

func isQiniuErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var info *qiniustorage.ErrorInfo
	return errors.As(err, &info) && info.Code == code
}
