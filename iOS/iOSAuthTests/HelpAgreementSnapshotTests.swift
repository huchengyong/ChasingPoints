import SwiftUI
import XCTest
@testable import iOS

/// 帮助、协议与 PK 报表页面快照：各渲染亮暗两张。
@MainActor
final class HelpAgreementSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private var server: MockHTTPServer!

    override func setUp() {
        super.setUp()
        server = MockHTTPServer()
    }

    func testFeedbackSnapshot() async throws {
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let model = FeedbackModel(session: session)

        try render("feedback", session: session, view: FeedbackView(session: session, model: model))
    }

    func testFeedbackErrorsSnapshot() async throws {
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let model = FeedbackModel(session: session)
        let toast = ToastCenter()
        toast.show("请输入投诉举报或反馈内容")

        try render("feedback-errors", session: session, toast: toast, view: FeedbackView(session: session, model: model))
    }

    func testUserAgreementSnapshot() throws {
        try render("user-agreement", view: AgreementView(document: AgreementContent.userAgreement))
    }

    func testPrivacyPolicySnapshot() throws {
        try render("privacy-policy", view: AgreementView(document: AgreementContent.privacyPolicy))
    }

    private func render(
        _ name: String,
        session: SessionStore? = nil,
        toast: ToastCenter? = nil,
        view: some View
    ) throws {
        let toast = toast ?? ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    view
                }
                .environmentObject(session ?? SessionStore(api: APIClient(
                    baseURL: URL(string: "https://example.test")!,
                    session: server.makeSession()
                ), credentialStore: InMemoryCredentialStore()))
                .environmentObject(AppNavigator())
                .environmentObject(toast)
                .toastOverlay(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }
}
