import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/season-{lifecycle,qualification,history}.test.mjs` 的纯规则用例。
@MainActor
final class SeasonRulesTests: XCTestCase {
    // MARK: - 赛季剩余天数与进度

    func testTimelineKeepsTheFullEndDateInTheHalfOpenWindow() {
        let season = makeSeason(startDate: "2026-07-01", endDate: "2026-07-31")
        let timeline = SeasonLifecycleRules.resolveTimeline(
            season,
            now: date("2026-07-31T12:00:00+08:00")
        )

        XCTAssertEqual(timeline.remainDays, 1)
        XCTAssertGreaterThan(timeline.progressPercent, 98)
        XCTAssertLessThan(timeline.progressPercent, 100)
    }

    func testTimelineReachesZeroAtTheNextBusinessDayBoundary() {
        let season = makeSeason(startDate: "2026-07-01", endDate: "2026-07-31")
        let timeline = SeasonLifecycleRules.resolveTimeline(
            season,
            now: date("2026-08-01T00:00:00+08:00")
        )

        XCTAssertEqual(timeline.remainDays, 0)
        XCTAssertEqual(timeline.progressPercent, 100)
    }

    func testTimelineHonorsAuthoritativeTimestampsFromTheConfiguredTimezone() {
        let season = makeSeason(
            startDate: "2026-07-01",
            endDate: "2026-07-31",
            startAt: "2026-07-01T00:00:00-04:00",
            endAtExclusive: "2026-08-01T00:00:00-04:00"
        )
        let timeline = SeasonLifecycleRules.resolveTimeline(
            season,
            now: date("2026-08-01T03:00:00Z")
        )

        XCTAssertEqual(timeline.remainDays, 1)
        XCTAssertLessThan(timeline.progressPercent, 100)
    }

    func testTimelineRejectsInvalidDateOnlyValues() {
        let season = makeSeason(
            startDate: "2026-07-01",
            endDate: "invalid",
            startAt: "",
            endAtExclusive: ""
        )
        let timeline = SeasonLifecycleRules.resolveTimeline(season)

        XCTAssertEqual(timeline.remainDays, 0)
        XCTAssertEqual(timeline.progressPercent, 0)
    }

    // MARK: - 达标进度、净变化与榜单

    func testLegacySeasonWithoutThresholdTreatsEveryRecordAsQualified() {
        let view = SeasonQualificationRules.qualification(
            season: makeSeason(),
            record: makeRecord(matchesPlayed: 1)
        )

        XCTAssertFalse(view.hasThreshold)
        XCTAssertTrue(view.qualified)
        XCTAssertEqual(view.progressText, "")
        XCTAssertEqual(view.thresholdText, "")
    }

    func testV2SeasonReportsCappedProgressUntilServerMarksRecordQualified() {
        let season = makeSeason(qualifyMinMatches: 5, qualifyMinOpponents: 3)
        let pending = SeasonQualificationRules.qualification(
            season: season,
            record: makeRecord(matchesPlayed: 7, distinctOpponents: 1, qualified: false)
        )

        XCTAssertFalse(pending.qualified)
        XCTAssertEqual(pending.progressText, "场次 5/5 · 不同对手 1/3")
        XCTAssertEqual(pending.thresholdText, "本季打满 5 场、遇到 3 位不同对手即参与赛季排名")
        XCTAssertTrue(SeasonQualificationRules.qualification(
            season: season,
            record: makeRecord(qualified: true)
        ).qualified)
        XCTAssertFalse(SeasonQualificationRules.qualification(season: season, record: nil).qualified)
    }

    func testNetChangeSignsTheSeasonDelta() {
        XCTAssertEqual(SeasonQualificationRules.netChange(makeRecord(start: 1250, end: 1310)), "+60")
        XCTAssertEqual(SeasonQualificationRules.netChange(makeRecord(start: 1250, end: 1200)), "-50")
        XCTAssertEqual(SeasonQualificationRules.netChange(makeRecord(start: 1250, end: 1250)), "0")
        XCTAssertEqual(SeasonQualificationRules.netChange(nil), "")
    }

    func testMergeLeaderboardKeepsTheFirstOccurrenceWhenRanksShift() {
        let merged = SeasonQualificationRules.mergeLeaderboard(
            [makeLeaderboardItem(rank: 1, userId: 1), makeLeaderboardItem(rank: 2, userId: 2)],
            [makeLeaderboardItem(rank: 3, userId: 2), makeLeaderboardItem(rank: 4, userId: 3)]
        )

        XCTAssertEqual(merged.map(\.userId), [1, 2, 3])
    }

    func testLeaderboardRankHidesUnqualifiedRowsAndKeepsOldResponsesWorking() {
        XCTAssertEqual(
            SeasonQualificationRules.leaderboardRank(makeLeaderboardItem(rank: 2, userId: 1, qualified: true)),
            SeasonLeaderboardRank(rank: 2, text: "2", unqualified: false)
        )
        XCTAssertEqual(
            SeasonQualificationRules.leaderboardRank(makeLeaderboardItem(rank: 0, userId: 2, qualified: false)),
            SeasonLeaderboardRank(rank: 0, text: "—", unqualified: true)
        )
        XCTAssertEqual(
            SeasonQualificationRules.leaderboardRank(makeLeaderboardItem(rank: 5, userId: 3, qualified: nil)),
            SeasonLeaderboardRank(rank: 5, text: "5", unqualified: false)
        )
    }

    // MARK: - 历届赛季

    func testHistoryRowExposesRankStatsAndReportOnlyWithRecord() {
        let ranked = SeasonHistoryRules.row(
            SeasonHistoryItem(
                season: makeSeason(id: 2, name: "S2", startDate: "2026-08-01", endDate: "2026-08-31", status: 2),
                record: makeRecord(seasonId: 2, matchesPlayed: 8, wins: 6, finalRank: 3, qualified: true)
            ),
            gameType: 1
        )
        XCTAssertEqual(ranked?.name, "S2")
        XCTAssertEqual(ranked?.dateRange, "2026-08-01 ~ 2026-08-31")
        XCTAssertEqual(ranked?.hasRecord, true)
        XCTAssertEqual(ranked?.rankText, "第 3 名")
        XCTAssertEqual(ranked?.matches, 8)
        XCTAssertEqual(ranked?.wins, 6)
        XCTAssertEqual(ranked?.winRate, 75)
        XCTAssertEqual(ranked?.reportURL, "/subPages/season/report?id=2&game_type=1")

        let absent = SeasonHistoryRules.row(
            SeasonHistoryItem(season: makeSeason(id: 1, status: 2), record: nil),
            gameType: 3
        )
        XCTAssertEqual(absent?.hasRecord, false)
        XCTAssertEqual(absent?.reportURL, "")
        XCTAssertEqual(absent?.matches, 0)

        let unranked = SeasonHistoryRules.row(
            SeasonHistoryItem(
                season: makeSeason(id: 4, status: 2),
                record: makeRecord(seasonId: 4, matchesPlayed: 0, wins: 0, finalRank: 0)
            ),
            gameType: 3
        )
        XCTAssertEqual(unranked?.rankText, "未排名")

        let unqualified = SeasonHistoryRules.row(
            SeasonHistoryItem(
                season: makeSeason(id: 5, status: 2, qualifyMinMatches: 5, qualifyMinOpponents: 3),
                record: makeRecord(seasonId: 5, matchesPlayed: 3, wins: 3, finalRank: 0, qualified: false)
            ),
            gameType: 3
        )
        XCTAssertEqual(unqualified?.rankText, "未达标")
        XCTAssertEqual(unqualified?.reportURL, "/subPages/season/report?id=5&game_type=3")

        XCTAssertNil(SeasonHistoryRules.row(SeasonHistoryItem(season: nil, record: nil), gameType: 3))
        XCTAssertEqual(SeasonHistoryRules.reportURL(seasonId: 9, gameType: 4), "/subPages/season/report?id=9&game_type=4")
    }

    func testHistoryHasMoreStopsAtTotalOrAfterAnEmptyPage() {
        XCTAssertTrue(SeasonHistoryRules.hasMore(loaded: 10, total: 25, lastPageCount: 10))
        XCTAssertFalse(SeasonHistoryRules.hasMore(loaded: 25, total: 25, lastPageCount: 5))
        XCTAssertFalse(SeasonHistoryRules.hasMore(loaded: 10, total: 25, lastPageCount: 0))
        XCTAssertFalse(SeasonHistoryRules.hasMore())
    }

    // MARK: - 辅助

    private func date(_ value: String) -> Date {
        ISO8601DateFormatter().date(from: value) ?? .distantPast
    }

    private func makeSeason(
        id: Int64 = 1,
        name: String = "S1",
        startDate: String = "2026-07-01",
        endDate: String = "2026-07-31",
        startAt: String = "",
        endAtExclusive: String = "",
        status: Int = 1,
        qualifyMinMatches: Int = 0,
        qualifyMinOpponents: Int = 0
    ) -> SeasonInfo {
        SeasonInfo(
            id: id,
            name: name,
            startDate: startDate,
            endDate: endDate,
            startAt: startAt,
            endAtExclusive: endAtExclusive,
            status: status,
            rankResetRatio: 1,
            qualifyMinMatches: qualifyMinMatches,
            qualifyMinOpponents: qualifyMinOpponents
        )
    }

    private func makeRecord(
        seasonId: Int64 = 1,
        start: Int = 1250,
        end: Int = 1310,
        matchesPlayed: Int = 1,
        wins: Int = 1,
        finalRank: Int = 0,
        distinctOpponents: Int = 0,
        qualified: Bool = true
    ) -> SeasonRecordInfo {
        SeasonRecordInfo(
            seasonId: seasonId,
            seasonName: "S1",
            startRankScore: start,
            endRankScore: end,
            peakRankScore: max(start, end),
            matchesPlayed: matchesPlayed,
            wins: wins,
            finalRank: finalRank,
            distinctOpponents: distinctOpponents,
            qualified: qualified
        )
    }

    private func makeLeaderboardItem(rank: Int, userId: Int64, qualified: Bool? = true) -> SeasonLeaderboardItem {
        SeasonLeaderboardItem(
            rank: rank,
            userId: userId,
            nickname: "球友",
            avatar: "",
            rankScore: 1400,
            peakScore: 1450,
            matchesPlayed: 10,
            wins: 7,
            qualified: qualified
        )
    }
}
