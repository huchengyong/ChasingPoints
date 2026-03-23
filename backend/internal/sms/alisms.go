package sms

import (
	"encoding/json"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

type AliSmsConfig struct {
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string
}

type AliSmsClient struct {
	client *dysmsapi20170525.Client
	config *AliSmsConfig
}

// NewAliSmsClient 创建阿里云短信客户端
func NewAliSmsClient(config *AliSmsConfig) (*AliSmsClient, error) {
	// 使用 AccessKey 创建凭据
	credConfig := &credential.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(config.AccessKeyId),
		AccessKeySecret: tea.String(config.AccessKeySecret),
	}

	cred, err := credential.NewCredential(credConfig)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云凭据失败: %w", err)
	}

	// 创建客户端配置
	clientConfig := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String(config.Endpoint),
	}

	client, err := dysmsapi20170525.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云短信客户端失败: %w", err)
	}

	return &AliSmsClient{
		client: client,
		config: config,
	}, nil
}

// SendVerificationCode 发送验证码短信
func (c *AliSmsClient) SendVerificationCode(phone, code string) error {
	templateParam := fmt.Sprintf(`{"code":"%s"}`, code)

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		SignName:      tea.String(c.config.SignName),
		TemplateCode:  tea.String(c.config.TemplateCode),
		PhoneNumbers:  tea.String(phone),
		TemplateParam: tea.String(templateParam),
	}

	runtime := &util.RuntimeOptions{}

	resp, err := c.client.SendSmsWithOptions(sendSmsRequest, runtime)
	if err != nil {
		return c.handleError(err)
	}

	// 检查返回结果
	if resp.Body.Code != nil && *resp.Body.Code != "OK" {
		return fmt.Errorf("发送短信失败: %s - %s",
			tea.StringValue(resp.Body.Code),
			tea.StringValue(resp.Body.Message))
	}

	return nil
}

// handleError 处理阿里云SDK错误
func (c *AliSmsClient) handleError(err error) error {
	if sdkErr, ok := err.(*tea.SDKError); ok {
		var data interface{}
		d := json.NewDecoder(strings.NewReader(tea.StringValue(sdkErr.Data)))
		d.Decode(&data)

		errMsg := tea.StringValue(sdkErr.Message)
		if m, ok := data.(map[string]interface{}); ok {
			if recommend, exists := m["Recommend"]; exists {
				errMsg = fmt.Sprintf("%s (建议: %v)", errMsg, recommend)
			}
		}
		return fmt.Errorf("阿里云短信错误: %s", errMsg)
	}
	return fmt.Errorf("发送短信失败: %w", err)
}
