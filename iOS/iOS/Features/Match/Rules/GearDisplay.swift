import Foundation

/// 装备类别与状态，取值与小程序 `utils/gear-display.js` 一致。
enum GearCategory {
    static let cue = "cue"
    static let breakJump = "break_jump"
    static let tip = "tip"
    static let chalk = "chalk"
}

enum GearItemStatus {
    static let active = 1
    static let retired = 2
    static let deleted = 3
}

/// 装备图片：优先目录图，缺失或不合规时回落到按类别选出的默认图。
struct GearImage: Equatable {
    let src: URL?
    let fallback: URL?
}

/// 逐条对应小程序 `utils/gear-display.js` 里「本场用杆」卡用到的部分。
enum GearDisplay {
    private static let pattern = try! NSRegularExpression(
        pattern: #"^/static/images/gear/[A-Za-z0-9][A-Za-z0-9._-]*\.(?:png|webp)$"#
    )

    static func isGearImagePath(_ path: String?) -> Bool {
        guard let path, !path.isEmpty else { return false }
        let range = NSRange(path.startIndex..., in: path)
        return pattern.firstMatch(in: path, range: range) != nil && CDNAsset.isCDNPath(path)
    }

    static func defaultImagePath(category: String, house: Bool = false, cueStyle: String = "", roles: [String]? = nil) -> String {
        switch category {
        case GearCategory.tip:
            return "/static/images/gear/gear-tip-default.png"
        case GearCategory.chalk:
            return "/static/images/gear/gear-chalk-default.png"
        case GearCategory.breakJump:
            return (roles ?? []).contains("break")
                ? "/static/images/gear/gear-break-cue-default.png"
                : "/static/images/gear/gear-jump-cue-default.png"
        default:
            if house { return "/static/images/gear/gear-house-cue-default.png" }
            return cueStyle == "pool"
                ? "/static/images/gear/gear-cue-pool-default.png"
                : "/static/images/gear/gear-cue-snooker-default.png"
        }
    }

    static func image(_ subject: GearItemInfo?) -> GearImage {
        guard let subject else { return GearImage(src: nil, fallback: nil) }
        return image(
            category: subject.category,
            house: subject.house,
            cueStyle: subject.cueStyle ?? "",
            roles: subject.roles,
            productImagePath: subject.product?.imagePath
        )
    }

    static func image(_ product: GearProductInfo?) -> GearImage {
        guard let product else { return GearImage(src: nil, fallback: nil) }
        return image(
            category: product.category,
            house: product.house,
            cueStyle: product.cueStyle ?? "",
            roles: product.roles,
            productImagePath: product.imagePath
        )
    }

    private static func image(
        category: String,
        house: Bool,
        cueStyle: String,
        roles: [String]?,
        productImagePath: String?
    ) -> GearImage {
        let fallbackPath = defaultImagePath(category: category, house: house, cueStyle: cueStyle, roles: roles)
        let imagePath = isGearImagePath(productImagePath) ? (productImagePath ?? "") : ""
        return GearImage(
            src: CDNAsset.url(for: imagePath.isEmpty ? fallbackPath : imagePath),
            fallback: CDNAsset.url(for: fallbackPath)
        )
    }

    /// 装备标题：品牌 + 型号；球房公用条目没有品牌，只显示型号名。
    static func itemTitle(_ item: GearItemInfo?) -> String {
        guard let item else { return "" }
        let parts = [item.brandName, item.modelName]
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        return parts.isEmpty ? "未命名装备" : parts.joined(separator: " ")
    }

    /// 装备副标题：球杆显示杆型，冲跳装备显示身份；未收录型号加一个提示。
    static func itemSubtitle(_ item: GearItemInfo?) -> String {
        guard let item else { return "" }
        let base = item.category == GearCategory.breakJump
            ? formatRoles(item.roles)
            : (item.house ? "球房公用" : cueStyleLabel(item.cueStyle ?? ""))
        return [base, item.unlisted ? "未收录" : ""].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    static func productTitle(_ product: GearProductInfo?) -> String {
        guard let product else { return "" }
        let parts = [product.brandName, product.name]
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        return parts.joined(separator: " ")
    }

    static func formatRoles(_ roles: [String]?) -> String {
        let labels = (roles ?? []).compactMap { role -> String? in
            switch role {
            case "break": "冲杆"
            case "jump": "跳杆"
            default: nil
            }
        }
        return labels.joined(separator: " · ")
    }

    static func cueStyleLabel(_ cueStyle: String) -> String {
        switch cueStyle {
        case "pool": "美式杆"
        case "snooker": "斯诺克 / 中式杆"
        default: ""
        }
    }
}
