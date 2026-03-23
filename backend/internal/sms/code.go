package sms

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 验证码长度
	CodeLength = 6
	// 验证码有效期（5分钟）
	CodeExpiration = 5 * time.Minute
	// 验证码发送间隔（60秒）
	SendInterval = 60 * time.Second
	// Redis key前缀
	CodeKeyPrefix     = "sms:code:"
	SendTimeKeyPrefix = "sms:send_time:"
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
	key := CodeKeyPrefix + phone
	return m.redis.Set(ctx, key, code, CodeExpiration).Err()
}

// VerifyCode 验证验证码
func (m *CodeManager) VerifyCode(ctx context.Context, phone, code string) (bool, error) {
	key := CodeKeyPrefix + phone
	savedCode, err := m.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, fmt.Errorf("验证码已过期或不存在")
	}
	if err != nil {
		return false, err
	}

	if savedCode != code {
		return false, nil
	}

	// 验证成功后删除验证码
	m.redis.Del(ctx, key)
	return true, nil
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

// RecordSendTime 记录发送时间
func (m *CodeManager) RecordSendTime(ctx context.Context, phone string) error {
	key := SendTimeKeyPrefix + phone
	now := time.Now().Format(time.RFC3339)
	return m.redis.Set(ctx, key, now, SendInterval).Err()
}
