import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class MemberCenterSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testActiveMemberCenterSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stub(server, status: ProfileFixtures.memberStatusJSON(
            isActive: true,
            memberExpiresAt: "2030-12-31 00:30:00",
            growthLevel: 4,
            growthPoints: 78,
            todayGrowthCount: 3
        ))
        let model = MemberCenterModel(session: session)
        await model.onAppear()
        XCTAssertTrue(model.identityPreview.active)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("member-center-active", colorScheme: scheme, size: pageSize) {
                page(session) { MemberCenterView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testExpiredMemberCenterSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stub(server, status: ProfileFixtures.memberStatusJSON(
            isActive: false,
            memberExpiresAt: "2020-01-02 10:00:00",
            growthLevel: 3
        ))
        let model = MemberCenterModel(session: session)
        await model.onAppear()
        XCTAssertTrue(model.identityPreview.frozen)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("member-center-expired", colorScheme: scheme, size: pageSize) {
                page(session) { MemberCenterView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testNeverMemberCenterSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stub(server, status: ProfileFixtures.memberStatusJSON(
            isActive: false,
            planCode: nil,
            planName: nil,
            memberExpiresAt: nil
        ))
        let model = MemberCenterModel(session: session)
        await model.onAppear()
        XCTAssertFalse(model.entryCard.visible)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("member-center-none", colorScheme: scheme, size: pageSize) {
                page(session) { MemberCenterView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    private func stub(_ server: MockHTTPServer, status: [String: Any]) {
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            memberStatus: status,
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON()
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([
            ProfileFixtures.rankInfoItem(gameType: 3, rankInfo: ProfileFixtures.rankInfoJSON()),
        ])))
    }

    private func page<Content: View>(
        _ session: SessionStore,
        @ViewBuilder content: () -> Content
    ) -> some View {
        NavigationStack {
            content()
        }
        .environmentObject(session)
        .environmentObject(AppNavigator())
    }
}
