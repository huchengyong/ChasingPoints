package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	// 应用环境：local, dev, prod
	AppEnv string `json:",env=APP_ENV,default=local"`

	// JWT配置
	Auth struct {
		AccessSecret  string
		AccessExpire  int64
		RefreshExpire int64 `json:",default=2592000"`
	}

	// 管理后台配置
	Admin struct {
		SetupToken string `json:",env=ADMIN_SETUP_TOKEN,optional"`
	}

	// 合规收口配置
	Compliance ComplianceConfig

	// MySQL配置
	MySQL struct {
		DataSource string
	}

	// Redis配置
	Redis struct {
		Host     string
		Password string
		DB       int
	}

	// 阿里云短信配置
	AliSms struct {
		AccessKeyId     string
		AccessKeySecret string
		SignName        string
		TemplateCode    string
		Endpoint        string
	}

	// 推送配置
	UniPush struct {
		AppId        string `json:",env=UNIPUSH_APP_ID,default="`
		AppKey       string `json:",env=UNIPUSH_APP_KEY,default="`
		MasterSecret string `json:",env=UNIPUSH_MASTER_SECRET,default="`
		Enabled      bool   `json:",env=UNIPUSH_ENABLED,default=false"`
	}

	// 七牛上传配置
	Qiniu struct {
		AccessKey    string `json:",env=QINIU_ACCESS_KEY,optional"`
		SecretKey    string `json:",env=QINIU_SECRET_KEY,optional"`
		Bucket       string `json:",env=QINIU_BUCKET,optional"`
		UploadUrl    string `json:",env=QINIU_UPLOAD_URL,default=https://up-z2.qiniup.com"`
		PublicDomain string `json:",env=QINIU_PUBLIC_DOMAIN,optional"`
	}

	// 球馆地理解析配置
	Geocode struct {
		WorkerEnabled    bool   `json:",env=GEOCODE_WORKER_ENABLED,default=true"`
		PollIntervalMs   int    `json:",env=GEOCODE_POLL_INTERVAL_MS,default=3000"`
		RequestTimeoutMs int    `json:",env=GEOCODE_REQUEST_TIMEOUT_MS,default=5000"`
		MaxAttempts      int    `json:",env=GEOCODE_MAX_ATTEMPTS,default=5"`
		BatchSize        int    `json:",env=GEOCODE_BATCH_SIZE,default=5"`
		AutoPublish      bool   `json:",env=GEOCODE_AUTO_PUBLISH,default=true"`
		EncryptSecret    string `json:",env=GEOCODE_ENCRYPT_SECRET,optional"`
	}

	WSTSync WSTSyncConfig

	// 支付宝支付配置
	Alipay struct {
		AppId      int64
		PrivateKey string
		PublicKey  string
		PublicCert string
		AppCert    string
		RootCert   string
		NotifyUrl  string
		ReturnUrl  string
		IsProd     bool
		Charset    string
		SignType   string
		UseCert    bool
	}

	// 微信支付配置
	WechatPay struct {
		AppId      int64
		MchId      int64
		ApiKey     string
		SerialNo   string
		ApiV3Key   string
		PrivateKey string
		NotifyUrl  string
		IsProd     bool
		UseCert    bool
	}

	// 微信小程序登录配置
	WechatMiniProgram struct {
		AppId            string
		AppSecret        string
		RequestTimeoutMs int `json:",default=5000"`
	}
}

type WSTSyncConfig struct {
	Enabled            bool `json:",env=WST_SYNC_ENABLED,default=true"`
	IntervalMinutes    int  `json:",env=WST_SYNC_INTERVAL_MINUTES,default=720"`
	LookbackDays       int  `json:",env=WST_SYNC_LOOKBACK_DAYS,default=30"`
	LookaheadDays      int  `json:",env=WST_SYNC_LOOKAHEAD_DAYS,default=7"`
	Publish            bool `json:",env=WST_SYNC_PUBLISH,default=true"`
	HotEnabled         bool `json:",env=WST_SYNC_HOT_ENABLED,default=true"`
	HotIntervalMinutes int  `json:",env=WST_SYNC_HOT_INTERVAL_MINUTES,default=15"`
	HotLookbackDays    int  `json:",env=WST_SYNC_HOT_LOOKBACK_DAYS,default=2"`
	HotLookaheadDays   int  `json:",env=WST_SYNC_HOT_LOOKAHEAD_DAYS,default=7"`
}
