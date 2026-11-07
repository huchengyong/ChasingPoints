package oauth

import "chasing_points/internal/config"

func NewProviderVerifier(security config.SecurityConfig) Verifier {
	var huawei Verifier
	if security.HuaweiOAuth.Enabled {
		huawei = NewHuaweiVerifier(security.HuaweiOAuth)
	}
	return ProviderVerifier{Huawei: huawei}
}
