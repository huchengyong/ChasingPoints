package sms

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCodeExpired = errors.New("验证码已过期或不存在")

// ErrSendLimited 表示手机号或来源 IP 的发送次数已达上限。
var ErrSendLimited = errors.New("验证码发送过于频繁，请稍后再试")

const (
	// 验证码长度
	CodeLength = 6
	// 验证码有效期（5分钟）
	CodeExpiration = 5 * time.Minute
	// 验证码发送间隔（60秒）
	SendInterval = 60 * time.Second
	// 同一个验证码最多允许输错的次数，达到后验证码作废，需重新获取
	MaxVerifyAttempts = 5
	// 同一手机号 24 小时内最多发送的条数
	PhoneDailySendLimit = 10
	// 同一来源 IP 每小时最多发送的条数；球房、运营商出口可能多人共用，不宜过低
	IPHourlySendLimit = 30
	// Redis key前缀
	CodeKeyPrefix           = "sms:code:"
	SendTimeKeyPrefix       = "sms:send_time:"
	verifyAttemptsKeyPrefix = "sms:verify_attempts:"
	phoneSendCountKeyPrefix = "sms:send_count:phone:"
	ipSendCountKeyPrefix    = "sms:send_count:ip:"

	SceneLogin         = "login"
	SceneBind          = "bind"
	SceneDeleteAccount = "delete_account"
)

type CodeManager struct {
	redis *redis.Client
}

func NewCodeManager(redisClient *redis.Client) *CodeManager {
	return &CodeManager{
		redis: redisClient,
	}
}

// GenerateCode 生成6位数字验证码
func (m *CodeManager) GenerateCode() string {
	// 使用crypto/rand生成安全的随机数
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// 如果生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// SaveCode 保存验证码到Redis
func (m *CodeManager) SaveCode(ctx context.Context, phone, code string) error {
	return m.SaveCodeForScene(ctx, phone, SceneLogin, code)
}

func (m *CodeManager) SaveCodeForScene(ctx context.Context, phone, scene, code string) error {
	key, err := codeKey(phone, scene)
	if err != nil {
		return err
	}
	// 新验证码的错误次数从零开始计
	pipe := m.redis.TxPipeline()
	pipe.Set(ctx, key, code, CodeExpiration)
	pipe.Del(ctx, verifyAttemptsKey(key))
	_, err = pipe.Exec(ctx)
	return err
}

// VerifyCode 验证验证码
func (m *CodeManager) VerifyCode(ctx context.Context, phone, code string) (bool, error) {
	return m.VerifyCodeForScene(ctx, phone, SceneLogin, code)
}

func (m *CodeManager) VerifyCodeForScene(ctx context.Context, phone, scene, code string) (bool, error) {
	key, err := codeKey(phone, scene)
	if err != nil {
		return false, err
	}
	// 输错累计到 MaxVerifyAttempts 次后验证码作废，防止在有效期内枚举 6 位数字。
	result, err := redis.NewScript(`
local saved = redis.call('GET', KEYS[1])
if not saved then return 0 end
if saved ~= ARGV[1] then
  local attempts = redis.call('INCR', KEYS[2])
  if attempts >= tonumber(ARGV[2]) then
    redis.call('DEL', KEYS[1], KEYS[2])
  else
    redis.call('EXPIRE', KEYS[2], ARGV[3])
  end
  return -1
end
redis.call('DEL', KEYS[1], KEYS[2])
return 1
`).Run(ctx, m.redis, []string{key, verifyAttemptsKey(key)}, code, MaxVerifyAttempts, int(CodeExpiration.Seconds())).Int()
	if err != nil {
		return false, err
	}
	switch result {
	case 1:
		return true, nil
	case 0:
		return false, ErrCodeExpired
	default:
		return false, nil
	}
}

// CheckSendInterval 检查发送间隔
func (m *CodeManager) CheckSendInterval(ctx context.Context, phone string) error {
	key := SendTimeKeyPrefix + phone
	lastSendTime, err := m.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// 第一次发送
		return nil
	}
	if err != nil {
		return err
	}

	lastTime, err := time.Parse(time.RFC3339, lastSendTime)
	if err != nil {
		return err
	}

	if time.Since(lastTime) < SendInterval {
		remaining := SendInterval - time.Since(lastTime)
		return fmt.Errorf("请等待%d秒后再试", int(remaining.Seconds()))
	}

	return nil
}

// ReserveSendQuota 在发送前占用一次手机号与来源 IP 的发送额度，任一超限返回 ErrSendLimited。
// 额度在发送前扣减：发送失败也计数，避免失败重试绕过上限。ip 为空时只按手机号限制。
func (m *CodeManager) ReserveSendQuota(ctx context.Context, phone, ip string) error {
	if err := m.reserveQuota(ctx, phoneSendCountKeyPrefix+phone, PhoneDailySendLimit, 24*time.Hour); err != nil {
		return err
	}
	if ip = strings.TrimSpace(ip); ip == "" {
		return nil
	}
	return m.reserveQuota(ctx, ipSendCountKeyPrefix+ip, IPHourlySendLimit, time.Hour)
}

func (m *CodeManager) reserveQuota(ctx context.Context, key string, limit int, window time.Duration) error {
	// 固定窗口计数：首次计数时设置过期时间，脚本内完成，不会留下没有过期时间的计数键。
	count, err := redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 or redis.call('TTL', KEYS[1]) < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`).Run(ctx, m.redis, []string{key}, int(window.Seconds())).Int()
	if err != nil {
		return err
	}
	if count > limit {
		return ErrSendLimited
	}
	return nil
}

// RecordSendTime 记录发送时间
func (m *CodeManager) RecordSendTime(ctx context.Context, phone string) error {
	key := SendTimeKeyPrefix + phone
	now := time.Now().Format(time.RFC3339)
	return m.redis.Set(ctx, key, now, SendInterval).Err()
}

func NormalizeScene(scene string) (string, error) {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return SceneLogin, nil
	}
	switch scene {
	case SceneLogin, SceneBind, SceneDeleteAccount:
		return scene, nil
	default:
		return "", fmt.Errorf("不支持的验证码场景")
	}
}

func verifyAttemptsKey(codeKey string) string {
	return verifyAttemptsKeyPrefix + strings.TrimPrefix(codeKey, CodeKeyPrefix)
}

func codeKey(phone, scene string) (string, error) {
	normalizedScene, err := NormalizeScene(scene)
	if err != nil {
		return "", err
	}
	return CodeKeyPrefix + normalizedScene + ":" + phone, nil
}
