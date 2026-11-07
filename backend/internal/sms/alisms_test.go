package sms

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
)

// 线上曾出现：HTTP 错误把完整请求 URL（含手机号与验证码）写进日志。
func TestSanitizeSmsErrorDropsURLAndMasksPhoneAndCode(t *testing.T) {
	const phone = "13800001111"
	const code = "196561"
	transportErr := &url.Error{
		Op:  "Post",
		URL: "https://dysmsapi.aliyuncs.com/?PhoneNumbers=" + phone + "&TemplateParam=%7B%22code%22%3A%22" + code + "%22%7D",
		Err: io.EOF,
	}

	got := sanitizeSmsError(transportErr, phone, code).Error()

	for _, secret := range []string{phone, code, "dysmsapi", "TemplateParam"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitized error still contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "EOF") {
		t.Fatalf("sanitized error lost the underlying reason: %s", got)
	}
}

func TestSanitizeSmsErrorMasksSecretsInPlainMessages(t *testing.T) {
	const phone = "13800001111"
	const code = "654321"
	got := sanitizeSmsError(errors.New("provider rejected "+phone+" with code "+code), phone, code).Error()

	if strings.Contains(got, phone) || strings.Contains(got, code) {
		t.Fatalf("plain message still exposes phone or code: %s", got)
	}
}
