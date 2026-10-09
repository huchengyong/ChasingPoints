import Foundation
import XCTest
@testable import iOS

@MainActor
final class MatchReportModelTests: XCTestCase {
    func testPlayerReportLoadsDetailAndRewardOnce() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail()))
        server.stub("GET", "/api/match/reward-summary", .json(MatchHistoryFixtures.rewardSummary(
            list: [MatchHistoryFixtures.rewardItem()]
        )))
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        guard case .loaded(let report) = model.phase else { return XCTFail("战报应加载成功：\(model.phase)") }
        XCTAssertEqual(report.id, 101)
        XCTAssertEqual(server.requests("/api/match/detail").first?.query["match_id"], "101")
        guard case .ready(let rewards) = model.reward else { return XCTFail("应展示本场奖励：\(model.reward)") }
        XCTAssertEqual(rewards.map(\.achievementName), ["夜猫子"])
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 1)
    }

    func testRefereeReportDoesNotRequestReward() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.refereeDetail()))
        let model = MatchReportModel(session: session, matchId: 201)

        await model.load()

        guard case .loaded(let report) = model.phase else { return XCTFail("执裁战报应加载成功") }
        XCTAssertTrue(report.isRefereeView)
        XCTAssertEqual(model.reward, .hidden)
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 0)
    }

    func testUnfinishedMatchDoesNotRequestReward() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail(status: 1)))
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        XCTAssertEqual(model.reward, .hidden)
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 0)
    }

    func testDetailFailureShowsErrorInsteadOfDefaultScore() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.businessFailure))
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        guard case .failed = model.phase else {
            return XCTFail("详情失败不得显示为已加载的默认战报：\(model.phase)")
        }
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 0)
    }

    func testRewardFailureHidesOnlyTheRewardSection() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail()))
        server.stub("GET", "/api/match/reward-summary", .json(MatchHistoryFixtures.businessFailure))
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        guard case .loaded = model.phase else { return XCTFail("奖励失败不应影响战报主体") }
        XCTAssertEqual(model.reward, .hidden)
    }

    func testPendingRewardIsReportedWithoutPolling() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail()))
        server.stub("GET", "/api/match/reward-summary", .json(MatchHistoryFixtures.rewardSummary(status: "pending")))
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        XCTAssertEqual(model.reward, .pending)
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 1, "历史战报只读取一次，不轮询")
    }

    func testLateDetailAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/detail") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(MatchHistoryFixtures.playerDetail())
        }
        let model = MatchReportModel(session: session, matchId: 101)

        let load = Task { await model.load() }
        await waitUntil { server.requestCount("/api/match/detail") == 1 }
        session.logout()
        gate.signal()
        await load.value

        guard case .loading = model.phase else {
            return XCTFail("退出后旧账号的战报不得写入：\(model.phase)")
        }
        XCTAssertEqual(server.requestCount("/api/match/reward-summary"), 0)
    }

    func testLoadWithoutSessionDoesNotRequest() async throws {
        let server = MockHTTPServer()
        let store = InMemoryCredentialStore(credentials: nil)
        let session = makeSessionStore(server, credentialStore: store)
        let model = MatchReportModel(session: session, matchId: 101)

        await model.load()

        XCTAssertEqual(server.requestCount("/api/match/detail"), 0)
    }
}

@MainActor
private func makeRestoredSession(_ server: MockHTTPServer) async -> SessionStore {
    server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
    let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
    let session = makeSessionStore(server, credentialStore: store)
    await session.restore()
    return session
}
