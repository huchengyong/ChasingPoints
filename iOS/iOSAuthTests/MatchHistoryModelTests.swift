import Foundation
import XCTest
@testable import iOS

final class MatchHistoryModelTests: XCTestCase {
    private func decode<T: Decodable>(_ type: T.Type, from object: [String: Any]) throws -> T {
        let data = try JSONSerialization.data(withJSONObject: object)
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try decoder.decode(type, from: data)
    }

    func testParticipantPageDecodesServerResultAndIgnoresUnknownFields() throws {
        let page = try decode(
            PagedFixtureResponse<MatchListItem>.self,
            from: MatchHistoryFixtures.participantPage(
                items: [MatchHistoryFixtures.participantItem(id: 7, result: 3, myScore: 2, opponentScore: 2)],
                total: 1
            )
        )

        XCTAssertEqual(page.total, 1)
        XCTAssertEqual(page.list.first?.id, 7)
        XCTAssertEqual(page.list.first?.gameTypeName, "中式八球")
        XCTAssertEqual(MatchOutcome(result: page.list.first?.result ?? 0), .draw)
    }

    func testUnknownServerResultDoesNotGuessWinOrLoss() {
        XCTAssertNil(MatchOutcome(result: 0))
        XCTAssertNil(MatchOutcome(result: 9))
        XCTAssertEqual(MatchOutcome(result: 1), .win)
        XCTAssertEqual(MatchOutcome(result: 2), .loss)
    }

    func testReportOutcomeUsesFinalScoreIncludingDraw() {
        XCTAssertEqual(MatchOutcome(myScore: 3, opponentScore: 1), .win)
        XCTAssertEqual(MatchOutcome(myScore: 1, opponentScore: 3), .loss)
        XCTAssertEqual(MatchOutcome(myScore: 2, opponentScore: 2), .draw)
        XCTAssertEqual(MatchOutcome.draw.title, "平局")
    }

    func testReportOutcomePrefersSettledResultOverFrameScore() throws {
        // 自由局数整场认输：本人局分 2:1 领先但认输，服务端结果为负。
        var fixture = MatchHistoryFixtures.playerDetail(myScore: 2, opponentScore: 1)
        var match = try XCTUnwrap(fixture["match"] as? [String: Any])
        match["result"] = 2
        match["snooker_frame_end_reason"] = "match_concession"
        fixture["match"] = match

        let report = try decode(MatchDetailEnvelope.self, from: fixture).match

        XCTAssertEqual(report.result, 2)
        XCTAssertEqual(report.outcome, .loss)
        XCTAssertTrue(report.isMatchConcession)
    }

    func testReportOutcomeFallsBackToScoreWithoutSettledResult() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(myScore: 1, opponentScore: 3)).match

        XCTAssertNil(report.result)
        XCTAssertEqual(report.outcome, .loss)
        XCTAssertFalse(report.isMatchConcession)
    }

    func testPlayerReportDecodesFieldsAndNullableArrays() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail()).match

        XCTAssertFalse(report.isRefereeView)
        XCTAssertEqual(report.myScore, 3)
        XCTAssertEqual(report.myWinRate, 62.5)
        XCTAssertEqual(report.myRankChange, 12)
        XCTAssertEqual(report.myRankDetails?.map(\.label), ["基础分", "高光排位分"])
        XCTAssertEqual(report.refereeUserId, 11)
        XCTAssertEqual(report.completionSource, "referee")
    }

    func testRankSettlementPendingShowsEffectiveClockInShanghaiTime() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(
            rankSettlementStatus: "pending", rankSettlementDueAt: "2026-08-25T16:10:00+08:00"
        )).match
        XCTAssertEqual(report.rankSettlement, .pending(effectiveClock: "16:10"))

        // 服务端换用其他偏移时仍按北京时间显示。
        let utc = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(
            rankSettlementStatus: "pending", rankSettlementDueAt: "2026-08-25T08:10:00Z"
        )).match
        XCTAssertEqual(utc.rankSettlement, .pending(effectiveClock: "16:10"))
    }

    func testRankSettlementPendingWithUnparsableTimeDoesNotInventAClock() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(
            rankSettlementStatus: "pending", rankSettlementDueAt: "不是时间"
        )).match
        XCTAssertEqual(report.rankSettlement, .pending(effectiveClock: nil))
        XCTAssertNil(MatchTimeFormatting.clock(nil))
        XCTAssertNil(MatchTimeFormatting.clock(""))
    }

    func testRankSettlementVoidedIsNotCountedAsZeroChange() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(
            rankSettlementStatus: "voided", rankSettlementDueAt: "2026-08-25T16:10:00+08:00"
        )).match
        XCTAssertEqual(report.rankSettlement, .voided)
    }

    func testRankSettlementSettledNoneAndLegacyKeepOrdinaryRankDisplay() throws {
        for status in ["settled", "none", "未知状态"] {
            let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail(
                rankSettlementStatus: status, rankSettlementDueAt: "2026-08-25T16:10:00+08:00"
            )).match
            XCTAssertEqual(report.rankSettlement, .counted, status)
        }
        // 旧版服务端不返回结算字段。
        let legacy = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.playerDetail()).match
        XCTAssertNil(legacy.rankSettlementStatus)
        XCTAssertEqual(legacy.rankSettlement, .counted)
    }

    func testRefereeReportTreatsNullArraysAsAbsent() throws {
        let report = try decode(MatchDetailEnvelope.self, from: MatchHistoryFixtures.refereeDetail()).match

        XCTAssertTrue(report.isRefereeView)
        XCTAssertNil(report.myRankDetails)
        XCTAssertNil(report.summaryHighlights)
        XCTAssertNil(report.summaryStats)
    }

    func testRewardStateMapsReadyEmptyAndPending() throws {
        let ready = try decode(
            MatchRewardSummary.self,
            from: MatchHistoryFixtures.rewardSummary(list: [MatchHistoryFixtures.rewardItem()])
        )
        XCTAssertEqual(ready.state, .ready(ready.list))
        XCTAssertEqual(ready.list.first?.rewardTitleName, "夜战王")

        let noReward = try decode(MatchRewardSummary.self, from: MatchHistoryFixtures.rewardSummary())
        XCTAssertEqual(noReward.state, .hidden)

        let pending = try decode(MatchRewardSummary.self, from: MatchHistoryFixtures.rewardSummary(status: "pending"))
        XCTAssertEqual(pending.state, .pending)

        let unknown = try decode(MatchRewardSummary.self, from: MatchHistoryFixtures.rewardSummary(status: "weird"))
        XCTAssertEqual(unknown.state, .hidden)
    }

    func testGameTypeNamesAndScoreUnits() {
        XCTAssertEqual(MatchFormatting.gameTypeName(1), "斯诺克")
        XCTAssertEqual(MatchFormatting.gameTypeName(2), "九球追分")
        XCTAssertEqual(MatchFormatting.gameTypeName(3), "中式八球")
        XCTAssertEqual(MatchFormatting.gameTypeName(4), "美式九球")
        XCTAssertEqual(MatchFormatting.gameTypeName(99), "台球")

        XCTAssertEqual(MatchFormatting.scoreUnit(2), "分")
        XCTAssertEqual(MatchFormatting.scoreUnit(1), "局")
        XCTAssertEqual(MatchFormatting.scoreUnit(4), "局")
    }

    func testCompletionSourceAttributionIsShownOnlyWhenReliable() {
        XCTAssertEqual(MatchFormatting.completionSourceLabel("referee"), "裁判结算")
        XCTAssertEqual(MatchFormatting.completionSourceLabel("player_direct"), "选手直接完成")
        XCTAssertEqual(MatchFormatting.completionSourceLabel("player_confirmed"), "双方确认完成")
        XCTAssertEqual(MatchFormatting.completionSourceLabel("unknown"), "未知")

        XCTAssertTrue(MatchFormatting.hasReliableCompletionSource("referee"))
        XCTAssertFalse(MatchFormatting.hasReliableCompletionSource("unknown"))
        XCTAssertFalse(MatchFormatting.hasReliableCompletionSource(""))
        XCTAssertFalse(MatchFormatting.hasReliableCompletionSource(nil))
    }

    func testDurationFormatsSecondsMinutesAndHours() {
        XCTAssertNil(MatchFormatting.duration(seconds: nil))
        XCTAssertNil(MatchFormatting.duration(seconds: 0))
        XCTAssertEqual(MatchFormatting.duration(seconds: 42), "42秒")
        XCTAssertEqual(MatchFormatting.duration(seconds: 60), "1分钟")
        XCTAssertEqual(MatchFormatting.duration(seconds: 1260), "21分钟")
        XCTAssertEqual(MatchFormatting.duration(seconds: 1275), "21分15秒")
        XCTAssertEqual(MatchFormatting.duration(seconds: 3600), "1小时")
        XCTAssertEqual(MatchFormatting.duration(seconds: 3900), "1小时5分")
    }

    func testRelativeTimeBoundariesMatchListRules() {
        let base = MatchTimeFormatting.parse("2026-10-09T20:30:00+08:00")!
        func text(_ secondsAgo: TimeInterval) -> String {
            MatchTimeFormatting.relative("2026-10-09T20:30:00+08:00", now: base.addingTimeInterval(secondsAgo))
        }

        XCTAssertEqual(text(0), "刚刚")
        XCTAssertEqual(text(59), "刚刚")
        XCTAssertEqual(text(60), "1分钟前")
        XCTAssertEqual(text(3600), "1小时前")
        XCTAssertEqual(text(86_400), "1天前")
        XCTAssertEqual(text(29 * 86_400), "29天前")
        // 超过一个月后显示对局本身的绝对时间，不再显示相对时间。
        XCTAssertEqual(text(30 * 86_400), "2026-10-09 20:30")
    }

    func testAbsoluteTimeUsesShanghaiWallClock() {
        XCTAssertEqual(MatchTimeFormatting.absolute("2026-10-09T20:30:00+08:00"), "2026-10-09 20:30")
        // 同一时刻的 UTC 表示也应显示北京时间。
        XCTAssertEqual(MatchTimeFormatting.absolute("2026-10-09T12:30:00Z"), "2026-10-09 20:30")
    }

    func testUnparsableOrEmptyTimeShowsPendingText() {
        XCTAssertEqual(MatchTimeFormatting.absolute(nil), "时间待同步")
        XCTAssertEqual(MatchTimeFormatting.absolute(""), "时间待同步")
        XCTAssertEqual(MatchTimeFormatting.absolute("not-a-date"), "时间待同步")
        XCTAssertEqual(MatchTimeFormatting.relative("not-a-date"), "时间待同步")
    }
}

/// 测试内用的分页解码形态，与 `MatchHistoryAPI` 的私有响应结构保持一致。
private struct PagedFixtureResponse<Item: Decodable>: Decodable {
    let total: Int
    let list: [Item]
}

private struct MatchDetailEnvelope: Decodable {
    let match: MatchReport
}
