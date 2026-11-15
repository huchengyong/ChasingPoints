import Foundation

/// 段位配置里的规则版本；`rules_effective_at` 为空表示未排期。
struct RankRulesMeta: Equatable {
    let rulesVersion: Int
    let rulesEffectiveAt: String

    static let legacy = RankRulesMeta(rulesVersion: 0, rulesEffectiveAt: "")
}

/// `GET /api/public/rank/configs`：段位列表与规则版本来自同一份公开配置。
struct RankConfigs: Equatable {
    let rankList: [RankItem]
    let rules: RankRulesMeta
}

/// 段位说明的读取入口：段位配置公开可读，用户段位仍需登录。
@MainActor
struct RankExplainAPI {
    let session: SessionStore

    func configs() async throws -> RankConfigs {
        let response: RankConfigsResponse = try await session.optionalAuthRequest("/api/public/rank/configs")
        return RankConfigs(
            rankList: response.list,
            rules: RankRulesMeta(
                rulesVersion: response.rulesVersion,
                rulesEffectiveAt: response.rulesEffectiveAt
            )
        )
    }
}

private struct RankConfigsResponse: Decodable {
    let list: [RankItem]
    let rulesVersion: Int
    let rulesEffectiveAt: String
}
