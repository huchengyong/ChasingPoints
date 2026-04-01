package config

type ComplianceConfig struct {
	RestrictedMode      bool `json:",env=COMPLIANCE_RESTRICTED_MODE,default=true"`
	AllowSocial         bool `json:",env=COMPLIANCE_ALLOW_SOCIAL,default=false"`
	AllowMemberPayment  bool `json:",env=COMPLIANCE_ALLOW_MEMBER_PAYMENT,default=false"`
	AllowUserTournament bool `json:",env=COMPLIANCE_ALLOW_USER_TOURNAMENT,default=false"`
}

func (c Config) SocialEnabled() bool {
	if !c.Compliance.RestrictedMode {
		return true
	}
	return c.Compliance.AllowSocial
}

func (c Config) MemberPaymentEnabled() bool {
	if !c.Compliance.RestrictedMode {
		return true
	}
	return c.Compliance.AllowMemberPayment
}

func (c Config) UserTournamentEnabled() bool {
	if !c.Compliance.RestrictedMode {
		return true
	}
	return c.Compliance.AllowUserTournament
}

func DisabledFeatureMessage(feature string) string {
	switch feature {
	case "social":
		return "当前版本暂未开放动态互动功能"
	case "member_payment":
		return "当前版本暂未开放会员订阅"
	case "tournament_user_action":
		return "当前版本暂未开放用户赛事操作"
	default:
		return "当前功能暂未开放"
	}
}
