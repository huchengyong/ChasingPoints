import SwiftUI
import XCTest
@testable import iOS

/// 消息页与消息 Tab 角标的渲染快照：先读完数据再注入页面，断言能正常渲染。
@MainActor
final class MessageCenterSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    private func bootstrap(unread: Int = 0, pendingFriend: Int = 0, received: Int = 0) -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": received,
            "server_time": "2026-01-01 12:00:00",
            "unread_count": unread,
            "pending_friend_request_count": pendingFriend,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private func note(_ id: Int64, type: String = "system", title: String, isRead: Bool = false) -> [String: Any] {
        [
            "id": id,
            "type": type,
            "title": title,
            "content": "这是一条消息的正文内容",
            "data": "",
            "is_read": isRead,
            "created_at": "2026-01-01 12:00:00",
        ]
    }

    private func listJSON(_ list: [[String: Any]]) -> [String: Any] {
        ["success": true, "total": list.count, "unread_count": list.filter { ($0["is_read"] as? Bool) != true }.count, "list": list]
    }

    private func makeSignedIn(_ server: MockHTTPServer) async -> SessionStore {
        await makeRestoredAccountSession(server)
    }

    func testMessageCenterSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 1, pendingFriend: 2, received: 1)))
        server.stub("GET", "/api/notification/list", .json(listJSON([
            note(1, type: "friend_request", title: "收到好友申请"),
            note(2, type: "match_result", title: "对局结果已生成", isRead: true),
        ])))
        let session = await makeSignedIn(server)
        let activity = ActivityStore(session: session)
        let model = MessageCenterModel(session: session, activity: activity)
        await model.onAppear()
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("messages", colorScheme: scheme, size: size) {
                NavigationStack {
                    MessageCenterView(session: session, activity: activity, model: model)
                }
                .environmentObject(session)
                .environmentObject(activity)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMessageCenterEmptySnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        server.stub("GET", "/api/notification/list", .json(listJSON([])))
        let session = await makeSignedIn(server)
        let activity = ActivityStore(session: session)
        let model = MessageCenterModel(session: session, activity: activity)
        await model.onAppear()
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("messages-empty", colorScheme: scheme, size: size) {
                NavigationStack {
                    MessageCenterView(session: session, activity: activity, model: model)
                }
                .environmentObject(session)
                .environmentObject(activity)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMessageCenterGuestSnapshot() throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("messages-guest", colorScheme: scheme, size: size) {
                NavigationStack {
                    MessageCenterView(session: session, activity: activity)
                }
                .environmentObject(session)
                .environmentObject(activity)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMainTabsBadgeCountSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(unread: 3)))
        let session = await makeSignedIn(server)
        let activity = ActivityStore(session: session)
        await activity.refresh()
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("main-tabs-badge-count", colorScheme: scheme, size: size) {
                MainTabView(session: session)
                    .environmentObject(session)
                    .environmentObject(activity)
                    .environmentObject(navigator)
                    .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMainTabsBadgeDotSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap(pendingFriend: 1)))
        let session = await makeSignedIn(server)
        let activity = ActivityStore(session: session)
        await activity.refresh()
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("main-tabs-badge-dot", colorScheme: scheme, size: size) {
                MainTabView(session: session)
                    .environmentObject(session)
                    .environmentObject(activity)
                    .environmentObject(navigator)
                    .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }
}
