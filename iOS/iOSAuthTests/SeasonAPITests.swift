import XCTest
@testable import iOS

@MainActor
final class SeasonAPITests: XCTestCase {
    private let currentPath = "/api/season/current"
    private let leaderboardPath = "/api/season/leaderboard"
    private let overviewPath = "/api/season/overview"
    private let reportPath = "/api/season/report"
    private let historyPath = "/api/season/history"

    func testGuestReadsCurrentSeasonAndLeaderboardWithoutAuthorization() async throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", currentPath, .json(SeasonFixtures.current()))
        server.stub("GET", leaderboardPath, .json(SeasonFixtures.leaderboard()))
        let api = SeasonAPI(session: session)

        let current = try await api.current()
        let leaderboard = try await api.leaderboard(seasonId: 2, gameType: 2, page: 2)

        XCTAssertEqual(current.seasonState, "active")
        XCTAssertEqual(current.season?.name, "S3")
        XCTAssertEqual(leaderboard.list.map(\.userId), [7])
        XCTAssertNil(server.requests(currentPath).first?.bearerToken)
        let request = server.requests(leaderboardPath).first
        XCTAssertNil(request?.bearerToken)
        XCTAssertEqual(request?.query["season_id"], "2")
        XCTAssertEqual(request?.query["game_type"], "2")
        XCTAssertEqual(request?.query["page"], "2")
        XCTAssertEqual(request?.query["page_size"], "20")
    }

    func testLoggedInReadsUseBearerAndExpectedQuery() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview()))
        server.stub("GET", reportPath, .json(SeasonFixtures.report()))
        server.stub("GET", historyPath, .json(SeasonFixtures.history()))
        let api = SeasonAPI(session: session)

        _ = try await api.overview(gameType: 2)
        _ = try await api.report(seasonId: 2, gameType: 1)
        _ = try await api.history(gameType: 3, page: 2)

        XCTAssertEqual(server.requests(overviewPath).first?.query["game_type"], "2")
        XCTAssertEqual(server.requests(reportPath).first?.query["season_id"], "2")
        XCTAssertEqual(server.requests(reportPath).first?.query["game_type"], "1")
        XCTAssertEqual(server.requests(historyPath).first?.query["game_type"], "3")
        XCTAssertEqual(server.requests(historyPath).first?.query["page"], "2")
        XCTAssertEqual(server.requests(historyPath).first?.query["page_size"], "10")
        for path in [overviewPath, reportPath, historyPath] {
            XCTAssertEqual(server.requests(path).first?.bearerToken, AuthFixtures.oldAccess)
        }
    }

    func testResponsesDecodeOptionalBlocksAndQualifiedFlag() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", overviewPath, .json(SeasonFixtures.overview(
            record: nil,
            leaderboard: [SeasonFixtures.leaderboardItem(rank: 0, userId: 8, qualified: false)],
            availability: ["record": false]
        )))
        server.stub("GET", reportPath, .json(SeasonFixtures.emptyReport))
        server.stub("GET", historyPath, .json(SeasonFixtures.history(list: [
            SeasonFixtures.historyItem(record: nil),
        ])))
        let api = SeasonAPI(session: session)

        let overview = try await api.overview(gameType: 3)
        let report = try await api.report(seasonId: 2, gameType: 3)
        let history = try await api.history(gameType: 3, page: 1)

        XCTAssertNil(overview.record)
        XCTAssertEqual(overview.availability?["record"], false)
        XCTAssertEqual(overview.leaderboard.first?.qualified, false)
        XCTAssertNil(report.report)
        XCTAssertNil(history.list.first?.record)
        XCTAssertEqual(history.list.first?.season?.id, 2)
    }
}
