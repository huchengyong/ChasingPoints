// 合规收口：上线前暂时隐藏用户公开内容运营入口，待取得相关资质或完成专项合规评估后再恢复。
export const ADMIN_COMPLIANCE_MODE = true

export const canShowSocialReviewMenu = () => !ADMIN_COMPLIANCE_MODE
