// 合规收口：上线前暂时隐藏用户UGC、订阅付费和用户赛事发起入口，待取得相关资质
// 或完成专项合规评估后，再按模块逐步恢复。
export const APP_COMPLIANCE_MODE = true

export const canShowSocialPublishing = () => !APP_COMPLIANCE_MODE

export const canShowPaidMembershipEntry = () => !APP_COMPLIANCE_MODE

export const canShowUserTournamentActions = () => !APP_COMPLIANCE_MODE

export const getSaiXunTabLabel = () => '赛讯'

export const getSaiXunNavigationTitle = () => '赛讯'
