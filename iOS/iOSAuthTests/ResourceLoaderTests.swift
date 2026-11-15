import Foundation
import XCTest
@testable import iOS

@MainActor
final class ResourceLoaderTests: XCTestCase {
    func testFirstLoadRequestsOnceAndLoadedDoesNotRequestAgain() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(7))
        let loader = makeLoader(session)

        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .loaded(7))
        XCTAssertEqual(loader.value, 7)
        XCTAssertFalse(loader.isLoading)

        await loader.loadIfNeeded()
        await loader.loadIfNeeded()

        XCTAssertEqual(server.requestCount("/api/resource"), 1)
    }

    func testSignedInWithoutIdentityStaysIdleAndDoesNotRequest() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let loader = makeLoader(session)

        await loader.loadIfNeeded()

        XCTAssertEqual(server.requestCount("/api/resource"), 0)
        XCTAssertEqual(loader.phase, .idle)
        XCTAssertNil(loader.failureMessage)
    }

    func testAnyoneAccessRequestsAsGuestWithoutAuthorization() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        server.stub("GET", "/api/public", resourceStub(3))
        let loader = ResourceLoader<Int>(session: session, access: .anyone) {
            let response: ResourceValue = try await session.optionalAuthRequest("/api/public")
            return response.value
        }

        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .loaded(3))
        XCTAssertEqual(server.requestCount("/api/public"), 1)
        XCTAssertNil(server.requests("/api/public").first?.bearerToken)
    }

    func testLateResultAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/resource") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return resourceStub(1)
        }
        let loader = makeLoader(session)

        let load = Task { await loader.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/resource") == 1 }
        session.logout()
        gate.signal()
        await load.value

        XCTAssertNil(loader.value, "退出后旧账号的结果不得写入")
        if case .loaded = loader.phase {
            XCTFail("退出后不得留下已加载状态：\(loader.phase)")
        }

        loader.syncIdentity()
        XCTAssertEqual(loader.phase, .idle)
        await loader.loadIfNeeded()
        XCTAssertEqual(server.requestCount("/api/resource"), 1, "退出后不得使用旧账号继续请求")
    }

    func testFirstLoadBusinessFailureIsFailureAndRetrySucceeds() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", .json(["success": false, "message": "读取失败"]))
        let loader = makeLoader(session)

        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .failed("读取失败"))
        XCTAssertNil(loader.value, "失败不能显示为已加载")
        XCTAssertEqual(loader.failureMessage, "读取失败")

        server.stub("GET", "/api/resource", resourceStub(9))
        await loader.retry()

        XCTAssertEqual(loader.phase, .loaded(9))
        XCTAssertNil(loader.failureMessage)
        XCTAssertEqual(server.requestCount("/api/resource"), 2)
    }

    func testRefreshFailureKeepsValueWithBannerThenRecovers() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        let loader = makeLoader(session)
        await loader.loadIfNeeded()

        server.stub("GET", "/api/resource", .raw("service down", status: 503))
        await loader.refresh()

        XCTAssertEqual(loader.phase, .loaded(1))
        XCTAssertEqual(loader.value, 1)
        XCTAssertFalse(loader.isRefreshing)
        XCTAssertEqual(loader.refreshError, "服务暂时不可用，请稍后重试")

        server.stub("GET", "/api/resource", resourceStub(2))
        await loader.refresh()

        XCTAssertEqual(loader.phase, .loaded(2))
        XCTAssertNil(loader.refreshError)
    }

    func testInvalidateKeepsOldDataWhileReloadingAndOnlyThenRequests() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        let loader = makeLoader(session)
        await loader.loadIfNeeded()
        await loader.loadIfNeeded()
        XCTAssertEqual(server.requestCount("/api/resource"), 1, "未过期时不得重复请求")

        loader.invalidate()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/resource") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return resourceStub(2)
        }
        let reload = Task { await loader.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/resource") == 2 }

        XCTAssertEqual(loader.phase, .loaded(1), "刷新期间旧数据仍可见")
        XCTAssertTrue(loader.isRefreshing)

        gate.signal()
        await reload.value
        XCTAssertEqual(loader.phase, .loaded(2))
    }

    func testInvalidateKeepsStaleWhenReloadFailsSoNextAppearanceRetries() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        let loader = makeLoader(session)
        await loader.loadIfNeeded()

        loader.invalidate()
        server.stub("GET", "/api/resource", .raw("service down", status: 503))
        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .loaded(1), "刷新失败时旧数据保持不变")
        XCTAssertEqual(loader.value, 1)
        XCTAssertEqual(loader.refreshError, "服务暂时不可用，请稍后重试")

        server.stub("GET", "/api/resource", resourceStub(2))
        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .loaded(2), "失败后过期标记仍在，下次出现应重新读取")
        XCTAssertNil(loader.refreshError)
        XCTAssertEqual(server.requestCount("/api/resource"), 3)
    }

    func testFirstLoadServerFailureCarriesCategoryAndRetryClearsIt() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", .raw("service down", status: 503))
        let loader = makeLoader(session)

        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .failed("服务暂时不可用，请稍后重试"))
        XCTAssertEqual(loader.failure?.category, .server)

        server.stub("GET", "/api/resource", resourceStub(9))
        await loader.retry()

        XCTAssertEqual(loader.phase, .loaded(9))
        XCTAssertNil(loader.failure)
    }

    func testRefreshFailureOnlySetsRefreshError() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        let loader = makeLoader(session)
        await loader.loadIfNeeded()
        XCTAssertNil(loader.failure)

        server.stub("GET", "/api/resource", .raw("service down", status: 503))
        await loader.refresh()

        XCTAssertEqual(loader.phase, .loaded(1))
        XCTAssertEqual(loader.refreshError, "服务暂时不可用，请稍后重试")
        XCTAssertNil(loader.failure, "刷新失败只记 refreshError")
    }

    func testCancelledFirstLoadReturnsToIdleWithoutError() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", .failure(.cancelled))
        let loader = makeLoader(session)

        await loader.loadIfNeeded()

        XCTAssertEqual(loader.phase, .idle)
        XCTAssertNil(loader.failureMessage)

        server.stub("GET", "/api/resource", resourceStub(5))
        await loader.loadIfNeeded()
        XCTAssertEqual(loader.phase, .loaded(5))
    }

    func testOnlyLatestRequestResultIsAccepted() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        let firstGate = DispatchSemaphore(value: 0)
        let secondGate = DispatchSemaphore(value: 0)
        let attempts = LockedCounter()
        server.stub("GET", "/api/resource") { _ in
            if attempts.next() == 1 {
                _ = firstGate.wait(timeout: .now() + 5)
                return resourceStub(1)
            }
            _ = secondGate.wait(timeout: .now() + 5)
            return resourceStub(2)
        }
        let loader = makeLoader(session)

        let first = Task { await loader.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/resource") == 1 }
        let second = Task { await loader.refresh() }
        await waitUntil { server.requestCount("/api/resource") == 2 }

        secondGate.signal()
        await second.value
        XCTAssertEqual(loader.phase, .loaded(2))

        firstGate.signal()
        await first.value
        XCTAssertEqual(loader.phase, .loaded(2), "先发的请求后返回时不得覆盖较新的结果")
    }

    func testLoadIfNeededSkipsRefreshWithinMaxAge() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        var now = Date(timeIntervalSince1970: 1_700_000_000)
        let loader = makeLoader(session, now: { now })

        await loader.loadIfNeeded(maxAge: 60)
        now = now.addingTimeInterval(59)
        await loader.loadIfNeeded(maxAge: 60)

        XCTAssertEqual(server.requestCount("/api/resource"), 1, "未超过 maxAge 时不得重读")
        XCTAssertEqual(loader.phase, .loaded(1))
    }

    func testLoadIfNeededRefreshesAfterMaxAgeKeepingOldValueVisible() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        var now = Date(timeIntervalSince1970: 1_700_000_000)
        let loader = makeLoader(session, now: { now })
        await loader.loadIfNeeded(maxAge: 60)

        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/resource") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return resourceStub(2)
        }
        now = now.addingTimeInterval(61)
        let reload = Task { await loader.loadIfNeeded(maxAge: 60) }
        await waitUntil { server.requestCount("/api/resource") == 2 }

        XCTAssertEqual(loader.phase, .loaded(1), "刷新期间旧数据仍可见")
        XCTAssertTrue(loader.isRefreshing)

        gate.signal()
        await reload.value
        XCTAssertEqual(loader.phase, .loaded(2))

        now = now.addingTimeInterval(30)
        await loader.loadIfNeeded(maxAge: 60)
        XCTAssertEqual(server.requestCount("/api/resource"), 2, "刷新成功后重新计算新鲜度")
    }

    func testLoadIfNeededWithoutMaxAgeNeverReloadsByTime() async throws {
        let server = MockHTTPServer()
        let session = await makeLoaderSession(server)
        server.stub("GET", "/api/resource", resourceStub(1))
        var now = Date(timeIntervalSince1970: 1_700_000_000)
        let loader = makeLoader(session, now: { now })
        await loader.loadIfNeeded()

        now = now.addingTimeInterval(10 * 24 * 60 * 60)
        await loader.loadIfNeeded()
        await loader.loadIfNeeded(maxAge: nil)

        XCTAssertEqual(server.requestCount("/api/resource"), 1, "不传 maxAge 时行为与按需加载一致")
        XCTAssertEqual(loader.phase, .loaded(1))
    }

    private func makeLoader(
        _ session: SessionStore,
        now: @escaping () -> Date = Date.init
    ) -> ResourceLoader<Int> {
        ResourceLoader(session: session, now: now) {
            let response: ResourceValue = try await session.authorizedRequest("/api/resource")
            return response.value
        }
    }
}

private struct ResourceValue: Decodable {
    let value: Int
}

private func resourceStub(_ value: Int) -> MockHTTPServer.Stub {
    .json(["success": true, "value": value])
}

@MainActor
private func makeLoaderSession(_ server: MockHTTPServer) async -> SessionStore {
    server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
    let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
    let session = makeSessionStore(server, credentialStore: store)
    await session.restore()
    return session
}
