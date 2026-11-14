import XCTest
@testable import iOS

@MainActor
final class SeasonModelTests: XCTestCase {
    private let overviewPath = "/api/season/overview"
    private let currentPath = "/api/season/current"
    private let leaderboardPath = "/api/season/leaderboard"

    func testFirstLoadReadsOverviewForTheCurrentGameType() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview()))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertTrue(model.isActiveSeason)
        XCTAssertEqual(model.season?.name, "S3")
        XCTAssertEqual(model.statusText, "进行中")
        XCTAssertEqual(model.myRecord?.matchesPlayed, 8)
        XCTAssertEqual(model.winRate, 75)
        XCTAssertEqual(model.netChangeText, "+60")
        XCTAssertEqual(model.myRank, 3)
        XCTAssertTrue(model.myRankAvailable)
        XCTAssertEqual(model.leaderboardRows.map(\.rank.text), ["1"])
        XCTAssertFalse(model.seasonDataUnavailable)

        let request = server.requests(overviewPath).first
        XCTAssertEqual(request?.query["game_type"], "3")
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
    }

    func testSecondAppearanceDoesNotRequestAgain() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview()))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()
        await model.loadIfNeeded()

        XCTAssertEqual(server.requestCount(overviewPath), 1)
    }

    func testGuestReadsOnlyPublicSeasonAndLeaderboard() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", currentPath, .json(SeasonFixtures.current()))
        server.stub("GET", leaderboardPath, .json(SeasonFixtures.leaderboard()))
        let model = SeasonModel(session: session, gameType: 2)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertTrue(model.isGuest)
        XCTAssertEqual(server.requestCount(overviewPath), 0)
        XCTAssertEqual(server.requestCount(currentPath), 1)
        XCTAssertEqual(server.requests(leaderboardPath).first?.query["game_type"], "2")
        XCTAssertNil(server.requests(leaderboardPath).first?.bearerToken)
        XCTAssertNil(model.myRecord)
        XCTAssertFalse(model.myRankAvailable)
        XCTAssertEqual(model.leaderboardRows.count, 1)
    }

    func testSwitchingGameTypeKeepsSeasonCardAndReplacesData() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath) { request in
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            return .json(SeasonFixtures.overview(
                record: SeasonFixtures.record(seasonName: "S3", matchesPlayed: gameType),
                leaderboard: [SeasonFixtures.leaderboardItem(rank: 1, userId: Int64(gameType))]
            ))
        }
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()

        await model.selectGameType(2)

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.currentGameTypeLabel, "九球追分")
        XCTAssertEqual(model.season?.name, "S3", "切换球种必须保留赛季卡片")
        XCTAssertEqual(model.myRecord?.matchesPlayed, 2)
        XCTAssertEqual(model.leaderboardRows.map(\.id), [2])
        XCTAssertFalse(model.isSwitchingGameType)
        XCTAssertEqual(server.requests(overviewPath).last?.query["game_type"], "2")
    }

    func testNotStartedAndUnavailableSeasonsShowLifecycleStateWithoutData() async {
        for state in ["not_started", "unavailable"] {
            let server = MockHTTPServer()
            let session = await makeRestoredAccountSession(server)
            server.stub("GET", overviewPath, .json(SeasonFixtures.overview(
                seasonState: state,
                season: nil,
                record: nil,
                myRank: 0,
                leaderboard: []
            )))
            let model = SeasonModel(session: session)

            await model.loadIfNeeded()

            XCTAssertEqual(model.phase, .ready)
            XCTAssertFalse(model.isActiveSeason)
            XCTAssertNil(model.season)
            XCTAssertNil(model.myRecord)
            XCTAssertTrue(model.leaderboardRows.isEmpty)
            XCTAssertFalse(model.hasMoreLeaderboard)
            XCTAssertEqual(
                model.seasonStateView.mode,
                state == "unavailable" ? .unavailable : .notStarted
            )
        }
    }

    func testEmptyRecordAndLeaderboardAreNotFailures() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(
            record: nil,
            myRank: 0,
            leaderboard: []
        )))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertTrue(model.isActiveSeason)
        XCTAssertNil(model.myRecord)
        XCTAssertEqual(model.winRate, 0)
        XCTAssertEqual(model.netChangeText, "")
        XCTAssertTrue(model.leaderboardRows.isEmpty)
        XCTAssertTrue(model.qualification.qualified, "无门槛赛季的空记录仍可参与排名")
    }

    func testInitialFailureThenRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .raw("service down", status: 503))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()

        guard case .failed(let failure) = model.phase else {
            XCTFail("首次加载失败应进入失败态，实际为 \(model.phase)")
            return
        }
        XCTAssertEqual(failure.category, .server)
        XCTAssertFalse(model.isActiveSeason)

        server.stub("GET", overviewPath, .json(SeasonFixtures.overview()))
        await model.retry()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.season?.name, "S3")
    }

    func testPartialAvailabilityHidesOnlyUnavailableBlocks() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(
            record: SeasonFixtures.record(matchesPlayed: 9),
            availability: ["record": false, "leaderboard": false, "my_rank": false]
        )))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.season?.name, "S3")
        XCTAssertTrue(model.recordUnavailable)
        XCTAssertTrue(model.leaderboardUnavailable)
        XCTAssertTrue(model.seasonDataUnavailable)
        XCTAssertNil(model.myRecord, "不可用的个人记录不能被当成真实数据")
        XCTAssertTrue(model.leaderboardRows.isEmpty)
        XCTAssertFalse(model.myRankAvailable)
        XCTAssertEqual(model.myRank, 0)
        XCTAssertFalse(model.hasMoreLeaderboard)
    }

    func testUnavailableRankKeepsVisibleRecord() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(
            record: SeasonFixtures.record(matchesPlayed: 4, wins: 3),
            myRank: 0,
            availability: ["my_rank": false]
        )))
        let model = SeasonModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.myRecord?.matchesPlayed, 4, "名次分区不可用时仍应保留记录")
        XCTAssertFalse(model.myRankAvailable)
        XCTAssertEqual(model.myRank, 0)
        XCTAssertFalse(model.seasonDataUnavailable)
    }

    func testLoadMoreLeaderboardMergesDuplicateRowsAndAdvancesPage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let firstPage = (1...20).map { SeasonFixtures.leaderboardItem(rank: $0, userId: Int64($0)) }
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(leaderboard: firstPage)))
        server.stub("GET", leaderboardPath, .json(SeasonFixtures.leaderboard(
            total: 23,
            list: [
                SeasonFixtures.leaderboardItem(rank: 20, userId: 20),
                SeasonFixtures.leaderboardItem(rank: 21, userId: 21),
                SeasonFixtures.leaderboardItem(rank: 22, userId: 22),
            ]
        )))
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()

        XCTAssertTrue(model.hasMoreLeaderboard)
        await model.loadMoreLeaderboard()

        XCTAssertEqual(model.leaderboardRows.map(\.id), (1...22).map { Int64($0) })
        XCTAssertEqual(model.page, 2)
        XCTAssertFalse(model.hasMoreLeaderboard)
        XCTAssertFalse(model.isLoadingMore)
        XCTAssertEqual(server.requests(leaderboardPath).first?.query["page"], "2")
    }

    func testLoadMoreFailureKeepsRowsAndKeepsPage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let firstPage = (1...20).map { SeasonFixtures.leaderboardItem(rank: $0, userId: Int64($0)) }
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(leaderboard: firstPage)))
        server.stub("GET", leaderboardPath, .raw("service down", status: 503))
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()

        await model.loadMoreLeaderboard()

        XCTAssertEqual(model.leaderboardRows.count, 20)
        XCTAssertEqual(model.page, 1)
        XCTAssertNotNil(model.refreshFailure)
        XCTAssertFalse(model.isLoadingMore)
    }

    func testLateOverviewAfterLogoutIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", overviewPath) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SeasonFixtures.overview())
        }
        let model = SeasonModel(session: session)

        let load = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount(self.overviewPath) == 1 }
        session.logout()
        model.syncIdentity()
        gate.signal()
        await load.value

        XCTAssertEqual(model.phase, .idle)
        XCTAssertNil(model.season)
        XCTAssertTrue(model.leaderboardRows.isEmpty)
        XCTAssertEqual(server.requestCount(overviewPath), 1, "退出后旧请求结果不得落到新身份")
    }
}
