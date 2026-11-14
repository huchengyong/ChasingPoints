import Foundation

/// 编辑草稿：只保存四个槽位当前选中的 id / 目录型号。
struct GearMatchDraft: Equatable {
    var cueItemId: Int64 = 0
    var tipProduct: GearProductInfo?
    var chalkProduct: GearProductInfo?
    var breakItemId: Int64 = 0
}

/// 展示行：球杆 / 皮头 / 巧克 / 冲杆之一。
struct GearMatchRow: Equatable {
    enum Subject: Equatable {
        case item(GearItemInfo)
        case product(GearProductInfo)
    }

    let key: String
    let label: String
    let title: String
    let subtitle: String
    let subject: Subject
}

struct GearMatchSideView: Equatable {
    let recorded: Bool
    let rows: [GearMatchRow]
}

/// `UpdateGearMatchReq`：四项都为 0 表示清空本场记录。
struct GearMatchPayload: Encodable, Equatable {
    let matchId: Int64
    let cueItemId: Int64
    let tipProductId: Int64
    let chalkProductId: Int64
    let breakItemId: Int64
}

/// 逐条对应小程序 `utils/gear-match.js`（不含皮头 / 巧克目录选择页的全局事件链路）。
enum GearMatchRules {
    /// 服务端用这些原因拒绝：不是选手 / 对局未结束 / 对局不存在。卡片直接隐藏，不当作错误。
    static let hiddenReasons: Set<String> = ["not_participant", "not_completed", "not_found"]
    static let completedStatus = 2

    static func shouldShowMatchGear(
        status: Int?,
        viewerRole: String = "",
        isPlayer1: Bool = false,
        isSpectate: Bool = false
    ) -> Bool {
        if isSpectate { return false }
        // 裁判不是参与者，is_player1 可能为 true，但用杆接口只对本场选手开放。
        if viewerRole == "referee" { return false }
        guard status == completedStatus else { return false }
        return isPlayer1 || viewerRole == "player1" || viewerRole == "player2"
    }

    static func isHiddenResponse(_ response: GearMatchResponse?) -> Bool {
        guard let response else { return false }
        return response.success == false && hiddenReasons.contains(response.reason ?? "")
    }

    static func describeSide(_ side: GearMatchSide?) -> GearMatchSideView {
        guard let side, side.recorded else { return GearMatchSideView(recorded: false, rows: []) }
        var rows: [GearMatchRow] = []
        if let cue = side.cue {
            rows.append(GearMatchRow(
                key: "cue",
                label: "球杆",
                title: GearDisplayRules.itemTitle(cue),
                subtitle: GearDisplayRules.itemSubtitle(cue),
                subject: .item(cue)
            ))
        }
        if let tip = side.tipProduct {
            rows.append(GearMatchRow(
                key: "tip",
                label: "皮头",
                title: GearDisplayRules.productTitle(tip),
                subtitle: "",
                subject: .product(tip)
            ))
        }
        if let chalk = side.chalkProduct {
            rows.append(GearMatchRow(
                key: "chalk",
                label: "巧克",
                title: GearDisplayRules.productTitle(chalk),
                subtitle: "",
                subject: .product(chalk)
            ))
        }
        if let breakCue = side.breakCue {
            rows.append(GearMatchRow(
                key: "break",
                label: "冲杆",
                title: GearDisplayRules.itemTitle(breakCue),
                subtitle: GearDisplayRules.itemSubtitle(breakCue),
                subject: .item(breakCue)
            ))
        }
        return GearMatchSideView(recorded: true, rows: rows)
    }

    /// 「5月3日前可修改」；无法解析返回空串，由调用方决定是否展示。
    static func formatEditDeadline(_ text: String?) -> String {
        guard let date = AppFormat.parseDateTime(text) else { return "" }
        let parts = AppFormat.beijingCalendar.dateComponents([.month, .day], from: date)
        return "\(String(parts.month ?? 0))月\(String(parts.day ?? 0))日前可修改"
    }

    /// 服务端给出 editable，同时按截止时间再核对一次，页面长时间停留后不会误导。
    static func canEdit(_ response: GearMatchResponse?, now: Date = Date()) -> Bool {
        guard let response, response.success, response.editable == true else { return false }
        guard let deadline = AppFormat.parseDateTime(response.editDeadline) else { return true }
        return now < deadline
    }

    static func draft(from side: GearMatchSide?) -> GearMatchDraft {
        GearMatchDraft(
            cueItemId: side?.cue?.id ?? 0,
            tipProduct: side?.tipProduct,
            chalkProduct: side?.chalkProduct,
            breakItemId: side?.breakCue?.id ?? 0
        )
    }

    /// 换球杆时用这根杆当前的皮头与巧克预填；选「不记录」一并清空。
    static func applyCue(_ item: GearItemInfo?, to draft: GearMatchDraft) -> GearMatchDraft {
        GearMatchDraft(
            cueItemId: item?.id ?? 0,
            tipProduct: item?.tipProduct,
            chalkProduct: item?.chalkProduct,
            breakItemId: draft.breakItemId
        )
    }

    static func isDraftEmpty(_ draft: GearMatchDraft) -> Bool {
        draft.cueItemId == 0 && draft.tipProduct == nil && draft.chalkProduct == nil && draft.breakItemId == 0
    }

    static func payload(matchId: Int64, draft: GearMatchDraft, supportsBreak: Bool = true) -> GearMatchPayload {
        GearMatchPayload(
            matchId: matchId,
            cueItemId: draft.cueItemId,
            tipProductId: draft.tipProduct?.id ?? 0,
            chalkProductId: draft.chalkProduct?.id ?? 0,
            breakItemId: supportsBreak ? draft.breakItemId : 0
        )
    }

    /// 补记历史对局：未删除的球杆都可选，已退役的也可以。
    static func cueOptions(_ items: [GearItemInfo]) -> [GearItemInfo] {
        items.filter { $0.category == GearCategory.cue.rawValue && $0.status != GearItemStatus.deleted.rawValue }
    }

    static func breakOptions(_ items: [GearItemInfo]) -> [GearItemInfo] {
        items.filter {
            $0.category == GearCategory.breakJump.rawValue
                && $0.status != GearItemStatus.deleted.rawValue
                && $0.roles.contains("break")
        }
    }
}

/// 带身份校验的读取器：只认最新一次读取；读取期间切号、退出或再次读取，旧响应一律丢弃。
@MainActor
final class GearMatchLoader {
    struct Identity: Equatable {
        let userId: Int64?
        let generation: Int?
    }

    private let request: (Int64) async throws -> GearMatchResponse
    private let readIdentity: () -> Identity
    private var sequence = 0

    init(
        request: @escaping (Int64) async throws -> GearMatchResponse,
        readIdentity: @escaping () -> Identity
    ) {
        self.request = request
        self.readIdentity = readIdentity
    }

    func load(_ matchId: Int64) async throws -> (stale: Bool, response: GearMatchResponse) {
        let identity = readIdentity()
        sequence += 1
        let current = sequence
        let response = try await request(matchId)
        if current != sequence || identity != readIdentity() {
            return (true, response)
        }
        return (false, response)
    }

    /// 页面卸载或本场对局变化时调用：进行中的读取不再回写。
    func cancel() {
        sequence += 1
    }
}
