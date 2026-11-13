import Foundation

/// 球馆公开信息，字段与 `backend/chasing_points.api` 的 `VenueInfo` 对齐。
struct VenueInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let name: String
    let address: String
    let city: String
    let district: String
    let latitude: Double
    let longitude: Double
    let phone: String
    let images: [String]
    let businessHours: String
    let tableCount: Int
    let priceRange: String
    let description: String
    /// 列表与附近接口按距离排序时返回；为 `omitempty`，可能缺省。
    let distance: Double?
    let checkinCount: Int
    let status: Int
}

/// 省 / 市 / 区选项，`area_id` 为 0 的是占位项。
struct VenueAreaOption: Decodable, Equatable, Identifiable {
    let areaId: Int64
    let parentId: Int64
    let name: String

    var id: Int64 { areaId }

    static func placeholder(_ name: String) -> VenueAreaOption {
        VenueAreaOption(areaId: 0, parentId: 0, name: name)
    }
}

enum VenueDistance {
    /// 与小程序 formatDistance 一致：不足 1 公里取整米数，否则保留一位小数。
    static func format(_ meters: Double) -> String {
        if meters < 1000 {
            return "\(Int(meters.rounded()))m"
        }
        return String(format: "%.1fkm", meters / 1000)
    }

    /// 小程序以真值判断是否展示距离，0 表示没有距离信息。
    static func isVisible(_ meters: Double?) -> Bool {
        guard let meters else { return false }
        return meters > 0
    }
}
