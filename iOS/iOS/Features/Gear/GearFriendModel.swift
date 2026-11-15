import Combine
import Foundation

/// 好友装备（只读）：好友关系随时可能变化，每次进入都回源，不进缓存。
@MainActor
final class GearFriendModel: ObservableObject {
    struct Failure: Equatable {
        let title: String
        let description: String
        let retryable: Bool
    }

    @Published private(set) var loading = true
    @Published private(set) var failure: Failure?
    @Published private(set) var items: [GearItemInfo] = []
    @Published private(set) var defaults: [GearDefaultInfo] = []

    let userId: Int64
    let ownerName: String

    private let session: SessionStore
    private let api: GearAPI
    private var requestSeq = 0

    init(session: SessionStore, userId: Int64, ownerName: String, api: GearAPI? = nil) {
        self.session = session
        self.userId = userId
        self.ownerName = ownerName
        self.api = api ?? GearAPI(session: session)
    }

    var groups: GearGroups { GearDisplayRules.groups(items) }
    /// 默认的「TA」是英文占位，和「的装备」之间留空格；中文昵称直接连着写。
    var ownerTitle: String { ownerName == "TA" ? "TA 的装备" : "\(ownerName)的装备" }
    func mainLabels(for item: GearItemInfo) -> [String] {
        GearDisplayRules.mainGameLabels(defaults, itemId: item.id)
    }

    /// 一个主力都没设置时不展示这块，避免一排「未设置」。
    var mainRows: [GearMainRow] {
        let rows = GearDisplayRules.defaultsView(defaults, items: items)
        return rows.contains { $0.cue != nil || $0.breakCue != nil } ? rows : []
    }

    func load() async {
        guard session.identity != nil else { return }
        guard userId > 0 else {
            loading = false
            failure = Failure(title: "用户不存在", description: "请返回后重新进入。", retryable: false)
            return
        }
        let identity = session.identity
        requestSeq += 1
        let seq = requestSeq
        loading = true
        failure = nil
        do {
            let response = try await api.userGear(userId: userId)
            guard seq == requestSeq, session.identity == identity else { return }
            items = response.items
            defaults = response.defaults
            loading = false
        } catch {
            guard seq == requestSeq, session.identity == identity else { return }
            if let apiError = error as? APIError, apiError.reason == "not_friend" {
                failure = Failure(title: "仅好友可以查看", description: "加为好友后，就能看到 TA 的装备。", retryable: false)
            } else {
                let feedback = LoadFailure(error).feedback(resource: "装备")
                failure = Failure(title: feedback.title, description: feedback.description, retryable: feedback.retryable)
            }
            loading = false
        }
    }
}
