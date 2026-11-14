import Combine
import Foundation

/// 装备详情：读取本人装备并处理皮头 / 巧克、主力、退役与删除。
@MainActor
final class GearItemModel: ObservableObject {
    struct Part: Identifiable {
        let category: GearCategory
        let label: String
        let product: GearProductInfo?

        var id: String { category.rawValue }
        var subject: GearImageSource {
            product.map(GearImageSource.init(product:)) ?? GearImageSource(category: category.rawValue)
        }
        var text: String {
            product.map(GearDisplayRules.productTitle) ?? "未设置"
        }
    }

    struct MainToggle: Identifiable {
        let gameType: Int
        let label: String
        let slot: GearSlot
        let active: Bool

        var id: Int { gameType }
    }

    @Published private(set) var data: GearMineResponse?
    @Published private(set) var isInitialLoading = true
    @Published private(set) var failure: LoadFailure.Feedback?
    @Published private(set) var refreshFailure: LoadFailure.Feedback?
    @Published private(set) var isBusy = false
    @Published private(set) var justAdded = false

    let itemId: Int64

    private let session: SessionStore
    private let store: GearStore
    private let api: GearAPI
    private var owner: SessionIdentity?

    init(session: SessionStore, store: GearStore, itemId: Int64, api: GearAPI? = nil) {
        self.session = session
        self.store = store
        self.itemId = itemId
        self.api = api ?? GearAPI(session: session)
    }

    // MARK: - 展示值

    var item: GearItemInfo? {
        data?.items.first { $0.id == itemId }
    }
    var title: String { GearDisplayRules.itemTitle(item) }
    var subtitle: String { GearDisplayRules.itemSubtitle(item) }
    var retired: Bool { item?.status == GearItemStatus.retired.rawValue }
    var isCue: Bool { item?.category == GearCategory.cue.rawValue }
    var specs: [GearSpec] { GearDisplayRules.specs(item?.product) }

    var parts: [Part] {
        guard let item else { return [] }
        return [
            Part(category: .tip, label: "皮头", product: item.tipProduct),
            Part(category: .chalk, label: "巧克", product: item.chalkProduct),
        ]
    }

    /// 能设为主力的球种：球杆有四个球种，冲杆没有斯诺克；退役装备不能设为主力。
    var mainRows: [MainToggle] {
        guard let item else { return [] }
        let slot: GearSlot = isCue ? .cue : .breakSlot
        let accepts = isCue ? GearDisplayRules.canBeMainCue(item) : GearDisplayRules.canBeMainBreakCue(item)
        guard accepts else { return [] }
        return GearDisplayRules.defaultsView(data?.defaults ?? [], items: data?.items ?? [])
            .filter { isCue || $0.supportsBreak }
            .map { row in
                MainToggle(
                    gameType: row.gameType,
                    label: row.label,
                    slot: slot,
                    active: (isCue ? row.cue : row.breakCue)?.id == item.id
                )
            }
    }

    // MARK: - 读取

    func onAppear() async {
        justAdded = store.consumeJustAdded(itemId)
        await load()
    }

    func retry() async {
        await load(force: true)
    }

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

    // MARK: - 写操作

    func setPart(_ category: GearCategory, product: GearProductInfo?) async -> GearWriteOutcome {
        guard let identity = session.identity, !isBusy else { return .stale }
        isBusy = true
        defer { isBusy = false }

        var request = GearUpdateRequest(itemId: itemId)
        if category == .tip {
            request.updateTip = true
            request.tipProductId = product?.id ?? 0
        } else {
            request.updateChalk = true
            request.chalkProductId = product?.id ?? 0
        }
        do {
            let updated = try await api.update(request)
            guard store.isCurrent(identity) else { return .stale }
            if let updated {
                store.applyUpdated(updated)
                data = store.peek()
            }
            return .success("已保存")
        } catch {
            guard store.isCurrent(identity) else { return .stale }
            return .failure(error.localizedDescription)
        }
    }

    func toggleMain(_ row: MainToggle) async -> GearWriteOutcome {
        guard let identity = session.identity, !isBusy else { return .stale }
        isBusy = true
        defer { isBusy = false }

        let nextItemId: Int64 = row.active ? 0 : itemId
        do {
            try await api.setDefault(gameType: row.gameType, slot: row.slot, itemId: nextItemId)
            guard store.isCurrent(identity) else { return .stale }
            store.applyDefault(gameType: row.gameType, slot: row.slot, itemId: nextItemId)
            data = store.peek()
            return .success(row.active ? "已取消主力" : "已设为\(row.label)主力")
        } catch {
            guard store.isCurrent(identity) else { return .stale }
            return .failure(error.localizedDescription)
        }
    }

    /// 退役需要确认；恢复使用不需要。
    func toggleRetire() async -> GearWriteOutcome {
        guard let identity = session.identity, !isBusy else { return .stale }
        isBusy = true
        defer { isBusy = false }

        let next = retired ? GearItemStatus.active : .retired
        do {
            let updated = try await api.update(GearUpdateRequest(itemId: itemId, status: next.rawValue))
            guard store.isCurrent(identity) else { return .stale }
            if let updated {
                store.applyUpdated(updated)
                data = store.peek()
            }
            return .success(next == .active ? "已恢复使用" : "已退役")
        } catch {
            guard store.isCurrent(identity) else { return .stale }
            return .failure(error.localizedDescription)
        }
    }

    func delete() async -> GearWriteOutcome {
        guard let identity = session.identity, !isBusy else { return .stale }
        isBusy = true
        defer { isBusy = false }

        do {
            try await api.delete(itemId: itemId)
            guard store.isCurrent(identity) else { return .stale }
            store.applyDeleted(itemId)
            data = store.peek()
            return .success("已删除")
        } catch {
            guard store.isCurrent(identity) else { return .stale }
            return .failure(error.localizedDescription)
        }
    }
}
