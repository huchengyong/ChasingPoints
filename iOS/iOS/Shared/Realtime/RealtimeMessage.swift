import Foundation

/// 与服务端 `ws.Message` 信封一致：`{ "type": ..., "data": ... }`。
struct RealtimeMessage: Equatable {
    let type: String
    /// 原始 `data` JSON；业务侧按各自模型解码，通道不持有业务结构。
    let payload: Data?

    init(type: String, payload: Data? = nil) {
        self.type = type
        self.payload = payload
    }

    /// 解析失败的消息返回 nil，由连接层忽略。
    init?(text: String) {
        guard let raw = text.data(using: .utf8),
              let object = (try? JSONSerialization.jsonObject(with: raw)) as? [String: Any],
              let type = object["type"] as? String else {
            return nil
        }
        self.type = type
        if let data = object["data"], !(data is NSNull),
           let payload = try? JSONSerialization.data(withJSONObject: data, options: .fragmentsAllowed) {
            self.payload = payload
        } else {
            self.payload = nil
        }
    }
}

/// `WS_MESSAGE_TYPES` 的全量类型常量。
enum WSMessageType {
    static let scoreUpdate = "score_update"
    static let roundEnd = "round_end"
    static let roundStart = "round_start"
    static let matchEnd = "match_end"
    static let matchStart = "match_start"
    static let matchRoleChanged = "match_role_changed"
    static let matchFinishRequest = "match_finish_request"
    static let matchFinishConfirm = "match_finish_confirm"
    static let matchFinishDispute = "match_finish_dispute"
    static let matchFinishWithdraw = "match_finish_withdraw"
    static let matchFinishExpired = "match_finish_expired"
    static let matchFinishTimeout = "match_finish_timeout"
    static let notificationUpdate = "notification_update"
    static let rankInfoUpdated = "rank_info_updated"
    static let userDataUpdated = "user_data_updated"
    static let sessionInvalid = "session_invalid"
    static let sync = "sync"
    static let ping = "ping"
    static let pong = "pong"

    /// 对应 JS 里的 `'*'` 全量订阅。
    static let any = "*"
}
