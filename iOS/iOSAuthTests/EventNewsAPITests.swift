import Foundation
import XCTest
@testable import iOS

/// 赛讯接口：迁移 `event-news-response.test.mjs` 的 2 条，并覆盖列表/详情的参数与访客行为。
@MainActor
final class EventNewsAPITests: XCTestCase {
    // MARK: - event-news-response

    func testPickEventNewsViewPayloadOnlyAcceptsCanonicalEventNewsField() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub(
            "GET",
            "/api/event-news/view",
            .json(EventNewsFixtures.viewResponse(
                eventNews: EventNewsFixtures.eventNewsJSON(id: 18, title: "赛事详情"),
                tournament: EventNewsFixtures.tournamentJSON(id: 3, name: "赛事实体"),
                matches: [EventNewsFixtures.matchJSON(id: 88)]
            ))
        )

        let view = try await EventNewsAPI(session: session).view(eventId: 18)

        XCTAssertEqual(view.eventNews?.id, 18)
        XCTAssertEqual(view.eventNews?.title, "赛事详情")
        XCTAssertEqual(view.tournament?.id, 3)
        XCTAssertEqual(view.tournament?.name, "赛事实体")
        XCTAssertEqual(view.matches.map(\.id), [88])
        XCTAssertEqual(view.pickedPayload, view)
    }

    func testPickEventNewsViewPayloadIgnoresMissingEventNewsPayload() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub(
            "GET",
            "/api/event-news/view",
            .json([
                "success": true,
                "event_news": NSNull(),
                "event": ["id": 99, "title": "旧字段"],
                "matches": [],
            ])
        )

        let view = try await EventNewsAPI(session: session).view(eventId: 18)

        XCTAssertNil(view.eventNews)
        XCTAssertNil(view.pickedPayload)
        XCTAssertNil(view.tournament)
        XCTAssertTrue(view.matches.isEmpty)
    }

    // MARK: - 列表参数

    func testListSendsOnlyPagingParamsWhenFilterIsEmpty() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/list", .json(EventNewsFixtures.listResponse()))

        _ = try await EventNewsAPI(session: session).list(filter: EventNewsFilter(), page: 1, pageSize: 10)

        XCTAssertEqual(
            server.requests("/api/event-news/list").first?.query,
            ["page": "1", "page_size": "10"]
        )
    }

    func testListSendsDateRangeInsteadOfYearAndOptionalFilters() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/list", .json(EventNewsFixtures.listResponse(total: 0, list: [])))

        _ = try await EventNewsAPI(session: session).list(
            filter: EventNewsFilter(
                year: 2026,
                from: "2026-01-01",
                to: "2026-04-30",
                gameType: 1,
                status: 2,
                city: " 深圳 "
            ),
            page: 2,
            pageSize: 20
        )

        XCTAssertEqual(
            server.requests("/api/event-news/list").first?.query,
            [
                "page": "2",
                "page_size": "20",
                "from": "2026-01-01",
                "to": "2026-04-30",
                "game_type": "1",
                "status": "2",
                "city": "深圳",
            ]
        )
    }

    func testListOmitsNegativeStatusAndZeroGameType() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/list", .json(EventNewsFixtures.listResponse()))

        _ = try await EventNewsAPI(session: session).list(
            filter: EventNewsFilter(year: 2025, gameType: 0, status: -1),
            page: 1,
            pageSize: 10
        )

        XCTAssertEqual(
            server.requests("/api/event-news/list").first?.query,
            ["page": "1", "page_size": "10", "year": "2025"]
        )
    }

    func testListReturnsItemsAndTotal() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub(
            "GET",
            "/api/event-news/list",
            .json(EventNewsFixtures.listResponse(
                total: 2,
                list: [EventNewsFixtures.eventNewsJSON(id: 1), EventNewsFixtures.eventNewsJSON(id: 2)]
            ))
        )

        let page = try await EventNewsAPI(session: session).list(
            filter: EventNewsFilter(year: 2026),
            page: 1,
            pageSize: 10
        )

        XCTAssertEqual(page.total, 2)
        XCTAssertEqual(page.items.map(\.id), [1, 2])
        XCTAssertEqual(page.items.first?.id, 1)
        XCTAssertNil(page.items.first?.startTime)
    }

    func testListByGuestSendsNoAuthorizationHeader() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server, signedIn: false)
        server.stub("GET", "/api/event-news/list", .json(EventNewsFixtures.listResponse()))

        _ = try await EventNewsAPI(session: session).list(filter: EventNewsFilter(), page: 1, pageSize: 10)

        XCTAssertNil(server.requests("/api/event-news/list").first?.bearerToken)
    }

    func testListBySignedInUserReusesSessionToken() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/list", .json(EventNewsFixtures.listResponse()))

        _ = try await EventNewsAPI(session: session).list(filter: EventNewsFilter(), page: 1, pageSize: 10)

        XCTAssertEqual(server.requests("/api/event-news/list").first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testListThrowsOnBusinessFailure() async {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/list", .json(["success": false, "message": "读取失败"]))

        do {
            _ = try await EventNewsAPI(session: session).list(filter: EventNewsFilter(), page: 1, pageSize: 10)
            XCTFail("success=false 应当抛错")
        } catch {
            assertAPIError(error, kind: .business, message: "读取失败")
        }
    }

    // MARK: - 详情

    func testViewSendsEventIdQueryParam() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/view", .json(EventNewsFixtures.viewResponse()))

        _ = try await EventNewsAPI(session: session).view(eventId: 18)

        XCTAssertEqual(server.requests("/api/event-news/view").first?.query, ["event_id": "18"])
    }

    func testViewAcceptsEmptyMatchesArray() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub("GET", "/api/event-news/view", .json(EventNewsFixtures.viewResponse(matches: [])))

        let view = try await EventNewsAPI(session: session).view(eventId: 18)

        XCTAssertTrue(view.matches.isEmpty)
        XCTAssertNotNil(view.eventNews)
        XCTAssertNotNil(view.pickedPayload)
    }

    func testViewUnwrapsNestedDataPayload() async throws {
        let server = MockHTTPServer()
        let session = makeSession(server)
        server.stub(
            "GET",
            "/api/event-news/view",
            .json(["success": true, "data": EventNewsFixtures.viewResponse(matches: [])])
        )

        let view = try await EventNewsAPI(session: session).view(eventId: 18)

        XCTAssertEqual(view.eventNews?.id, 18)
        XCTAssertTrue(view.matches.isEmpty)
    }

    private func makeSession(_ server: MockHTTPServer, signedIn: Bool = true) -> SessionStore {
        let credentials = signedIn ? AuthFixtures.credentials() : nil
        return makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: credentials)
        )
    }
}
