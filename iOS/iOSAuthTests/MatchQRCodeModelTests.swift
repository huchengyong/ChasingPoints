import Foundation
import XCTest
@testable import iOS

@MainActor
final class MatchQRCodeModelTests: XCTestCase {
    func testLoadRendersTheScannableImage() async throws {
        let server = MockHTTPServer()
        let session = makeQRCodeSession(server)
        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode()))
        let model = MatchQRCodeModel(session: session)

        await model.load()

        guard case .ready(let image) = model.phase else {
            return XCTFail("期望二维码生成成功，实际为 \(model.phase)")
        }
        XCTAssertGreaterThan(image.size.width, 0)
        XCTAssertEqual(server.requestCount("/api/match/qrcode"), 1)
    }

    func testEmptyContentFails() async {
        let server = MockHTTPServer()
        let session = makeQRCodeSession(server)
        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode(qrcodeData: "")))
        let model = MatchQRCodeModel(session: session)

        await model.load()

        XCTAssertEqual(model.phase, .failed)
    }

    func testRequestFailureFails() async {
        let server = MockHTTPServer()
        let session = makeQRCodeSession(server)
        server.stub("GET", "/api/match/qrcode", .raw("service down", status: 503))
        let model = MatchQRCodeModel(session: session)

        await model.load()

        XCTAssertEqual(model.phase, .failed)
    }

    func testResetDropsLateResultAndKeepsInitialPhase() async {
        let server = MockHTTPServer()
        let session = makeQRCodeSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/qrcode") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(ProfileFixtures.matchQRCode())
        }
        let model = MatchQRCodeModel(session: session)

        let load = Task { await model.load() }
        await waitUntil { server.requestCount("/api/match/qrcode") == 1 }

        model.reset()
        gate.signal()
        await load.value

        XCTAssertEqual(model.phase, .loading, "关闭弹层后到达的二维码必须丢弃")
    }

    func testRetrySucceedsAfterFailure() async {
        let server = MockHTTPServer()
        let session = makeQRCodeSession(server)
        server.stub("GET", "/api/match/qrcode", .raw("service down", status: 503))
        let model = MatchQRCodeModel(session: session)

        await model.load()
        XCTAssertEqual(model.phase, .failed)

        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode()))
        await model.load()

        if case .ready = model.phase {
            XCTAssertEqual(server.requestCount("/api/match/qrcode"), 2)
        } else {
            XCTFail("重试后应生成二维码，实际为 \(model.phase)")
        }
    }

    private func makeQRCodeSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
    }
}
