import Foundation

/// 写动作负载里的标量值。规则只负责原样搬运调用方给的字段，所以这里不引入通用的 JSON 模型。
enum MatchActionValue: Equatable, Encodable {
    case int(Int)
    case double(Double)
    case string(String)
    case bool(Bool)

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .int(let value): try container.encode(value)
        case .double(let value): try container.encode(value)
        case .string(let value): try container.encode(value)
        case .bool(let value): try container.encode(value)
        }
    }
}

/// 逐条对应小程序 `utils/match-action.js` 的纯规则：客户端动作 ID 与写动作负载。
enum MatchAction {
    /// 客户端动作序号，跨调用单调递增并对 1000000 取模。
    private static var sequence = 0

    static func createActionId(now: Date = Date()) -> String {
        sequence = (sequence + 1) % 1_000_000
        let timePart = String(Int64(now.timeIntervalSince1970 * 1000), radix: 36)
        let sequenceText = String(sequence, radix: 36)
        let sequencePart = String(repeating: "0", count: max(0, 4 - sequenceText.count)) + sequenceText
        let randomPart = String(UInt64.random(in: 0..<2_176_782_336), radix: 36)
        let randomText = String(repeating: "0", count: max(0, 6 - randomPart.count)) + randomPart
        return "ma_\(timePart)_\(sequencePart)_\(randomText)"
    }

    /// 附加 `client_action_id` 与 `base_revision`；客户端还没同步到快照时回退为 0。
    static func buildPayload(
        _ payload: [String: MatchActionValue],
        baseRevision: Int?,
        createActionId: () -> String = { MatchAction.createActionId() }
    ) -> [String: MatchActionValue] {
        var result = payload
        result["client_action_id"] = .string(createActionId())
        result["base_revision"] = .int(baseRevision ?? 0)
        return result
    }
}
