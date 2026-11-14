import Foundation

/// 写操作结果：message 用于 toast；身份已变化时不提示也不回写。
struct GearWriteOutcome {
    let succeeded: Bool
    let message: String?

    static let stale = GearWriteOutcome(succeeded: false, message: nil)
    static func success(_ message: String? = nil) -> GearWriteOutcome { .init(succeeded: true, message: message) }
    static func failure(_ message: String) -> GearWriteOutcome { .init(succeeded: false, message: message) }
}

/// 装备接口门面，全部经由 `SessionStore` 的认证请求。
@MainActor
struct GearAPI {
    static let pageSize = 20

    let session: SessionStore

    func brands(category: GearCategory) async throws -> [GearBrandInfo] {
        let path = APIQuery.path("/api/gear/catalog/brands", ["category": category.rawValue])
        let response: GearBrandsResponse = try await session.authorizedRequest(path)
        return response.list
    }

    func products(
        category: GearCategory,
        brandId: Int64 = 0,
        keyword: String = "",
        house: Bool = false,
        page: Int = 1,
        pageSize: Int = GearAPI.pageSize
    ) async throws -> GearProductsResponse {
        // 空值不拼进查询串，避免后端把空字符串当成筛选条件。
        var query: [String: CustomStringConvertible?] = [
            "category": category.rawValue,
            "page": page,
            "page_size": pageSize,
        ]
        if brandId > 0 { query["brand_id"] = brandId }
        let word = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !word.isEmpty { query["keyword"] = word }
        if house { query["house"] = "true" }
        return try await session.authorizedRequest(APIQuery.path("/api/gear/catalog/products", query))
    }

    func mine() async throws -> GearMineResponse {
        try await session.authorizedRequest("/api/gear/mine")
    }

    func create(_ request: GearCreateRequest) async throws -> GearItemInfo? {
        let response: GearItemResponse = try await session.authorizedPost("/api/gear/item/create", body: request)
        return response.item
    }

    func update(_ request: GearUpdateRequest) async throws -> GearItemInfo? {
        let response: GearItemResponse = try await session.authorizedPost("/api/gear/item/update", body: request)
        return response.item
    }

    func delete(itemId: Int64) async throws {
        let _: GearCommonResponse = try await session.authorizedPost(
            "/api/gear/item/delete",
            body: GearDeleteRequest(itemId: itemId)
        )
    }

    func setDefault(gameType: Int, slot: GearSlot, itemId: Int64) async throws {
        let _: GearCommonResponse = try await session.authorizedPost(
            "/api/gear/default/set",
            body: GearSetDefaultRequest(gameType: gameType, slot: slot.rawValue, itemId: itemId)
        )
    }

    func userGear(userId: Int64) async throws -> GearUserResponse {
        try await session.authorizedRequest(APIQuery.path("/api/gear/user", ["user_id": userId]))
    }
}
