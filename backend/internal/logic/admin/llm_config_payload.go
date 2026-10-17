package admin

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const (
	llmEndpointURLMaxLen = 512
	llmAPIKeyMaxLen      = 512
	llmModelMaxLen       = 128
	llmMaxTokensLimit    = 1000000
	// 密钥至少这么长才展示末 4 位，避免短密钥被脱敏值泄露大半。
	llmAPIKeyRevealMinLen = 12
)

func maskLlmAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	runes := []rune(apiKey)
	if len(runes) < llmAPIKeyRevealMinLen {
		return "****"
	}
	return "****" + string(runes[len(runes)-4:])
}

func buildAdminLlmConfigResp(cfg *model.LlmConfig) *types.AdminLlmConfigResp {
	resp := &types.AdminLlmConfigResp{
		Code:    0,
		Success: true,
		Message: "success",
	}
	if cfg == nil {
		return resp
	}

	resp.ApiFormat = cfg.APIFormat
	resp.EndpointUrl = cfg.EndpointURL
	resp.HasApiKey = cfg.APIKey != ""
	resp.ApiKeyMasked = maskLlmAPIKey(cfg.APIKey)
	resp.Model = cfg.Model
	resp.MaxTokens = cfg.MaxTokens
	return resp
}

// containsSpaceOrControl 拒绝空白与控制字符，避免密钥、模型名后续被写入请求头或请求体时出现注入。
func containsSpaceOrControl(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0
}

// buildLlmConfigModel 校验更新请求并生成待保存的模型。
// req.ApiKey 为空时沿用 storedAPIKey；两者都为空视为未提供密钥。
func buildLlmConfigModel(req *types.AdminLlmConfigUpdateReq, storedAPIKey string, updatedBy int64) (*model.LlmConfig, error) {
	if req == nil {
		return nil, errors.New("请求参数不能为空")
	}

	apiFormat := strings.TrimSpace(req.ApiFormat)
	switch apiFormat {
	case model.LlmAPIFormatAnthropic, model.LlmAPIFormatOpenAI:
	case "":
		return nil, errors.New("请选择接口格式")
	default:
		return nil, errors.New("接口格式只支持 anthropic 或 openai")
	}

	endpointURL := strings.TrimSpace(req.EndpointUrl)
	if endpointURL == "" {
		return nil, errors.New("请填写接口地址")
	}
	if utf8.RuneCountInString(endpointURL) > llmEndpointURLMaxLen {
		return nil, errors.New("接口地址过长")
	}
	parsed, err := url.Parse(endpointURL)
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" {
		return nil, errors.New("接口地址格式不正确")
	}
	if parsed.Scheme != "https" {
		return nil, errors.New("接口地址必须使用 https")
	}
	if parsed.User != nil {
		return nil, errors.New("接口地址不能包含账号密码，请在 API Key 中填写密钥")
	}

	apiKey := strings.TrimSpace(req.ApiKey)
	if apiKey == "" {
		apiKey = storedAPIKey
	}
	if apiKey == "" {
		return nil, errors.New("请填写 API Key")
	}
	if utf8.RuneCountInString(apiKey) > llmAPIKeyMaxLen {
		return nil, errors.New("API Key 过长")
	}
	if containsSpaceOrControl(apiKey) {
		return nil, errors.New("API Key 不能包含空白或控制字符")
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		return nil, errors.New("请填写模型名称")
	}
	if utf8.RuneCountInString(modelName) > llmModelMaxLen {
		return nil, errors.New("模型名称过长")
	}
	if containsSpaceOrControl(modelName) {
		return nil, errors.New("模型名称不能包含空白或控制字符")
	}

	if req.MaxTokens < 1 || req.MaxTokens > llmMaxTokensLimit {
		return nil, errors.New("max_tokens 需在 1 到 1000000 之间")
	}

	return &model.LlmConfig{
		ConfigKey:   model.DefaultLlmConfigKey,
		APIFormat:   apiFormat,
		EndpointURL: endpointURL,
		APIKey:      apiKey,
		Model:       modelName,
		MaxTokens:   req.MaxTokens,
		UpdatedBy:   updatedBy,
	}, nil
}
