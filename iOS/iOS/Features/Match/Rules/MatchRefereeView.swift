import Foundation

/// 战报 / 详情页的裁判身份卡；未绑定裁判时 `hasReferee == false`。
struct RefereeIdentityCard: Equatable {
    var hasReferee = false
    var refereeUserId: Int64 = 0
    var refereeName = ""
    var refereeAvatar = ""
    var refereeJoinedAt = ""
    var refereeJoinedAtText = ""
    var refereeDurationSeconds: Int64 = 0
    var refereeDurationText = ""
    var completedByUserId: Int64 = 0
    var completionSource = ""
    var completionLabel = ""
    var hasReliableAttribution = false
    var neutralLabel = "本场裁判"
}

/// 赛果页按查看者角色切换标题、可见模块与底部操作。
struct RefereeResultViewConfig: Equatable {
    let pageTitle: String
    let showWinLossBanner: Bool
    let showRankChange: Bool
    let showRematchButton: Bool
    let showShareResultButton: Bool
    let showH2HActions: Bool
    let showRefereeCard: Bool
    /// 亮点与数据按选手「我 / 对手」视角生成，裁判没有个人视角。
    let showPersonalInsights: Bool
    let roleHint: String
}

/// 逐条对应小程序 `utils/match-referee-view.js` 里战报与详情页用到的部分。
enum MatchRefereeView {
    static func completionSourceLabel(_ source: String?) -> String {
        MatchFormatting.completionSourceLabel(source)
    }

    static func hasReliableAttribution(_ source: String?) -> Bool {
        MatchFormatting.hasReliableCompletionSource(source)
    }

    static func identityCard(
        refereeBound: Bool = false,
        refereeUserId: Int64 = 0,
        refereeName: String = "",
        refereeAvatar: String = "",
        refereeJoinedAt: String = "",
        refereeDurationSeconds: Int64 = 0,
        completedByUserId: Int64 = 0,
        completionSource: String = "",
        status: Int = 0
    ) -> RefereeIdentityCard {
        guard refereeBound, refereeUserId > 0 else { return RefereeIdentityCard() }

        let reliable = status == 2 && hasReliableAttribution(completionSource)
        return RefereeIdentityCard(
            hasReferee: true,
            refereeUserId: refereeUserId,
            refereeName: refereeName.isEmpty ? "本场裁判" : refereeName,
            refereeAvatar: refereeAvatar,
            refereeJoinedAt: refereeJoinedAt,
            refereeJoinedAtText: AppFormat.dateTime(refereeJoinedAt),
            refereeDurationSeconds: refereeDurationSeconds,
            refereeDurationText: MatchFormatting.duration(seconds: refereeDurationSeconds) ?? "",
            completedByUserId: completedByUserId,
            completionSource: completionSource,
            completionLabel: reliable ? completionSourceLabel(completionSource) : "",
            hasReliableAttribution: reliable,
            neutralLabel: "本场裁判"
        )
    }

    static func resultViewConfig(viewerRole: String = "player1", status: Int = 1) -> RefereeResultViewConfig {
        let isReferee = viewerRole == "referee"
        let isCompleted = status == 2

        if isReferee {
            return RefereeResultViewConfig(
                pageTitle: isCompleted ? "执裁详情" : "执裁完成",
                showWinLossBanner: false,
                showRankChange: false,
                showRematchButton: false,
                showShareResultButton: false,
                showH2HActions: false,
                showRefereeCard: true,
                showPersonalInsights: false,
                roleHint: isCompleted ? "裁判视角，展示双方选手和执裁归因" : "已完赛，可查看执裁详情"
            )
        }

        return RefereeResultViewConfig(
            pageTitle: "战报",
            showWinLossBanner: true,
            showRankChange: true,
            showRematchButton: true,
            showShareResultButton: true,
            showH2HActions: true,
            showRefereeCard: true,
            showPersonalInsights: true,
            roleHint: ""
        )
    }
}
