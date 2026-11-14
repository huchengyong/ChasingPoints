import Foundation

/// 模型只依赖这几个操作，测试用替身推进消息；生产实现是 `MatchRealtime`。
@MainActor
protocol LiveMatchChannel: AnyObject {
    var isConnected: Bool { get }
    func openLive(matchId: Int64, allowAnonymous: Bool) async throws
    func requestLiveSync() async
    func setLiveActive(_ active: Bool)
    func closeLive()
    @discardableResult
    func observeLive(_ handler: @escaping (RealtimeMessage) -> Void) -> UUID
    func stopObservingLive(_ token: UUID)
}

extension MatchRealtime: LiveMatchChannel {
    func openLive(matchId: Int64, allowAnonymous: Bool) async throws {
        try await connect(matchId: matchId, allowAnonymous: allowAnonymous)
    }

    func requestLiveSync() async {
        await requestSync()
    }

    func setLiveActive(_ active: Bool) {
        setTargetActive(active)
    }

    func closeLive() {
        disconnect()
    }

    func observeLive(_ handler: @escaping (RealtimeMessage) -> Void) -> UUID {
        on(WSMessageType.any, handler)
    }

    func stopObservingLive(_ token: UUID) {
        off(token)
    }
}
