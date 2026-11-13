import Foundation

/// 球种目录，顺序与小程序 `utils/game-types.js` 一致。
struct GameTypeOption: Equatable, Identifiable {
    let value: Int
    /// chinese_eight / nine_ball / snooker / american_nine。
    let key: String
    let label: String

    var id: Int { value }
}

enum GameTypeCatalog {
    static let ordered: [GameTypeOption] = [
        GameTypeOption(value: 3, key: "chinese_eight", label: "中式八球"),
        GameTypeOption(value: 2, key: "nine_ball", label: "九球追分"),
        GameTypeOption(value: 1, key: "snooker", label: "斯诺克"),
        GameTypeOption(value: 4, key: "american_nine", label: "美式九球"),
    ]

    static func label(for gameType: Int, fallback: String = "台球") -> String {
        ordered.first { $0.value == gameType }?.label ?? fallback
    }

    static func key(for gameType: Int) -> String? {
        ordered.first { $0.value == gameType }?.key
    }

    static func value(forKey key: String) -> Int? {
        ordered.first { $0.key == key }?.value
    }

    static func isSupported(_ gameType: Int) -> Bool {
        ordered.contains { $0.value == gameType }
    }
}
