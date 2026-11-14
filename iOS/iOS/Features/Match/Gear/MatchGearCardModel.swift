import Combine
import Foundation

/// 「本场用杆」卡片状态：读取失败只影响这张卡片，服务端不认本场选手时整卡隐藏。
@MainActor
final class MatchGearCardModel: ObservableObject {
    enum Phase: Equatable {
        case hidden
        case loading
        case failed(String)
        case ready(GearMatchResponse)
    }

    /// 修改弹层的草稿与装备列表；草稿只随弹层生命周期存在。
    struct Editor: Equatable {
        var visible = false
        var saving = false
        var draft = GearMatchDraft()
        var items: [GearItemInfo] = []
        var itemsLoading = false
        var itemsError = ""
    }

    @Published private(set) var phase: Phase = .hidden
    @Published private(set) var editor = Editor()

    let matchId: Int64
    private let enabled: Bool
    private let session: SessionStore
    private let api: GearAPI
    private let loader: GearMatchLoader
    private var itemsToken = 0

    init(session: SessionStore, matchId: Int64, enabled: Bool) {
        self.session = session
        self.matchId = matchId
        self.enabled = enabled
        let api = GearAPI(session: session)
        self.api = api
        self.loader = GearMatchLoader(
            request: { try await api.match(matchId: $0) },
            readIdentity: {
                let identity = session.identity
                return GearMatchLoader.Identity(userId: identity?.userId, generation: identity?.generation)
            }
        )
    }

    private var response: GearMatchResponse? {
        if case .ready(let data) = phase { return data }
        return nil
    }

    var canEdit: Bool { GearMatchRules.canEdit(response) }

    var deadlineText: String { GearMatchRules.formatEditDeadline(response?.editDeadline) }

    var responseSupportsBreak: Bool { response?.supportsBreak == true }

    var cueOptions: [GearItemInfo] { GearMatchRules.cueOptions(editor.items) }

    var breakOptions: [GearItemInfo] { GearMatchRules.breakOptions(editor.items) }

    func load() async {
        guard enabled, matchId > 0, (session.identity?.userId ?? 0) > 0 else {
            loader.cancel()
            phase = .hidden
            return
        }
        phase = .loading
        do {
            let result = try await loader.load(matchId)
            guard !result.stale else { return }
            let response = result.response
            if response.success {
                phase = .ready(response)
            } else if GearMatchRules.isHiddenResponse(response) {
                phase = .hidden
            } else {
                phase = .failed(response.message.isEmpty ? "本场用杆加载失败" : response.message)
            }
        } catch {
            // 页面离开或切号会取消读取，此时保留原状态。
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            phase = .failed("本场用杆加载失败，请检查网络后重试")
        }
    }

    func cancel() {
        loader.cancel()
    }

    func openEditor() {
        guard let data = response else { return }
        itemsToken += 1
        editor = Editor(
            visible: true,
            draft: GearMatchRules.draft(from: data.me),
            itemsLoading: true
        )
    }

    func closeEditor() {
        itemsToken += 1
        editor.visible = false
        editor.saving = false
    }

    func loadItems() async {
        itemsToken += 1
        let token = itemsToken
        editor.itemsLoading = true
        editor.itemsError = ""
        do {
            let response = try await api.mine()
            guard token == itemsToken, editor.visible else { return }
            editor.items = response.items
            editor.itemsLoading = false
        } catch {
            guard token == itemsToken else { return }
            editor.itemsLoading = false
            editor.itemsError = "装备加载失败"
        }
    }

    func chooseCue(_ item: GearItemInfo?) {
        editor.draft = GearMatchRules.applyCue(item, to: editor.draft)
    }

    func chooseBreak(_ item: GearItemInfo?) {
        editor.draft.breakItemId = item?.id ?? 0
    }

    func clearPart(tip: Bool) {
        setPart(nil, tip: tip)
    }

    func setPart(_ product: GearProductInfo?, tip: Bool) {
        if tip {
            editor.draft.tipProduct = product
        } else {
            editor.draft.chalkProduct = product
        }
    }

    /// 保存失败返回提示文案；成功返回 nil，由页面弹出「已保存」。
    func save() async -> String? {
        guard !editor.saving, let data = response else { return nil }
        editor.saving = true
        let identity = session.identity
        let payload = GearMatchRules.payload(
            matchId: matchId,
            draft: editor.draft,
            supportsBreak: data.supportsBreak == true
        )
        do {
            let result = try await api.updateMatch(payload)
            guard identity == session.identity else { return nil }
            guard result.success else {
                editor.saving = false
                return result.message.isEmpty ? "保存失败" : result.message
            }
            closeEditor()
            if result.me != nil {
                phase = .ready(result)
            } else {
                await load()
            }
            return nil
        } catch {
            guard identity == session.identity else { return nil }
            editor.saving = false
            return "保存失败，请重试"
        }
    }
}
