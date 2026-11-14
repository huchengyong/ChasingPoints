import Combine
import Foundation

/// 我的装备页的读取与主力设置。
@MainActor
final class GearIndexModel: ObservableObject {
    @Published private(set) var data: GearMineResponse?
    @Published private(set) var isInitialLoading = true
    @Published private(set) var failure: LoadFailure.Feedback?
    @Published private(set) var refreshFailure: LoadFailure.Feedback?
    @Published private(set) var isSaving = false

    private let session: SessionStore
    private let store: GearStore
    private let api: GearAPI
    private var owner: SessionIdentity?

    init(session: SessionStore, store: GearStore, api: GearAPI? = nil) {
        self.session = session
        self.store = store
        self.api = api ?? GearAPI(session: session)
    }

    // MARK: - 展示值

    var items: [GearItemInfo] { data?.items ?? [] }
    var defaults: [GearDefaultInfo] { data?.defaults ?? [] }
    var groups: GearGroups { GearDisplayRules.groups(items) }
    var defaultsView: [GearMainRow] { GearDisplayRules.defaultsView(defaults, items: items) }
    var maxItems: Int { data?.maxItems ?? GearDisplayRules.maxItems }
    var countText: String { "\(String(items.count))/\(String(maxItems))" }
    var atLimit: Bool { items.count >= maxItems }
    var isEmpty: Bool { items.isEmpty }

    func mainLabels(for item: GearItemInfo) -> [String] {
        GearDisplayRules.mainGameLabels(defaults, itemId: item.id)
    }

    // MARK: - 读取

    func onAppear() async {
        await load()
    }

    func retry() async {
        await load(force: true)
    }

    /// 同身份的缓存先撑住页面，没有缓存才显示加载态。
    func load(force: Bool = false) async {
        let identity = session.identity
        if identity != owner {
            owner = identity
            data = nil
            failure = nil
            refreshFailure = nil
        }
        guard identity != nil else { return }
        if let cached = store.peek() { data = cached }
        isInitialLoading = data == nil
        do {
            guard let response = try await store.load(force: force) else { return }
            data = response
            failure = nil
            refreshFailure = nil
            isInitialLoading = false
        } catch {
            let feedback = LoadFailure(error).feedback(resource: "装备")
            if data == nil {
                failure = feedback
            } else {
                refreshFailure = feedback
            }
            isInitialLoading = false
        }
    }

    // MARK: - 主力设置

    /// 成功返回 nil；失败返回给 toast 的文案；身份变化时静默丢弃。
    func setDefault(row: GearMainRow, slot: GearSlot, itemId: Int64) async -> String? {
        guard let identity = session.identity, !isSaving else { return nil }
        isSaving = true
        defer { isSaving = false }
        do {
            try await api.setDefault(gameType: row.gameType, slot: slot, itemId: itemId)
            guard store.isCurrent(identity) else { return nil }
            store.applyDefault(gameType: row.gameType, slot: slot, itemId: itemId)
            data = store.peek()
            return nil
        } catch {
            guard store.isCurrent(identity) else { return nil }
            return error.localizedDescription
        }
    }
}
