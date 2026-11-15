import SwiftUI
import XCTest
@testable import iOS

/// PK 报表与海报快照，及海报尺寸约束。
@MainActor
final class PKReportSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)
    private var server: MockHTTPServer!
    private let target = FriendHomepageTarget(
        userId: 18,
        name: "阿杰",
        avatar: "",
        rankName: "",
        vipLevel: 0,
        rankLevel: 0
    )

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
    }

    func testReportSnapshot() async throws {
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [
                MatchHistoryFixtures.participantItem(id: 1, result: 1, myScore: 7, opponentScore: 5),
                MatchHistoryFixtures.participantItem(id: 2, result: 2, myScore: 3, opponentScore: 6),
            ],
            total: 12
        )))
        let model = PKReportModel(session: session, target: target)
        await model.onAppear()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("pk-report", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    PKReportView(session: session, target: target, model: model)
                }
                .environmentObject(session)
                .environmentObject(AppNavigator())
                .environmentObject(ToastCenter())
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testPosterSnapshot() throws {
        let qr = QRCodeImage.make(content: EntryLink.base, size: 108)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("pk-report-poster", colorScheme: scheme, size: PKReportPoster.size) {
                PKReportPoster(payload: payload, qrImage: qr)
            }
            XCTAssertEqual(image.size, PKReportPoster.size)
        }
    }

    func testImageRendererProducesPosterAtFixedSize() throws {
        let renderer = ImageRenderer(content: PKReportPoster(payload: payload, qrImage: nil))
        renderer.scale = 1

        let image = try XCTUnwrap(renderer.uiImage)

        XCTAssertEqual(image.size, PKReportPoster.size)
    }

    private var payload: PKPosterPayload {
        PKPosterPayload(
            myName: "我",
            opponentName: "阿杰",
            myWins: 7,
            opponentWins: 5,
            totalMatches: 12,
            winRateLabel: "58%",
            avgScoreDiffLabel: "+1.8",
            maxWinStreak: 3,
            summaryText: "你对阿杰略占上风 · 近 5 场 4 胜 1 负 · 当前最长连胜 3 场",
            gameTypeLabel: "中式八球"
        )
    }
}
