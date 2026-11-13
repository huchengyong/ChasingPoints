import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/chasing-score.test.mjs`。
@MainActor
final class ChasingScoreTests: XCTestCase {
    private var breakdown: ChasingScoreBreakdown {
        ChasingScoreBreakdown(
            available: true,
            player1: ChasingScoreSide(
                score: 16, normalWins: 2, smallGolds: 1, bigGolds: 0,
                foulCount: 3, foulPointsGiven: 3, otherPoints: 0
            ),
            player2: ChasingScoreSide(
                score: 13, normalWins: 0, smallGolds: 0, bigGolds: 1,
                foulCount: 1, foulPointsGiven: 1, otherPoints: 0
            ),
            events: [
                ChasingScoreEvent(
                    seq: 1, kind: "foul", scorer: 1, points: 1,
                    player1Score: 1, player2Score: 0, createdAt: "2026-11-10 12:00:01"
                ),
                ChasingScoreEvent(
                    seq: 2, kind: "normal", scorer: 1, points: 4,
                    player1Score: 5, player2Score: 0, createdAt: "2026-11-10 12:00:02"
                ),
                ChasingScoreEvent(
                    seq: 3, kind: "big_gold", scorer: 2, points: 10,
                    player1Score: 5, player2Score: 10, createdAt: "2026-11-10 12:00:03"
                ),
            ],
            eventsTruncated: false
        )
    }

    func testReturnsNilForAnythingThatIsNotABreakdownObject() {
        XCTAssertNil(ChasingScore.resolveView(nil))
    }

    func testPlayer1PerspectiveShowsMeAndOpponentWithFixedRowOrder() throws {
        let view = try XCTUnwrap(ChasingScore.resolveView(breakdown))
        XCTAssertTrue(view.available)
        XCTAssertEqual([view.leftLabel, view.rightLabel], ["我", "对手"])
        XCTAssertEqual([view.leftScore, view.rightScore], [16, 13])
        XCTAssertEqual(view.rows.map { [$0.key, $0.label, $0.left, $0.right] }, [
            ["score", "总分", "16", "13"],
            ["normal_wins", "普胜", "2", "0"],
            ["small_golds", "小金", "1", "0"],
            ["big_golds", "大金", "0", "1"],
            ["foul_count", "犯规", "3", "1"],
            ["foul_points_given", "犯规送分", "3", "1"],
        ])
        XCTAssertEqual(view.notice, "")
        XCTAssertFalse(view.eventsTruncated)
    }

    func testPlayer2PerspectiveSwapsRowsAndEventScores() throws {
        let view = try XCTUnwrap(ChasingScore.resolveView(breakdown, swap: true))
        XCTAssertEqual([view.leftScore, view.rightScore], [13, 16])
        XCTAssertEqual(view.rows.prefix(3).map { [$0.label, $0.left, $0.right] }, [
            ["总分", "13", "16"],
            ["普胜", "0", "2"],
            ["小金", "0", "1"],
        ])
        let latest = view.events[0]
        let fouled = view.events[2]
        XCTAssertEqual(latest.kindLabel, "大金")
        XCTAssertEqual([latest.side, latest.sideLabel], ["left", "我"])
        XCTAssertEqual(latest.scoreText, "10 : 5")
        // 犯规方（选手2）在右列，犯规条目的得分方在左列。
        XCTAssertEqual([fouled.side, fouled.desc], ["right", "我犯规"])
        XCTAssertEqual(fouled.scoreText, "0 : 1")
    }

    func testNeutralPerspectiveIgnoresSwapAndLabelsSidesPlayer1AndPlayer2() throws {
        let view = try XCTUnwrap(ChasingScore.resolveView(breakdown, swap: true, neutral: true))
        XCTAssertEqual([view.leftLabel, view.rightLabel], ["选手1", "选手2"])
        XCTAssertEqual([view.leftScore, view.rightScore], [16, 13])
        let latest = view.events[0]
        let fouled = view.events[2]
        XCTAssertEqual([latest.side, latest.sideLabel], ["right", "选手2"])
        XCTAssertEqual(latest.scoreText, "5 : 10")
        XCTAssertEqual([fouled.side, fouled.desc], ["left", "选手2犯规"])
    }

    func testEventsAreListedNewestFirstWithTimeAndDescriptionText() throws {
        let view = try XCTUnwrap(ChasingScore.resolveView(breakdown, neutral: true))
        XCTAssertEqual(view.events.map(\.seq), [3, 2, 1])
        XCTAssertEqual(view.events.map(\.kindLabel), ["大金", "普胜", "犯规"])
        XCTAssertEqual(view.events.map(\.pointsText), ["+10", "+4", "选手1 +1"])
        XCTAssertEqual(view.events.map(\.desc), ["选手2 大金", "选手1 普胜", "选手2犯规"])
        XCTAssertEqual(view.events.map(\.timeText), ["12:00:03", "12:00:02", "12:00:01"])
    }

    func testOtherPointsRowOnlyAppearsWhenEitherSideScoredSome() throws {
        let withoutOther = try XCTUnwrap(ChasingScore.resolveView(breakdown))
        XCTAssertFalse(withoutOther.rows.contains { $0.key == "other_points" })

        var withOther = breakdown
        withOther.player2.otherPoints = 3
        let view = try XCTUnwrap(ChasingScore.resolveView(withOther))
        XCTAssertEqual(view.rows.last, ChasingScoreRow(
            key: "other_points", label: "其他得分", left: "0", right: "3"
        ))

        var otherEvent = breakdown
        otherEvent.events = [ChasingScoreEvent(
            seq: 4, kind: "other", scorer: 1, points: 2,
            player1Score: 18, player2Score: 13, createdAt: "2026-11-10 12:00:04"
        )]
        let otherEventView = try XCTUnwrap(ChasingScore.resolveView(otherEvent))
        XCTAssertEqual(otherEventView.events[0].desc, "我 得分")
    }

    func testUnavailableBreakdownDegradesToTheTotalScoreOnly() throws {
        var unavailable = breakdown
        unavailable.available = false
        let view = try XCTUnwrap(ChasingScore.resolveView(unavailable))
        XCTAssertFalse(view.available)
        XCTAssertEqual(view.rows, [ChasingScoreRow(key: "score", label: "总分", left: "16", right: "13")])
        XCTAssertEqual(view.events, [])
        XCTAssertEqual(view.notice, "记分明细暂不可用，仅显示总分")
    }

    func testTruncatedEventsSayOnlyTheLatest200AreShown() throws {
        var truncated = breakdown
        truncated.eventsTruncated = true
        let view = try XCTUnwrap(ChasingScore.resolveView(truncated))
        XCTAssertTrue(view.eventsTruncated)
        XCTAssertEqual(view.notice, "仅显示最近 200 条记分")
    }

    /// JS 还会传入字符串 / Infinity / null 事件；Swift 的窄类型不允许这些值，
    /// 这里只覆盖「未知 kind、异常 scorer、空时间」这些仍可表达的容错分支。
    func testDirtyPayloadsDoNotThrowAndFallBackToZeroValues() throws {
        let view = try XCTUnwrap(ChasingScore.resolveView(ChasingScoreBreakdown(
            available: true,
            player1: ChasingScoreSide(),
            events: [ChasingScoreEvent(seq: 1, kind: "mystery", scorer: 9, points: 4)]
        )))
        XCTAssertEqual(view.leftScore, 0)
        XCTAssertEqual(view.rows[0].right, "0")
        XCTAssertEqual(view.rows.first { $0.key == "foul_count" }?.left, "0")
        XCTAssertEqual(view.events[0].kindLabel, "其他")
        XCTAssertEqual(view.events[0].side, "left")
        XCTAssertEqual(view.events[0].pointsText, "+4")
        XCTAssertEqual(view.events[0].timeText, "")
    }
}
