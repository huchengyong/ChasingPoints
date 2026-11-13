import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class RulesSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testRulesIndexSnapshot() throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rules-index", colorScheme: scheme, size: size) {
                NavigationStack {
                    RulesIndexView(session: session)
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testRulesSearchSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let navigator = AppNavigator()
        let toast = ToastCenter()
        server.stub("GET", "/api/rules/search", .json([
            "success": true,
            "list": [
                ["id": 1, "title": "开球", "content": "开球时白球必须放在开球线后。", "sort_order": 1],
                ["id": 2, "title": "斯诺克", "content": "白球被遮挡时的击球规则。", "sort_order": 2],
                ["id": 3, "title": "定杆", "content": "击球后白球停在原位。", "sort_order": 3],
            ],
        ]))
        let model = RulesSearchModel(api: RulesAPI(session: session))
        model.keyword = "开球"
        await model.search()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rules-search", colorScheme: scheme, size: size) {
                NavigationStack {
                    RulesIndexView(session: session, search: model, expandedResultId: 1)
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testRuleDetailSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let navigator = AppNavigator()
        let toast = ToastCenter()
        server.stub("GET", "/api/rules/content", .json([
            "success": true,
            "list": [
                ["id": 1, "title": "球台与球", "content": "斯诺克台面长 3569 毫米。", "sort_order": 1],
                ["id": 2, "title": "开球", "content": "开球方须将白球放在开球线后。", "sort_order": 2],
                ["id": 3, "title": "击球顺序", "content": "红球与彩球交替击打。", "sort_order": 3],
                ["id": 4, "title": "犯规", "content": "未击中目标球即为犯规。", "sort_order": 4],
                ["id": 5, "title": "计分", "content": "红球 1 分，黑球 7 分。", "sort_order": 5],
            ],
        ]))
        let model = RuleDetailModel(session: session, category: "snooker")
        await model.select(.rule)
        model.toggle(2)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rule-detail", colorScheme: scheme, size: size) {
                NavigationStack {
                    RuleDetailView(session: session, category: "snooker", model: model)
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testRuleDetailErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let navigator = AppNavigator()
        let toast = ToastCenter()
        server.stub("GET", "/api/rules/content", .raw("service down", status: 503))
        let model = RuleDetailModel(session: session, category: "snooker")
        await model.select(.rule)
        XCTAssertEqual(model.loader(for: .rule).failure?.category, .server)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rule-detail-error", colorScheme: scheme, size: size) {
                NavigationStack {
                    RuleDetailView(session: session, category: "snooker", model: model)
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testGlossarySnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let navigator = AppNavigator()
        let toast = ToastCenter()
        server.stub("GET", "/api/rules/glossary", .json([
            "success": true,
            "list": [
                ["id": 1, "title": "开球", "content": "Break Shot，开球时白球必须放在开球线后。", "category": "snooker"],
                ["id": 2, "title": "定杆", "content": "Stun，击球后白球停在原位。", "category": "chinese_eight"],
                ["id": 3, "title": "缩杆", "content": "Draw，击球后白球向后回拉。", "category": "chinese_eight"],
                ["id": 4, "title": "斯诺克", "content": "Snooker，白球被其他球遮挡。", "category": "snooker"],
                ["id": 5, "title": "黄金开球", "content": "开球直接打进九号球获胜。", "category": "nine_ball"],
                ["id": 6, "title": "自由球", "content": "Free Ball，犯规后可指定任意球为目标球。", "category": ""],
            ],
        ]))
        let api = RulesAPI(session: session)
        let loader = ResourceLoader<[GlossaryItem]>(session: session, access: .anyone) {
            try await api.glossary()
        }
        await loader.loadIfNeeded()
        XCTAssertEqual(loader.value?.count, 6)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("glossary", colorScheme: scheme, size: size) {
                NavigationStack {
                    GlossaryView(session: session, loader: loader)
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }
}
