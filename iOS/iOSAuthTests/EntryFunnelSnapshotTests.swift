import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class EntryFunnelSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testWelcomeSnapshot() throws {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let agreement = EntryAgreementModel(defaults: defaults)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("welcome", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    WelcomeView(agreement: agreement)
                        .toolbar(.hidden, for: .navigationBar)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testWelcomeAgreementSheetSnapshot() throws {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let agreement = EntryAgreementModel(defaults: defaults)
        agreement.requestConsent()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("welcome-agreement-sheet", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    WelcomeView(agreement: agreement)
                        .toolbar(.hidden, for: .navigationBar)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testLoginSnapshot() throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let agreement = EntryAgreementModel(defaults: defaults)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("login", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    LoginView(agreement: agreement)
                }
                .environmentObject(session)
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testLoginAuthenticatingSnapshot() throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let agreement = EntryAgreementModel(defaults: defaults)
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let feedback = AuthFeedbackModel(now: { now })
        feedback.begin()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("login-authenticating", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    LoginView(agreement: agreement, feedback: feedback)
                }
                .environmentObject(session)
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testLoginErrorSnapshot() throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let agreement = EntryAgreementModel(defaults: defaults)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("login-error", colorScheme: scheme, size: pageSize) {
                NavigationStack {
                    LoginView(
                        agreement: agreement,
                        initialError: "验证码错误或已过期，请重新获取"
                    )
                }
                .environmentObject(session)
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    private func makeDefaults() -> (UserDefaults, String) {
        let name = "EntryFunnelSnapshotTests.\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: name)!
        defaults.removePersistentDomain(forName: name)
        return (defaults, name)
    }
}
