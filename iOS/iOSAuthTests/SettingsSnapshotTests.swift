import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class SettingsSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testSignedInSettingsSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/privacy", .json([
            "success": true,
            "hide_match_record": true,
            "friends_only_challenges": false,
        ]))
        let privacy = PrivacySettingsModel(session: session)
        await privacy.load()
        XCTAssertEqual(privacy.phase, .ready)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("settings-signed-in", colorScheme: scheme, size: pageSize) {
                page(session, toast: ToastCenter()) {
                    SettingsView(session: session, privacyModel: privacy)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
        XCTAssertEqual(server.requestCount("/api/user/privacy"), 1)
    }

    func testGuestSettingsSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("settings-guest", colorScheme: scheme, size: pageSize) {
                page(session, toast: ToastCenter()) {
                    SettingsView(session: session)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
        XCTAssertEqual(server.requests.count, 0, "访客设置页不发起读取")
    }

    func testDeleteAccountVerifySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = DeleteAccountModel(session: session)
        model.open()
        model.continueToVerify()
        XCTAssertEqual(model.step, .verify)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("delete-account-verify", colorScheme: scheme, size: pageSize) {
                DeleteAccountDialog(model: model, onClose: {})
                    .environmentObject(ToastCenter())
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testProfileEditSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("profile-edit", colorScheme: scheme, size: pageSize) {
                page(session, toast: ToastCenter()) {
                    ProfileEditView(session: session)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testNotificationSettingsSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/preferences", .json([
            "success": true,
            "match_result_enabled": true,
            "friend_request_enabled": true,
            "challenge_enabled": false,
            "tournament_enabled": true,
            "follow_enabled": true,
        ]))
        let model = NotificationPreferencesModel(session: session)
        await model.load()
        XCTAssertEqual(model.phase, .ready)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("notification-settings", colorScheme: scheme, size: pageSize) {
                page(session, toast: ToastCenter()) {
                    NotificationSettingsView(session: session, model: model)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
        XCTAssertEqual(server.requestCount("/api/notification/preferences"), 1)
    }

    private func page<Content: View>(
        _ session: SessionStore,
        toast: ToastCenter,
        @ViewBuilder content: () -> Content
    ) -> some View {
        NavigationStack {
            content()
        }
        .environmentObject(session)
        .environmentObject(toast)
        .environmentObject(AppNavigator())
    }
}
