import SwiftUI
import XCTest
@testable import iOS

/// 好友五个页面的快照：有数据 / 空 / 失败，各渲染亮暗两张。
@MainActor
final class FriendSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private var server: MockHTTPServer!
    private let homepageTarget = FriendHomepageTarget(
        userId: 52,
        name: "球友 B",
        avatar: "",
        rankName: "星耀",
        vipLevel: 2,
        rankLevel: 3
    )

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
    }

    // MARK: - 好友列表

    func testFriendListDataSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18, nickname: "球友 A", rankLevel: 3, rankName: "星耀", vipLevel: 2),
            FriendFixtures.friend(id: 2, userId: 19, nickname: "球友 B"),
        ])))
        let model = FriendListModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-list-data", session: session, activity: activity, view: FriendListView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    func testFriendListEmptySnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([])))
        let model = FriendListModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-list-empty", session: session, activity: activity, view: FriendListView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    func testFriendListFailureSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/list", .raw("down", status: 503))
        let model = FriendListModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-list-failure", session: session, activity: activity, view: FriendListView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    // MARK: - 好友申请

    func testFriendRequestsDataSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52, nickname: "球友 A"),
            FriendFixtures.request(id: 10, fromUserId: 53, nickname: "球友 B", message: "一起打球"),
        ])))
        let model = FriendRequestsModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-requests-data", session: session, activity: activity, view: FriendRequestsView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    func testFriendRequestsEmptySnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([])))
        let model = FriendRequestsModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-requests-empty", session: session, activity: activity, view: FriendRequestsView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    func testFriendRequestsFailureSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/requests", .raw("down", status: 503))
        let model = FriendRequestsModel(session: session, activity: activity)
        await model.onAppear()

        try render("friend-requests-failure", session: session, activity: activity, view: FriendRequestsView(
            session: session,
            activity: activity,
            model: model
        ))
    }

    // MARK: - 添加好友

    func testAddFriendDataSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([
            FriendFixtures.searchItem(userId: 1, nickname: "球友 A", isFriend: true),
            FriendFixtures.searchItem(userId: 2, nickname: "球友 B", hasPendingRequest: true),
            FriendFixtures.searchItem(userId: 3, nickname: "球友 C"),
        ])))
        let model = AddFriendModel(session: session)
        model.keyword = "球友"
        await model.search()

        try render("add-friend-data", session: session, activity: activity, view: AddFriendView(
            session: session,
            model: model
        ))
    }

    func testAddFriendEmptySnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([])))
        let model = AddFriendModel(session: session)
        model.keyword = "不存在"
        await model.search()

        try render("add-friend-empty", session: session, activity: activity, view: AddFriendView(
            session: session,
            model: model
        ))
    }

    func testAddFriendFailureSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/search", .raw("down", status: 503))
        let model = AddFriendModel(session: session)
        model.keyword = "球友"
        await model.search()

        try render("add-friend-failure", session: session, activity: activity, view: AddFriendView(
            session: session,
            model: model
        ))
    }

    // MARK: - 好友主页

    func testFriendHomepageDataSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [
                SubPageFixtures.opponentItem(id: 18, name: "阿杰"),
                SubPageFixtures.opponentItem(id: 19, name: "小北"),
            ],
            totalOpponents: 2,
            totalWins: 9,
            total: 2
        )))
        let model = FriendHomepageModel(session: session, target: homepageTarget)
        await model.onAppear()

        try render("friend-homepage-data", session: session, activity: activity, view: FriendHomepageView(
            session: session,
            target: homepageTarget,
            model: model
        ))
    }

    func testFriendHomepageEmptySnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [],
            totalOpponents: 0,
            totalWins: 0,
            total: 0
        )))
        let model = FriendHomepageModel(session: session, target: homepageTarget)
        await model.onAppear()

        try render("friend-homepage-empty", session: session, activity: activity, view: FriendHomepageView(
            session: session,
            target: homepageTarget,
            model: model
        ))
    }

    func testFriendHomepageFailureSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/opponent/list", .raw("down", status: 503))
        let model = FriendHomepageModel(session: session, target: homepageTarget)
        await model.onAppear()

        try render("friend-homepage-failure", session: session, activity: activity, view: FriendHomepageView(
            session: session,
            target: homepageTarget,
            model: model
        ))
    }

    // MARK: - 黑名单

    func testBlacklistDataSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([
            FriendFixtures.blacklistItem(userId: 66, nickname: "球友 A"),
            FriendFixtures.blacklistItem(userId: 67, nickname: "球友 B"),
        ])))
        let model = BlacklistModel(session: session)
        await model.onAppear()

        try render("blacklist-data", session: session, activity: activity, view: BlacklistView(
            session: session,
            model: model
        ))
    }

    func testBlacklistEmptySnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([])))
        let model = BlacklistModel(session: session)
        await model.onAppear()

        try render("blacklist-empty", session: session, activity: activity, view: BlacklistView(
            session: session,
            model: model
        ))
    }

    func testBlacklistFailureSnapshot() async throws {
        let (session, activity) = try await makeSession()
        server.stub("GET", "/api/friend/blacklist/list", .raw("down", status: 503))
        let model = BlacklistModel(session: session)
        await model.onAppear()

        try render("blacklist-failure", session: session, activity: activity, view: BlacklistView(
            session: session,
            model: model
        ))
    }

    // MARK: - 辅助

    private func makeSession() async throws -> (SessionStore, ActivityStore) {
        server.stub("GET", "/api/user/bootstrap", .json(FriendFixtures.bootstrap(pendingFriend: 2)))
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
