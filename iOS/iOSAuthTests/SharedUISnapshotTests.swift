import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class SharedUISnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testUIKitSnapshot() throws {
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("ui-kit", colorScheme: scheme, size: size) {
                uiKit(scheme)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMainTabsGuestSnapshot() throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("main-tabs-guest", colorScheme: scheme, size: size) {
                MainTabView(session: session)
                    .environmentObject(session)
                    .environmentObject(activity)
                    .environmentObject(navigator)
                    .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testMyPageSignedInSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        XCTAssertNotNil(session.identity)

        let activity = ActivityStore(session: session)
        let navigator = AppNavigator()
        navigator.select(.me, isSignedIn: true)
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("my-page-signed-in", colorScheme: scheme, size: size) {
                MainTabView(session: session)
                    .environmentObject(session)
                    .environmentObject(activity)
                    .environmentObject(navigator)
                    .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testToastSnapshot() throws {
        let center = ToastCenter(duration: 5)
        center.show("已复制到剪贴板")

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("toast", colorScheme: scheme, size: size) {
                AppTheme.palette(for: scheme).page
                    .ignoresSafeArea()
                    .toastOverlay(center)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func uiKit(_ scheme: ColorScheme) -> some View {
        let palette = AppTheme.palette(for: scheme)

        return ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                VStack(spacing: 12) {
                    Button("确认对局") {}
                        .buttonStyle(.appPrimary)
                    Button("暂不可用") {}
                        .buttonStyle(.appPrimary)
                        .disabled(true)
                    Button("取消") {}
                        .buttonStyle(.appSecondary)
                    Button("去看看") {}
                        .buttonStyle(.appPill)
                }

                SectionHeader(title: "最近对局", caption: "近 7 天", moreTitle: "更多") {}

                VStack(alignment: .leading, spacing: 6) {
                    Text("卡片标题")
                        .font(.system(size: 17, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("卡片里的辅助说明文字。")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .appCard()

                HStack(spacing: 12) {
                    AvatarView(avatar: "", userId: 3, size: 48)
                    AvatarView(avatar: "", userId: 12, size: 48)
                }

                HStack(spacing: 16) {
                    ForEach([0, 2, 4, 5], id: \.self) { level in
                        MemberAvatarView(
                            avatar: "",
                            userId: 3,
                            vipLevel: level,
                            rankLevel: 3,
                            size: 56,
                            reserveFrame: true
                        )
                    }
                }

                StateMessageView(
                    icon: "trophy.fill",
                    title: "暂无荣誉",
                    detail: "完成对局后可以获得成就",
                    actionTitle: "去看看"
                ) {}
            }
            .padding(20)
        }
        .background(palette.page)
    }
}
