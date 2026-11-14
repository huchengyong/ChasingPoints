import Foundation

/// 目录型号；规格任一项为 0 表示品牌未提供。
struct GearProductInfo: Decodable, Equatable {
    var id: Int64 = 0
    var category = ""
    var brandId: Int64 = 0
    var brandName = ""
    var name = ""
    var house = false
    var cueStyle: String?
    var roles: [String]?
    var lengthMm: Int = 0
    var weightG: Int = 0
    var tipDiameterMm: Double = 0
    var imagePath: String?

    private enum CodingKeys: String, CodingKey {
        case id, category, brandId, brandName, name, house, cueStyle, roles
        case lengthMm, weightG, tipDiameterMm, imagePath
    }

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decodeIfPresent(Int64.self, forKey: .id) ?? 0
        category = try c.decodeIfPresent(String.self, forKey: .category) ?? ""
        brandId = try c.decodeIfPresent(Int64.self, forKey: .brandId) ?? 0
        brandName = try c.decodeIfPresent(String.self, forKey: .brandName) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        house = try c.decodeIfPresent(Bool.self, forKey: .house) ?? false
        cueStyle = try c.decodeIfPresent(String.self, forKey: .cueStyle)
        roles = try c.decodeIfPresent([String].self, forKey: .roles)
        lengthMm = try c.decodeIfPresent(Int.self, forKey: .lengthMm) ?? 0
        weightG = try c.decodeIfPresent(Int.self, forKey: .weightG) ?? 0
        tipDiameterMm = try c.decodeIfPresent(Double.self, forKey: .tipDiameterMm) ?? 0
        imagePath = try c.decodeIfPresent(String.self, forKey: .imagePath)
    }
}

/// 用户装备；`product` / `tipProduct` / `chalkProduct` 为已关联的目录型号。
struct GearItemInfo: Decodable, Equatable {
    var id: Int64 = 0
    var category = ""
    var status = 0
    var unlisted = false
    var house = false
    var brandName = ""
    var modelName = ""
    var cueStyle: String?
    var roles: [String]?
    var product: GearProductInfo?
    var tipProduct: GearProductInfo?
    var chalkProduct: GearProductInfo?
    var createdAt = ""

    private enum CodingKeys: String, CodingKey {
        case id, category, status, unlisted, house, brandName, modelName
        case cueStyle, roles, product, tipProduct, chalkProduct, createdAt
    }

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decodeIfPresent(Int64.self, forKey: .id) ?? 0
        category = try c.decodeIfPresent(String.self, forKey: .category) ?? ""
        status = try c.decodeIfPresent(Int.self, forKey: .status) ?? 0
        unlisted = try c.decodeIfPresent(Bool.self, forKey: .unlisted) ?? false
        house = try c.decodeIfPresent(Bool.self, forKey: .house) ?? false
        brandName = try c.decodeIfPresent(String.self, forKey: .brandName) ?? ""
        modelName = try c.decodeIfPresent(String.self, forKey: .modelName) ?? ""
        cueStyle = try c.decodeIfPresent(String.self, forKey: .cueStyle)
        roles = try c.decodeIfPresent([String].self, forKey: .roles)
        product = try c.decodeIfPresent(GearProductInfo.self, forKey: .product)
        tipProduct = try c.decodeIfPresent(GearProductInfo.self, forKey: .tipProduct)
        chalkProduct = try c.decodeIfPresent(GearProductInfo.self, forKey: .chalkProduct)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

/// 一方选手的本场用杆。
struct GearMatchSide: Decodable, Equatable {
    var userId: Int64 = 0
    var recorded = false
    var source: String?
    var cue: GearItemInfo?
    var tipProduct: GearProductInfo?
    var chalkProduct: GearProductInfo?
    var breakCue: GearItemInfo?

    private enum CodingKeys: String, CodingKey {
        case userId, recorded, source, cue, tipProduct, chalkProduct, breakCue
    }

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        userId = try c.decodeIfPresent(Int64.self, forKey: .userId) ?? 0
        recorded = try c.decodeIfPresent(Bool.self, forKey: .recorded) ?? false
        source = try c.decodeIfPresent(String.self, forKey: .source)
        cue = try c.decodeIfPresent(GearItemInfo.self, forKey: .cue)
        tipProduct = try c.decodeIfPresent(GearProductInfo.self, forKey: .tipProduct)
        chalkProduct = try c.decodeIfPresent(GearProductInfo.self, forKey: .chalkProduct)
        breakCue = try c.decodeIfPresent(GearItemInfo.self, forKey: .breakCue)
    }
}

/// `GetGearMatchResp`：本场用杆视图，`editable` 与 `editDeadline` 决定能否修改。
struct GearMatchResponse: Decodable, Equatable {
    var success = false
    var message: String?
    var reason: String?
    var matchId: Int64 = 0
    var gameType: Int = 0
    var supportsBreak = false
    var me: GearMatchSide?
    var hasOpponent = false
    var opponent: GearMatchSide?
    var editable = false
    var editDeadline: String?

    private enum CodingKeys: String, CodingKey {
        case success, message, reason, matchId, gameType, supportsBreak
        case me, hasOpponent, opponent, editable, editDeadline
    }

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        success = try c.decodeIfPresent(Bool.self, forKey: .success) ?? false
        message = try c.decodeIfPresent(String.self, forKey: .message)
        reason = try c.decodeIfPresent(String.self, forKey: .reason)
        matchId = try c.decodeIfPresent(Int64.self, forKey: .matchId) ?? 0
        gameType = try c.decodeIfPresent(Int.self, forKey: .gameType) ?? 0
        supportsBreak = try c.decodeIfPresent(Bool.self, forKey: .supportsBreak) ?? false
        me = try c.decodeIfPresent(GearMatchSide.self, forKey: .me)
        hasOpponent = try c.decodeIfPresent(Bool.self, forKey: .hasOpponent) ?? false
        opponent = try c.decodeIfPresent(GearMatchSide.self, forKey: .opponent)
        editable = try c.decodeIfPresent(Bool.self, forKey: .editable) ?? false
        editDeadline = try c.decodeIfPresent(String.self, forKey: .editDeadline)
    }
}

struct GearDefaultInfo: Decodable, Equatable {
    var gameType = 0
    var slot = ""
    var itemId: Int64 = 0
}

/// `GetGearMineResp`：编辑器只需要装备列表。
struct GearMineResponse: Decodable, Equatable {
    var success = false
    var message: String?
    var maxItems = 0
    var items: [GearItemInfo] = []
    var defaults: [GearDefaultInfo] = []
}
