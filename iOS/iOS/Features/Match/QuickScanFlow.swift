import Foundation

/// 确认页的两种模式：扫码开局与扫码执裁。
enum ScanConfirmMode: String, Equatable {
    case start
    case referee
}

/// 相机扫码失败的原因；文案与小程序 `match-scan.js` 逐字一致。
enum ScanFailure: Equatable {
    case permissionDenied
    case unsupported
    case failed(String?)

    var message: String? {
        switch self {
        case .permissionDenied: "需要相机权限才能扫码"
        case .unsupported: "当前端暂不支持扫码"
        case .failed(let message): message?.isEmpty == false ? message : nil
        }
    }
}

/// 相机一端的原始结果；解析与请求由流程完成。
enum ScanOutcome: Equatable {
    case scanned(String)
    case cancelled
    case failure(ScanFailure)
}

/// 赛果页「再来一局」带入的上下文，只接受原对手的匹配码。
struct ScanRematch: Equatable {
    let opponentId: Int64
    let gameType: Int
    let matchMode: String?
    let visibility: String?
}

struct ScanRematchPreset: Equatable {
    let gameType: Int
    let matchMode: String
    let visibility: String
}

/// 扫码结果到确认页的一次性交接；凭证不进路由、本地存储与日志。
struct ScanHandoff: Equatable, CustomStringConvertible {
    let identity: SessionIdentity
    let mode: ScanConfirmMode
    let inviteToken: String?
    let opponent: MatchScanOpponent?
    let refereeJoin: RefereeJoin?
    let preset: ScanRematchPreset?
    let rematch: ScanRematch?

    var description: String { "ScanHandoff(mode: \(mode.rawValue), <redacted>)" }
}

/// 逐条对应小程序 `utils/quick-scan.js`：先扫后解析，再把凭证一次性交给确认页。
@MainActor
final class QuickScanFlow {
    /// 与 `quick-scan-entry.js` 的模块级单例一致；应用启动时 `bind(session:)` 接上真实会话。
    static let shared = QuickScanFlow()

    enum StartResult: Equatable {
        case busy
        case login
        case stale
        case cancelled
        /// 需要轻提示；nil 表示默认「扫码失败」。
        case error(message: String?)
        /// 观赛详情尚未在原生端开放，调用方只需关闭扫码页。
        case spectate(matchId: Int64)
        case mismatch
        case confirm(mode: ScanConfirmMode)
    }

    private var currentIdentity: () -> SessionIdentity?
    private var api: MatchStartAPI?
    private var scanning = false
    // 凭证只在内存里交给确认页使用一次。
    private var handoff: ScanHandoff?

    init(
        currentIdentity: @escaping () -> SessionIdentity? = { nil },
        api: MatchStartAPI? = nil
    ) {
        self.currentIdentity = currentIdentity
        self.api = api
    }

    func bind(session: SessionStore, api: MatchStartAPI? = nil) {
        self.currentIdentity = { session.identity }
        self.api = api ?? MatchStartAPI(session: session)
    }

    /// rematch：赛果页「再来一局」传入上一场上下文。`scan` 由相机视图提供，测试可直接注入结果。
    func start(rematch: ScanRematch? = nil, scan: () async throws -> ScanOutcome) async throws -> StartResult {
        if scanning { return .busy }
        guard let identity = currentIdentity() else { return .login }

        scanning = true
        defer { scanning = false }

        let outcome = try await scan()
        guard currentIdentity() == identity else { return .stale }

        switch outcome {
        case .cancelled:
            return .cancelled
        case .failure(let failure):
            return .error(message: failure.message)
        case .scanned(let rawValue):
            return await resolve(rawValue: rawValue, identity: identity, rematch: rematch)
        }
    }

    /// 只交给扫码时的身份；取走或身份不符都会清掉，凭证不会滞留。
    func takeHandoff(identity: SessionIdentity?) -> ScanHandoff? {
        let pending = handoff
        handoff = nil
        guard let pending, let identity else { return nil }
        return pending.identity == identity ? pending : nil
    }

    /// 承接页（入口链接）兑换成功后交接凭证，不走扫码；身份已变化时丢弃。
    @discardableResult
    func handoffResolved(identity: SessionIdentity?, inviteToken: String?, opponent: MatchScanOpponent) -> Bool {
        guard let identity, currentIdentity() == identity else { return false }
        guard let inviteToken, !inviteToken.isEmpty else { return false }
        handoff = ScanHandoff(
            identity: identity,
            mode: .start,
            inviteToken: inviteToken,
            opponent: opponent,
            refereeJoin: nil,
            preset: nil,
            rematch: nil
        )
        return true
    }

    private func resolve(rawValue: String, identity: SessionIdentity, rematch: ScanRematch?) async -> StartResult {
        guard let api else { return .error(message: "匹配二维码服务暂不可用") }

        let result = await MatchScan.resolvePayload(
            rawValue: rawValue,
            previewMatchInvite: { try await api.previewMatchInvite($0) },
            resolveMatchInviteCode: { try await api.resolveMatchInviteCode($0) }
        )
        guard currentIdentity() == identity else { return .stale }

        switch result {
        case .error(let message):
            return .error(message: message)
        case .openMatch(let matchId):
            return .spectate(matchId: matchId)
        case .joinReferee(let refereeJoin):
            handoff = ScanHandoff(
                identity: identity,
                mode: .referee,
                inviteToken: nil,
                opponent: nil,
                refereeJoin: refereeJoin,
                preset: nil,
                rematch: nil
            )
            return .confirm(mode: .referee)
        case .startMatch(let inviteToken, let opponent):
            let expectedOpponentId = rematch?.opponentId ?? 0
            if expectedOpponentId > 0, opponent.id > 0, opponent.id != expectedOpponentId {
                return .mismatch
            }
            handoff = ScanHandoff(
                identity: identity,
                mode: .start,
                inviteToken: inviteToken,
                opponent: opponent,
                refereeJoin: nil,
                preset: rematch.map(Self.buildRematchPreset),
                rematch: rematch
            )
            return .confirm(mode: .start)
        case .resolveInviteCode, .previewMatchInvite:
            // resolvePayload 已消费这两种解析结果；保留分支只为穷尽。
            return .error(message: "匹配二维码服务暂不可用")
        }
    }

    private static func buildRematchPreset(_ rematch: ScanRematch) -> ScanRematchPreset {
        let options = StartMatch.normalizeOptions(matchMode: rematch.matchMode, visibility: rematch.visibility)
        return ScanRematchPreset(
            gameType: rematch.gameType,
            matchMode: options.matchMode,
            visibility: options.visibility
        )
    }
}
