import XCTest
@testable import iOS

@MainActor
final class GearIndexModelTests: XCTestCase {
    private let minePath = "/api/gear/mine"

    private func makeModel(_ server: MockHTTPServer) async -> (GearIndexModel, GearStore, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        let store = GearStore(session: session)
        return (GearIndexModel(session: session, store: store), store, session)
    }

    func testLoadShowsGroupsCountAndMainLabels() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [
                GearFixtures.itemJSON(1, product: GearFixtures.productJSON()),
                GearFixtures.itemJSON(3, category: "break_jump", roles: ["break"]),
                GearFixtures.itemJSON(5, category: "cue", status: 2),
            ],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()

        XCTAssertEqual(model.items.count, 3)
        XCTAssertEqual(model.groups.cues.map(\.id), [1])
        XCTAssertEqual(model.groups.breakJump.map(\.id), [3])
        XCTAssertEqual(model.groups.retired.map(\.id), [5])
        XCTAssertEqual(model.countText, "3/30")
        XCTAssertFalse(model.atLimit)
        XCTAssertEqual(model.mainLabels(for: GearFixtures.item(1)), ["中式八球 主力"])
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertNil(model.failure)
    }

    func testSecondPageReusesStoreCache() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(1)])))
        let (first, store, session) = await makeModel(server)
        await first.onAppear()

        let second = GearIndexModel(session: session, store: store)
        await second.onAppear()

        XCTAssertEqual(second.items.map(\.id), [1])
        XCTAssertEqual(server.requestCount(minePath), 1)
    }

    func testSetDefaultUpdatesStoreAndRows() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(1)])))
        server.stub("POST", "/api/gear/default/set", .json(GearFixtures.commonJSON()))
        let (model, store, _) = await makeModel(server)
        await model.onAppear()

        let row = model.defaultsView[0]
        let failure = await model.setDefault(row: row, slot: .cue, itemId: 1)

        XCTAssertNil(failure)
        XCTAssertEqual(store.peek()?.defaults, [GearFixtures.mainDefault(3, slot: .cue, itemId: 1)])
        XCTAssertEqual(model.defaultsView[0].cue?.id, 1)
        XCTAssertEqual(server.requests("/api/gear/default/set").first?.bodyJSON["game_type"] as? Int, 3)
    }

    func testSetDefaultFailureReturnsMessageAndKeepsData() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(1)])))
        server.stub("POST", "/api/gear/default/set", .json(GearFixtures.commonJSON(success: false, message: "装备已退役")))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        let failure = await model.setDefault(row: model.defaultsView[0], slot: .cue, itemId: 1)

        XCTAssertEqual(failure, "装备已退役")
        XCTAssertNil(model.defaultsView[0].cue)
    }

    func testEmptyListShowsNoGroups() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(maxItems: 30)))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()

        XCTAssertTrue(model.isEmpty)
        XCTAssertEqual(model.countText, "0/30")
        XCTAssertEqual(model.groups, GearGroups())
        XCTAssertEqual(model.defaultsView.count, 4)
    }

    func testInitialFailureAndRefreshFailureAreSeparated() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .raw("service down", status: 503))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()
        XCTAssertNotNil(model.failure)
        XCTAssertNil(model.refreshFailure)

        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(1)])))
        await model.retry()
        XCTAssertNil(model.failure)
        XCTAssertEqual(model.items.map(\.id), [1])

        server.stub("GET", minePath, .raw("service down", status: 503))
        await model.retry()
        XCTAssertNil(model.failure, "已有数据时刷新失败不打断页面")
        XCTAssertNotNil(model.refreshFailure)
        XCTAssertEqual(model.items.map(\.id), [1])
    }
}

@MainActor
final class GearFriendModelTests: XCTestCase {
    private let userPath = "/api/gear/user"

    func testSuccessShowsGroupsDefaultsAndOwnerTitle() async {
        let server = MockHTTPServer()
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON(
            items: [
                GearFixtures.itemJSON(1),
                GearFixtures.itemJSON(3, category: "break_jump", roles: ["break"]),
            ],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")

        await model.load()

        XCTAssertEqual(model.ownerTitle, "TA 的装备")
        XCTAssertEqual(model.groups.cues.map(\.id), [1])
        XCTAssertEqual(model.groups.breakJump.map(\.id), [3])
        XCTAssertEqual(model.mainLabels(for: GearFixtures.item(1)), ["中式八球 主力"])
        XCTAssertEqual(model.mainRows.map(\.gameType), [3, 2, 1, 4])
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.loading)
        XCTAssertEqual(server.requests(userPath).first?.query["user_id"], "42")
    }

    func testNoDefaultsHidesMainRows() async {
        let server = MockHTTPServer()
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON(items: [GearFixtures.itemJSON(1)])))
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 42, ownerName: "老王")

        await model.load()

        XCTAssertEqual(model.ownerTitle, "老王的装备")
        XCTAssertTrue(model.mainRows.isEmpty, "一个主力都没有时不展示整块")
    }

    func testNotFriendFailureIsNotRetryable() async {
        let server = MockHTTPServer()
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON(
            success: false,
            message: "仅好友可以查看",
            reason: "not_friend"
        )))
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")

        await model.load()

        XCTAssertEqual(model.failure?.title, "仅好友可以查看")
        XCTAssertEqual(model.failure?.description, "加为好友后，就能看到 TA 的装备。")
        XCTAssertEqual(model.failure?.retryable, false)
    }

    func testMissingUserIdFailsWithoutRequest() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 0, ownerName: "TA")

        await model.load()

        XCTAssertEqual(model.failure?.title, "用户不存在")
        XCTAssertEqual(model.failure?.description, "请返回后重新进入。")
        XCTAssertEqual(model.failure?.retryable, false)
        XCTAssertEqual(server.requestCount(userPath), 0)
    }

    func testServerFailureIsRetryable() async {
        let server = MockHTTPServer()
        server.stub("GET", userPath, .raw("service down", status: 503))
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")

        await model.load()

        XCTAssertNotNil(model.failure?.title)
        XCTAssertEqual(model.failure?.retryable, true)
    }

    func testEmptyListIsSuccessful() async {
        let server = MockHTTPServer()
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON()))
        let session = await makeRestoredAccountSession(server)
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")

        await model.load()

        XCTAssertTrue(model.items.isEmpty)
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.loading)
    }
}
