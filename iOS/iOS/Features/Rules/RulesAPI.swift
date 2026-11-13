import Foundation

/// 规则百科的读取入口，全部为访客可读接口。
@MainActor
struct RulesAPI {
    let session: SessionStore

    func content(category: String, type: RuleContentType) async throws -> [RuleContentItem] {
        let path = APIQuery.path("/api/rules/content", [
            "category": category,
            "content_type": type.rawValue,
        ])
        let response: RuleListResponse<RuleContentItem> = try await session.optionalAuthRequest(path)
        return response.list
    }

    func glossary() async throws -> [GlossaryItem] {
        let response: RuleListResponse<GlossaryItem> = try await session.optionalAuthRequest("/api/rules/glossary")
        return response.list
    }

    func search(keyword: String) async throws -> [RuleContentItem] {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return [] }
        let path = APIQuery.path("/api/rules/search", ["keyword": trimmed])
        let response: RuleListResponse<RuleContentItem> = try await session.optionalAuthRequest(path)
        return response.list
    }
}

private struct RuleListResponse<Item: Decodable>: Decodable {
    let list: [Item]
}
