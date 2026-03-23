# 阿里云短信服务

## 概述

本模块封装了阿里云短信服务，用于发送验证码短信。

## 配置

在 `backend/etc/cloudcoupon-api.yaml` 中配置：

```yaml
AliSms:
  AccessKeyId: "${ALI_SMS_ACCESS_KEY_ID}"
  AccessKeySecret: "${ALI_SMS_ACCESS_KEY_SECRET}"
  SignName: ""
  TemplateCode: ""
  Endpoint: "dysmsapi.aliyuncs.com"

Redis:
  Host: "127.0.0.1:6379"
  Pass: ""
  DB: 0
```

## API接口

### 发送验证码

**接口地址**: `POST /api/v1/auth/sendSms`

**请求参数**:
```json
{
  "phone": "18856578827"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "验证码已发送"
}
```

**错误响应**:
```json
{
  "success": false,
  "message": "请等待60秒后再试"
}
```

## 功能特性

1. **验证码生成**: 自动生成6位数字验证码
2. **发送限制**: 60秒内只能发送一次
3. **有效期**: 验证码5分钟内有效
4. **自动清理**: 验证成功后自动删除验证码

## 使用示例

### 在注册逻辑中验证验证码

```go
// 在 register_logic.go 中
func (l *RegisterLogic) Register(req *types.RegisterRequest) (*types.RegisterResponse, error) {
    // 验证验证码
    smsLogic := NewSendSmsLogic(l.ctx, l.svcCtx)
    if err := smsLogic.VerifyCode(req.Phone, req.Code); err != nil {
        return &types.RegisterResponse{
            Success: false,
            Message: err.Error(),
        }, nil
    }
    
    // 继续注册逻辑...
}
```

## 模块结构

- `alisms.go`: 阿里云短信客户端封装
- `code.go`: 验证码生成和管理
- `README.md`: 使用文档

## 依赖

- `github.com/alibabacloud-go/dysmsapi-20170525/v5`: 阿里云短信SDK
- `github.com/redis/go-redis/v9`: Redis客户端

## 注意事项

1. 确保Redis服务已启动
2. 阿里云短信模板需要提前在控制台配置
3. 生产环境请妥善保管AccessKey信息
4. 建议使用环境变量或密钥管理服务存储敏感信息
