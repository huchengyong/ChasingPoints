/// 合规收口：与小程序 `utils/compliance-mode.js` 保持一致。
enum AppFlags {
    static let complianceMode = true
    static var canShowSocialPublishing: Bool { !complianceMode }
    static var canShowPaidMembershipEntry: Bool { !complianceMode }
    static var canShowUserTournamentActions: Bool { !complianceMode }
}
