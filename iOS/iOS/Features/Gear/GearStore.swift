import Combine
import Foundation

/// 本人装备的用户级缓存与跨页交接，对应小程序 `utils/gear-cache.js` + `store/gear.js`。
/// 只缓存本人装备；好友装备每次回源，不进这里。
@MainActor
final class GearStore: ObservableObject {
    static let ttl: TimeInterval = 60

    @Published private(set) var data: GearMineResponse?
    /// 刚登记成功的装备，供详情页显示一次「已添加」提示。
    @Published private(set) var justAddedItemId: Int64?

    private let session: SessionStore
    private let api: GearAPI
    private let now: () -> Date

    private var owner: SessionIdentity?
    private var fetchedAt: Date?
    /// 每次失效 +1；缓存只在「版本一致且未过期」时复用。
    private var version = 0
    private var dataVersion = -1
    private var inFlight: (version: Int, token: Int, task: Task<GearMineResponse, Error>)?
    private var inFlightToken = 0
    private var pendingPick: (token: String, handler: (GearProductInfo?) -> Void)?

    init(session: SessionStore, api: GearAPI? = nil, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.api = api ?? GearAPI(session: session)
        self.now = now
    }

    // MARK: - 身份

    /// 退出、切号或认证代次变化时整体清空；返回当前身份。
    @discardableResult
    func syncIdentity() -> SessionIdentity? {
        let current = session.identity
        guard current != owner else { return current }
        owner = current
        reset()
        return current
    }

    func isCurrent(_ identity: SessionIdentity) -> Bool {
        syncIdentity() == identity
    }

    /// 读取已有缓存而不触发请求；身份不一致时返回 nil。
    func peek() -> GearMineResponse? {
        guard syncIdentity() != nil else { return nil }
        return data
    }

    // MARK: - 读取

    /// force 跳过 TTL；并发读取共用同一个请求。返回 nil 表示读取期间身份已变化。
    func load(force: Bool = false) async throws -> GearMineResponse? {
        guard let identity = syncIdentity() else { return nil }

        if !force, let data, dataVersion == version, let fetchedAt,
           now().timeIntervalSince(fetchedAt) < Self.ttl {
            return data
        }
        if !force, let inFlight, inFlight.version == version {
            return try await inFlight.task.value
        }

        let startVersion = version
        inFlightToken += 1
        let token = inFlightToken
        let task = Task { [api] in try await api.mine() }
        inFlight = (version: startVersion, token: token, task: task)

        do {
            let response = try await task.value
            if inFlight?.token == token { inFlight = nil }
            guard isCurrent(identity) else { return nil }
            // 请求期间已有更新的读取：旧响应不写缓存。若新读取已落缓存，返回它，避免旧数据盖回去。
            if version == startVersion {
                data = response
                dataVersion = startVersion
                fetchedAt = now()
                return response
            }
            return data ?? response
        } catch {
            if inFlight?.token == token { inFlight = nil }
            // 退登会作废代次并让在途请求抛错：身份已变化的迟到失败同样丢弃。
            guard isCurrent(identity) else { return nil }
            throw error
        }
    }

    /// 请求进行中失效：旧响应既不写缓存，也不当作最新结果。
    func invalidate() {
        version += 1
    }

    // MARK: - 写操作后的本地更新

    func applyCreated(_ item: GearItemInfo) {
        guard var current = data else { return }
        current.items.removeAll { $0.id == item.id }
        current.items.append(item)
        data = current
    }

    func applyUpdated(_ item: GearItemInfo) {
        guard var current = data, let index = current.items.firstIndex(where: { $0.id == item.id }) else { return }
        current.items[index] = item
        // 退役会由服务端取消主力设置，本地同步去掉。
        if item.status != GearItemStatus.active.rawValue {
            current.defaults.removeAll { $0.itemId == item.id }
        }
        data = current
    }

    func applyDeleted(_ itemId: Int64) {
        guard var current = data else { return }
        current.items.removeAll { $0.id == itemId }
        current.defaults.removeAll { $0.itemId == itemId }
        data = current
    }

    func applyDefault(gameType: Int, slot: GearSlot, itemId: Int64) {
        guard var current = data else { return }
        current.defaults.removeAll { $0.gameType == gameType && $0.slot == slot.rawValue }
        if itemId != 0, current.items.contains(where: { $0.id == itemId }) {
            current.defaults.append(GearDefaultInfo(gameType: gameType, slot: slot.rawValue, itemId: itemId))
        }
        data = current
    }

    // MARK: - 刚添加的交接

    func markJustAdded(_ itemId: Int64) {
        justAddedItemId = itemId
    }

    func consumeJustAdded(_ itemId: Int64) -> Bool {
        guard justAddedItemId == itemId else { return false }
        justAddedItemId = nil
        return true
    }

    // MARK: - 皮头 / 巧克选择交接

    func beginPick(token: String, handler: @escaping (GearProductInfo?) -> Void) {
        pendingPick = (token, handler)
    }

    func finishPick(token: String, product: GearProductInfo?) {
        guard let pick = pendingPick, pick.token == token else { return }
        pendingPick = nil
        pick.handler(product)
    }

    func cancelPick(token: String) {
        if pendingPick?.token == token { pendingPick = nil }
    }

    private func reset() {
        data = nil
        dataVersion = -1
        fetchedAt = nil
        version = 0
        inFlight = nil
        inFlightToken += 1
        pendingPick = nil
        justAddedItemId = nil
    }
}
