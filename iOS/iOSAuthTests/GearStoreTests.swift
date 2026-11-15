import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/gear-cache.test.mjs`：TTL、single-flight、失效与身份隔离。
/// 网络时序用带超时的 DispatchSemaphore 控制，每个用例不超过 2 秒。
@MainActor
final class GearStoreTests: XCTestCase {
    private var now = Date(timeIntervalSince1970: 1_700_000_000)
    private let minePath = "/api/gear/mine"

    private func makeStore(_ server: MockHTTPServer) async -> (GearStore, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        return (GearStore(session: session, now: { self.now }), session)
    }

    func testFirstLoadFetchesOnceAndTTLReusesCache() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "first")))
        let (store, _) = await makeStore(server)

        let first = try await store.load()
        now = now.addingTimeInterval(GearStore.ttl - 1)
        let again = try await store.load()

        XCTAssertEqual(first?.message, "first")
        XCTAssertEqual(again, first)
        XCTAssertEqual(server.requestCount(minePath), 1)
        XCTAssertEqual(store.peek()?.message, "first")
    }

    func testExpiredEntryIsFetchedAgain() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-1")))
        let (store, _) = await makeStore(server)
        _ = try await store.load()

        now = now.addingTimeInterval(GearStore.ttl)
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-2")))
        let refreshed = try await store.load()

        XCTAssertEqual(refreshed?.message, "call-2")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testForceBypassesTTL() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-1")))
        let (store, _) = await makeStore(server)
        _ = try await store.load()

        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-2")))
        let forced = try await store.load(force: true)

        XCTAssertEqual(forced?.message, "call-2")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testConcurrentReadsShareSingleRequest() async throws {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", minePath) { _ in
            _ = gate.wait(timeout: .now() + 2)
            return .json(GearFixtures.mineJSON(message: "shared"))
        }
        let (store, _) = await makeStore(server)

        let reads = (0..<3).map { _ in Task { try await store.load() } }
        await waitUntil { server.requestCount(self.minePath) == 1 }
        gate.signal()
        var results: [GearMineResponse?] = []
        for read in reads { results.append(try await read.value) }

        XCTAssertTrue(results.allSatisfy { $0?.message == "shared" })
        XCTAssertEqual(server.requestCount(minePath), 1)
    }

    func testInvalidateMakesNextReadGoToServer() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-1")))
        let (store, _) = await makeStore(server)
        _ = try await store.load()

        store.invalidate()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-2")))
        let next = try await store.load()

        XCTAssertEqual(next?.message, "call-2")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testWriteDuringReadIsNotHiddenByOlderResponse() async throws {
        let server = MockHTTPServer()
        let counter = LockedCounter()
        let gates = [DispatchSemaphore(value: 0), DispatchSemaphore(value: 0)]
        let messages = ["before-write", "after-write"]
        server.stub("GET", minePath) { _ in
            let index = counter.next() - 1
            _ = gates[index].wait(timeout: .now() + 2)
            return .json(GearFixtures.mineJSON(message: messages[index]))
        }
        let (store, _) = await makeStore(server)

        let stale = Task { try await store.load() }
        await waitUntil { server.requestCount(self.minePath) == 1 }
        // 请求进行中发生写操作：失效后立刻再读，必须发起新请求而不是复用旧请求。
        store.invalidate()
        let fresh = Task { try await store.load() }
        await waitUntil { server.requestCount(self.minePath) == 2 }

        gates[1].signal()
        let freshResult = try await fresh.value
        XCTAssertEqual(freshResult?.message, "after-write")

        gates[0].signal()
        let staleResult = try await stale.value
        XCTAssertEqual(staleResult?.message, "after-write", "旧响应必须交回最新结果")
        XCTAssertEqual(store.peek()?.message, "after-write")
    }

    func testOldResponseAfterInvalidationIsShownOnceButNotCached() async throws {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        let counter = LockedCounter()
        server.stub("GET", minePath) { _ in
            if counter.next() == 1 {
                _ = gate.wait(timeout: .now() + 2)
                return .json(GearFixtures.mineJSON(message: "old"))
            }
            return .json(GearFixtures.mineJSON(message: "second"))
        }
        let (store, _) = await makeStore(server)

        let read = Task { try await store.load() }
        await waitUntil { server.requestCount(self.minePath) == 1 }
        store.invalidate()
        gate.signal()

        let readResult = try await read.value
        XCTAssertEqual(readResult?.message, "old")
        XCTAssertNil(store.peek())

        let next = try await store.load()
        XCTAssertEqual(next?.message, "second")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testSwitchingAccountsClearsCacheAndDropsLateResponse() async throws {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        let counter = LockedCounter()
        server.stub("GET", minePath) { _ in
            if counter.next() == 1 {
                _ = gate.wait(timeout: .now() + 2)
                return .json(GearFixtures.mineJSON(message: "old-account"))
            }
            return .json(GearFixtures.mineJSON(message: "new-account"))
        }
        let (store, session) = await makeStore(server)

        let late = Task { try await store.load() }
        await waitUntil { server.requestCount(self.minePath) == 1 }

        session.logout()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            access: "second-access",
            refresh: "second-refresh",
            user: AuthFixtures.userInfo(id: 2, nickname: "另一个人")
        )))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        let fresh = try await store.load()
        XCTAssertEqual(fresh?.message, "new-account")

        gate.signal()
        let lateResult = try await late.value
        XCTAssertNil(lateResult, "旧账号的迟到响应必须丢弃")
        XCTAssertEqual(store.peek()?.message, "new-account")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testNewAuthGenerationOfSameUserAlsoClearsCache() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-1")))
        let (store, session) = await makeStore(server)
        _ = try await store.load()

        session.logout()
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        XCTAssertNil(store.peek())
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "call-2")))
        let after = try await store.load()
        XCTAssertEqual(after?.message, "call-2")
        XCTAssertEqual(server.requestCount(minePath), 2)
    }

    func testLoggedOutStoreNeverReads() async throws {
        let server = MockHTTPServer()
        let (store, session) = await makeStore(server)
        session.logout()

        let result = try await store.load()
        XCTAssertNil(result)
        XCTAssertNil(store.peek())
        XCTAssertEqual(server.requestCount(minePath), 0)
    }

    func testFailedRequestThrowsAndKeepsCacheWithoutSticking() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "good")))
        let (store, _) = await makeStore(server)
        _ = try await store.load()

        server.stub("GET", minePath, .json(GearFixtures.mineJSON(success: false, message: "服务繁忙")))
        do {
            _ = try await store.load(force: true)
            XCTFail("应抛出业务失败")
        } catch {
            XCTAssertEqual((error as? APIError)?.message, "服务繁忙")
        }
        XCTAssertEqual(store.peek()?.message, "good")

        server.stub("GET", minePath, .json(GearFixtures.mineJSON(message: "again")))
        let retried = try await store.load(force: true)
        XCTAssertEqual(retried?.message, "again")
    }

    func testLocalWritesUpdateCacheWithoutRefetch() async throws {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [GearFixtures.itemJSON(1)],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let (store, _) = await makeStore(server)
        _ = try await store.load()

        store.applyCreated(GearFixtures.item(2, category: "break_jump", roles: ["break"]))
        XCTAssertEqual(store.peek()?.items.map(\.id), [1, 2])

        store.applyUpdated(GearFixtures.item(1, status: .retired))
        XCTAssertEqual(store.peek()?.items.first?.status, GearItemStatus.retired.rawValue)
        XCTAssertTrue(store.peek()?.defaults.isEmpty == true, "退役后本地去掉主力设置")

        store.applyDefault(gameType: 2, slot: .breakSlot, itemId: 2)
        XCTAssertEqual(store.peek()?.defaults, [GearFixtures.mainDefault(2, slot: .breakSlot, itemId: 2)])
        store.applyDefault(gameType: 2, slot: .breakSlot, itemId: 0)
        XCTAssertTrue(store.peek()?.defaults.isEmpty == true)

        store.applyDeleted(2)
        XCTAssertEqual(store.peek()?.items.map(\.id), [1])
        XCTAssertEqual(server.requestCount(minePath), 1)
    }

    func testJustAddedAndPickHandoffUseTokens() async throws {
        let server = MockHTTPServer()
        let (store, _) = await makeStore(server)

        store.markJustAdded(7)
        XCTAssertTrue(store.consumeJustAdded(7))
        XCTAssertFalse(store.consumeJustAdded(7))

        var picked: GearProductInfo?
        store.beginPick(token: "t1") { picked = $0 }
        store.finishPick(token: "wrong", product: GearFixtures.product())
        XCTAssertNil(picked)
        store.finishPick(token: "t1", product: GearFixtures.product(5))
        XCTAssertEqual(picked?.id, 5)

        store.beginPick(token: "t2") { picked = $0 }
        store.cancelPick(token: "t2")
        store.finishPick(token: "t2", product: GearFixtures.product(6))
        XCTAssertEqual(picked?.id, 5, "取消后的结果不再回传")
    }
}
