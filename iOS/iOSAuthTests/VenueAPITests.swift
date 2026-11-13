import Foundation
import XCTest
@testable import iOS

@MainActor
final class VenueAPITests: XCTestCase {
    func testListSendsPageAndCoordinatesWithoutAuthorizationForGuests() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([VenueFixtures.venueJSON()])))

        let page = try await VenueAPI(session: session).list(
            page: 2,
            coordinate: Coordinate(latitude: 22.540796, longitude: 113.917176)
        )

        XCTAssertEqual(page.items.map(\.id), [1])
        XCTAssertEqual(page.total, 1)
        let request = try XCTUnwrap(server.requests("/api/venue/list").first)
        XCTAssertEqual(request.query, [
            "page": "2",
            "page_size": "10",
            "latitude": "22.540796",
            "longitude": "113.917176",
        ])
        XCTAssertNil(request.bearerToken)
    }

    func testListOmitsCoordinatesWhenLocationIsUnavailable() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([])))

        _ = try await VenueAPI(session: session).list(page: 1, coordinate: nil)

        let request = try XCTUnwrap(server.requests("/api/venue/list").first)
        XCTAssertEqual(request.query.keys.sorted(), ["page", "page_size"])
    }

    func testNearbyUsesFixedRadiusAndLimit() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/nearby", .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(distance: 860),
        ])))

        let page = try await VenueAPI(session: session).nearby(Coordinate(latitude: 22.54, longitude: 113.91))

        XCTAssertEqual(page.items.first?.distance, 860)
        let request = try XCTUnwrap(server.requests("/api/venue/nearby").first)
        XCTAssertEqual(request.query, [
            "latitude": "22.54",
            "longitude": "113.91",
            "radius": "10000",
            "limit": "20",
        ])
        XCTAssertNil(request.bearerToken)
    }

    func testDetailSendsVenueIdAndReadsPublicVenue() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(id: 7, name: "南山台球会"),
        ]))

        let venue = try await VenueAPI(session: session).detail(id: 7)

        XCTAssertEqual(venue.id, 7)
        XCTAssertEqual(venue.name, "南山台球会")
        XCTAssertEqual(server.requests("/api/venue/detail").first?.query, ["venue_id": "7"])
    }

    func testDetailRejectsMissingIdWithoutRequest() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)

        do {
            _ = try await VenueAPI(session: session).detail(id: 0)
            XCTFail("缺少 venue_id 不应请求")
        } catch {
            assertAPIError(error, kind: .business, statusCode: 404)
        }
        XCTAssertEqual(server.requestCount("/api/venue/detail"), 0)
    }

    func testDetailReportsMissingVenueAsNotFound() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json(["success": true, "venue": NSNull()]))

        do {
            _ = try await VenueAPI(session: session).detail(id: 99)
            XCTFail("venue 为空应当抛错")
        } catch {
            XCTAssertEqual(LoadFailure(error).category, .notFound)
        }
    }

    func testAreasSendParentId() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/areas", .json(VenueFixtures.areaJSON([
            (id: 19, parentId: 0, name: "广东省"),
        ])))

        let areas = try await VenueAPI(session: session).areas(parentId: 0)

        XCTAssertEqual(areas, [VenueAreaOption(areaId: 19, parentId: 0, name: "广东省")])
        XCTAssertEqual(server.requests("/api/venue/areas").first?.query, ["parent_id": "0"])
    }

    func testCreatePostsOnlyBasicFieldsWithAuthorization() async throws {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("POST", "/api/venue/create", .json([
            "success": true,
            "venue_id": 42,
            "message": "提交成功，等待审核",
        ]))
        let payload = VenueSubmitPayload(
            name: "星轨台球",
            city: "深圳市",
            district: "南山区",
            address: "科苑路 8 号",
            latitude: 22.54,
            longitude: 113.91
        )

        let result = try await VenueAPI(session: session).create(payload)

        XCTAssertEqual(result, VenueCreateResult(venueId: 42, message: "提交成功，等待审核"))
        let request = try XCTUnwrap(server.requests("/api/venue/create").first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request.bodyJSON.keys.sorted(), ["address", "city", "district", "latitude", "longitude", "name"])
        XCTAssertEqual(request.bodyJSON["name"] as? String, "星轨台球")
        XCTAssertEqual(request.bodyJSON["district"] as? String, "南山区")
    }

    func testCreateOmitsCoordinatesWhenLocationWasNotPicked() async throws {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("POST", "/api/venue/create", .json(["success": true, "venue_id": 42, "message": ""]))
        let payload = VenueSubmitPayload(
            name: "星轨台球",
            city: "深圳市",
            district: "",
            address: "科苑路 8 号",
            latitude: nil,
            longitude: nil
        )

        _ = try await VenueAPI(session: session).create(payload)

        let request = try XCTUnwrap(server.requests("/api/venue/create").first)
        XCTAssertEqual(request.bodyJSON.keys.sorted(), ["address", "city", "district", "name"])
    }

    func testCheckinPostsVenueIdWithAuthorization() async throws {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("POST", "/api/venue/checkin", .json(["success": true, "message": "签到成功！"]))

        let message = try await VenueAPI(session: session).checkin(venueId: 7)

        XCTAssertEqual(message, "签到成功！")
        let request = try XCTUnwrap(server.requests("/api/venue/checkin").first)
        XCTAssertEqual(request.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request.bodyJSON["venue_id"] as? Int, 7)
    }
}
