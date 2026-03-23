package push

import (
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Config 推送服务配置
type Config struct {
	AppId        string
	AppKey       string
	MasterSecret string
	Enabled      bool
}

// PushService 推送服务
type PushService struct {
	config Config
	client *http.Client
}

// NewPushService 创建推送服务
func NewPushService(cfg Config) *PushService {
	return &PushService{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendPush 发送推送通知（best-effort，失败仅记录日志）
func (s *PushService) SendPush(pushClientId, title, content string, data map[string]interface{}) {
	if !s.config.Enabled || pushClientId == "" {
		return
	}

	logx.Infof("[UniPush] Sending push to cid=%s title=%s content=%s",
		pushClientId, title, content)

	// TODO: 接入 UniPush REST API
	// POST https://restapi.getui.com/v2/{appId}/push/single/cid
	// 当前仅记录日志，实际对接时替换为HTTP调用
	_ = fmt.Sprintf("push to %s: %s - %s", pushClientId, title, content)
}
