import XCTest
@testable import iOS

/// 约球接口门面：路径、查询与请求体。
@MainActor
final class ChallengeAPITests: XCTestCase {
    private func makeAPI(_ server: MockHTTPServer) async -> (ChallengeAPI, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        return (ChallengeAPI(session: session), session)
    }

    func testReadsUseExpectedPathsAndQueries() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary()))
        server.stub("GET", "/api/challenge/pending", .json(ChallengeFixtures.pendingList([])))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(nil)))
        server.stub("GET", "/api/challenge/history", .json(ChallengeFixtures.historyList([])))
        let (api, _) = await makeAPI(server)

        _ = try await api.summary()
        _ = try await api.pending()
        _ = try await api.detail(id: 9)
        _ = try await api.history()

        XCTAssertEqual(server.requests("/api/challenge/detail").first?.query["id"], "9")
        XCTAssertEqual(server.requests("/api/challenge/history").first?.query["page_size"], "20")
        XCTAssertNil(server.requests("/api/challenge/history").first?.query["before_id"])
    }

    func testHistoryPassesBeforeIdCursor() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/history", .json(ChallengeFixtures.historyList([], nextBeforeId: 12)))
        let (api, _) = await makeAPI(server)

        let response = try await api.history(beforeId: 30)

        XCTAssertEqual(server.requests("/api/challenge/history").first?.query["before_id"], "30")
        XCTAssertEqual(response.nextBeforeId, 12)
    }

    func testSendEncodesSnakeCaseBody() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/send", .json(ChallengeFixtures.sendResponse(challengeId: 77)))
        let (api, _) = await makeAPI(server)

        let response = try await api.send(SendChallengeBody(
            toUserId: 29,
            gameType: 2,
            matchMode: "practice",
            message: "今晚来一局",
            scheduledDate: "2026-10-12",
            startHour: 20,
            endHour: 22
        ))

        let body = try XCTUnwrap(server.requests("/api/challenge/send").first?.bodyJSON)
        XCTAssertEqual(body["to_user_id"] as? Int, 29)
        XCTAssertEqual(body["game_type"] as? Int, 2)
        XCTAssertEqual(body["match_mode"] as? String, "practice")
        XCTAssertEqual(body["message"] as? String, "今晚来一局")
        XCTAssertEqual(body["scheduled_date"] as? String, "2026-10-12")
        XCTAssertEqual(body["start_hour"] as? Int, 20)
        XCTAssertEqual(body["end_hour"] as? Int, 22)
        XCTAssertEqual(response.challengeId, 77)
    }

    func testAcceptAndEnterOnlySendConfirmationWhenRequested() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 7, matchId: 88)))
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.enterResponse(action: "waiting")))
        let (api, _) = await makeAPI(server)

        _ = try await api.accept(challengeId: 7)
        _ = try await api.accept(challengeId: 7, confirmCloseOwnPending: true)
        _ = try await api.enter(challengeId: 7)
        _ = try await api.enter(challengeId: 7, confirmEarly: true)

        let accepts = server.requests("/api/challenge/accept")
        XCTAssertEqual(accepts.count, 2)
        XCTAssertNil(accepts[0].bodyJSON["confirm_close_own_pending"])
        XCTAssertEqual(accepts[1].bodyJSON["confirm_close_own_pending"] as? Bool, true)

        let enters = server.requests("/api/challenge/enter")
        XCTAssertEqual(enters.count, 2)
        XCTAssertNil(enters[0].bodyJSON["confirm_early"])
        XCTAssertEqual(enters[1].bodyJSON["confirm_early"] as? Bool, true)
    }

    func testRejectCancelAbandonLeavePostChallengeId() async throws {
        let server = MockHTTPServer()
        for path in ["reject", "cancel", "abandon", "leave"] {
            server.stub("POST", "/api/challenge/\(path)", .json(ChallengeFixtures.actionResponse()))
        }
        let (api, _) = await makeAPI(server)

        _ = try await api.reject(challengeId: 11)
        _ = try await api.cancel(challengeId: 11)
        _ = try await api.abandon(challengeId: 11)
        _ = try await api.leave(challengeId: 11)

        for path in ["reject", "cancel", "abandon", "leave"] {
            XCTAssertEqual(server.requests("/api/challenge/\(path)").first?.bodyJSON["challenge_id"] as? Int, 11)
        }
    }

    func testBusinessFailureSurfacesMessageOnly() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.businessFailure(
            "你们发起的约球类型不同，不能直接匹配",
            typeConflict: true
        )))
        let (api, _) = await makeAPI(server)

        do {
            _ = try await api.accept(challengeId: 7)
            XCTFail("业务失败应当抛错")
        } catch {
            let apiError = try XCTUnwrap(error as? APIError)
            XCTAssertEqual(apiError.message, "你们发起的约球类型不同，不能直接匹配")
            XCTAssertEqual(apiError.decodeBody(ChallengeFailureDetails.self)?.typeConflict, true)
        }
    }

    func testBusinessFailureBodyCarriesStructuredFields() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/send", .json(ChallengeFixtures.businessFailure(
            "你发出的约球还在等待回应，请先处理",
            pendingChallengeId: 55
        )))
        let (api, _) = await makeAPI(server)

        do {
            _ = try await api.send(SendChallengeBody(
                toUserId: 29,
                gameType: 3,
                matchMode: "ranked",
                message: "",
                scheduledDate: "2026-10-12",
                startHour: 20,
                endHour: 22
            ))
            XCTFail("业务失败应当抛错")
        } catch {
            let details = try XCTUnwrap((error as? APIError)?.decodeBody(ChallengeFailureDetails.self))
            XCTAssertEqual(details.pendingChallengeId, 55)
        }
    }
}
