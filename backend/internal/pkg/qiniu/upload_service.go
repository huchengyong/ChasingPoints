package qiniu

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	qbox "github.com/qiniu/go-sdk/v7/auth/qbox"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"
)

const (
	defaultUploadExpiry = 10 * time.Minute
	defaultUploadURL    = "https://up-z2.qiniup.com"
	defaultAvatarExt    = ".jpg"
)

type Config struct {
	AccessKey    string
	SecretKey    string
	Bucket       string
	UploadURL    string
	PublicDomain string
}

type UploadCredential struct {
	UploadToken string
	Key         string
	UploadURL   string
	Domain      string
}

type UploadService struct {
	accessKey    string
	secretKey    string
	bucket       string
	uploadURL    string
	publicDomain string
	imageMirror  *imageMirrorClient
}

func NewUploadService(cfg Config) *UploadService {
	service := &UploadService{
		accessKey:    strings.TrimSpace(cfg.AccessKey),
		secretKey:    strings.TrimSpace(cfg.SecretKey),
		bucket:       strings.TrimSpace(cfg.Bucket),
		uploadURL:    normalizeUploadURL(cfg.UploadURL),
		publicDomain: strings.TrimSpace(cfg.PublicDomain),
	}
	service.imageMirror = newImageMirrorClient(service)
	return service
}

func (s *UploadService) Enabled() bool {
	return s != nil && s.accessKey != "" && s.secretKey != "" && s.bucket != "" && s.publicDomain != ""
}

func (s *UploadService) IssueAvatarUpload(userID int64, fileExt string, now time.Time) (*UploadCredential, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("qiniu upload service is not configured")
	}

	key := BuildAvatarObjectKey(userID, fileExt, now)
	policy := qiniustorage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s", s.bucket, key),
		Expires: uint64(defaultUploadExpiry.Seconds()),
	}

	token := policy.UploadToken(qbox.NewMac(s.accessKey, s.secretKey))
	return &UploadCredential{
		UploadToken: token,
		Key:         key,
		UploadURL:   s.uploadURL,
		Domain:      s.publicDomain,
	}, nil
}

func BuildAvatarObjectKey(userID int64, fileExt string, now time.Time) string {
	ext := normalizeFileExt(fileExt)
	return fmt.Sprintf("avatars/%d/%s/%d%s", userID, now.Format("20060102"), now.UnixNano(), ext)
}

func JoinPublicURL(domain, key string) string {
	trimmedDomain := strings.TrimRight(strings.TrimSpace(domain), "/")
	trimmedKey := strings.TrimLeft(strings.TrimSpace(key), "/")
	if trimmedDomain == "" || trimmedKey == "" {
		return ""
	}
	return trimmedDomain + "/" + trimmedKey
}

func normalizeUploadURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultUploadURL
	}
	return value
}

func normalizeFileExt(raw string) string {
	ext := strings.ToLower(strings.TrimSpace(raw))
	if ext == "" {
		return defaultAvatarExt
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == "." {
		return defaultAvatarExt
	}
	cleanExt := filepath.Ext("x" + ext)
	if cleanExt == "" || cleanExt == "." {
		return defaultAvatarExt
	}
	return cleanExt
}
