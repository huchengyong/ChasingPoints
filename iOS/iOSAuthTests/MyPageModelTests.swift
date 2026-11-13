import Foundation
import XCTest
@testable import iOS

@MainActor
final class MyPageModelTests: XCTestCase {
    private let userId: Int64 = 8801
    private var suiteName = ""
    private var defaults: UserDefaults!

    override func setUp() {
        super.setUp()
        suiteName = "MyPageModelTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
        defaults.removePersistentDomain(forName: suiteName)
    }

    override func tearDown() {
        defaults.removePersistentDomain(forName: suiteName)
        defaults = nil
        super.tearDown()
    }

    // MARK: - 页面出现

    func testGuestOnAppearMakesNoRequestsAndStaysInGuestState() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(server.requestCount("/api/user/overview"), 0)
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 0)
        XCTAssertNil(model.overview.value)
        XCTAssertNil(model.ranks.value)
        XCTAssertEqual(model.preferredGameType, 0)
        XCTAssertEqual(model.selectedRankGameType, 0)
        XCTAssertEqual(model.currentRankGameType, 3)
        XCTAssertFalse(model.showRankGamePicker)
        XCTAssertFalse(model.isRankSkeleton)
        XCTAssertEqual(model.rankCard.name, "未定级")
        XCTAssertFalse(model.memberIdentity.showBadge)
        XCTAssertEqual(model.careerState, .loading)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["—", "—", "—", "—"])
        XCTAssertEqual(model.profileLevelText, "Lv1")
        XCTAssertFalse(model.floatRewardVisible)
        XCTAssertFalse(model.compactRewardEntry.visible)
    }

    func testSignedInOnAppearLoadsOnceThenReloadsAfterFreshness() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        stubHomepage(server)
        var now = ProfileFixtures.time("2026-07-20T12:00:00+08:00")
        let model = makeModel(session, now: { now })

        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/overview"), 1)
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 1)
        XCTAssertEqual(model.careerState, .ready)

        now = now.addingTimeInterval(30)
        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/overview"), 1, "未超过 60 秒时不得重复读取")
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 1)

        now = now.addingTimeInterval(31)
        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/overview"), 2)
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 2)
    }

    // MARK: - 生涯数据状态

    func testCareerStateIsLoadingWhileOverviewIsInFlight() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/overview") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(ProfileFixtures.overview(stats: ProfileFixtures.userStats()))
        }
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/user/overview") == 1 }

        XCTAssertEqual(model.careerState, .loading)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["—", "—", "—", "—"])

        gate.signal()
        await load.value
        XCTAssertEqual(model.careerState, .ready)
    }

    func testCareerStateIsReadyWithMatchesAndEmptyWithoutThem() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        stubHomepage(server)
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()
        XCTAssertEqual(model.careerState, .ready)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["86", "51", "59.3%", "7"])

        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(totalMatches: 0, wins: 0, losses: 0, winRate: 0, maxWinStreak: 0)
        )))
        await model.refresh()

        XCTAssertEqual(model.careerState, .empty)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["0", "0", "0%", "0"])
    }

    func testCareerStateIsErrorWhenOverviewFails() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .raw("service down", status: 503))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(model.careerState, .error)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["—", "—", "—", "—"])
    }

    func testCareerStateIsErrorWhenStatsSectionIsUnavailable() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(success: false)
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(model.careerState, .error)
    }

    // MARK: - 段位卡球种

    func testSelectedRankGameTypeOnlyAppliesInsideTabs() async {
        GameTypePreference(defaults: defaults).save(userId: userId, gameType: 3)
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        stubHomepage(server, ranks: [
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 3, name: "黄金", totalWins: 10, totalLosses: 2)
            ),
            ProfileFixtures.rankInfoItem(
                gameType: 2,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 2, name: "白银", totalWins: 5, totalLosses: 1)
            ),
        ])
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(model.preferredGameType, 3)
        XCTAssertEqual(model.rankGameView.tabs.map(\.value), [3, 2])
        XCTAssertEqual(model.currentRankGameType, 3)
        XCTAssertEqual(model.rankCard.name, "黄金")

        model.selectedRankGameType = 2
        XCTAssertEqual(model.currentRankGameType, 2)
        XCTAssertEqual(model.rankCard.name, "白银")

        model.selectedRankGameType = 1
        XCTAssertEqual(model.currentRankGameType, 3, "不在 tabs 里的球种回落到 primaryGameType")
    }

    func testPickPreferredGameTypeSavesPreferenceAndHidesPicker() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        stubHomepage(server, ranks: GameTypeCatalog.ordered.map {
            ProfileFixtures.rankInfoItem(
                gameType: $0.value,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 1, name: "青铜")
            )
        })
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()
        XCTAssertTrue(model.showRankGamePicker)

        model.pickPreferredGameType(4)

        XCTAssertEqual(model.preferredGameType, 4)
        XCTAssertEqual(GameTypePreference(defaults: defaults).read(userId: userId), 4)
        XCTAssertFalse(model.showRankGamePicker)
        XCTAssertEqual(model.currentRankGameType, 4)
        XCTAssertEqual(model.selectedRankGameType, 0)
    }

    // MARK: - 分区独立

    func testOverviewFailureKeepsRankCard() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .raw("service down", status: 503))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 3, name: "黄金", totalWins: 4, totalLosses: 1)
            ),
        ])))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(model.careerState, .error)
        XCTAssertTrue(model.rankCard.ranked)
        XCTAssertEqual(model.rankCard.name, "黄金")
    }

    func testRankFailureKeepsCareerMetrics() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats()
        )))
        server.stub("GET", "/api/rank/infos", .raw("service down", status: 503))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertEqual(model.careerState, .ready)
        XCTAssertEqual(model.coreMetrics.map(\.value), ["86", "51", "59.3%", "7"])
        XCTAssertFalse(model.rankCard.ranked)
        XCTAssertFalse(model.isRankSkeleton)
        XCTAssertFalse(model.showRankGamePicker)
    }

    // MARK: - 悬浮奖励

    func testFloatRewardHidesForThreeDaysAfterSnoozeThenReturns() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(status: "not_started")
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))
        var now = ProfileFixtures.time("2026-07-20T12:00:00+08:00")
        let model = makeModel(session, now: { now })

        await model.onAppear()
        XCTAssertTrue(model.floatRewardVisible)

        model.snoozeFloatReward()
        XCTAssertFalse(model.floatRewardVisible)

        let key = FavoriteVenueReward.floatSnoozeKey(userId: userId, status: "not_started")
        XCTAssertNotNil(defaults.object(forKey: key))

        now = now.addingTimeInterval(2 * 24 * 60 * 60)
        await model.onAppear()
        XCTAssertFalse(model.floatRewardVisible, "3 天内保持隐藏")

        now = now.addingTimeInterval(2 * 24 * 60 * 60)
        await model.onAppear()
        XCTAssertTrue(model.floatRewardVisible)
        XCTAssertNil(defaults.object(forKey: key), "过期的稍后提醒记录应被删除")
    }

    func testPendingReviewShowsInlineEntryWithoutFloat() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(
                status: "pending_review", submittedVenueName: "测试球馆"
            )
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        await model.onAppear()

        XCTAssertFalse(model.floatRewardVisible)
        XCTAssertTrue(model.compactRewardEntry.inline)
        XCTAssertEqual(model.compactRewardEntry.statusText, "审核中")
    }

    // MARK: - 身份变化

    func testIdentityChangeResetsDerivedStateAndDropsLateOverview() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 3, name: "黄金", totalWins: 6, totalLosses: 2)
            ),
        ])))
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/overview") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(ProfileFixtures.overview(
                stats: ProfileFixtures.userStats(),
                favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(status: "not_started")
            ))
        }
        let model = makeModel(session, now: { Date(timeIntervalSince1970: 1_700_000_000) })

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/user/overview") == 1 }

        session.logout()
        await model.onIdentityChanged()
        gate.signal()
        await load.value

        XCTAssertNil(model.overview.value, "退出后迟到的概览结果不得写入")
        XCTAssertNil(model.ranks.value)
        XCTAssertEqual(model.preferredGameType, 0)
        XCTAssertEqual(model.careerState, .loading)
        XCTAssertFalse(model.memberIdentity.showBadge)
        XCTAssertFalse(model.floatRewardVisible)
        XCTAssertEqual(model.currentRankGameType, 3)
    }

    // MARK: - 辅助

    private func makeModel(_ session: SessionStore, now: @escaping () -> Date) -> MyPageModel {
        MyPageModel(session: session, defaults: defaults, now: now)
    }

    private func makeSignedInSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub(
            "GET",
            "/api/user/info",
            .json(AuthFixtures.userInfoResponse(AuthFixtures.userInfo(id: userId)))
        )
        let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        let session = makeSessionStore(server, credentialStore: store)
        await session.restore()
        return session
    }

    private func stubHomepage(
        _ server: MockHTTPServer,
        stats: [String: Any] = ProfileFixtures.userStats(),
        ranks: [[String: Any]] = [
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 3, name: "黄金", totalWins: 4, totalLosses: 1)
            ),
        ]
    ) {
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(stats: stats)))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos(ranks)))
    }
}
