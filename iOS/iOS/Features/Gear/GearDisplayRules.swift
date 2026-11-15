import Foundation

/// 装备类别，取值与后端一致。
enum GearCategory: String, CaseIterable {
    case cue
    case breakJump = "break_jump"
    case tip
    case chalk

    var label: String {
        switch self {
        case .cue: "球杆"
        case .breakJump: "冲杆 / 跳杆"
        case .tip: "皮头"
        case .chalk: "巧克"
        }
    }
}

enum GearSlot: String {
    case cue
    case breakSlot = "break"
}

enum GearItemStatus: Int {
    case active = 1
    case retired = 2
    case deleted = 3
}

/// 主力设置行：某个球种的主力球杆与主力冲杆。
struct GearMainRow: Equatable, Identifiable {
    let gameType: Int
    let label: String
    let supportsBreak: Bool
    let cue: GearItemInfo?
    let breakCue: GearItemInfo?

    var id: Int { gameType }
}

/// 装备页分组：退役单独一组，其余按球杆、冲杆跳杆分组；已删除的不展示。
struct GearGroups: Equatable {
    var cues: [GearItemInfo] = []
    var breakJump: [GearItemInfo] = []
    var retired: [GearItemInfo] = []
}

struct GearSpec: Equatable {
    let key: String
    let label: String
    let value: String
}

/// 图片主体：装备取所属目录型号的图，目录型号取自身。
struct GearImageSource: Equatable {
    let category: String
    let house: Bool
    let cueStyle: String?
    let roles: [String]
    let imagePath: String?

    init(category: String, house: Bool = false, cueStyle: String? = nil, roles: [String] = [], imagePath: String? = nil) {
        self.category = category
        self.house = house
        self.cueStyle = cueStyle
        self.roles = roles
        self.imagePath = imagePath
    }

    init(item: GearItemInfo) {
        self.init(
            category: item.category,
            house: item.house,
            cueStyle: item.cueStyle,
            roles: item.roles,
            imagePath: item.product?.imagePath
        )
    }

    init(product: GearProductInfo) {
        self.init(
            category: product.category,
            house: product.house,
            cueStyle: product.cueStyle,
            roles: product.roles,
            imagePath: product.imagePath
        )
    }
}

struct GearImageURLs: Equatable {
    let src: URL?
    let fallback: URL?
}

/// 装备展示规则，对应小程序 `utils/gear-display.js`，全部是纯函数。
enum GearDisplayRules {
    static let maxItems = 30
    static let unlistedModelName = "未收录型号"

    static let snookerCueImage = "/static/images/gear/gear-cue-snooker-default.png"
    static let poolCueImage = "/static/images/gear/gear-cue-pool-default.png"
    static let houseCueImage = "/static/images/gear/gear-house-cue-default.png"
    static let breakCueImage = "/static/images/gear/gear-break-cue-default.png"
    static let jumpCueImage = "/static/images/gear/gear-jump-cue-default.png"
    static let tipImage = "/static/images/gear/gear-tip-default.png"
    static let chalkImage = "/static/images/gear/gear-chalk-default.png"

    /// 后台只允许这个形态的目录图路径；不合规一律回退默认图，不信任服务端返回的任意地址。
    private static let imagePathRegex = try! NSRegularExpression(
        pattern: #"^/static/images/gear/[A-Za-z0-9][A-Za-z0-9._-]*\.(?:png|webp)$"#
    )

    static func isGearImagePath(_ path: String?) -> Bool {
        guard let path, !path.isEmpty, CDNAsset.isCDNPath(path) else { return false }
        let range = NSRange(path.startIndex..., in: path)
        return imagePathRegex.firstMatch(in: path, range: range) != nil
    }

    static func defaultImagePath(category: String, house: Bool = false, cueStyle: String? = nil, roles: [String] = []) -> String {
        switch category {
        case GearCategory.tip.rawValue:
            return tipImage
        case GearCategory.chalk.rawValue:
            return chalkImage
        case GearCategory.breakJump.rawValue:
            // 冲跳装备带冲杆身份时用冲杆图，否则用跳杆图。
            return roles.contains("break") ? breakCueImage : jumpCueImage
        default:
            if house { return houseCueImage }
            return cueStyle == "pool" ? poolCueImage : snookerCueImage
        }
    }

    static func image(for source: GearImageSource) -> GearImageURLs {
        let fallbackPath = defaultImagePath(
            category: source.category,
            house: source.house,
            cueStyle: source.cueStyle,
            roles: source.roles
        )
        let path = isGearImagePath(source.imagePath) ? source.imagePath! : fallbackPath
        return GearImageURLs(src: CDNAsset.url(for: path), fallback: CDNAsset.url(for: fallbackPath))
    }

    /// 官方规格：品牌没有提供的项目不展示（后端用 0 表示未提供），重量同时给出克与盎司。
    static func specs(_ product: GearProductInfo?) -> [GearSpec] {
        guard let product else { return [] }
        var specs: [GearSpec] = []
        if let length = product.lengthMm, length > 0 {
            specs.append(GearSpec(key: "length", label: "长度", value: "\(String(length)) mm"))
        }
        if let grams = product.weightG, grams > 0 {
            let ounces = String(format: "%.1f", Double(grams) / 28.3495)
            specs.append(GearSpec(key: "weight", label: "重量", value: "\(String(grams)) g（\(ounces) oz）"))
        }
        if let diameter = product.tipDiameterMm, diameter > 0 {
            specs.append(GearSpec(key: "tipDiameter", label: "皮头直径", value: "\(trimNumber(diameter)) mm"))
        }
        return specs
    }

    /// 最多两位小数并去掉尾零，与 JS `Number(v.toFixed(2))` 的展示一致。
    private static func trimNumber(_ value: Double) -> String {
        var text = String(format: "%.2f", (value * 100).rounded() / 100)
        while text.hasSuffix("0") { text.removeLast() }
        if text.hasSuffix(".") { text.removeLast() }
        return text
    }

    static func roles(_ roles: [String]) -> String {
        var labels: [String] = []
        if roles.contains("break") { labels.append("冲杆") }
        if roles.contains("jump") { labels.append("跳杆") }
        return labels.joined(separator: " · ")
    }

    static func cueStyle(_ cueStyle: String?) -> String {
        switch cueStyle {
        case "pool": "美式杆"
        case "snooker": "斯诺克 / 中式杆"
        default: ""
        }
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
        let base = item.category == GearCategory.breakJump.rawValue
            ? roles(item.roles)
            : (item.house ? "球房公用" : cueStyle(item.cueStyle))
        return [base, item.unlisted ? "未收录" : ""].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    static func productTitle(_ product: GearProductInfo?) -> String {
        guard let product else { return "" }
        return [product.brandName, product.name]
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .joined(separator: " ")
    }

    /// 目录卡片副标题：杆型或身份 + 长度与皮头直径（重量只取克数）。
    static func productSubtitle(_ product: GearProductInfo?) -> String {
        guard let product else { return "" }
        var parts: [String] = []
        if product.category == GearCategory.cue.rawValue {
            let style = cueStyle(product.cueStyle)
            if !style.isEmpty { parts.append(style) }
        } else if product.category == GearCategory.breakJump.rawValue {
            let roles = roles(product.roles)
            if !roles.isEmpty { parts.append(roles) }
        }
        for spec in specs(product) {
            parts.append(spec.key == "weight" ? spec.value.components(separatedBy: "（").first ?? spec.value : spec.value)
        }
        return parts.joined(separator: " · ")
    }

    static func groups(_ items: [GearItemInfo]) -> GearGroups {
        var groups = GearGroups()
        for item in items {
            if item.status == GearItemStatus.deleted.rawValue { continue }
            if item.status == GearItemStatus.retired.rawValue {
                groups.retired.append(item)
            } else if item.category == GearCategory.breakJump.rawValue {
                groups.breakJump.append(item)
            } else {
                groups.cues.append(item)
            }
        }
        return groups
    }

    static func isActive(_ item: GearItemInfo?) -> Bool {
        item?.status == GearItemStatus.active.rawValue
    }

    static func canBeMainCue(_ item: GearItemInfo?) -> Bool {
        isActive(item) && item?.category == GearCategory.cue.rawValue
    }

    static func canBeMainBreakCue(_ item: GearItemInfo?) -> Bool {
        isActive(item)
            && item?.category == GearCategory.breakJump.rawValue
            && item?.roles.contains("break") == true
    }

    /// 某个槽位可选的主力装备：只有在用的球杆能做主力球杆，只有在用且带冲杆身份的冲跳装备能做主力冲杆。
    static func mainCandidates(_ items: [GearItemInfo], slot: GearSlot) -> [GearItemInfo] {
        items.filter { slot == .breakSlot ? canBeMainBreakCue($0) : canBeMainCue($0) }
    }

    /// 主力设置视图：按球种固定顺序列出，斯诺克没有冲杆槽位。
    static func defaultsView(_ defaults: [GearDefaultInfo], items: [GearItemInfo]) -> [GearMainRow] {
        let itemById = Dictionary(items.map { ($0.id, $0) }, uniquingKeysWith: { first, _ in first })
        func pick(_ gameType: Int, _ slot: GearSlot) -> GearItemInfo? {
            guard let entry = defaults.first(where: { $0.gameType == gameType && $0.slot == slot.rawValue }) else { return nil }
            return itemById[entry.itemId]
        }
        return GameTypeCatalog.ordered.map { option in
            let supportsBreak = option.value != 1
            return GearMainRow(
                gameType: option.value,
                label: option.label,
                supportsBreak: supportsBreak,
                cue: pick(option.value, .cue),
                breakCue: supportsBreak ? pick(option.value, .breakSlot) : nil
            )
        }
    }

    /// 装备卡片上的主力标签：球杆写「中式八球 主力」，冲杆写「中式八球 冲杆主力」。
    static func mainGameLabels(_ defaults: [GearDefaultInfo], itemId: Int64) -> [String] {
        GameTypeCatalog.ordered.flatMap { option in
            defaults
                .filter { $0.itemId == itemId && $0.gameType == option.value }
                .map { "\(option.label) \($0.slot == GearSlot.breakSlot.rawValue ? "冲杆主力" : "主力")" }
        }
    }
}
