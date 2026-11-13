import Foundation
import XCTest
@testable import iOS

@MainActor
final class RulesModelTests: XCTestCase {
    func testCategoryMetadataMatchesRulePages() {
        XCTAssertEqual(RuleCategory.ordered.map { $0.key }, ["chinese_eight", "nine_ball", "snooker", "american_nine"])
        XCTAssertEqual(RuleCategory.ordered.map { $0.name }, ["中式八球", "九球追分", "斯诺克", "美式九球"])
        XCTAssertEqual(RuleCategory.ordered.map { $0.desc }, [
            "中国流行的花色球比赛",
            "9球制，必须按顺序击球",
            "22球制台球，得分制比赛",
            "赛局制九球，支持普胜、小金和大金",
        ])
        XCTAssertEqual(RuleCategory.ordered.map { $0.emoji }, ["⚫", "🟡", "🔴", "🟠"])

        XCTAssertEqual(RuleCategory.label(for: "snooker"), "斯诺克")
        XCTAssertEqual(RuleCategory.label(for: "unknown"), "unknown")
        XCTAssertEqual(RuleCategory.label(for: "unknown", fallback: "规则"), "规则")
        XCTAssertEqual(RuleCategory.emoji(for: "american_nine"), "🟠")
        XCTAssertEqual(RuleCategory.emoji(for: "unknown"), "🎱")
    }

    func testContentCategoryMapsAmericanNineToNineBall() {
        XCTAssertEqual(RuleCategory.contentCategory(for: "american_nine"), "nine_ball")
        XCTAssertEqual(RuleCategory.contentCategory(for: "chinese_eight"), "chinese_eight")
        XCTAssertEqual(RuleCategory.contentCategory(for: "nine_ball"), "nine_ball")
        XCTAssertEqual(RuleCategory.contentCategory(for: "snooker"), "snooker")
    }

    func testGlossaryFilterTrimsAndIgnoresCase() {
        let items = [
            GlossaryItem(id: 1, title: "开球", content: "Break Shot", category: "snooker"),
            GlossaryItem(id: 2, title: "定杆", content: "击球后白球停在原位", category: ""),
            GlossaryItem(id: 3, title: "Kick", content: "踢球犯规", category: "nine_ball"),
        ]

        XCTAssertEqual(GlossaryFilter.filter(items, keyword: "  break ").map(\.id), [1])
        XCTAssertEqual(GlossaryFilter.filter(items, keyword: "白球").map(\.id), [2])
        XCTAssertEqual(GlossaryFilter.filter(items, keyword: "kick").map(\.id), [3])
        XCTAssertEqual(GlossaryFilter.filter(items, keyword: "   "), items)
        XCTAssertEqual(GlossaryFilter.filter(items, keyword: "不存在"), [])
    }

    func testSearchModelLoadsResultsAndTrimsKeyword() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/search", .json([
            "success": true,
            "list": [itemJSON(id: 1, title: "开球")],
        ]))
        let model = RulesSearchModel(api: RulesAPI(session: session))
        model.keyword = "  开球  "

        await model.search()

        XCTAssertEqual(model.phase, .loaded([RuleContentItem(id: 1, title: "开球", content: "正文", sortOrder: 1)]))
        XCTAssertEqual(server.requests("/api/rules/search").first?.query["keyword"], "开球")
    }

    func testSearchModelShowsEmptyState() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/search", .json(["success": true, "list": []]))
        let model = RulesSearchModel(api: RulesAPI(session: session))
        model.keyword = "无结果"

        await model.search()

        XCTAssertEqual(model.phase, .empty)
    }

    func testSearchModelFailureKeepsCategory() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/search", .raw("down", status: 503))
        let model = RulesSearchModel(api: RulesAPI(session: session))
        model.keyword = "开球"

        await model.search()

        guard case .failed(let failure) = model.phase else {
            return XCTFail("应进入失败态：\(model.phase)")
        }
        XCTAssertEqual(failure.category, .server)
        XCTAssertEqual(failure.message, "服务暂时不可用，请稍后重试")
    }

    func testSearchModelOnlyAcceptsLatestRequest() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let firstGate = DispatchSemaphore(value: 0)
        let secondGate = DispatchSemaphore(value: 0)
        let attempts = LockedCounter()
        server.stub("GET", "/api/rules/search") { _ in
            if attempts.next() == 1 {
                _ = firstGate.wait(timeout: .now() + 5)
                return .json(["success": true, "list": [itemJSON(id: 1, title: "旧")]])
            }
            _ = secondGate.wait(timeout: .now() + 5)
            return .json(["success": true, "list": [itemJSON(id: 2, title: "新")]])
        }
        let model = RulesSearchModel(api: RulesAPI(session: session))

        model.keyword = "旧"
        let first = Task { await model.search() }
        await waitUntil { server.requestCount("/api/rules/search") == 1 }

        model.keyword = "新"
        let second = Task { await model.search() }
        await waitUntil { server.requestCount("/api/rules/search") == 2 }

        secondGate.signal()
        await second.value
        XCTAssertEqual(model.phase, .loaded([RuleContentItem(id: 2, title: "新", content: "正文", sortOrder: 1)]))

        firstGate.signal()
        await first.value
        XCTAssertEqual(model.phase, .loaded([RuleContentItem(id: 2, title: "新", content: "正文", sortOrder: 1)]))
    }

    func testSearchModelDropsResultAfterClear() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/rules/search") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true, "list": [itemJSON(id: 1, title: "开球")]])
        }
        let model = RulesSearchModel(api: RulesAPI(session: session))
        model.keyword = "开球"

        let task = Task { await model.search() }
        await waitUntil { server.requestCount("/api/rules/search") == 1 }

        model.clear()
        gate.signal()
        await task.value

        XCTAssertEqual(model.phase, .idle)
        XCTAssertEqual(model.keyword, "")
    }

    func testDetailModelLoadsSelectedTypeAndReusesLoadedTab() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/content") { request in
            .json([
                "success": true,
                "list": [itemJSON(id: 1, title: request.query["content_type"] ?? "")],
            ])
        }
        let model = RuleDetailModel(session: session, category: "snooker")

        await model.select(.foul)
        XCTAssertEqual(model.type, .foul)
        XCTAssertEqual(model.loader(for: .foul).value?.first?.title, "foul")

        await model.select(.rule)
        XCTAssertEqual(model.loader(for: .rule).value?.first?.title, "rule")

        await model.select(.foul)
        XCTAssertEqual(server.requests("/api/rules/content").filter { $0.query["content_type"] == "foul" }.count, 1)
        XCTAssertEqual(server.requests("/api/rules/content").filter { $0.query["content_type"] == "rule" }.count, 1)
        XCTAssertNil(model.expandedId, "切换 Tab 时收起展开项")
    }

    func testDetailModelRequestsNineBallForAmericanNine() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", "/api/rules/content", .json(["success": true, "list": []]))
        let model = RuleDetailModel(session: session, category: "american_nine")

        await model.select(.glossary)

        XCTAssertEqual(server.requests("/api/rules/content").first?.query["category"], "nine_ball")
        XCTAssertEqual(server.requests("/api/rules/content").first?.query["content_type"], "glossary")
    }

    func testDetailModelTogglesSingleExpandedItem() {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let model = RuleDetailModel(session: session, category: "snooker")

        model.toggle(1)
        XCTAssertEqual(model.expandedId, 1)
        model.toggle(2)
        XCTAssertEqual(model.expandedId, 2)
        model.toggle(2)
        XCTAssertNil(model.expandedId)
    }

    private func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }
}

private func itemJSON(id: Int64, title: String) -> [String: Any] {
    ["id": id, "title": title, "content": "正文", "sort_order": 1]
}
