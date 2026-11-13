import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class VenueSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testVenueListSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/nearby", .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(
                id: 1,
                name: "星轨台球俱乐部",
                address: "科苑路 8 号",
                tableCount: 18,
                priceRange: "30-60 元/小时",
                distance: 860,
                checkinCount: 12
            ),
            VenueFixtures.venueJSON(
                id: 2,
                name: "南山台球会",
                address: "海德三道 100 号",
                tableCount: 12,
                priceRange: "40-80 元/小时",
                distance: 2400,
                checkinCount: 3
            ),
            VenueFixtures.venueJSON(
                id: 3,
                name: "后海桌球馆",
                address: "后海滨路 6 号",
                tableCount: 0,
                priceRange: "",
                distance: 5300,
                checkinCount: 0
            ),
        ])))
        let model = VenueListModel(
            session: session,
            location: StubLocationProvider(result: .success(Coordinate(latitude: 22.54, longitude: 113.91))),
            api: VenueAPI(session: session)
        )
        await model.loadIfNeeded()
        try await render("venue-list", session: session, view: VenueListView(session: session, model: model))
    }

    func testVenueListLocationDeniedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/list", .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(id: 1, name: "星轨台球俱乐部", checkinCount: 12),
            VenueFixtures.venueJSON(id: 2, name: "南山台球会", checkinCount: 3),
        ])))
        let model = VenueListModel(
            session: session,
            location: StubLocationProvider(result: .failure(.denied)),
            api: VenueAPI(session: session)
        )
        await model.loadIfNeeded()
        XCTAssertEqual(model.mode, .list)
        try await render("venue-list-location-denied", session: session, view: VenueListView(session: session, model: model))
    }

    func testVenueListEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/nearby", .json(VenueFixtures.listResponse([])))
        let model = VenueListModel(
            session: session,
            location: StubLocationProvider(result: .success(Coordinate(latitude: 22.54, longitude: 113.91))),
            api: VenueAPI(session: session)
        )
        await model.loadIfNeeded()
        try await render("venue-list-empty", session: session, view: VenueListView(session: session, model: model))
    }

    func testVenueDetailSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/detail", .json([
            "success": true,
            "venue": VenueFixtures.venueJSON(
                id: 7,
                name: "星轨台球俱乐部",
                address: "科苑路 8 号",
                businessHours: "10:00-23:00",
                tableCount: 18,
                priceRange: "30-60 元/小时",
                checkinCount: 12
            ),
        ]))
        let model = VenueDetailModel(session: session, venueId: 7)
        await model.loadIfNeeded()
        try await render(
            "venue-detail",
            session: session,
            view: VenueDetailView(session: session, venueId: 7, model: model, showsMap: false)
        )
    }

    func testVenueSubmitSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        stubAreas(server)
        let model = VenueSubmitModel(session: session, rewardStatusLoader: { nil })
        await model.loadIfNeeded()
        model.form.name = "星轨台球俱乐部"
        model.form.regionText = "广东省 深圳市 南山区"
        model.form.city = "深圳市"
        model.form.district = "南山区"
        model.form.areaIds = [19, 321, 2723]
        try await render("venue-submit", session: session, view: VenueSubmitView(session: session, model: model))
    }

    func testVenueSubmitErrorsSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        stubAreas(server)
        let model = VenueSubmitModel(session: session, rewardStatusLoader: { nil })
        await model.loadIfNeeded()
        XCTAssertEqual(model.validate(), "请输入球馆名称")
        try await render("venue-submit-errors", session: session, view: VenueSubmitView(session: session, model: model))
    }

    private func render<Content: View>(
        _ name: String,
        session: SessionStore,
        view: Content
    ) async throws {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    view
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func stubAreas(_ server: MockHTTPServer) {
        server.stub("GET", "/api/venue/areas") { request in
            switch request.query["parent_id"] {
            case "0":
                return .json(VenueFixtures.areaJSON([(id: 19, parentId: 0, name: "广东省")]))
            case "19":
                return .json(VenueFixtures.areaJSON([(id: 321, parentId: 19, name: "深圳市")]))
            default:
                return .json(VenueFixtures.areaJSON([(id: 2723, parentId: 321, name: "南山区")]))
            }
        }
    }
}
