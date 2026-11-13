import Foundation
import XCTest
@testable import iOS

@MainActor
final class PrivacySettingsModelTests: XCTestCase {
    func testLoadReadsBothPrivacySwitches() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": true,
            "friends_only_challenges": true,
        ]))
        let model = PrivacySettingsModel(session: session)

        await model.load()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertTrue(model.hideMatchRecord)
        XCTAssertTrue(model.friendsOnlyChallenges)
    }

    func testLoadFailureThenRetry() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .raw("service unavailable", status: 503))
        let model = PrivacySettingsModel(session: session)

        await model.load()
        XCTAssertEqual(model.phase, .failed)

        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": true,
        ]))
        await model.load(force: true)

        XCTAssertEqual(model.phase, .ready)
        XCTAssertFalse(model.hideMatchRecord)
        XCTAssertTrue(model.friendsOnlyChallenges)
    }

    func testHideMatchUpdatesOptimisticallyAndUsesServerValue() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/privacy") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json([
                "success": true,
                "message": "",
                "hide_match_record": true,
                "friends_only_challenges": false,
            ])
        }
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let toggle = Task { await model.toggleHideMatch() }
        await waitUntil { server.requestCount("/api/user/privacy") == 2 }
        XCTAssertTrue(model.hideMatchRecord, "隐藏战绩应先乐观更新")

        gate.signal()
        let message = await toggle.value

        XCTAssertEqual(message, "已隐藏战绩")
        XCTAssertTrue(model.hideMatchRecord)
    }

    func testHideMatchSuccessFollowsServerValue() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        server.stub("POST", "/api/user/privacy", .json([
            "success": true,
            "message": "",
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let message = await model.toggleHideMatch()

        XCTAssertEqual(message, "已公开战绩")
        XCTAssertFalse(model.hideMatchRecord)
    }

    func testHideMatchFailureRollsBack() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        server.stub("POST", "/api/user/privacy", .raw("service unavailable", status: 503))
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let message = await model.toggleHideMatch()

        XCTAssertEqual(message, "服务暂时不可用，请稍后重试")
        XCTAssertFalse(model.hideMatchRecord)
    }

    func testFriendsOnlyDoesNotUpdateOptimistically() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/privacy") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json([
                "success": true,
                "message": "",
                "hide_match_record": false,
                "friends_only_challenges": true,
            ])
        }
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let toggle = Task { await model.toggleFriendsOnlyChallenges() }
        await waitUntil { server.requestCount("/api/user/privacy") == 2 }
        XCTAssertFalse(model.friendsOnlyChallenges, "仅允许好友约球不应乐观更新")

        gate.signal()
        let message = await toggle.value

        XCTAssertEqual(message, "已开启仅好友约球")
        XCTAssertTrue(model.friendsOnlyChallenges)
    }

    func testRequestsOnlyContainTheModifiedField() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        server.stub("POST", "/api/user/privacy") { request in
            let body = request.bodyJSON
            if body["hide_match_record"] != nil {
                return .json([
                    "success": true,
                    "message": "",
                    "hide_match_record": true,
                    "friends_only_challenges": false,
                ])
            }
            return .json([
                "success": true,
                "message": "",
                "hide_match_record": true,
                "friends_only_challenges": true,
            ])
        }
        let model = PrivacySettingsModel(session: session)
        await model.load()

        _ = await model.toggleHideMatch()
        _ = await model.toggleFriendsOnlyChallenges()

        let posts = server.requests("/api/user/privacy").filter { $0.method == "POST" }
        XCTAssertEqual(posts.count, 2)
        XCTAssertEqual(posts[0].bodyJSON.keys.sorted(), ["hide_match_record"])
        XCTAssertEqual(posts[1].bodyJSON.keys.sorted(), ["friends_only_challenges"])
    }

    func testRepeatedTapWhileSavingSendsOnlyOneRequest() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/privacy") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json([
                "success": true,
                "message": "",
                "hide_match_record": true,
                "friends_only_challenges": false,
            ])
        }
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let first = Task { await model.toggleHideMatch() }
        await waitUntil { server.requestCount("/api/user/privacy") == 2 }
        let second = await model.toggleHideMatch()
        XCTAssertNil(second)

        gate.signal()
        _ = await first.value

        let posts = server.requests("/api/user/privacy").filter { $0.method == "POST" }
        XCTAssertEqual(posts.count, 1)
    }

    func testLateResultAfterLogoutDoesNotChangeState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": false,
            "friends_only_challenges": false,
        ]))
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/user/privacy") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json([
                "success": true,
                "message": "",
                "hide_match_record": true,
                "friends_only_challenges": true,
            ])
        }
        let model = PrivacySettingsModel(session: session)
        await model.load()

        let toggle = Task { await model.toggleHideMatch() }
        await waitUntil { server.requestCount("/api/user/privacy") == 2 }
        session.logout()
        await model.identityChanged()
        gate.signal()

        let message = await toggle.value

        XCTAssertNil(message)
        XCTAssertEqual(model.phase, .idle)
        XCTAssertFalse(model.hideMatchRecord)
        XCTAssertFalse(model.friendsOnlyChallenges)
    }
}
