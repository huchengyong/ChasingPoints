import XCTest
@testable import iOS

/// 战绩海报：文案/映射规则、读取模型降级与位图渲染。
@MainActor
final class ShareResultTests: XCTestCase {
    // MARK: - 规则

    func testRankChangeLabelFollowsModeAndSign() {
        XCTAssertEqual(ShareResultRules.rankChangeLabel(matchMode: "practice", rankChange: 3), "练习赛")
        XCTAssertEqual(ShareResultRules.rankChangeLabel(matchMode: "ranked", rankChange: 3), "排位 +3")
        XCTAssertEqual(ShareResultRules.rankChangeLabel(matchMode: "ranked", rankChange: -2), "排位 -2")
        XCTAssertEqual(ShareResultRules.rankChangeLabel(matchMode: nil, rankChange: 0), "排位 ±0")
    }

    func testDisplayNameTruncatesAfterSevenCharactersWithoutGap() {
        XCTAssertEqual(ShareResultRules.displayName("", fallback: "我"), "我")
        XCTAssertEqual(ShareResultRules.displayName("一二三四五六七", fallback: "我"), "一二三四五六七")
        XCTAssertEqual(ShareResultRules.displayName("一二三四五六七八", fallback: "我"), "一二三四五六…")
    }

    func testPosterQrTargetUsesMatchLinkOnlyForPublicMatches() {
        let matchId: Int64 = 42
        XCTAssertEqual(
            ShareResultRules.resolvePosterQrTarget(visibility: "public", matchId: matchId).url,
            EntryLink.buildLink(.match(matchId: matchId))
        )
        XCTAssertEqual(
            ShareResultRules.resolvePosterQrTarget(visibility: "public", matchId: matchId).caption,
            "微信扫码查看这场比赛"
        )
        for visibility in ["private", nil] {
            let target = ShareResultRules.resolvePosterQrTarget(visibility: visibility, matchId: matchId)
            XCTAssertEqual(target.url, EntryLink.buildLink(.home))
            XCTAssertEqual(target.caption, "微信扫码打开追分")
        }
        XCTAssertEqual(ShareResultRules.resolvePosterQrTarget(visibility: "public", matchId: 0).url, EntryLink.base)
    }

    func testPayloadMapsFallbacksCapsAndOpponentBadge() {
        let data = makeShareData(
            gameType: 2,
            gameTypeName: "",
            myName: "一二三四五六七八",
            opponentName: "",
            opponentVipLevel: 3,
            opponentRankLevel: 4,
            result: "",
            rankChange: 2,
            highlights: (1...5).map { item("亮点\($0)") },
            stats: (1...7).map { item("数据\($0)") }
        )

        let payload = ShareResultRules.payload(from: data)

        XCTAssertEqual(payload.gameTypeName, "九球追分")
        XCTAssertEqual(payload.resultText, "已完赛")
        XCTAssertEqual(payload.rankChangeLabel, "排位 +2")
        XCTAssertEqual(payload.myName, "一二三四五六…")
        XCTAssertEqual(payload.opponentName, "对手")
        XCTAssertEqual(payload.highlights.map(\.label), ["亮点1", "亮点2", "亮点3", "亮点4"])
        XCTAssertEqual(payload.stats.map(\.label), ["数据1", "数据2", "数据3", "数据4", "数据5", "数据6"])
        XCTAssertEqual(payload.opponentBadge, SharePosterBadge(text: "VIP3", tier: "advanced"))
        XCTAssertEqual(payload.myScore, 7)
        XCTAssertEqual(payload.opponentScore, 5)
    }

    func testPayloadNormalizesEmptyCardFieldsAndKeepsTone() {
        let data = makeShareData(
            highlights: [MatchSummaryItem(label: "", value: "", desc: "", tone: "gold")]
        )

        let payload = ShareResultRules.payload(from: data)

        XCTAssertEqual(payload.highlights, [SharePosterCard(label: "--", value: "--", desc: nil, tone: "gold")])
        XCTAssertNil(payload.opponentBadge)
    }

    func testFallbackPayloadKeepsPosterUsableWithoutData() {
        let date = Calendar.current.date(from: DateComponents(year: 2026, month: 11, day: 15))!
        let payload = ShareResultRules.fallbackPayload(date: date)

        XCTAssertEqual(payload.gameTypeName, "中式八球")
        XCTAssertEqual(payload.resultText, "已完赛")
        XCTAssertEqual(payload.rankChangeLabel, "排位 ±0")
        XCTAssertEqual(payload.myName, "我")
        XCTAssertEqual(payload.opponentName, "对手")
        XCTAssertEqual(payload.myScore, 0)
        XCTAssertEqual(payload.opponentScore, 0)
        XCTAssertEqual(payload.matchTime, "2026/11/15")
        XCTAssertTrue(payload.highlights.isEmpty)
        XCTAssertTrue(payload.stats.isEmpty)
        XCTAssertNil(payload.opponentBadge)
        XCTAssertEqual(payload.qrURL, EntryLink.base)
        XCTAssertEqual(payload.qrCaption, "微信扫码打开追分")
    }

    func testPosterHeightGrowsWithRowsAndNeverBelowMinimum() {
        XCTAssertEqual(ShareResultRules.statsTop(highlightCount: 0), 818)
        XCTAssertEqual(ShareResultRules.statsTop(highlightCount: 4), 988)
        XCTAssertEqual(ShareResultRules.posterHeight(highlightCount: 0, statCount: 0), 1320)
        XCTAssertEqual(ShareResultRules.posterHeight(highlightCount: 4, statCount: 6), 1794)
    }

    func testScoreTextNeverAddsThousandsSeparator() {
        XCTAssertEqual(ShareResultRules.scoreText(1000), "1000")
    }

    // MARK: - 读取模型

    func testLoadRendersPosterFromShareResponse() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/share/match", .json([
            "success": true,
            "data": shareJSON(
                myName: "阿豪",
                myScore: 9,
                opponentName: "阿杰",
                opponentScore: 4,
                opponentVipLevel: 5
            ),
        ]))
        var rendered: SharePosterPayload?
        let model = ShareResultModel(
            session: session,
            matchId: 42,
            api: ShareResultAPI(session: session),
            render: { payload in
                rendered = payload
                return Self.pixel
            }
        )

        await model.load()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.payload?.myName, "阿豪")
        XCTAssertEqual(model.payload?.opponentBadge, SharePosterBadge(text: "VIP5", tier: "premium"))
        XCTAssertEqual(rendered, model.payload)
        XCTAssertNotNil(model.posterImage)
        XCTAssertEqual(server.requests("/api/share/match").first?.query, ["match_id": "42"])
        XCTAssertNotNil(server.requests("/api/share/match").first?.bearerToken)
    }

    func testLoadFailureFallsBackToPosterInsteadOfErrorPage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/share/match", .raw("down", status: 503))
        let model = ShareResultModel(
            session: session,
            matchId: 42,
            api: ShareResultAPI(session: session),
            render: { _ in Self.pixel }
        )

        await model.load()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.payload?.gameTypeName, "中式八球")
        XCTAssertEqual(model.payload?.myScore, 0)
        XCTAssertEqual(model.payload?.qrURL, EntryLink.base)
    }

    func testBusinessFailureAlsoFallsBackToPoster() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/share/match", .json(["success": false, "message": "无权查看该对局"]))
        let model = ShareResultModel(
            session: session,
            matchId: 42,
            api: ShareResultAPI(session: session),
            render: { _ in Self.pixel }
        )

        await model.load()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.payload?.myName, "我")
    }

    func testRenderFailureIsTheOnlyPathToFailedState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/share/match", .json(["success": true, "data": shareJSON()]))
        let model = ShareResultModel(
            session: session,
            matchId: 42,
            api: ShareResultAPI(session: session),
            render: { _ in nil }
        )

        await model.load()

        XCTAssertEqual(model.phase, .failed)
        XCTAssertNotNil(model.payload)
        XCTAssertNil(model.posterImage)
    }

    // MARK: - 位图

    func testPosterRendersAtCanvasSizeTwiceScale() throws {
        let payload = ShareResultRules.payload(from: makeShareData())

        let image = try XCTUnwrap(ShareResultPoster.render(payload, qrImage: nil, scale: 2))

        XCTAssertEqual(image.scale, 2)
        XCTAssertEqual(image.size, CGSize(width: 750, height: payload.height))
    }

    // MARK: - 支撑

    private static let pixel = UIGraphicsImageRenderer(size: CGSize(width: 1, height: 1)).image { _ in }

    private func item(_ label: String) -> MatchSummaryItem {
        MatchSummaryItem(label: label, value: "1", desc: nil, tone: nil)
    }

    private func makeShareData(
        matchId: Int64 = 42,
        gameType: Int = 3,
        gameTypeName: String = "中式八球",
        myName: String = "我",
        opponentName: String = "对手",
        myScore: Int = 7,
        opponentScore: Int = 5,
        opponentVipLevel: Int? = nil,
        opponentRankLevel: Int? = nil,
        result: String = "胜利",
        matchTime: String = "2026/11/10 20:00",
        rankChange: Int = 0,
        visibility: String? = "public",
        matchMode: String? = "ranked",
        highlights: [MatchSummaryItem]? = nil,
        stats: [MatchSummaryItem]? = nil
    ) -> ShareResultData {
        ShareResultData(
            matchId: matchId,
            gameType: gameType,
            gameTypeName: gameTypeName,
            myName: myName,
            myAvatar: "",
            myScore: myScore,
            opponentName: opponentName,
            opponentAvatar: "",
            opponentScore: opponentScore,
            myVipLevel: nil,
            opponentVipLevel: opponentVipLevel,
            myRankLevel: nil,
            opponentRankLevel: opponentRankLevel,
            result: result,
            matchTime: matchTime,
            rankChange: rankChange,
            visibility: visibility,
            matchMode: matchMode,
            summaryHighlights: highlights,
            summaryStats: stats
        )
    }

    private func shareJSON(
        myName: String = "我",
        myScore: Int = 7,
        opponentName: String = "对手",
        opponentScore: Int = 5,
        opponentVipLevel: Int = 0
    ) -> [String: Any] {
        [
            "match_id": 42,
            "game_type": 3,
            "game_type_name": "中式八球",
            "my_name": myName,
            "my_avatar": "",
            "my_score": myScore,
            "opponent_name": opponentName,
            "opponent_avatar": "",
            "opponent_score": opponentScore,
            "my_vip_level": 0,
            "opponent_vip_level": opponentVipLevel,
            "my_rank_level": 0,
            "opponent_rank_level": 0,
            "result": "胜利",
            "match_time": "2026/11/10 20:00",
            "rank_change": 2,
            "visibility": "public",
            "match_mode": "ranked",
            "summary_highlights": [["label": "清台", "value": "1 次"]],
            "summary_stats": [["label": "进球", "value": "7"]],
        ]
    }
}
