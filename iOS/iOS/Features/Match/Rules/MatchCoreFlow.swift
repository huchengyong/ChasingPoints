import Foundation

/// 对应小程序 `utils/match-core-flow.js` 的 `buildRematchContext`。
/// 原生端只保留扫码流程实际使用的字段：对手由扫码结果带出，昵称与头像不从旧对局猜测。
enum MatchCoreFlow {
    static func rematch(_ report: MatchReport) -> ScanRematch? {
        let opponentId = report.opponentId ?? report.player2Id ?? 0
        guard opponentId > 0 else { return nil }
        return ScanRematch(
            opponentId: opponentId,
            gameType: report.gameType,
            matchMode: report.matchMode == "ranked" ? "ranked" : "practice",
            visibility: visibility(report.visibility)
        )
    }

    private static func visibility(_ value: String?) -> String? {
        guard let value, !value.isEmpty else { return nil }
        return value == "public" ? "public" : "private"
    }
}
