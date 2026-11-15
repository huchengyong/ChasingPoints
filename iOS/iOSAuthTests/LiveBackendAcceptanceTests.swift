// 真实后端只读验收：本文件只允许 GET。
//
// - 不调用任何写接口（保存 / 删除 / 发送 / 接受 / 拒绝 / 上传 / 注销 / 标记已读）。
// - 不连接 WebSocket：对局详情只调 `load()`，不调 `start()`；也不触发 `AppRealtime` / `MatchRealtime`。
// - 不请求相机、定位、相册权限：定位与地图一律用替身或关闭。
// - 不在断言消息、日志与快照文件名里输出 token；本文件不打印凭据，也不拼请求头。
//
// 仅在显式提供 `LIVE_API_BASE_URL`、`LIVE_ACCESS_TOKEN_A`、`LIVE_REFRESH_TOKEN_A`、
// `LIVE_ACCESS_TOKEN_B`、`LIVE_REFRESH_TOKEN_B` 时运行；日常测试全部跳过且不访问网络。
// 唯一例外：`SessionStore.restore()` 在 access token 失效时会自动走
// `/api/auth/refresh-token` 续期，那是会话机制而非业务写入，本文件不主动触发。

import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class LiveBackendAcceptanceTests: XCTestCase {
    /// 验收环境只允许这套预发域名，绝不能指向生产 `api.zhuifen.cn`。
    private static let allowedBaseURL = "https://api-zhuifen.kekemate.cn"
    /// 上海人民广场；用固定坐标替代系统定位，避免真实定位权限。
    private static let shanghai = Coordinate(latitude: 31.2304, longitude: 121.4737)

    private let pageSize = CGSize(width: 393, height: 852)

    private struct LiveConfig {
        let baseURL: URL
        let accessA: String
        let refreshA: String
        let accessB: String
        let refreshB: String
    }

    private struct LiveEnv {
        let session: SessionStore
        let activity: ActivityStore
        let gear: GearStore
    }

    private var config: LiveConfig?
    private var cachedA: SessionStore?
    private var cachedB: SessionStore?

    override func setUp() async throws {
        try await super.setUp()

        let env = ProcessInfo.processInfo.environment
        guard let rawBaseURL = env["LIVE_API_BASE_URL"], !rawBaseURL.isEmpty else {
            throw XCTSkip("未提供真实后端验收参数")
        }
        let accessA = env["LIVE_ACCESS_TOKEN_A"] ?? ""
        let refreshA = env["LIVE_REFRESH_TOKEN_A"] ?? ""
        let accessB = env["LIVE_ACCESS_TOKEN_B"] ?? ""
        let refreshB = env["LIVE_REFRESH_TOKEN_B"] ?? ""
        guard !accessA.isEmpty, !refreshA.isEmpty, !accessB.isEmpty, !refreshB.isEmpty else {
            throw XCTSkip("未提供真实后端验收参数")
        }
        guard rawBaseURL == Self.allowedBaseURL, let baseURL = URL(string: rawBaseURL) else {
            XCTFail("LIVE_API_BASE_URL 必须是验收环境 \(Self.allowedBaseURL)，当前值被拒绝")
            throw XCTSkip("LIVE_API_BASE_URL 不是允许的验收环境")
        }
        config = LiveConfig(
            baseURL: baseURL,
            accessA: accessA,
            refreshA: refreshA,
            accessB: accessB,
            refreshB: refreshB
        )
    }

    // MARK: - 会话与页面辅助

    private func sessionA() async throws -> SessionStore {
        if let cachedA { return cachedA }
        let session = try await makeSession(label: "账号 A", access: liveConfig().accessA, refresh: liveConfig().refreshA)
        cachedA = session
        return session
    }

    private func sessionB() async throws -> SessionStore {
        if let cachedB { return cachedB }
        let session = try await makeSession(label: "账号 B", access: liveConfig().accessB, refresh: liveConfig().refreshB)
        cachedB = session
        return session
    }

    private func liveConfig() throws -> LiveConfig {
        guard let config else { throw XCTSkip("未提供真实后端验收参数") }
        return config
    }

    private func makeSession(label: String, access: String, refresh: String) async throws -> SessionStore {
        let config = try liveConfig()
        let api = APIClient(baseURL: config.baseURL, session: URLSession(configuration: .ephemeral))
        let store = InMemoryCredentialStore(credentials: Credentials(accessToken: access, refreshToken: refresh))
        let session = SessionStore(api: api, credentialStore: store)
        await session.restore()
        XCTAssertNotNil(session.identity, "\(label)会话恢复失败")
        guard session.identity != nil else {
            throw XCTSkip("\(label)会话恢复失败，缺少真实身份无法继续")
        }
        return session
    }

    private func makeEnv(_ session: SessionStore) -> LiveEnv {
        LiveEnv(session: session, activity: ActivityStore(session: session), gear: GearStore(session: session))
    }

    private func render<Content: View>(
        _ name: String,
        env: LiveEnv,
        @ViewBuilder content: () -> Content
    ) throws {
        for scheme in [ColorScheme.light, .dark] {
            let suffix = scheme == .dark ? "dark" : "light"
            let image = try renderSnapshot("live-\(name)-\(suffix)", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    content()
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(env.session)
                .environmentObject(AppNavigator())
                .environmentObject(ToastCenter())
                .environmentObject(env.activity)
                .environmentObject(env.gear)
            }
            XCTAssertEqual(image.size, pageSize, "\(name) \(suffix) 快照尺寸异常")
        }
    }

    // MARK: - 断言

    /// 断言「已加载或空」；失败时把 LoadFailure 的分类与文案写进消息，便于对照真实后端。
    private func assertLoaded<T>(
        _ loader: ResourceLoader<T>,
        _ resource: String,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        if loader.value != nil { return }
        if let failure = loader.failure {
            XCTFail("\(resource)未加载：\(describe(failure, resource: resource))", file: file, line: line)
        } else {
            XCTFail("\(resource)未加载：\(phaseName(loader.phase))", file: file, line: line)
        }
    }

    private func assertFirstPage<Item: Identifiable>(
        _ list: PagedHistoryList<Item>,
        _ resource: String,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        switch list.phase {
        case .loaded, .empty:
            return
        case .failed(let message):
            XCTFail("\(resource)加载失败：\(message)", file: file, line: line)
        case .idle, .loading:
            XCTFail("\(resource)未完成加载：\(String(describing: list.phase))", file: file, line: line)
        }
    }

    private func assertHomeSection(
        _ phase: HomeSectionPhase,
        _ resource: String,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        switch phase {
        case .loaded:
            return
        case .failed(let failure):
            XCTFail("\(resource)加载失败：\(describe(failure, resource: resource))", file: file, line: line)
        case .idle, .loading:
            XCTFail("\(resource)未完成加载", file: file, line: line)
        }
    }

    private func describe(_ failure: LoadFailure, resource: String) -> String {
        let feedback = failure.feedback(resource: resource)
        return "分类 \(failure.category.rawValue)，原文「\(failure.message)」，界面提示「\(feedback.title)：\(feedback.description)」"
    }

    private func phaseName<T>(_ phase: ResourceLoader<T>.Phase) -> String {
        switch phase {
        case .idle: return "idle"
        case .loading: return "loading"
        case .loaded: return "loaded"
        case .failed(let message): return "failed(\(message))"
        }
    }

    /// 「我的」与荣誉墙的换季弹窗在页面消失时会标记通知已读（POST）。
    /// 渲染前先只读预检：存在未读换季通知就跳过快照，守住本文件的只读约束。
    private func requireNoUnreadSeasonRollover(_ session: SessionStore) async throws {
        let notifications: [NotificationInfo]
        do {
            notifications = try await AchievementAPI(session: session).latestSeasonRolloverNotifications()
        } catch {
            XCTFail("换季通知预检失败：\(describe(LoadFailure(error), resource: "换季通知"))")
            throw XCTSkip("换季通知预检失败，无法确认渲染不会触发写请求")
        }
        if HonorWallRules.latestUnreadSeasonRollover(notifications) != nil {
            throw XCTSkip("账号存在未读换季通知：渲染该页会在消失时标记已读（写请求），跳过快照以守住只读约束")
        }
    }

    // MARK: - 我的

    func testMyPageAndSubPages() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)

        let myPage = MyPageModel(session: session)
        await myPage.onAppear()
        assertLoaded(myPage.overview, "我的-个人概览")
        assertLoaded(myPage.ranks, "我的-段位")
        try await requireNoUnreadSeasonRollover(session)
        try render("my-page", env: env) { MyPage(session: session, model: myPage) }

        let member = MemberCenterModel(session: session)
        await member.onAppear()
        assertLoaded(member.overview, "会员中心-概览")
        assertLoaded(member.ranks, "会员中心-段位")
        try render("member-center", env: env) { MemberCenterView(session: session, model: member) }

        let reputation = ReputationModel(session: session)
        await reputation.onAppear()
        assertLoaded(reputation.status, "信誉-状态")
        assertFirstPage(reputation.logs, "信誉-变更记录")
        try render("reputation", env: env) { ReputationView(session: session, model: reputation) }

        let opponents = OpponentRecordModel(session: session)
        await opponents.onAppear()
        assertFirstPage(opponents.list, "过往对手")
        try render("opponent-record", env: env) { OpponentRecordView(session: session, model: opponents) }

        let stats = StatsDetailModel(session: session, gameType: 3)
        await stats.loadIfNeeded()
        if let failure = stats.failure {
            XCTFail("数据统计加载失败：\(describe(failure, resource: "数据统计"))")
        } else {
            XCTAssertNotNil(stats.overview, "数据统计未加载到数据（仍在加载：\(stats.isFetching)）")
        }
        try render("stats-detail", env: env) { StatsDetailView(session: session, gameType: 3, model: stats) }

        let performance = PerformanceModel(session: session, gameType: 3)
        await performance.loadIfNeeded()
        if let failure = performance.failure {
            XCTFail("表现分析加载失败：\(describe(failure, resource: "表现分析"))")
        } else {
            XCTAssertTrue(performance.data != nil || performance.isLocked, "表现分析未加载到数据也未标记未解锁")
        }
        try render("performance", env: env) { PerformanceView(session: session, gameType: 3, model: performance) }

        let rankExplain = RankExplainModel(session: session, gameType: 3)
        await rankExplain.loadIfNeeded()
        XCTAssertTrue(rankExplain.hasLoadedOnce, "段位说明未完成加载")
        XCTAssertFalse(rankExplain.hasLoadError, "段位说明加载失败（模型只给出布尔标记）")
        try render("rank-explain", env: env) { RankExplainView(session: session, gameType: 3, model: rankExplain) }

        let notifications = NotificationPreferencesModel(session: session)
        await notifications.load()
        XCTAssertEqual(notifications.phase, .ready, "通知设置未加载：\(notifications.errorMessage ?? "无错误文案")")
        try render("notification-settings", env: env) {
            NotificationSettingsView(session: session, model: notifications)
        }

        let privacy = PrivacySettingsModel(session: session)
        await privacy.load()
        XCTAssertEqual(privacy.phase, .ready, "隐私设置未加载")
        try render("settings", env: env) { SettingsView(session: session, privacyModel: privacy) }
    }

    // MARK: - 对局记录与衍生页面

    func testMatchHistoryAndDerivedPages() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let api = MatchHistoryAPI(session: session)

        let participant = PagedHistoryList<MatchListItem>(session: session) { page in
            try await api.participantPage(page: page)
        }
        let referee = PagedHistoryList<RefereeHistoryItem>(session: session) { page in
            try await api.refereePage(page: page)
        }
        await participant.loadIfNeeded()
        await referee.loadIfNeeded()
        assertFirstPage(participant, "对局记录-参赛第一页")
        assertFirstPage(referee, "对局记录-执裁第一页")
        try render("match-history", env: env) {
            MatchHistoryView(session: session, participantList: participant, refereeList: referee)
        }

        guard let match = participant.items.first else {
            throw XCTSkip("参赛记录为空，缺少对局 id，无法生成衍生页面")
        }

        // 参赛记录都是已结束对局；页面 `.task` 会调 start()，已结束对局不会建立 WebSocket。
        let detail = MatchDetailModel(session: session, matchId: match.id)
        await detail.load()
        switch detail.phase {
        case .loaded:
            break
        case .failed(let failure):
            XCTFail("对局详情加载失败：\(describe(failure, resource: "对局详情"))")
        case .loading:
            XCTFail("对局详情未完成加载")
        }
        try render("match-detail", env: env) { MatchDetailView(session: session, matchId: match.id, model: detail) }

        let report = MatchReportModel(session: session, matchId: match.id)
        await report.load()
        switch report.phase {
        case .loaded:
            break
        case .failed(let failure):
            XCTFail("战报加载失败：\(describe(failure, resource: "战报"))")
        case .loading:
            XCTFail("战报未完成加载")
        }
        try render("match-report", env: env) {
            MatchReportView(session: session, matchId: match.id, fromHistory: true, model: report)
        }

        let replay = MatchReplayModel(session: session, matchId: match.id)
        await replay.load()
        switch replay.phase {
        case .loaded:
            break
        case .failed(let failure):
            XCTFail("复盘加载失败：\(describe(failure, resource: "复盘"))")
        case .loading:
            XCTFail("复盘未完成加载")
        }
        try render("match-replay", env: env) {
            MatchReplayView(session: session, matchId: match.id, model: replay)
        }

        let share = ShareResultModel(session: session, matchId: match.id)
        await share.load()
        XCTAssertEqual(share.phase, .ready, "分享海报未就绪（模型只给出状态枚举）")
        try render("share-result", env: env) {
            ShareResultView(session: session, matchId: match.id, model: share)
        }

        let gearCard = MatchGearCardModel(session: session, matchId: match.id, enabled: true)
        await gearCard.load()
        switch gearCard.phase {
        case .ready, .hidden:
            break
        case .failed(let message):
            XCTFail("对局装备加载失败：\(message)")
        case .loading:
            XCTFail("对局装备未完成加载")
        }
        try render("match-gear", env: env) {
            MatchGearCardView(session: session, matchId: match.id, enabled: true, model: gearCard)
        }
    }

    // MARK: - 首页与赛讯

    func testHomeAndNews() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let location = StubLocationProvider(result: .success(Self.shanghai))

        let home = HomeModel(session: session, location: location)
        await home.onAppear()
        assertHomeSection(home.leaderboardPhase, "首页-排行榜")
        assertHomeSection(home.eventPhase, "首页-赛事新闻")
        assertHomeSection(home.livePhase, "首页-进行中对局")
        assertHomeSection(home.nearbyPhase, "首页-附近球房")
        try render("home", env: env) { HomeView(session: session, model: home) }

        let news = SaiXunListModel(session: session)
        await news.loadIfNeeded()
        switch news.phase {
        case .loaded, .empty:
            break
        case .failed(let failure):
            XCTFail("赛讯列表加载失败：\(describe(failure, resource: "赛讯列表"))")
        case .idle, .loading:
            XCTFail("赛讯列表未完成加载")
        }
        try render("news-home", env: env) { NewsHomeView(session: session, model: news) }

        guard let event = news.items.first else {
            throw XCTSkip("赛讯列表为空，缺少资讯 id，无法生成资讯详情页")
        }
        // 详情页有 30 秒自动刷新循环；传一个真正挂起的 sleep，避免测试期间空转请求。
        let detail = SaiXunDetailModel(
            session: session,
            eventId: event.id,
            sleep: { _ in try await Task.sleep(for: .seconds(3600)) }
        )
        await detail.loadIfNeeded()
        switch detail.phase {
        case .loaded:
            break
        case .failed(let failure):
            XCTFail("资讯详情加载失败：\(describe(failure, resource: "资讯详情"))")
        case .idle, .loading:
            XCTFail("资讯详情未完成加载")
        }
        try render("news-detail", env: env) {
            NewsDetailView(session: session, eventId: event.id, model: detail)
        }
    }

    // MARK: - 规则

    func testRulesSearchIndexDetailAndGlossary() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)

        let search = RulesSearchModel(api: RulesAPI(session: session))
        search.keyword = "斯诺克"
        await search.search()
        switch search.phase {
        case .loaded, .empty:
            break
        case .failed(let failure):
            XCTFail("规则搜索失败：\(describe(failure, resource: "规则搜索"))")
        case .idle, .loading:
            XCTFail("规则搜索未完成")
        }
        try render("rules-index", env: env) { RulesIndexView(session: session, search: search) }

        let detail = RuleDetailModel(session: session, category: "snooker")
        await detail.select(.rule)
        assertLoaded(detail.loader(for: .rule), "规则详情-规则")
        try render("rule-detail", env: env) {
            RuleDetailView(session: session, category: "snooker", model: detail)
        }

        let glossary = ResourceLoader<[GlossaryItem]>(session: session, access: .anyone) {
            try await RulesAPI(session: session).glossary()
        }
        await glossary.loadIfNeeded()
        assertLoaded(glossary, "术语词典")
        try render("glossary", env: env) { GlossaryView(session: session, loader: glossary) }
    }

    // MARK: - 赛季与荣誉

    func testSeasonAndHonorWall() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)

        let season = SeasonModel(session: session, gameType: 3)
        await season.loadIfNeeded()
        switch season.phase {
        case .ready:
            break
        case .failed(let failure):
            XCTFail("赛季主页加载失败：\(describe(failure, resource: "赛季主页"))")
        case .idle, .loading:
            XCTFail("赛季主页未完成加载")
        }
        try render("season", env: env) { SeasonView(session: session, model: season) }

        let history = SeasonHistoryModel(session: session, gameType: 3)
        await history.loadIfNeeded()
        switch history.phase {
        case .ready, .empty:
            break
        case .failed(let failure):
            XCTFail("赛季历史加载失败：\(describe(failure, resource: "赛季历史"))")
        case .idle, .loading:
            XCTFail("赛季历史未完成加载")
        }
        try render("season-history", env: env) {
            SeasonHistoryView(session: session, gameType: 3, model: history)
        }

        guard let seasonId = season.season?.id ?? history.items.first(where: { $0.hasRecord })?.id else {
            throw XCTSkip("没有可用的赛季 id，无法生成赛季报告")
        }
        let report = SeasonReportModel(session: session, seasonId: seasonId, gameType: 3)
        await report.loadIfNeeded()
        switch report.phase {
        case .ready, .empty:
            break
        case .failed(let failure):
            XCTFail("赛季报告加载失败：\(describe(failure, resource: "赛季报告"))")
        case .idle, .loading:
            XCTFail("赛季报告未完成加载")
        }
        try render("season-report", env: env) {
            SeasonReportView(session: session, seasonId: seasonId, gameType: 3, model: report)
        }

        let honor = HonorWallModel(session: session, gameType: 3)
        await honor.loadIfNeeded()
        if let failure = honor.failure {
            XCTFail("荣誉墙加载失败：\(describe(failure, resource: "荣誉墙"))")
        } else {
            XCTAssertTrue(honor.hasLoaded, "荣誉墙未加载到数据")
        }
        try await requireNoUnreadSeasonRollover(session)
        try render("honor-wall", env: env) {
            HonorWallView(session: session, gameType: 3, model: honor)
        }

        guard let achievement = honor.wall?.careerAchievements.first else {
            throw XCTSkip("荣誉墙没有成就条目，缺少 achievementId")
        }
        let achievementModel = AchievementDetailModel(session: session, achievementId: achievement.id)
        await achievementModel.loadIfNeeded()
        if let failure = achievementModel.failure {
            XCTFail("成就详情加载失败：\(describe(failure, resource: "成就详情"))")
        } else {
            XCTAssertTrue(achievementModel.hasLoaded, "成就详情未加载到数据")
        }
        try render("achievement-detail", env: env) {
            AchievementDetailView(session: session, achievementId: achievement.id, model: achievementModel)
        }
    }

    // MARK: - 社交

    func testSocialPages() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)

        let friends = FriendListModel(session: session, activity: env.activity)
        await friends.onAppear()
        assertFirstPage(friends.list, "好友列表")
        try render("friends", env: env) {
            FriendListView(session: session, activity: env.activity, model: friends)
        }

        let requests = FriendRequestsModel(session: session, activity: env.activity)
        await requests.onAppear()
        assertFirstPage(requests.list, "好友申请")
        try render("friend-requests", env: env) {
            FriendRequestsView(session: session, activity: env.activity, model: requests)
        }

        let blacklist = BlacklistModel(session: session)
        await blacklist.onAppear()
        assertFirstPage(blacklist.list, "黑名单")
        try render("blacklist", env: env) { BlacklistView(session: session, model: blacklist) }

        let challenges = ChallengesModel(session: session, activity: env.activity, initialTab: .received)
        await challenges.onAppear()
        if let failure = challenges.activeFailure {
            XCTFail("约球列表加载失败：\(describe(failure, resource: "约球"))")
        }
        XCTAssertFalse(challenges.activeLoading, "约球列表未完成加载")
        try render("challenges", env: env) {
            ChallengesView(session: session, activity: env.activity, initialTab: .received, model: challenges)
        }

        guard let friend = friends.items.first else {
            throw XCTSkip("好友列表为空，缺少好友 id，无法生成好友衍生页面")
        }
        let target = FriendHomepageTarget(
            userId: friend.friendUserId,
            name: friend.nickname,
            avatar: friend.avatar,
            rankName: friend.rankName,
            vipLevel: friend.vipLevel,
            rankLevel: friend.rankLevel
        )

        let homepage = FriendHomepageModel(session: session, target: target)
        await homepage.onAppear()
        assertLoaded(homepage.loader, "好友主页")
        try render("friend-homepage", env: env) {
            FriendHomepageView(session: session, target: target, model: homepage)
        }

        let pk = PKReportModel(session: session, target: target)
        await pk.onAppear()
        switch pk.phase {
        case .ready, .empty:
            break
        case .failed(let failure):
            XCTFail("PK 战绩加载失败：\(describe(failure, resource: "PK 战绩"))")
        case .loading:
            XCTFail("PK 战绩未完成加载")
        }
        try render("pk-report", env: env) { PKReportView(session: session, target: target, model: pk) }

        let h2h = H2HRecordModel(
            session: session,
            targetUserId: target.userId,
            targetName: target.name,
            opponentId: friend.friendUserId,
            opponentName: friend.nickname
        )
        await h2h.onAppear()
        if let failure = h2h.failure {
            XCTFail("交手记录加载失败：\(describe(failure, resource: "交手记录"))")
        } else {
            XCTAssertTrue(h2h.statsLoaded || h2h.historyLoaded, "交手记录未加载到任何数据")
        }
        try render("h2h-record", env: env) { H2HRecordView(session: session, model: h2h) }

        let gearFriend = GearFriendModel(session: session, userId: friend.friendUserId, ownerName: friend.nickname)
        await gearFriend.load()
        if let failure = gearFriend.failure {
            XCTFail("好友装备加载失败：\(failure.title)：\(failure.description)")
        }
        XCTAssertFalse(gearFriend.loading, "好友装备仍在加载")
        try render("gear-friend", env: env) {
            GearFriendView(session: session, userId: friend.friendUserId, name: friend.nickname, model: gearFriend)
        }
    }

    // MARK: - 约球三分栏

    func testChallengeTabs() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let model = ChallengesModel(session: session, activity: env.activity, initialTab: .received)
        await model.onAppear()
        XCTAssertFalse(model.activeLoading, "约球「收到」栏未完成加载")
        try render("challenges-received", env: env) {
            ChallengesView(session: session, activity: env.activity, initialTab: .received, model: model)
        }

        await model.switchTab(.sent)
        XCTAssertFalse(model.activeLoading, "约球「发出」栏未完成加载")
        try render("challenges-sent", env: env) {
            ChallengesView(session: session, activity: env.activity, initialTab: .sent, model: model)
        }

        await model.switchTab(.history)
        XCTAssertFalse(model.historyLoading, "约球「记录」栏未完成加载")
        if let failure = model.historyFailure {
            XCTFail("约球记录加载失败：\(describe(failure, resource: "约球记录"))")
        }
        try render("challenges-history", env: env) {
            ChallengesView(session: session, activity: env.activity, initialTab: .history, model: model)
        }
    }

    // MARK: - 消息中心

    func testMessageCenterLoadsWithoutMarkingRead() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let model = MessageCenterModel(session: session, activity: env.activity)
        await model.onAppear()
        assertFirstPage(model.list, "消息中心")

        // 只读核对：加载不得在本地标记已读或删除；每条消息的已读状态必须与接口返回一致。
        XCTAssertTrue(model.readIDs.isEmpty, "加载过程写入了本地已读标记")
        XCTAssertTrue(model.deletedIDs.isEmpty, "加载过程写入了本地删除标记")
        for item in model.items {
            XCTAssertEqual(model.isRead(item), item.isRead, "消息 \(item.id) 的已读状态在加载后被改动")
        }
        try render("message-center", env: env) {
            MessageCenterView(session: session, activity: env.activity, model: model)
        }
    }

    // MARK: - 装备

    func testGearPages() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let store = env.gear

        let index = GearIndexModel(session: session, store: store)
        await index.onAppear()
        if let failure = index.failure {
            XCTFail("装备列表加载失败：\(failure.title)：\(failure.description)")
        }
        XCTAssertFalse(index.isInitialLoading, "装备列表仍在加载")
        XCTAssertNotNil(index.data, "装备列表未加载到数据")
        try render("gear-index", env: env) {
            GearIndexView(session: session, store: store, model: index)
        }

        guard let item = index.items.first else {
            throw XCTSkip("账号没有装备条目，跳过装备详情")
        }
        let itemModel = GearItemModel(session: session, store: store, itemId: item.id)
        await itemModel.onAppear()
        if let failure = itemModel.failure {
            XCTFail("装备详情加载失败：\(failure.title)：\(failure.description)")
        }
        XCTAssertNotNil(itemModel.item, "装备详情未加载到条目")
        try render("gear-item", env: env) {
            GearItemView(session: session, store: store, itemId: item.id, model: itemModel)
        }

        // pickToken 传 nil：页面消失时不会调 cancelPick（那是写操作）。
        let add = GearAddModel(session: session, store: store, requestedCategory: "cue", isPick: false, pickToken: nil)
        await add.start()
        XCTAssertNil(add.listError, "装备目录加载失败：\(add.listError ?? "无错误文案")")
        XCTAssertFalse(add.brands.isEmpty, "装备目录未返回品牌")
        try render("gear-add", env: env) {
            GearAddView(session: session, store: store, category: "cue", pickToken: nil, model: add)
        }
    }

    // MARK: - 球馆

    func testVenueListAndDetail() async throws {
        let session = try await sessionA()
        let env = makeEnv(session)
        let location = StubLocationProvider(result: .success(Self.shanghai))
        let list = VenueListModel(session: session, location: location)
        await list.loadIfNeeded()
        switch list.phase {
        case .loaded, .empty:
            break
        case .failed(let failure):
            XCTFail("球馆列表加载失败：\(describe(failure, resource: "球馆列表"))")
        case .idle, .loading:
            XCTFail("球馆列表未完成加载")
        }
        try render("venues", env: env) { VenueListView(session: session, model: list) }

        guard let venue = list.items.first else {
            throw XCTSkip("附近球房为空，缺少球馆 id，无法生成详情页")
        }
        let detail = VenueDetailModel(session: session, venueId: venue.id)
        await detail.loadIfNeeded()
        assertLoaded(detail.loader, "球馆详情")
        // showsMap=false：快照不加载地图瓦片，避免额外网络与渲染成本。
        try render("venue-detail", env: env) {
            VenueDetailView(session: session, venueId: venue.id, model: detail, showsMap: false)
        }
    }

    // MARK: - 账号 B

    func testAccountBCorePages() async throws {
        let session = try await sessionB()
        let env = makeEnv(session)

        let myPage = MyPageModel(session: session)
        await myPage.onAppear()
        assertLoaded(myPage.overview, "账号 B-个人概览")
        assertLoaded(myPage.ranks, "账号 B-段位")

        let history = PagedHistoryList<MatchListItem>(session: session) { page in
            try await MatchHistoryAPI(session: session).participantPage(page: page)
        }
        await history.loadIfNeeded()
        assertFirstPage(history, "账号 B-对局记录第一页")

        let friends = FriendListModel(session: session, activity: env.activity)
        await friends.onAppear()
        assertFirstPage(friends.list, "账号 B-好友列表")

        let messages = MessageCenterModel(session: session, activity: env.activity)
        await messages.onAppear()
        assertFirstPage(messages.list, "账号 B-消息中心")

        try await requireNoUnreadSeasonRollover(session)
        try render("account-b-my-page", env: env) { MyPage(session: session, model: myPage) }
    }
}
