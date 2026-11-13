import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/member-replay-view.test.mjs`。
final class MemberReplayViewTests: XCTestCase {
    // MARK: - 夹具

    private static let snookerResponse = MatchReplayResponse(
        gameType: 1,
        frameCount: 2,
        summary: ReplaySummary(
            myFrameWins: 1,
            opponentFrameWins: 1,
            myMaxRun: 68,
            opponentMaxRun: 41
        ),
        frames: [
            ReplayFrame(
                frameNo: 1,
                winner: 1,
                myScore: 72,
                opponentScore: 40,
                durationSeconds: 1260,
                maxDeficit: 12,
                keyTags: ["comeback"],
                myMaxRun: 68,
                opponentMaxRun: 20,
                myFoulCount: 1,
                opponentFoulCount: 0,
                myFoulPoints: 4,
                opponentFoulPoints: 0,
                trend: [
                    ReplayTrendPoint(step: 0, diff: 0),
                    ReplayTrendPoint(step: 1, diff: -12),
                    ReplayTrendPoint(step: 2, diff: 6),
                    ReplayTrendPoint(step: 3, diff: 32),
                ]
            ),
            ReplayFrame(
                frameNo: 2,
                winner: 2,
                myScore: 30,
                opponentScore: 61,
                durationSeconds: 45,
                keyTags: ["decider"],
                myMaxRun: 18,
                opponentMaxRun: 41,
                trend: []
            ),
        ]
    )

    private static let nineBallBreakdown = ChasingScoreBreakdown(
        available: true,
        player1: ChasingScoreSide(
            score: 21, normalWins: 3, smallGolds: 1, bigGolds: 1,
            foulCount: 2, foulPointsGiven: 2, otherPoints: 0
        ),
        player2: ChasingScoreSide(
            score: 15, normalWins: 1, smallGolds: 2, bigGolds: 0,
            foulCount: 1, foulPointsGiven: 1, otherPoints: 0
        ),
        events: [],
        eventsTruncated: false
    )

    private static let nineBallFrame = ReplayFrame(
        frameNo: 1,
        winner: 1,
        myScore: 21,
        opponentScore: 15,
        durationSeconds: 1500,
        maxDeficit: 4,
        myMaxRun: 12,
        opponentMaxRun: 5,
        myFoulCount: 2,
        opponentFoulCount: 1,
        myFoulPoints: 2,
        opponentFoulPoints: 1,
        trend: [
            ReplayTrendPoint(step: 0, diff: 0),
            ReplayTrendPoint(step: 1, diff: -4),
            ReplayTrendPoint(step: 2, diff: 3),
            ReplayTrendPoint(step: 3, diff: 6),
        ]
    )

    private static let nineBallResponse = MatchReplayResponse(
        gameType: 2,
        viewerActor: 1,
        frameCount: 3,
        summary: ReplaySummary(
            myMaxRun: 12,
            opponentMaxRun: 5,
            scoreBreakdown: nineBallBreakdown
        ),
        frames: [nineBallFrame]
    )

    private func rowLabels(_ rows: [ReplaySummaryRow]?) -> [[String]] {
        (rows ?? []).map { [$0.label, $0.mine, $0.opponent] }
    }

    // MARK: - 用例

    func testLockedReplayCarriesOnlyFrameCountAndPurchaseEntryFollowsComplianceSwitch() {
        let response = MatchReplayResponse(
            gameType: 3,
            locked: true,
            frameCount: 7,
            summary: ReplaySummary(myFrameWins: 4, opponentFrameWins: 3),
            frames: [ReplayFrame(frameNo: 1, winner: 1)]
        )
        let view = MemberReplayView.resolveReplayView(response, canShowPurchase: false)
        XCTAssertTrue(view.locked)
        XCTAssertEqual(view.frameCount, 7)
        XCTAssertFalse(view.showPurchase)
        XCTAssertTrue(MemberReplayView.resolveReplayView(response, canShowPurchase: true).showPurchase)
        XCTAssertFalse(MemberReplayView.resolveReplayView(response).showPurchase)
        // 锁定时即使响应里误带了明细，也不会进入展示模型。
        XCTAssertNil(view.frames)
        XCTAssertNil(view.summaryRows)
        XCTAssertFalse(String(describing: view).contains("第 1 局"))
    }

    func testSnookerReplayShowsFrameScoresBreaksFoulsTagsAndComebackDescription() {
        let view = MemberReplayView.resolveReplayView(Self.snookerResponse)
        XCTAssertFalse(view.wholeMatch)
        XCTAssertEqual(view.title, "斯诺克复盘")
        XCTAssertEqual(view.scoreText, "1 : 1")
        XCTAssertEqual(rowLabels(view.summaryRows), [
            ["赢下局数", "1", "1"],
            ["单杆最高", "68", "41"],
        ])

        let frames = view.frames ?? []
        XCTAssertEqual(frames.count, 2)
        let first = frames[0]
        XCTAssertEqual(first.scoreText, "72 : 40")
        XCTAssertEqual(first.winnerText, "我赢")
        XCTAssertEqual(first.durationText, "21分")
        XCTAssertEqual(first.tags.map(\.label), ["逆转局"])
        XCTAssertEqual(first.deficitText, "我曾落后 12 分后赢下")
        XCTAssertEqual(first.details.map(\.label), ["单杆最高", "犯规（次 / 罚分）"])
        XCTAssertEqual(
            first.details[1],
            ReplayDetailRow(label: "犯规（次 / 罚分）", mine: "1 / 4", opponent: "0 / 0")
        )
        XCTAssertTrue(first.trend.hasData)

        let second = frames[1]
        XCTAssertEqual(second.winnerText, "对手赢")
        XCTAssertEqual(second.durationText, "45秒")
        XCTAssertEqual(second.deficitText, "", "赢家从未落后时没有落后描述")
        XCTAssertFalse(second.trend.hasData)
        XCTAssertEqual((view.keyFrames ?? []).map(\.frameNo), [1, 2])
        XCTAssertEqual(view.notice, "")
        XCTAssertEqual(view.hasDetail, true)
    }

    func testNineBallChaseReplaysTheWholeMatchInsteadOfPerFrameResults() {
        let view = MemberReplayView.resolveReplayView(Self.nineBallResponse)
        XCTAssertTrue(view.wholeMatch)
        XCTAssertEqual(view.scoreText, "21 : 15")
        XCTAssertEqual(rowLabels(view.summaryRows), [
            ["总分", "21", "15"],
            ["普胜", "3", "1"],
            ["小金", "1", "2"],
            ["大金", "1", "0"],
            ["犯规", "2", "1"],
            ["犯规送分", "2", "1"],
            ["最大连续得分", "12", "5"],
        ])
        XCTAssertFalse((view.summaryRows ?? []).contains { $0.label.contains("赢下局数") })
        XCTAssertEqual(view.winTypeRows, [])
        XCTAssertEqual(view.sequence, [])
        XCTAssertEqual(view.keyFrames, [])

        let frames = view.frames ?? []
        XCTAssertEqual(frames.count, 1)
        let frame = frames[0]
        XCTAssertEqual(frame.title, "整场走势")
        XCTAssertEqual(frame.winTypeText, "")
        XCTAssertEqual(frame.tags, [])
        XCTAssertFalse(frame.isKey)
        XCTAssertEqual(frame.winnerText, "我赢")
        XCTAssertEqual(frame.scoreText, "21 : 15")
        XCTAssertEqual(frame.durationText, "25分")
        XCTAssertEqual(frame.deficitText, "我曾落后 4 分后赢下")
        XCTAssertTrue(frame.trend.hasData)
        XCTAssertEqual(view.notice, "")
        XCTAssertEqual(view.hasDetail, true)
        XCTAssertEqual(view.empty, false)
    }

    func testNineBallChaseSwapsTheComparisonForThePlayer2Perspective() {
        let view = MemberReplayView.resolveReplayView(MatchReplayResponse(
            gameType: 2,
            viewerActor: 2,
            frameCount: 3,
            summary: ReplaySummary(
                myMaxRun: 12,
                opponentMaxRun: 5,
                scoreBreakdown: Self.nineBallBreakdown
            ),
            frames: [ReplayFrame(
                frameNo: 1,
                winner: 2,
                myScore: 15,
                opponentScore: 21,
                durationSeconds: 1500,
                maxDeficit: 4,
                myMaxRun: 12,
                opponentMaxRun: 5
            )]
        ))
        XCTAssertEqual(view.scoreText, "15 : 21")
        XCTAssertEqual(Array(rowLabels(view.summaryRows).prefix(4)), [
            ["总分", "15", "21"],
            ["普胜", "1", "3"],
            ["小金", "2", "1"],
            ["大金", "0", "1"],
        ])
        XCTAssertEqual(view.frames?.first?.winnerText, "对手赢")
        // 连续得分只在整场数据里出现一次，片段不重复。
        XCTAssertEqual(view.frames?.first?.details, [])
        XCTAssertEqual(view.summaryRows?.last?.label, "最大连续得分")
    }

    func testNineBallChaseSaysWhyOnlyTheTotalScoreIsShown() {
        var frameOnlyResponse = Self.nineBallResponse
        frameOnlyResponse.frameOnly = true
        let frameOnly = MemberReplayView.resolveReplayView(frameOnlyResponse)
        XCTAssertEqual(frameOnly.notice, "这场对局的记分记录不完整，只显示总分。")
        XCTAssertEqual(frameOnly.hasDetail, false)

        var truncatedResponse = Self.nineBallResponse
        truncatedResponse.detailTruncated = true
        truncatedResponse.frameOnly = true
        let truncated = MemberReplayView.resolveReplayView(truncatedResponse)
        XCTAssertEqual(truncated.notice, "这场对局的记录太长，只显示总分。")
        XCTAssertEqual(truncated.hasDetail, false)
    }

    func testLockedNineBallChaseListsWholeMatchFeaturesWithoutFrameCounts() {
        let view = MemberReplayView.resolveReplayView(
            MatchReplayResponse(gameType: 2, locked: true, frameCount: 7),
            canShowPurchase: true
        )
        XCTAssertTrue(view.wholeMatch)
        XCTAssertEqual(view.frameCount, 7)
        XCTAssertTrue(view.showPurchase)
        XCTAssertEqual(view.features, [
            "整场走势：谁领先、什么时候被追上",
            "双方普胜、小金、大金次数对照",
            "犯规次数与送分",
            "最大连续得分",
        ])
        XCTAssertFalse(String(describing: view).contains("局数"))
    }

    func testChineseEightBallShowsWinSequenceLongestStreakWinTypesAndNoScorePerFrame() {
        let view = MemberReplayView.resolveReplayView(MatchReplayResponse(
            gameType: 3,
            frameCount: 8,
            summary: ReplaySummary(
                myFrameWins: 5,
                opponentFrameWins: 3,
                myLongestStreak: 4,
                opponentLongestStreak: 2,
                winTypeStats: [
                    ReplayWinTypeStat(winType: "normal", label: "普胜", myCount: 3, opponentCount: 3),
                    ReplayWinTypeStat(winType: "break_clear", label: "炸清", myCount: 2, opponentCount: 0),
                ]
            ),
            frames: [
                ReplayFrame(frameNo: 1, winner: 1, myScore: 1, winType: "normal"),
                ReplayFrame(frameNo: 2, winner: 2, opponentScore: 1, winType: "break_clear"),
                ReplayFrame(frameNo: 3, winner: 1, myScore: 1, winType: "normal", keyTags: ["decider"]),
            ]
        ))
        XCTAssertEqual((view.sequence ?? []).map(\.winner), ["me", "opponent", "me"])
        XCTAssertFalse(view.wholeMatch)
        XCTAssertEqual((view.summaryRows ?? []).map(\.label), ["赢下局数", "最长连胜（局）"])
        XCTAssertEqual(view.winTypeRows, [
            ReplaySummaryRow(label: "普胜", mine: "3", opponent: "3"),
            ReplaySummaryRow(label: "炸清", mine: "2", opponent: "0"),
        ])
        XCTAssertEqual(view.frames?[0].scoreText, "")
        XCTAssertEqual(view.frames?[1].winTypeText, "炸清")
        XCTAssertEqual((view.keyFrames ?? []).map(\.frameNo), [3])
    }

    func testFrameOnlyAndTruncatedReplaysSayWhyThereIsNoPerFrameDetail() {
        let frameOnly = MemberReplayView.resolveReplayView(MatchReplayResponse(
            gameType: 1,
            frameCount: 1,
            frameOnly: true,
            summary: ReplaySummary(myFrameWins: 1),
            frames: [ReplayFrame(frameNo: 1, winner: 1, myScore: 60, opponentScore: 20)]
        ))
        XCTAssertTrue(frameOnly.notice?.contains("没有逐杆记录") == true)
        XCTAssertEqual(frameOnly.hasDetail, false)

        let truncated = MemberReplayView.resolveReplayView(MatchReplayResponse(
            gameType: 2,
            frameCount: 1,
            frameOnly: true,
            detailTruncated: true,
            frames: [ReplayFrame(frameNo: 1, winner: 2)]
        ))
        XCTAssertTrue(truncated.notice?.contains("记录太长") == true)
        XCTAssertEqual(truncated.hasDetail, false)

        let none = MemberReplayView.resolveReplayView(MatchReplayResponse(gameType: 3, frames: []))
        XCTAssertEqual(none.empty, true)
    }

    func testTrendColumnsAreCenteredOnZeroWithHeightsNormalizedToTheLargestGap() {
        let trend = MemberReplayView.buildTrendColumns([
            ReplayTrendPoint(step: 0, diff: 0),
            ReplayTrendPoint(step: 1, diff: 20),
            ReplayTrendPoint(step: 2, diff: -10),
            ReplayTrendPoint(step: 3, diff: 5),
        ])
        XCTAssertTrue(trend.hasData)
        XCTAssertEqual(trend.maxAbs, 20)
        XCTAssertEqual(trend.columns.map(\.direction), ["flat", "up", "down", "up"])
        XCTAssertEqual(trend.columns.map(\.heightPercent), [0, 100, 50, 25])

        // 单点或全零没有可画的走势。
        XCTAssertFalse(MemberReplayView.buildTrendColumns([ReplayTrendPoint()]).hasData)
        XCTAssertFalse(MemberReplayView.buildTrendColumns([
            ReplayTrendPoint(step: 0, diff: 0),
            ReplayTrendPoint(step: 1, diff: 0),
        ]).hasData)
        XCTAssertFalse(MemberReplayView.buildTrendColumns(nil).hasData)

        let many = MemberReplayView.buildTrendColumns((0..<120).map {
            ReplayTrendPoint(step: $0, diff: $0 - 60)
        })
        XCTAssertEqual(many.columns.count, MemberReplayView.maxTrendColumns)
    }

    func testFrameSequenceAndDurationHelpersAreDefensiveAboutMissingValues() {
        XCTAssertEqual(
            MemberReplayView.buildFrameSequence([
                ReplayFrame(frameNo: 1, winner: 1),
                ReplayFrame(frameNo: 2, winner: 0),
                ReplayFrame(),
            ]),
            [
                ReplaySequenceItem(frameNo: 1, winner: "me"),
                ReplaySequenceItem(frameNo: 2, winner: "none"),
                ReplaySequenceItem(frameNo: 0, winner: "none"),
            ]
        )
        XCTAssertEqual(
            [0, nil, 45, 60, 61, 1260].map(MemberReplayView.formatFrameDuration),
            ["", "", "45秒", "1分", "1分1秒", "21分"]
        )
    }
}
