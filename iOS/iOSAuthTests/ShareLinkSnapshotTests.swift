import SwiftUI
import XCTest
@testable import iOS

/// 战绩海报与邀请承接页快照，只断言能渲染，不做像素比对。
@MainActor
final class ShareLinkSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)
    private var server: MockHTTPServer!

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
    }

    func testPosterSnapshots() throws {
        for scheme in [ColorScheme.light, .dark] {
            let win = try renderSnapshot(
                "share-result-poster-win",
                colorScheme: scheme,
                size: CGSize(width: 750, height: payload(for: .win).height)
            ) {
                ShareResultPoster(payload: payload(for: .win), qrImage: QRCodeImage.make(content: EntryLink.base, size: 108))
            }
            XCTAssertEqual(win.size.width, 750)

            let lose = try renderSnapshot(
                "share-result-poster-lose",
                colorScheme: scheme,
                size: CGSize(width: 750, height: payload(for: .lose).height)
            ) {
                ShareResultPoster(payload: payload(for: .lose), qrImage: nil)
            }
            XCTAssertEqual(lose.size.width, 750)

            let memberPayload = payload(for: .member)
            let member = try renderSnapshot(
                "share-result-poster-member",
                colorScheme: scheme,
                size: CGSize(width: 750, height: memberPayload.height)
            ) {
                ShareResultPoster(
                    payload: memberPayload,
                    qrImage: QRCodeImage.make(content: EntryLink.base, size: 108),
                    assets: SharePosterAssets(
                        myAvatar: Self.solidImage(color: .systemTeal),
                        opponentAvatar: Self.solidImage(color: .systemIndigo),
                        frame: Self.frameImage()
                    )
                )
            }
            XCTAssertEqual(member.size.width, 750)
        }
    }

    func testShareResultPageSnapshot() async throws {
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/share/match", .json(["success": true, "data": shareJSON()]))
        let model = ShareResultModel(session: session, matchId: 42)
        await model.load()
        XCTAssertEqual(model.phase, .ready)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("share-result-page", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    ShareResultView(session: session, matchId: 42, model: model)
                }
                .environmentObject(session)
                .environmentObject(ToastCenter())
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testLinkEntrySnapshots() async throws {
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let navigator = AppNavigator()
        let loading = LinkEntryModel(flow: unusedFlow())
        let error = await errorModel()

        for scheme in [ColorScheme.light, .dark] {
            let loadingImage = try renderSnapshot("link-entry-loading", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    LinkEntryView(
                        session: session,
                        url: EntryLink.base,
                        quickScan: QuickScanFlow(session: session),
                        model: loading
                    )
                }
                .environmentObject(session)
                .environmentObject(navigator)
            }
            XCTAssertEqual(loadingImage.size, pageSize)

            let errorImage = try renderSnapshot("link-entry-error", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    LinkEntryView(
                        session: session,
                        url: EntryLink.base,
                        quickScan: QuickScanFlow(session: session),
                        model: error
                    )
                }
                .environmentObject(session)
                .environmentObject(navigator)
            }
            XCTAssertEqual(errorImage.size, pageSize)
        }
    }

    // MARK: - 支撑

    /// 快照不联网：会员模板需要的头像与头像框用本地生成的图片代替。
    private static func solidImage(color: UIColor) -> UIImage {
        UIGraphicsImageRenderer(size: CGSize(width: 88, height: 88)).image { context in
            color.setFill()
            context.fill(CGRect(x: 0, y: 0, width: 88, height: 88))
        }
    }

    private static func frameImage() -> UIImage {
        UIGraphicsImageRenderer(size: CGSize(width: 143, height: 143)).image { context in
            UIColor(red: 0.88, green: 0.65, blue: 0.13, alpha: 1).setStroke()
            let path = UIBezierPath(ovalIn: CGRect(x: 8, y: 8, width: 127, height: 127))
            path.lineWidth = 10
            path.stroke()
        }
    }

    private enum ResultKind {
        case win
        case lose
        case member
    }

    private func payload(for kind: ResultKind) -> SharePosterPayload {
        let data: ShareResultData
        switch kind {
        case .win:
            data = shareData(
                myName: "阿豪",
                myScore: 9,
                opponentName: "阿杰",
                opponentScore: 4,
                opponentVipLevel: 3,
                result: "胜利",
                rankChange: 3,
                highlights: [
                    item("连下三局", value: "3", desc: "从 3:4 追到 6:4", tone: "gold"),
                    item("单杆最高分", value: "78", desc: "个人本赛季新高", tone: "emerald"),
                    item("清台", value: "1", desc: "关键第八局", tone: "purple"),
                ],
                stats: [
                    item("进球数", value: "41"),
                    item("成功率", value: "86%", tone: "cyan"),
                    item("平均出杆", value: "14 秒"),
                    item("罚分", value: "2", tone: "orange"),
                ]
            )
        case .lose:
            data = shareData(
                myName: "阿豪",
                myScore: 3,
                opponentName: "阿杰",
                opponentScore: 7,
                result: "失利",
                rankChange: -2,
                highlights: [],
                stats: [
                    item("进球数", value: "23"),
                    item("成功率", value: "71%", tone: "slate"),
                ]
            )
        case .member:
            data = shareData(
                myName: "阿豪",
                myAvatar: "https://img.example/me.png",
                myScore: 9,
                opponentName: "阿杰",
                opponentAvatar: "https://img.example/op.png",
                opponentScore: 4,
                myVipLevel: 3,
                opponentVipLevel: 5,
                myRankLevel: 3,
                opponentRankLevel: 6,
                result: "胜利",
                rankChange: 3,
                highlights: [
                    item("连下三局", value: "3", desc: "从 3:4 追到 6:4", tone: "gold"),
                    item("清台", value: "1", desc: "关键第八局", tone: "purple"),
                ],
                stats: [
                    item("进球数", value: "41"),
                    item("成功率", value: "86%", tone: "cyan"),
                ]
            )
        }
        return ShareResultRules.payload(from: data)
    }

    private func unusedFlow() -> LinkEntryFlow {
        LinkEntryFlow(
            currentIdentity: { nil },
            previewInvite: { _ in throw URLError(.notConnectedToInternet) },
            resolveInvite: { _ in throw URLError(.notConnectedToInternet) },
            handoffResolved: { _, _, _ in false }
        )
    }

    private func errorModel() async -> LinkEntryModel {
        let flow = LinkEntryFlow(
            currentIdentity: { nil },
            previewInvite: { _ in throw URLError(.notConnectedToInternet) },
            resolveInvite: { _ in throw URLError(.notConnectedToInternet) },
            handoffResolved: { _, _, _ in false }
        )
        let model = LinkEntryModel(flow: flow)
        await model.begin(code: "Ab3dEf6hIj9k")
        XCTAssertEqual(model.stage, .error)
        return model
    }

    private func item(
        _ label: String,
        value: String = "1",
        desc: String? = nil,
        tone: String? = nil
    ) -> MatchSummaryItem {
        MatchSummaryItem(label: label, value: value, desc: desc, tone: tone)
    }

    private func shareData(
        myName: String = "我",
        myAvatar: String = "",
        myScore: Int = 7,
        opponentName: String = "对手",
        opponentAvatar: String = "",
        opponentScore: Int = 5,
        myVipLevel: Int? = nil,
        opponentVipLevel: Int? = nil,
        myRankLevel: Int? = nil,
        opponentRankLevel: Int? = nil,
        result: String = "胜利",
        rankChange: Int = 0,
        highlights: [MatchSummaryItem]? = nil,
        stats: [MatchSummaryItem]? = nil
    ) -> ShareResultData {
        ShareResultData(
            matchId: 42,
            gameType: 3,
            gameTypeName: "中式八球",
            myName: myName,
            myAvatar: myAvatar,
            myScore: myScore,
            opponentName: opponentName,
            opponentAvatar: opponentAvatar,
            opponentScore: opponentScore,
            myVipLevel: myVipLevel,
            opponentVipLevel: opponentVipLevel,
            myRankLevel: myRankLevel,
            opponentRankLevel: opponentRankLevel,
            result: result,
            matchTime: "2026/11/10 20:00",
            rankChange: rankChange,
            visibility: "public",
            matchMode: "ranked",
            summaryHighlights: highlights,
            summaryStats: stats
        )
    }

    private func shareJSON() -> [String: Any] {
        [
            "match_id": 42,
            "game_type": 3,
            "game_type_name": "中式八球",
            "my_name": "阿豪",
            "my_avatar": "",
            "my_score": 9,
            "opponent_name": "阿杰",
            "opponent_avatar": "",
            "opponent_score": 4,
            "my_vip_level": 0,
            "opponent_vip_level": 3,
            "my_rank_level": 0,
            "opponent_rank_level": 4,
            "result": "胜利",
            "match_time": "2026/11/10 20:00",
            "rank_change": 3,
            "visibility": "public",
            "match_mode": "ranked",
            "summary_highlights": [
                ["label": "连下三局", "value": "3", "desc": "从 3:4 追到 6:4", "tone": "gold"],
                ["label": "清台", "value": "1", "desc": "关键第八局", "tone": "purple"],
            ],
            "summary_stats": [
                ["label": "进球数", "value": "41"],
                ["label": "成功率", "value": "86%", "tone": "cyan"],
            ],
        ]
    }
}
