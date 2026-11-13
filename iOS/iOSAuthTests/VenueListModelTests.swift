import XCTest
@testable import iOS

@MainActor
final class VenueListModelTests: XCTestCase {
    func testLocatedListUsesNearbyEndpoint() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .success(Coordinate(latitude: 22.54, longitude: 113.91)))
        server.stub("GET", "/api/venue/nearby", .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(id: 1, distance: 860),
            VenueFixtures.venueJSON(id: 2, distance: 2400),
        ])))

        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))
        await model.loadIfNeeded()

        XCTAssertEqual(model.mode, .nearby)
        XCTAssertEqual(model.locationText, "已获取位置")
        XCTAssertEqual(model.items.map(\.id), [1, 2])
        XCTAssertFalse(model.hasMore, "附近接口不再分页")
        guard case .loaded = model.phase else {
            return XCTFail("应进入已加载态：\(model.phase)")
        }
        XCTAssertEqual(server.requests("/api/venue/nearby").first?.query, [
            "latitude": "22.54",
            "longitude": "113.91",
            "radius": "10000",
            "limit": "20",
        ])
        XCTAssertEqual(server.requestCount("/api/venue/list"), 0)
    }

    func testDeniedLocationFallsBackToListAndNotifies() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .failure(.denied))
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([VenueFixtures.venueJSON()])))

        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))
        await model.loadIfNeeded()

        XCTAssertEqual(model.mode, .list)
        XCTAssertEqual(model.locationText, "未授权定位")
        XCTAssertEqual(model.notice, "定位权限未开启，将仅展示全部球馆")
        XCTAssertEqual(model.items.map(\.id), [1])
        XCTAssertEqual(server.requestCount("/api/venue/nearby"), 0)
        XCTAssertEqual(server.requestCount("/api/venue/list"), 1)
    }

    func testUnavailableLocationFallsBackToListWithFailureCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .failure(.unavailable))
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([])))

        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))
        await model.loadIfNeeded()

        XCTAssertEqual(model.mode, .list)
        XCTAssertEqual(model.locationText, "定位失败")
        XCTAssertEqual(model.notice, "定位失败，将仅展示全部球馆")
        XCTAssertEqual(model.phase, .empty)
    }

    func testListModePaginatesUntilTotal() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .failure(.denied))
        let firstPage = (1...10).map { VenueFixtures.venueJSON(id: Int64($0)) }
        let secondPage = (11...15).map { VenueFixtures.venueJSON(id: Int64($0)) }
        server.stub("GET", "/api/venue/list") { request in
            request.query["page"] == "1"
                ? .json(VenueFixtures.listResponse(firstPage, total: 15))
                : .json(VenueFixtures.listResponse(secondPage, total: 15))
        }
        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))

        await model.switchMode(.list)
        XCTAssertEqual(model.items.count, 10)
        XCTAssertTrue(model.hasMore)

        await model.loadMore()
        XCTAssertEqual(model.items.count, 15)
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests("/api/venue/list").map { $0.query["page"] }, ["1", "2"])
        XCTAssertEqual(server.requests("/api/venue/list").map { $0.query["page_size"] }, ["10", "10"])
    }

    func testEmptyNearbyShowsNearbyEmptyCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .success(Coordinate(latitude: 22.54, longitude: 113.91)))
        server.stub("GET", "/api/venue/nearby", .json(VenueFixtures.listResponse([])))

        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))
        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertEqual(model.emptyTitle, "附近暂无球馆")
    }

    func testEmptyListModeShowsListEmptyCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .failure(.denied))
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([])))

        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))
        await model.switchMode(.list)

        XCTAssertEqual(model.phase, .empty)
        XCTAssertEqual(model.emptyTitle, "暂无球馆数据")
    }

    func testFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let provider = StubLocationProvider(result: .success(Coordinate(latitude: 22.54, longitude: 113.91)))
        let attempts = LockedCounter()
        server.stub("GET", "/api/venue/nearby") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(VenueFixtures.listResponse([VenueFixtures.venueJSON(id: 3)]))
        }
        let model = VenueListModel(session: session, location: provider, api: VenueAPI(session: session))

        await model.loadIfNeeded()
        guard case .failed(let failure) = model.phase else {
            return XCTFail("首次应失败：\(model.phase)")
        }
        XCTAssertEqual(failure.category, .server)

        await model.retry()
        XCTAssertEqual(model.items.map(\.id), [3])
        guard case .loaded = model.phase else {
            return XCTFail("重试后应加载成功：\(model.phase)")
        }
        XCTAssertEqual(server.requestCount("/api/venue/nearby"), 2)
    }
}
