import Foundation
import XCTest
@testable import iOS

@MainActor
final class RankExplainModelTests: XCTestCase {
    private let infosPath = "/api/rank/infos"
    private let configsPath = "/api/public/rank/configs"

    func testFirstLoadRequestsRankInfoAndConfigs() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        server.stub("GET", configsPath, RankingFixtures.configs(rulesVersion: 2))
        let model = RankExplainModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertEqual(model.rankInfo?.level, 3)
        XCTAssertEqual(model.rankList.map(\.level), [1, 2, 3, 4, 5, 6])
        XCTAssertEqual(model.loadedGameType, 3)
        XCTAssertEqual(model.currentGameTypeLabel, "中式八球")
        XCTAssertEqual(model.heroModel?.name, "黄金球手")
        XCTAssertEqual(model.heroModel?.placementRemaining, 7)
        XCTAssertEqual(model.growthPath.count, 6)
        XCTAssertEqual(model.rulesMeta.rulesVersion, 2)
        XCTAssertEqual(model.rankRules.version, 2)
        XCTAssertEqual(server.requestCount(infosPath), 1)
        XCTAssertEqual(server.requestCount(configsPath), 1)
        XCTAssertEqual(server.requests(infosPath).first?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(server.requests(configsPath).first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testInitialLoadingModeOnlyWhileFirstFetchIsInFlight() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let gate = DispatchSemaphore(value: 0)
        let stub = infosStub
        server.stub("GET", infosPath) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return stub
        }
        let model = RankExplainModel(session: session, gameType: 3)

        let load = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount(infosPath) == 1 }

        XCTAssertTrue(model.isInitialLoading)
        XCTAssertFalse(model.isRefreshing)

        gate.signal()
        await load.value

        XCTAssertFalse(model.isInitialLoading)
        XCTAssertFalse(model.isRefreshing)
        XCTAssertTrue(model.hasLoadedOnce)
    }

    func testStaleGameTypeResponseIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        let gate = DispatchSemaphore(value: 0)
        let calls = LockedCounter()
        let response = RankingFixtures.configs(rulesVersion: 1)
        server.stub("GET", configsPath) { _ in
            if calls.next() == 1 {
                _ = gate.wait(timeout: .now() + 5)
            }
            return response
        }

        let stale = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(configsPath) == 2 }
        await model.selectGameType(4)
        gate.signal()
        await stale.value

        XCTAssertEqual(model.currentGameType, 4)
        XCTAssertEqual(model.loadedGameType, 4)
        XCTAssertEqual(model.currentGameTypeLabel, "美式九球")
        XCTAssertEqual(model.heroModel?.name, "美式段位")
        XCTAssertFalse(model.isRefreshing)
    }

    func testConfigsFailureShowsInitialErrorAndRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        server.stub("GET", configsPath, .raw("service down", status: 503))
        let model = RankExplainModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isInitialError)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertNil(model.heroModel)

        stubContent(server)
        await model.retry()

        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertEqual(model.heroModel?.name, "黄金球手")
    }

    func testSwitchingGameTypeKeepsOldContentVisibleUntilNewDataArrives() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.heroModel?.name, "黄金球手")

        let gate = DispatchSemaphore(value: 0)
        let calls = LockedCounter()
        let response = RankingFixtures.configs(rulesVersion: 1)
        server.stub("GET", configsPath) { _ in
            if calls.next() == 1 {
                _ = gate.wait(timeout: .now() + 5)
            }
            return response
        }

        let switching = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(configsPath) == 2 }

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.loadedGameType, 3)
        XCTAssertEqual(model.currentGameTypeLabel, "中式八球", "刷新中仍展示已加载球种的文案")
        XCTAssertTrue(model.isRefreshing)
        XCTAssertEqual(model.heroModel?.name, "黄金球手", "新数据到达前保留旧内容")

        gate.signal()
        await switching.value

        XCTAssertEqual(model.loadedGameType, 2)
        XCTAssertEqual(model.currentGameTypeLabel, "九球追分")
        XCTAssertEqual(model.heroModel?.name, "九球段位")
        XCTAssertFalse(model.isRefreshing)
    }

    func testFirstFailureShowsErrorAndRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        server.stub("GET", infosPath, .raw("service down", status: 500))
        server.stub("GET", configsPath, RankingFixtures.configs(rulesVersion: 1))
        let model = RankExplainModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isInitialError)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertNil(model.heroModel)

        stubContent(server)
        await model.retry()

        XCTAssertFalse(model.isInitialError)
        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertEqual(model.heroModel?.name, "黄金球手")
    }

    func testRefreshFailureRevertsToLoadedGameTypeWithNotice() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.heroModel?.name, "黄金球手")

        server.stub("GET", configsPath, .raw("service down", status: 503))
        await model.selectGameType(2)

        XCTAssertEqual(model.currentGameType, 3, "刷新失败回到已加载的球种")
        XCTAssertEqual(model.loadedGameType, 3)
        XCTAssertEqual(model.heroModel?.name, "黄金球手")
        XCTAssertNotNil(model.notice)
        XCTAssertFalse(model.isInitialError)

        model.clearNotice()
        XCTAssertNil(model.notice)
    }

    func testUnsupportedGameTypeFallsBackToDefault() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let model = RankExplainModel(session: session, gameType: 9)

        await model.loadIfNeeded()

        XCTAssertEqual(model.currentGameType, 3)
        XCTAssertEqual(model.loadedGameType, 3)
        XCTAssertEqual(server.requestCount(configsPath), 1)
    }

    func testGuestDoesNotRequestRankData() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = RankExplainModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(server.requests.isEmpty)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertFalse(model.isInitialError)
        XCTAssertNil(model.heroModel)
    }

    func testLogoutClearsContentWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubContent(server)
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        let totalRequests = server.requests.count

        session.logout()
        await model.reloadForIdentityChange()

        XCTAssertNil(model.rankInfo)
        XCTAssertTrue(model.rankList.isEmpty)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertEqual(model.rulesMeta, RankRulesMeta.legacy)
        XCTAssertEqual(server.requests.count, totalRequests, "退出后不得继续请求")
    }

    // MARK: - 辅助

    private var infosStub: MockHTTPServer.Stub {
        RankingFixtures.rankInfos([
            (gameType: 3, info: RankingFixtures.rankInfoJSON(
                level: 3,
                name: "黄金球手",
                rankScore: 1280,
                nextLevel: 4,
                nextName: "铂金大师",
                nextScore: 1500,
                progress: 56,
                placementRemaining: 7
            )),
            (gameType: 2, info: RankingFixtures.rankInfoJSON(
                level: 4,
                name: "九球段位",
                rankScore: 1600,
                nextLevel: 5,
                nextName: "钻石",
                nextScore: 2000,
                progress: 20
            )),
            (gameType: 4, info: RankingFixtures.rankInfoJSON(
                level: 2,
                name: "美式段位",
                rankScore: 700,
                nextLevel: 3,
                nextName: "黄金球手",
                nextScore: 1000,
                progress: 40
            )),
        ])
    }

    private func stubContent(_ server: MockHTTPServer) {
        server.stub("GET", infosPath, infosStub)
        server.stub("GET", configsPath, RankingFixtures.configs(rulesVersion: 1))
    }
}
