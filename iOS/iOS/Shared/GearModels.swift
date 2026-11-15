import Foundation

// 装备目录与用户装备的接口模型，字段与 `backend/chasing_points.api` 的 gear 模块一致。
// 标了 optional 的标量用可选类型；数组在服务端总会返回（可能为空），按非可选解码。

struct GearBrandInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let name: String
}

/// 目录型号。长度、重量、皮头直径为 0 或缺失都表示品牌未提供。
struct GearProductInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let category: String
    let brandId: Int64
    let brandName: String
    let name: String
    let house: Bool
    let cueStyle: String?
    let roles: [String]
    let lengthMm: Int?
    let weightG: Int?
    let tipDiameterMm: Double?
    let imagePath: String?
}

/// 用户装备。展示字段由服务端按查看者组装。
struct GearItemInfo: Decodable, Equatable, Identifiable {
    let id: Int64
    let category: String
    let status: Int
    let unlisted: Bool
    let house: Bool
    let brandName: String
    let modelName: String
    let cueStyle: String?
    let roles: [String]
    let product: GearProductInfo?
    let tipProduct: GearProductInfo?
    let chalkProduct: GearProductInfo?
    let createdAt: String
}

/// 主力设置：game_type 1=斯诺克 2=九球追分 3=中式八球 4=美式九球；slot 为 cue / break。
struct GearDefaultInfo: Decodable, Equatable {
    let gameType: Int
    let slot: String
    let itemId: Int64
}

struct GearBrandsResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let list: [GearBrandInfo]
}

struct GearProductsResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let total: Int64
    let list: [GearProductInfo]
}

struct GearMineResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let maxItems: Int
    var items: [GearItemInfo]
    var defaults: [GearDefaultInfo]
}

struct GearItemResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let item: GearItemInfo?
}

struct GearUserResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let reason: String?
    let items: [GearItemInfo]
    let defaults: [GearDefaultInfo]
}

/// 一方选手的本场用杆。
struct GearMatchSide: Decodable, Equatable {
    let userId: Int64
    let recorded: Bool
    let source: String?
    let cue: GearItemInfo?
    let tipProduct: GearProductInfo?
    let chalkProduct: GearProductInfo?
    let breakCue: GearItemInfo?
}

/// `GetGearMatchResp`：本场用杆视图，`editable` 与 `editDeadline` 决定能否修改。
struct GearMatchResponse: Decodable, Equatable {
    let success: Bool
    let message: String
    let reason: String?
    let matchId: Int64?
    let gameType: Int?
    let supportsBreak: Bool?
    let me: GearMatchSide?
    let hasOpponent: Bool?
    let opponent: GearMatchSide?
    let editable: Bool?
    let editDeadline: String?
}

/// 登记、删除、设主力共用的成功响应。
struct GearCommonResponse: Decodable, Equatable {
    let success: Bool
    let message: String
}

// MARK: - 请求体

/// 登记装备：productId > 0 选目录型号，否则登记未收录型号。
struct GearCreateRequest: Encodable, Equatable {
    let category: String
    var productId: Int64?
    var brandId: Int64?
    var brandName: String?
    var modelName: String?
    var cueStyle: String?
    var roles: [String]?
    var tipProductId: Int64?
    var chalkProductId: Int64?
}

/// 修改装备：status 为 nil 表示不改状态；updateTip / updateChalk 为 true 时才改对应子属性，product id 为 0 表示清空。
struct GearUpdateRequest: Encodable, Equatable {
    let itemId: Int64
    var status: Int?
    var updateTip: Bool?
    var tipProductId: Int64?
    var updateChalk: Bool?
    var chalkProductId: Int64?
}

struct GearDeleteRequest: Encodable, Equatable {
    let itemId: Int64
}

struct GearSetDefaultRequest: Encodable, Equatable {
    let gameType: Int
    let slot: String
    let itemId: Int64
}
