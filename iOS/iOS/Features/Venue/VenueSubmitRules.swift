import Foundation

struct VenueRegionSelection: Equatable {
    let areaIds: [Int64]
    let regionText: String
    let city: String
    let district: String
}

struct PickedLocation: Equatable {
    let latitude: Double
    let longitude: Double
    let name: String
    let address: String
}

/// 上传球馆表单，对应 submit.vue 的 `form`。
struct VenueSubmitForm: Equatable {
    var name = ""
    var regionText = ""
    var areaIds: [Int64] = []
    var city = ""
    var district = ""
    var address = ""
    var location: PickedLocation?
}

/// 只提交后端 `CreateVenueReq` 接收的字段；未选点时坐标不编码。
struct VenueSubmitPayload: Encodable, Equatable {
    let name: String
    let city: String
    let district: String
    let address: String
    let latitude: Double?
    let longitude: Double?
}

struct VenueSubmitIntro: Equatable {
    let title: String
    let desc: String
    let sectionTip: String
}

struct VenueSubmitBarCopy: Equatable {
    let title: String
    let tip: String
}

/// 迁移小程序 `utils/venue-submit.js` 的纯规则。
enum VenueSubmitRules {
    static func regionSelection(_ path: [VenueAreaOption]) -> VenueRegionSelection {
        let normalized = path.filter { !trim($0.name).isEmpty }
        let names = normalized.map { trim($0.name) }
        return VenueRegionSelection(
            areaIds: normalized.map(\.areaId).filter { $0 > 0 },
            regionText: names.joined(separator: " "),
            city: names.count > 1 ? names[1] : "",
            district: names.count > 2 ? names[2] : ""
        )
    }

    /// 与小程序一致：经纬度必须有限且都非 0。
    static func normalizePickedLocation(
        latitude: Double?,
        longitude: Double?,
        name: String = "",
        address: String = ""
    ) -> PickedLocation? {
        guard let latitude, let longitude,
              latitude.isFinite, longitude.isFinite,
              latitude != 0, longitude != 0 else { return nil }
        return PickedLocation(
            latitude: latitude,
            longitude: longitude,
            name: trim(name),
            address: trim(address)
        )
    }

    /// 与小程序一致：按长度从长到短逐个去掉地址开头的地区名，每项只去一次。
    static func stripRegionPrefix(_ address: String, regionNames: [String]) -> String {
        var rest = trim(address)
        var seen = Set<String>()
        let unique = regionNames.map { trim($0) }.filter { !$0.isEmpty && seen.insert($0).inserted }
        var candidates = unique.enumerated()
            .sorted { lhs, rhs in
                lhs.element.count == rhs.element.count
                    ? lhs.offset < rhs.offset
                    : lhs.element.count > rhs.element.count
            }
            .map(\.element)

        while !candidates.isEmpty {
            guard let prefix = candidates.first(where: { rest.hasPrefix($0) }) else { break }
            rest = String(rest.dropFirst(prefix.count))
            candidates.removeAll { $0 == prefix }
        }
        return trim(rest)
    }

    static func payload(_ form: VenueSubmitForm) -> VenueSubmitPayload {
        let location = form.location.flatMap {
            normalizePickedLocation(
                latitude: $0.latitude,
                longitude: $0.longitude,
                name: $0.name,
                address: $0.address
            )
        }
        return VenueSubmitPayload(
            name: trim(form.name),
            city: trim(form.city),
            district: trim(form.district),
            address: trim(form.address),
            latitude: location?.latitude,
            longitude: location?.longitude
        )
    }

    /// 只有常玩球馆奖励可领取时才提 VIP，否则只说明普通审核流程。
    static func resolveIntro(_ rewardStatus: FavoriteVenueRewardStatus?) -> VenueSubmitIntro {
        let duration = FavoriteVenueReward.formatDuration(rewardDays: FavoriteVenueReward.rewardDays(rewardStatus))
        if FavoriteVenueReward.isClaimable(rewardStatus) {
            return VenueSubmitIntro(
                title: "免费领取 \(duration)",
                desc: "填写球馆真实的位置信息，审核通过后会自动发放VIP。虚假信息将不予通过。",
                sectionTip: "请填写球馆名称、地区和详细地址，审核通过后会自动发放VIP。"
            )
        }
        return VenueSubmitIntro(
            title: "提交常去球馆",
            desc: "填写球馆真实的位置信息，审核通过后会展示给附近球友。虚假信息将不予通过。",
            sectionTip: "请填写球馆名称、地区和详细地址，审核通过后会展示给附近球友。"
        )
    }

    static func resolveBarCopy(
        missingRequiredLabels: [String],
        rewardStatus: FavoriteVenueRewardStatus?
    ) -> VenueSubmitBarCopy {
        if !missingRequiredLabels.isEmpty {
            return VenueSubmitBarCopy(
                title: "还有 \(missingRequiredLabels.count) 项未填写",
                tip: "请先填写：\(missingRequiredLabels.joined(separator: "、"))"
            )
        }
        let tip = FavoriteVenueReward.isClaimable(rewardStatus)
            ? "首次有效补充并审核通过，送 \(FavoriteVenueReward.formatDuration(rewardDays: FavoriteVenueReward.rewardDays(rewardStatus)))。"
            : "审核通过后会展示给附近球友。"
        return VenueSubmitBarCopy(title: "确认常玩球馆后提交", tip: tip)
    }

    private static func trim(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
