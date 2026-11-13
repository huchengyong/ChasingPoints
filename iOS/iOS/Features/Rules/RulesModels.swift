import Combine
import Foundation

struct RuleContentItem: Decodable, Equatable, Identifiable {
    let id: Int64
    let title: String
    let content: String
    let sortOrder: Int
}

struct GlossaryItem: Decodable, Equatable, Identifiable {
    let id: Int64
    let title: String
    let content: String
    let category: String
}

/// 规则 / 犯规 / 术语，对应接口的 content_type。
enum RuleContentType: String, CaseIterable {
    case rule
    case foul
    case glossary

    var label: String {
        switch self {
        case .rule: "规则"
        case .foul: "犯规"
        case .glossary: "术语"
        }
    }
}

/// 规则百科的球种分类。
enum RuleCategory {
    static let ordered: [(key: String, name: String, desc: String, emoji: String)] = [
        (key: "chinese_eight", name: label(for: "chinese_eight"), desc: "中国流行的花色球比赛", emoji: "⚫"),
        (key: "nine_ball", name: label(for: "nine_ball"), desc: "9球制，必须按顺序击球", emoji: "🟡"),
        (key: "snooker", name: label(for: "snooker"), desc: "22球制台球，得分制比赛", emoji: "🔴"),
        (key: "american_nine", name: label(for: "american_nine"), desc: "赛局制九球，支持普胜、小金和大金", emoji: "🟠"),
    ]

    /// 对应 getRuleCategoryLabel：查不到时返回 fallback，fallback 为空则返回 category 本身。
    static func label(for category: String, fallback: String = "") -> String {
        if let option = GameTypeCatalog.ordered.first(where: { $0.key == category }) {
            return option.label
        }
        return fallback.isEmpty ? category : fallback
    }

    static func emoji(for category: String) -> String {
        ordered.first { $0.key == category }?.emoji ?? "🎱"
    }

    /// 美式九球没有单独的规则内容，读取时按九球追分取。
    static func contentCategory(for category: String) -> String {
        category == "american_nine" ? "nine_ball" : category
    }
}

/// 术语词典的本地过滤：标题或内容包含关键词，不区分大小写并去掉首尾空白。
enum GlossaryFilter {
    static func filter(_ items: [GlossaryItem], keyword: String) -> [GlossaryItem] {
        let keyword = keyword.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !keyword.isEmpty else { return items }
        return items.filter {
            $0.title.lowercased().contains(keyword) || $0.content.lowercased().contains(keyword)
        }
    }
}

/// 首页搜索状态。只采纳最新一次搜索的结果，clear() 后到达的结果丢弃。
@MainActor
final class RulesSearchModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded([RuleContentItem])
        case empty
        case failed(LoadFailure)
    }

    @Published var keyword = ""
    @Published private(set) var phase: Phase = .idle

    private let api: RulesAPI
    private var requestToken = 0

    init(api: RulesAPI) {
        self.api = api
    }

    func search() async {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        requestToken += 1
        let token = requestToken
        guard !trimmed.isEmpty else {
            phase = .idle
            return
        }

        phase = .loading
        do {
            let items = try await api.search(keyword: trimmed)
            guard token == requestToken else { return }
            phase = items.isEmpty ? .empty : .loaded(items)
        } catch {
            guard token == requestToken else { return }
            phase = .failed(LoadFailure(error))
        }
    }

    func clear() {
        keyword = ""
        requestToken += 1
        phase = .idle
    }
}

/// 规则详情：每个 Tab 一个读取器，读过的内容在本页内保留。
@MainActor
final class RuleDetailModel: ObservableObject {
    let category: String
    @Published private(set) var type: RuleContentType = .rule
    @Published private(set) var expandedId: Int64?

    private let rules: ResourceLoader<[RuleContentItem]>
    private let fouls: ResourceLoader<[RuleContentItem]>
    private let glossaries: ResourceLoader<[RuleContentItem]>

    init(session: SessionStore, category: String) {
        self.category = category
        let api = RulesAPI(session: session)
        let contentCategory = RuleCategory.contentCategory(for: category)
        rules = ResourceLoader(session: session, access: .anyone) {
            try await api.content(category: contentCategory, type: .rule)
        }
        fouls = ResourceLoader(session: session, access: .anyone) {
            try await api.content(category: contentCategory, type: .foul)
        }
        glossaries = ResourceLoader(session: session, access: .anyone) {
            try await api.content(category: contentCategory, type: .glossary)
        }
    }

    func loader(for type: RuleContentType) -> ResourceLoader<[RuleContentItem]> {
        switch type {
        case .rule: rules
        case .foul: fouls
        case .glossary: glossaries
        }
    }

    /// 切换 Tab 时收起展开项并读取该类型；已读过的内容不会重复请求。
    func select(_ type: RuleContentType) async {
        if type != self.type {
            self.type = type
            expandedId = nil
        }
        await loader(for: type).loadIfNeeded()
    }

    /// 同一时间只展开一条。
    func toggle(_ id: Int64) {
        expandedId = expandedId == id ? nil : id
    }
}
