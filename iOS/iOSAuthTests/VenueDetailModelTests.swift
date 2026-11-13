import XCTest
@testable import iOS

@MainActor
final class VenueDetailModelTests: XCTestCase {
    func testLoadsPublicDetailWithoutLogin() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(id: 7, name: "南山台球会"),
        ]))

        let model = VenueDetailModel(session: session, venueId: 7)
        await model.loadIfNeeded()

        XCTAssertEqual(model.venue?.name, "南山台球会")
        XCTAssertNil(server.requests("/api/venue/detail").first?.bearerToken)
        XCTAssertEqual(server.requests("/api/venue/detail").first?.query, ["venue_id": "7"])
    }

    func testCheckinRequiresLoginWithoutSendingRequest() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let model = VenueDetailModel(session: session, venueId: 7)

        let result = await model.checkin()

        XCTAssertEqual(result, .requiresLogin)
        XCTAssertEqual(server.requestCount("/api/venue/checkin"), 0)
        XCTAssertFalse(model.hasCheckedIn)
    }

    func testSuccessfulCheckinRefreshesDetail() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(id: 7, checkinCount: 3),
        ]))
        server.stub("POST", "/api/venue/checkin", .json(["success": true, "message": "签到成功！"]))
        let model = VenueDetailModel(session: session, venueId: 7)
        await model.loadIfNeeded()

        let result = await model.checkin()

        XCTAssertEqual(result, .success)
        XCTAssertTrue(model.hasCheckedIn)
        XCTAssertEqual(model.notice, "签到成功！")
        let request = server.requests("/api/venue/checkin").first
        XCTAssertEqual(request?.bodyJSON["venue_id"] as? Int, 7)
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(server.requestCount("/api/venue/detail"), 2, "成功后应刷新详情")
    }

    func testFailedCheckinKeepsStateAndShowsFailureCopy() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(id: 7),
        ]))
        server.stub("POST", "/api/venue/checkin", .raw("down", status: 503))
        let model = VenueDetailModel(session: session, venueId: 7)
        await model.loadIfNeeded()

        let result = await model.checkin()

        XCTAssertEqual(result, .failed)
        XCTAssertFalse(model.hasCheckedIn)
        XCTAssertEqual(model.notice, "签到失败")
        XCTAssertEqual(server.requestCount("/api/venue/detail"), 1)
    }

    func testSecondCheckinOnlyShowsAlreadyCheckedInNotice() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(id: 7),
        ]))
        server.stub("POST", "/api/venue/checkin", .json(["success": true, "message": "签到成功！"]))
        let model = VenueDetailModel(session: session, venueId: 7)
        await model.loadIfNeeded()

        _ = await model.checkin()
        let result = await model.checkin()

        XCTAssertEqual(result, .ignored)
        XCTAssertEqual(model.notice, "今日已签到")
        XCTAssertEqual(server.requestCount("/api/venue/checkin"), 1)
    }

    func testMissingVenueReportsNotFound() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json(["success": true, "venue": NSNull()]))

        let model = VenueDetailModel(session: session, venueId: 99)
        await model.loadIfNeeded()

        XCTAssertEqual(model.loader.failure?.category, .notFound)
        XCTAssertNil(model.venue)
    }
}
