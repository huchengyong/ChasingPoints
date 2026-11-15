import Foundation
@testable import iOS

/// 装备模块的虚构夹具，字段与 `backend/chasing_points.api` 的 gear 模块一致；不含任何真实凭据。
enum GearFixtures {
    static let cdn = AppConfiguration.cdnBaseURL

    // MARK: - 结构体（规则与模型测试）

    static func brand(_ id: Int64 = 1, name: String = "Alpha") -> GearBrandInfo {
        GearBrandInfo(id: id, name: name)
    }

    static func product(
        _ id: Int64 = 101,
        category: String = "cue",
        brandId: Int64 = 1,
        brandName: String = "Alpha",
        name: String = "A-100",
        house: Bool = false,
        cueStyle: String? = "pool",
        roles: [String] = [],
        lengthMm: Int? = nil,
        weightG: Int? = nil,
        tipDiameterMm: Double? = nil,
        imagePath: String? = nil
    ) -> GearProductInfo {
        GearProductInfo(
            id: id,
            category: category,
            brandId: brandId,
            brandName: brandName,
            name: name,
            house: house,
            cueStyle: cueStyle,
            roles: roles,
            lengthMm: lengthMm,
            weightG: weightG,
            tipDiameterMm: tipDiameterMm,
            imagePath: imagePath
        )
    }

    static func item(
        _ id: Int64 = 1,
        category: String = "cue",
        status: GearItemStatus = .active,
        unlisted: Bool = false,
        house: Bool = false,
        brandName: String = "Alpha",
        modelName: String = "A-100",
        cueStyle: String? = "snooker",
        roles: [String] = [],
        product: GearProductInfo? = nil,
        tip: GearProductInfo? = nil,
        chalk: GearProductInfo? = nil,
        createdAt: String = "2026-01-01 00:00:00"
    ) -> GearItemInfo {
        GearItemInfo(
            id: id,
            category: category,
            status: status.rawValue,
            unlisted: unlisted,
            house: house,
            brandName: brandName,
            modelName: modelName,
            cueStyle: cueStyle,
            roles: roles,
            product: product,
            tipProduct: tip,
            chalkProduct: chalk,
            createdAt: createdAt
        )
    }

    static func mainDefault(_ gameType: Int, slot: GearSlot, itemId: Int64) -> GearDefaultInfo {
        GearDefaultInfo(gameType: gameType, slot: slot.rawValue, itemId: itemId)
    }

    static func mine(
        items: [GearItemInfo] = [],
        defaults: [GearDefaultInfo] = [],
        maxItems: Int = 30
    ) -> GearMineResponse {
        GearMineResponse(success: true, message: "", maxItems: maxItems, items: items, defaults: defaults)
    }

    // MARK: - JSON（接口与页面模型测试）

    static func brandJSON(_ id: Int64 = 1, name: String = "Alpha") -> [String: Any] {
        ["id": id, "name": name]
    }

    static func productJSON(
        _ id: Int64 = 101,
        category: String = "cue",
        brandId: Int64 = 1,
        brandName: String = "Alpha",
        name: String = "A-100",
        house: Bool = false,
        cueStyle: String? = "pool",
        roles: [String] = [],
        lengthMm: Int? = nil,
        weightG: Int? = nil,
        tipDiameterMm: Double? = nil,
        imagePath: String = ""
    ) -> [String: Any] {
        var json: [String: Any] = [
            "id": id,
            "category": category,
            "brand_id": brandId,
            "brand_name": brandName,
            "name": name,
            "house": house,
            "roles": roles,
            "image_path": imagePath,
        ]
        if let cueStyle { json["cue_style"] = cueStyle }
        if let lengthMm { json["length_mm"] = lengthMm }
        if let weightG { json["weight_g"] = weightG }
        if let tipDiameterMm { json["tip_diameter_mm"] = tipDiameterMm }
        return json
    }

    static func itemJSON(
        _ id: Int64 = 1,
        category: String = "cue",
        status: Int = 1,
        unlisted: Bool = false,
        house: Bool = false,
        brandName: String = "Alpha",
        modelName: String = "A-100",
        cueStyle: String? = "snooker",
        roles: [String] = [],
        product: [String: Any]? = nil,
        tip: [String: Any]? = nil,
        chalk: [String: Any]? = nil
    ) -> [String: Any] {
        [
            "id": id,
            "category": category,
            "status": status,
            "unlisted": unlisted,
            "house": house,
            "brand_name": brandName,
            "model_name": modelName,
            "cue_style": cueStyle ?? "",
            "roles": roles,
            "product": product ?? NSNull(),
            "tip_product": tip ?? NSNull(),
            "chalk_product": chalk ?? NSNull(),
            "created_at": "2026-01-01 00:00:00",
        ]
    }

    static func mineJSON(
        items: [[String: Any]] = [],
        defaults: [[String: Any]] = [],
        maxItems: Int = 30,
        success: Bool = true,
        message: String = ""
    ) -> [String: Any] {
        [
            "success": success,
            "message": message,
            "max_items": maxItems,
            "items": items,
            "defaults": defaults,
        ]
    }

    static func defaultJSON(gameType: Int, slot: String = "cue", itemId: Int64) -> [String: Any] {
        ["game_type": gameType, "slot": slot, "item_id": itemId]
    }

    static func brandsJSON(_ brands: [[String: Any]] = []) -> [String: Any] {
        ["success": true, "message": "", "list": brands]
    }

    static func productsJSON(_ list: [[String: Any]] = [], total: Int64 = 0) -> [String: Any] {
        ["success": true, "message": "", "total": total, "list": list]
    }

    static func itemResponseJSON(_ item: [String: Any]?) -> [String: Any] {
        ["success": true, "message": "", "item": item ?? NSNull()]
    }

    static func userGearJSON(
        items: [[String: Any]] = [],
        defaults: [[String: Any]] = [],
        success: Bool = true,
        message: String = "",
        reason: String? = nil
    ) -> [String: Any] {
        var json: [String: Any] = [
            "success": success,
            "message": message,
            "items": items,
            "defaults": defaults,
        ]
        if let reason { json["reason"] = reason }
        return json
    }

    static func commonJSON(success: Bool = true, message: String = "") -> [String: Any] {
        ["success": success, "message": message]
    }
}
