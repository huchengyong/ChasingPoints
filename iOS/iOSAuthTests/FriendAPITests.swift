import Foundation
import XCTest
@testable import iOS

/// 好友接口门面：请求路径、参数与响应映射。全部走 MockHTTPServer。
@MainActor
final class FriendAPITests: XCTestCase {
    func testListUsesPagedQuery() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/friend/list", .json(FriendFixtures.friendList([
            FriendFixtures.friend(id: 1, userId: 18, nickname: "球友 A"),
        ], total: 1)))
        let api = FriendAPI(session: session)

        let response = try await api.list(page: 2)

        XCTAssertEqual(response.list?.map(\.friendUserId), [18])
        XCTAssertEqual(server.requests("/api/friend/list").first?.query, [
            "page": "2",
            "page_size": "20",
        ])
    }

    func testRequestsAndBlacklistUseDedicatedPaths() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/friend/requests", .json(FriendFixtures.requestList([
            FriendFixtures.request(id: 9, fromUserId: 52),
        ])))
        server.stub("GET", "/api/friend/blacklist/list", .json(FriendFixtures.blacklistList([
            FriendFixtures.blacklistItem(userId: 66),
        ])))
        let api = FriendAPI(session: session)

        let requests = try await api.requests(page: 1)
        let blacklist = try await api.blacklist(page: 1)

        XCTAssertEqual(requests.list?.first?.applicantUserId, 52)
        XCTAssertEqual(blacklist.list?.first?.friendUserId, 66)
        XCTAssertEqual(server.requests("/api/friend/requests").first?.query["page_size"], "20")
        XCTAssertEqual(server.requests("/api/friend/blacklist/list").first?.query["page"], "1")
    }

    func testSearchOnlySendsKeyword() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/friend/search", .json(FriendFixtures.searchList([
            FriendFixtures.searchItem(userId: 5, hasPendingRequest: true),
        ])))
        let api = FriendAPI(session: session)

        let response = try await api.search(keyword: "阿杰")

        XCTAssertEqual(response.list?.first?.hasPendingRequest, true)
        XCTAssertEqual(server.requests("/api/friend/search").first?.query, ["keyword": "阿杰"])
    }

    func testWriteRequestsSendContractBodies() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        for path in [
            "/api/friend/request",
            "/api/friend/accept",
            "/api/friend/reject",
            "/api/friend/delete",
            "/api/friend/blacklist",
            "/api/friend/unblacklist",
        ] {
            server.stub("POST", path, .json(["success": true, "message": "ok"]))
        }
        let api = FriendAPI(session: session)

        _ = try await api.sendRequest(toUserId: 8)
        _ = try await api.accept(requestId: 3)
        _ = try await api.reject(requestId: 4)
        _ = try await api.delete(friendUserId: 5)
        _ = try await api.addToBlacklist(friendUserId: 6)
        _ = try await api.removeFromBlacklist(friendUserId: 7)

        XCTAssertEqual(server.requests("/api/friend/request").first?.bodyJSON["to_user_id"] as? Int, 8)
        XCTAssertEqual(server.requests("/api/friend/request").first?.bodyJSON["message"] as? String, "")
        XCTAssertEqual(server.requests("/api/friend/accept").first?.bodyJSON["request_id"] as? Int, 3)
        XCTAssertEqual(server.requests("/api/friend/reject").first?.bodyJSON["request_id"] as? Int, 4)
        XCTAssertEqual(server.requests("/api/friend/delete").first?.bodyJSON["friend_user_id"] as? Int, 5)
        XCTAssertEqual(server.requests("/api/friend/blacklist").first?.bodyJSON["friend_user_id"] as? Int, 6)
        XCTAssertEqual(server.requests("/api/friend/unblacklist").first?.bodyJSON["friend_user_id"] as? Int, 7)
    }

    func testBusinessFailureSurfacesServerMessage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/friend/request", .json(["success": false, "message": "对方已是你的好友"]))
        let api = FriendAPI(session: session)

        do {
            _ = try await api.sendRequest(toUserId: 8)
            XCTFail("业务失败应抛错")
        } catch {
            XCTAssertEqual(error.localizedDescription, "对方已是你的好友")
        }
    }

    func testHomepageReusesTargetModeOpponentList() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [
                SubPageFixtures.opponentItem(id: 18, lastMatchAt: "2026-10-08 18:30:00"),
                SubPageFixtures.opponentItem(id: 19),
            ],
            totalOpponents: 2,
            totalWins: 9,
            total: 2
        )))
        let api = FriendAPI(session: session)

        let data = try await api.homepage(userId: 52)

        XCTAssertEqual(data.totalMatches, 9)
        XCTAssertEqual(data.totalWins, 9)
        XCTAssertEqual(data.lastMatchAt, "2026-10-08 18:30:00")
        XCTAssertFalse(data.hidden)
        XCTAssertEqual(data.opponents.map(\.id), [18, 19])
        XCTAssertEqual(server.requests("/api/opponent/list").first?.query, [
            "page": "1",
            "page_size": "100",
            "keyword": "",
            "target_user_id": "52",
        ])
    }

    func testHomepageMapsHiddenResponseToEmptyList() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        var payload = SubPageFixtures.opponentList(items: [SubPageFixtures.opponentItem(id: 18)], total: 1)
        payload["hidden"] = true
        payload["message"] = "对方已隐藏战绩"
        server.stub("GET", "/api/opponent/list", .json(payload))
        let api = FriendAPI(session: session)

        let data = try await api.homepage(userId: 52)

        XCTAssertTrue(data.hidden)
        XCTAssertTrue(data.opponents.isEmpty, "隐藏战绩不展示列表")
    }

    func testHomepageRejectsMissingUserIdWithoutRequest() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let api = FriendAPI(session: session)

        do {
            _ = try await api.homepage(userId: 0)
            XCTFail("缺少目标用户应抛错")
        } catch {
            XCTAssertEqual(error.localizedDescription, "好友信息异常，请返回好友列表后重试。")
        }
        XCTAssertEqual(server.requestCount("/api/opponent/list"), 0)
    }
}
