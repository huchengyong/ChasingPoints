import SwiftUI
import XCTest
@testable import iOS

/// 约球三个页面的快照：发起、等待、列表，各渲染亮暗两张。
@MainActor
final class ChallengeSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private var server: MockHTTPServer!
    private let target = ChallengeComposeTarget(userId: 29, nickname: "球友A", avatar: "")

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
    }

    func testComposeSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        let model = ChallengeComposeModel(session: session, activity: activity, target: target)
        await model.onAppear()

        try render("challenge-compose", session: session, activity: activity, view: ChallengeComposeView(
            session: session,
            activity: activity,
            target: target,
            model: model
        ))
    }

    /// 未选对手时不能提交的初始状态。
    func testComposeWithoutOpponentSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        let model = ChallengeComposeModel(session: session, activity: activity)
        await model.onAppear()

        try render("challenge-compose-errors", session: session, activity: activity, view: ChallengeComposeView(
            session: session,
            activity: activity,
            target: nil,
            model: model
        ))
    }

    func testWaitingSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7,
            status: 0,
            fromUserId: 18,
            fromNickname: "球友B",
            toUserId: 1,
            toNickname: "测试用户"
        ))))
        let model = ChallengeWaitingModel(session: session, activity: activity, challengeId: 7, sleep: neverSleep)
        await model.load()

        try render("challenge-waiting", session: session, activity: activity, view: ChallengeWaitingView(
            session: session,
            activity: activity,
            challengeId: 7,
            model: model
        ))
    }

    func testWaitingAcceptedSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7,
            status: 1,
            fromUserId: 1,
            fromNickname: "测试用户",
            toUserId: 29,
            toNickname: "球友A",
            waitingUserId: 1
        ))))
        let model = ChallengeWaitingModel(session: session, activity: activity, challengeId: 7, sleep: neverSleep)
        await model.load()

        try render("challenge-waiting-accepted", session: session, activity: activity, view: ChallengeWaitingView(
            session: session,
            activity: activity,
            challengeId: 7,
            model: model
        ))
    }

    func testReceivedListSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([
            ChallengeFixtures.challenge(id: 1, fromUserId: 18, fromNickname: "球友B", toUserId: 1, toNickname: "测试用户"),
            ChallengeFixtures.challenge(
                id: 2,
                fromUserId: 19,
                fromNickname: "球友C",
                toUserId: 1,
                toNickname: "测试用户",
                gameType: 2,
                message: "晚上有空吗"
            ),
        ])))
        let model = ChallengesModel(session: session, activity: activity, initialTab: .received)
        await model.onAppear()

        try render("challenges-received", session: session, activity: activity, view: ChallengesView(
            session: session,
            activity: activity,
            initialTab: .received,
            model: model
        ))
    }

    func testSentListSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([
            ChallengeFixtures.challenge(id: 1, fromUserId: 1, fromNickname: "测试用户", toUserId: 29, toNickname: "球友A"),
            ChallengeFixtures.challenge(
                id: 2,
                status: 1,
                fromUserId: 1,
                fromNickname: "测试用户",
                toUserId: 30,
                toNickname: "球友B",
                waitingUserId: 30
            ),
            ChallengeFixtures.challenge(
                id: 3,
                status: 6,
                matchId: 88,
                fromUserId: 1,
                fromNickname: "测试用户",
                toUserId: 31,
                toNickname: "球友C"
            ),
        ])))
        let model = ChallengesModel(session: session, activity: activity, initialTab: .sent)
        await model.onAppear()

        try render("challenges-sent", session: session, activity: activity, view: ChallengesView(
            session: session,
            activity: activity,
            initialTab: .sent,
            model: model
        ))
    }

    func testHistoryListSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        server.stub("GET", "/api/challenge/history", .json(ChallengeFixtures.historyList([
            ChallengeFixtures.challenge(id: 11, status: 7, fromUserId: 1, toUserId: 29, toNickname: "球友A"),
            ChallengeFixtures.challenge(id: 10, status: 2, fromUserId: 18, fromNickname: "球友B", toUserId: 1, closeReason: "拒绝"),
            ChallengeFixtures.challenge(id: 9, status: 4, fromUserId: 1, toUserId: 30, toNickname: "球友C", closeReason: "取消"),
            ChallengeFixtures.challenge(id: 8, status: 3, fromUserId: 19, fromNickname: "球友D", toUserId: 1, closeReason: "过期"),
        ], nextBeforeId: 8)))
        let model = ChallengesModel(session: session, activity: activity, initialTab: .history)
        await model.onAppear()

        try render("challenges-history", session: session, activity: activity, view: ChallengesView(
            session: session,
            activity: activity,
            initialTab: .history,
            model: model
        ))
    }

    func testEmptyListSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        let model = ChallengesModel(session: session, activity: activity, initialTab: .received)
        await model.onAppear()

        try render("challenges-empty", session: session, activity: activity, view: ChallengesView(
            session: session,
            activity: activity,
            initialTab: .received,
            model: model
        ))
    }

    // MARK: - 辅助

    private func bootstrap() -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-08-25 12:00:00",
            "unread_count": 0,
            "pending_friend_request_count": 0,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    /// 快照不需要轮询：挂起一天，避免渲染期间反复请求。
    private func neverSleep(_: TimeInterval) async throws {
        try await Task.sleep(for: .seconds(3600))
    }

    private func makeSession() async throws -> (SessionStore, ActivityStore) {
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        let session = await makeRestoredAccountSession(server)
        return (session, ActivityStore(session: session))
    }

    private func render(
        _ name: String,
        session: SessionStore,
        activity: ActivityStore,
        view: some View
    ) throws {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    view
                }
                .environmentObject(session)
                .environmentObject(activity)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }
}
