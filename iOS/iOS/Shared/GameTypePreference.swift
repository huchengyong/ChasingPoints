import Foundation

/// 用户的常用球种，按账号保存在本机；0 表示「每次询问」。对应小程序 `utils/game-type-preference.js`。
struct GameTypePreference {
    private static let storagePrefix = "default-game-type"

    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    /// `default-game-type:<userId>`；没有有效账号时不落盘。
    static func storageKey(userId: Int64) -> String? {
        userId > 0 ? "\(storagePrefix):\(userId)" : nil
    }

    /// 不支持的球种一律按「每次询问」处理。
    static func normalize(_ value: Int) -> Int {
        GameTypeCatalog.isSupported(value) ? value : 0
    }

    func read(userId: Int64) -> Int {
        guard let key = Self.storageKey(userId: userId) else { return 0 }
        return Self.normalize(defaults.integer(forKey: key))
    }

    /// 0 或不支持的值会删除已保存的偏好并返回 0。
    @discardableResult
    func save(userId: Int64, gameType: Int) -> Int {
        guard let key = Self.storageKey(userId: userId) else { return 0 }
        let normalized = Self.normalize(gameType)
        if normalized > 0 {
            defaults.set(normalized, forKey: key)
            return normalized
        }
        defaults.removeObject(forKey: key)
        return 0
    }
}
