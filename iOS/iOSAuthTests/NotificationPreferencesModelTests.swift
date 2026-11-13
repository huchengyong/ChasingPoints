import Foundation
import XCTest
@testable import iOS

@MainActor
final class NotificationPreferencesModelTests: XCTestCase {
    func testLoadReadsPreferences() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/preferences", .json(preferences(
            matchResult: false,
            friendRequest: true,
            challenge: false,
            tournament: true,
            follow: false
        )))
        let model = NotificationPreferencesModel(session: session)

        await model.load()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertFalse(model.preferences.matchResult)
        XCTAssertTrue(model.preferences.friendRequest)
        XCTAssertFalse(model.preferences.challenge)
        XCTAssertTrue(model.preferences.tournament)
        XCTAssertFalse(model.preferences.follow)
    }

    func testToggleSubmitsAllFiveFieldsAndUsesServerResponse() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/preferences", .json(preferences()))
        server.stub("POST", "/api/notification/preferences", .json(preferences(
            matchResult: true,
            friendRequest: false,
            challenge: true,
            tournament: true,
            follow: true
        )))
        let model = NotificationPreferencesModel(session: session)
        await model.load()

        let message = await model.toggle(.friendRequest, value: false)

        XCTAssertNil(message)
        XCTAssertFalse(model.preferences.friendRequest)
        let request = server.requests("/api/notification/preferences").first { $0.method == "POST" }
        let body = request?.bodyJSON ?? [:]
        XCTAssertEqual(
            body.keys.sorted(),
            [
                "challenge_enabled",
                "follow_enabled",
                "friend_request_enabled",
                "match_result_enabled",
                "tournament_enabled",
            ]
        )
        XCTAssertEqual(body["friend_request_enabled"] as? Bool, false)
        XCTAssertEqual(body["match_result_enabled"] as? Bool, true)
    }

    func testFailureRollsBackToPreviousPreferences() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/preferences", .json(preferences()))
        server.stub("POST", "/api/notification/preferences", .raw("service unavailable", status: 503))
        let model = NotificationPreferencesModel(session: session)
        await model.load()

        let message = await model.toggle(.challenge, value: false)

        XCTAssertEqual(message, "保存失败，请重试")
        XCTAssertTrue(model.preferences.challenge)
    }

    func testSavingDisablesRepeatedToggles() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/preferences", .json(preferences()))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/notification/preferences") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json([
                "success": true,
                "match_result_enabled": true,
                "friend_request_enabled": true,
                "challenge_enabled": true,
                "tournament_enabled": false,
                "follow_enabled": true,
            ])
        }
        let model = NotificationPreferencesModel(session: session)
        await model.load()

        let first = Task { await model.toggle(.tournament, value: false) }
        await waitUntil { server.requestCount("/api/notification/preferences") == 2 }
        XCTAssertTrue(model.isSaving)
        let second = await model.toggle(.matchResult, value: false)
        XCTAssertNil(second)

        gate.signal()
        _ = await first.value

        let posts = server.requests("/api/notification/preferences").filter { $0.method == "POST" }
        XCTAssertEqual(posts.count, 1)
    }

    private func preferences(
        matchResult: Bool = true,
        friendRequest: Bool = true,
        challenge: Bool = true,
        tournament: Bool = true,
        follow: Bool = true
    ) -> [String: Any] {
        [
            "success": true,
            "match_result_enabled": matchResult,
            "friend_request_enabled": friendRequest,
            "challenge_enabled": challenge,
            "tournament_enabled": tournament,
            "follow_enabled": follow,
        ]
    }
}
