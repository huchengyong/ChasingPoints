import Foundation
import XCTest
@testable import iOS

@MainActor
final class ProfileAPITests: XCTestCase {
    func testOverviewReturnsEverySuccessfulSection() async throws {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(),
            reputation: ProfileFixtures.reputation(),
            memberStatus: ProfileFixtures.memberStatusJSON(growthLevel: 3, growthPoints: 78, todayGrowthCount: 3),
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(
                popupEnabled: false, submittedVenueName: "南山台球会"
            )
        )))

        let overview = try await ProfileAPI(session: session).overview()

        XCTAssertEqual(overview.availableStats?.totalMatches, 86)
        XCTAssertEqual(overview.availableStats?.wins, 51)
        XCTAssertEqual(overview.availableStats?.losses, 35)
        XCTAssertEqual(overview.availableStats?.winRate, 59.3)
        XCTAssertEqual(overview.availableStats?.maxWinStreak, 7)
        XCTAssertEqual(overview.availableReputation?.statusText, "良好")
        XCTAssertEqual(overview.availableMemberStatus?.planCode, "monthly")
        XCTAssertEqual(overview.availableMemberStatus?.growthLevel, 3)
        XCTAssertEqual(overview.availableMemberStatus?.growthPoints, 78)
        XCTAssertEqual(overview.availableMemberStatus?.todayGrowthCount, 3)
        XCTAssertEqual(overview.availableFavoriteVenueReward?.rewardDays, 30)
        XCTAssertEqual(overview.availableFavoriteVenueReward?.popupEnabled, false)
        XCTAssertEqual(overview.availableFavoriteVenueReward?.submittedVenueName, "南山台球会")
        XCTAssertEqual(overview.availableFavoriteVenueReward?.status, "not_started")
        XCTAssertEqual(server.requests("/api/user/overview").first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testOverviewDropsFailedAndMissingSectionsWithoutAffectingOthers() async throws {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(success: true),
            reputation: ProfileFixtures.reputation(success: false),
            memberStatus: NSNull(),
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(success: false)
        )))

        let overview = try await ProfileAPI(session: session).overview()

        XCTAssertNotNil(overview.availableStats, "成功的分区不受其他分区影响")
        XCTAssertNotNil(overview.stats, "失败但存在的分区仍保留原始响应")
        XCTAssertNil(overview.availableReputation)
        XCTAssertNil(overview.availableMemberStatus)
        XCTAssertNil(overview.availableFavoriteVenueReward)
    }

    func testOverviewTopLevelFailureThrows() async {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/user/overview", .json(["success": false, "message": "读取失败"]))

        do {
            _ = try await ProfileAPI(session: session).overview()
            XCTFail("顶层 success=false 应当抛错")
        } catch {
            assertAPIError(error, kind: .business, message: "读取失败")
        }
    }

    func testRankInfosKeepOnlySupportedGameTypesWithRankInfo() async throws {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([
            ProfileFixtures.rankInfoItem(gameType: 1, rankInfo: ProfileFixtures.rankInfoJSON(level: 2)),
            ProfileFixtures.rankInfoItem(gameType: 2, rankInfo: nil),
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(level: 4, placementRemaining: 2)
            ),
            ProfileFixtures.rankInfoItem(gameType: 4, rankInfo: ProfileFixtures.rankInfoJSON(level: 1)),
            ProfileFixtures.rankInfoItem(gameType: 5, rankInfo: ProfileFixtures.rankInfoJSON(level: 6)),
            ProfileFixtures.rankInfoItem(gameType: 0, rankInfo: ProfileFixtures.rankInfoJSON(level: 6)),
        ])))

        let map = try await ProfileAPI(session: session).rankInfos()

        XCTAssertEqual(map.keys.sorted(), [1, 3, 4])
        XCTAssertEqual(map[1]?.level, 2)
        XCTAssertEqual(map[3]?.level, 4)
        XCTAssertEqual(map[3]?.placementRemaining, 2)
        XCTAssertNil(map[1]?.placementRemaining, "旧服务端不返回 placement_remaining")
        XCTAssertEqual(server.requests("/api/rank/infos").first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testRankInfosEmptyListReturnsEmptyMap() async throws {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([])))

        let map = try await ProfileAPI(session: session).rankInfos()

        XCTAssertTrue(map.isEmpty)
    }

    func testMatchQRCodeReturnsContentAndDropsInviteToken() async throws {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode()))

        let code = try await ProfileAPI(session: session).matchQRCode()

        XCTAssertEqual(code.qrcodeData, "https://zhuifen.cn/invite?code=ABC123")
        XCTAssertEqual(code.expiresInSeconds, 300)
        XCTAssertEqual(server.requests("/api/match/qrcode").first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testMatchQRCodeEmptyContentThrows() async {
        let server = MockHTTPServer()
        let session = makeProfileSession(server)
        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode(qrcodeData: "")))

        do {
            _ = try await ProfileAPI(session: session).matchQRCode()
            XCTFail("空二维码正文应当抛错")
        } catch {
            assertAPIError(error, kind: .invalidResponse, message: "二维码加载失败")
        }
    }

    func testSignedOutReadsThrowWithoutAnyRequest() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let api = ProfileAPI(session: session)

        await assertSignedOut { _ = try await api.overview() }
        await assertSignedOut { _ = try await api.rankInfos() }
        await assertSignedOut { _ = try await api.matchQRCode() }

        XCTAssertEqual(server.requestCount("/api/user/overview"), 0)
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 0)
        XCTAssertEqual(server.requestCount("/api/match/qrcode"), 0)
    }

    private func makeProfileSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
    }

    private func assertSignedOut(
        file: StaticString = #filePath,
        line: UInt = #line,
        _ operation: () async throws -> Void
    ) async {
        do {
            try await operation()
            XCTFail("未登录时不应成功", file: file, line: line)
        } catch {
            XCTAssertEqual(error.localizedDescription, "登录状态已变更", file: file, line: line)
        }
    }
}
