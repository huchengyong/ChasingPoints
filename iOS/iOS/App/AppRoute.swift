import SwiftUI

/// 应用内页面路由，与小程序 pages.json 的页面一一对应；未迁移的页面先落到占位页。
enum AppRoute: Hashable {
    case login
    case settings
    case editProfile
    case editNickname
    case matchHistory
    case matchReport(matchId: Int64)
    case ranking
    case spectatorLobby
    case rankExplain(gameType: Int)
    case notificationSettings
    case opponentRecord
    case h2hRecord(
        opponentId: Int64,
        targetUserId: Int64,
        targetName: String,
        targetAvatar: String,
        opponentName: String
    )
    case memberCenter
    case reputation
    case statsDetail(gameType: Int)
    case performance
    case matchReplay(matchId: Int64)
    case challengeCompose
    case challenges(showHistory: Bool)
    case feedback
    case achievementWall(gameType: Int)
    case achievementDetail(achievementId: Int64)
    case friendList
    case friendHomepage(userId: Int64)
    case addFriend
    case friendRequests
    case blacklist
    case tournamentDetail(id: Int64)
    case venueList
    case venueDetail(id: Int64)
    case venueSubmit
    case rulesIndex
    case ruleDetail(category: String)
    case glossary
    case seasonIndex
    case seasonHistory
    case gearIndex

    /// 导航栏标题，取自 pages.json 的 navigationBarTitleText。
    var title: String {
        switch self {
        case .login: "登录"
        case .settings: "设置"
        case .editProfile: "编辑资料"
        case .editNickname: "编辑昵称"
        case .matchHistory: "对局记录"
        case .matchReport: "对局详情"
        case .ranking: "排行榜"
        case .spectatorLobby: "观赛大厅"
        case .rankExplain: "段位说明"
        case .notificationSettings: "通知管理"
        case .opponentRecord: "过往对手"
        case .h2hRecord: "交锋记录"
        case .memberCenter: "VIP中心"
        case .reputation: "信誉明细"
        case .statsDetail: "竞技分析"
        case .performance: "表现分析"
        case .matchReplay: "单场复盘"
        case .challengeCompose: "发起约球"
        case .challenges: "约球"
        case .feedback: "帮助、投诉与举报"
        case .achievementWall: "荣誉墙"
        case .achievementDetail: "成就详情"
        case .friendList: "好友列表"
        case .friendHomepage: "好友主页"
        case .addFriend: "添加好友"
        case .friendRequests: "好友申请"
        case .blacklist: "黑名单"
        case .tournamentDetail: "赛事详情"
        case .venueList: "附近球馆"
        case .venueDetail: "球馆详情"
        case .venueSubmit: "上传球馆"
        case .rulesIndex: "规则百科"
        case .ruleDetail: "规则详情"
        case .glossary: "术语词典"
        case .seasonIndex: "赛季"
        case .seasonHistory: "历届赛季"
        case .gearIndex: "我的装备"
        }
    }
}

/// 路由 → 页面。已迁移的返回真实页面，其余返回占位页。
struct AppRouteDestination: View {
    @EnvironmentObject private var session: SessionStore

    let route: AppRoute

    var body: some View {
        switch route {
        case .login:
            LoginView()
        case .settings:
            SettingsView(session: session)
        case .editProfile:
            ProfileEditView(session: session)
        case .editNickname:
            NicknameEditView()
        case .notificationSettings:
            NotificationSettingsView(session: session)
        case .reputation:
            ReputationView(session: session)
        case .opponentRecord:
            OpponentRecordView(session: session)
        case .h2hRecord(let opponentId, let targetUserId, let targetName, let targetAvatar, let opponentName):
            H2HRecordView(
                session: session,
                targetUserId: targetUserId,
                targetName: targetName,
                targetAvatar: targetAvatar,
                opponentId: opponentId,
                opponentName: opponentName
            )
        case .matchHistory:
            MatchHistoryView(session: session)
        case .matchReport(let matchId):
            MatchReportView(session: session, matchId: matchId)
        case .venueList:
            VenueListView(session: session)
        case .venueDetail(let id):
            VenueDetailView(session: session, venueId: id)
        case .venueSubmit:
            VenueSubmitView(session: session)
        case .rulesIndex:
            RulesIndexView(session: session)
        case .ruleDetail(let category):
            RuleDetailView(session: session, category: category)
        case .glossary:
            GlossaryView(session: session)
        default:
            MigrationPlaceholderView(title: route.title)
        }
    }
}
