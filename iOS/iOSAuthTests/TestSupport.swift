import Foundation
import XCTest
@testable import iOS

final class InMemoryCredentialStore: CredentialStoring {
    var credentials: Credentials?
    var saveError: Error?
    private(set) var saveCount = 0
    private(set) var deleteCount = 0

    init(credentials: Credentials? = nil) {
        self.credentials = credentials
    }

    func load() -> Credentials? { credentials }

    func save(_ credentials: Credentials) throws {
        saveCount += 1
        if let saveError { throw saveError }
        self.credentials = credentials
    }

    func delete() {
        deleteCount += 1
        credentials = nil
    }
}

final class LockedCounter: @unchecked Sendable {
    private let lock = NSLock()
    private var value = 0

    func next() -> Int {
        lock.lock()
        defer { lock.unlock() }
        value += 1
        return value
    }

    var current: Int {
        lock.lock()
        defer { lock.unlock() }
        return value
    }
}

struct TestUserInfoResponse: Decodable {
    let userInfo: UserInfo?
}

@MainActor
func makeSessionStore(_ server: MockHTTPServer, credentialStore: CredentialStoring) -> SessionStore {
    let client = APIClient(baseURL: URL(string: "https://example.test")!, session: server.makeSession())
    return SessionStore(api: client, credentialStore: credentialStore)
}

@MainActor
func waitUntil(
    timeout: TimeInterval = 3,
    file: StaticString = #filePath,
    line: UInt = #line,
    _ condition: () -> Bool
) async {
    let deadline = Date().addingTimeInterval(timeout)
    while !condition() && Date() < deadline {
        try? await Task.sleep(nanoseconds: 10_000_000)
    }
    XCTAssertTrue(condition(), "等待测试条件超时", file: file, line: line)
}

@MainActor
func assertAPIError(
    _ error: Error,
    kind: APIError.Kind,
    message: String? = nil,
    statusCode: Int? = nil,
    file: StaticString = #filePath,
    line: UInt = #line
) {
    guard let apiError = error as? APIError else {
        XCTFail("期望 APIError，实际为 \(error)", file: file, line: line)
        return
    }
    XCTAssertEqual(apiError.kind, kind, file: file, line: line)
    if let message {
        XCTAssertEqual(apiError.message, message, file: file, line: line)
    }
    if let statusCode {
        XCTAssertEqual(apiError.statusCode, statusCode, file: file, line: line)
    }
}
