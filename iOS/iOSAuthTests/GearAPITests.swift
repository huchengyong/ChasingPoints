import XCTest
@testable import iOS

/// 装备接口门面的路径、查询串与请求体；全部经由 MockHTTPServer，不访问真实后端。
@MainActor
final class GearAPITests: XCTestCase {
    private func makeAPI() async -> (GearAPI, MockHTTPServer) {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        return (GearAPI(session: session), server)
    }

    func testBrandsUsesCategoryQuery() async throws {
        let (api, server) = await makeAPI()
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON([GearFixtures.brandJSON(7, name: "其他")])))

        let brands = try await api.brands(category: .breakJump)

        XCTAssertEqual(brands, [GearFixtures.brand(7, name: "其他")])
        XCTAssertEqual(server.requests("/api/gear/catalog/brands").first?.query["category"], "break_jump")
    }

    func testProductsOmitsEmptyFiltersAndTrimsKeyword() async throws {
        let (api, server) = await makeAPI()
        server.stub("GET", "/api/gear/catalog/products", .json(GearFixtures.productsJSON([GearFixtures.productJSON()], total: 1)))

        _ = try await api.products(category: .cue)
        var query = try XCTUnwrap(server.requests("/api/gear/catalog/products").first?.query)
        XCTAssertEqual(query["category"], "cue")
        XCTAssertEqual(query["page"], "1")
        XCTAssertEqual(query["page_size"], "20")
        XCTAssertNil(query["brand_id"])
        XCTAssertNil(query["keyword"])
        XCTAssertNil(query["house"])

        _ = try await api.products(category: .cue, brandId: 3, keyword: "  A-1  ", house: true, page: 2)
        query = try XCTUnwrap(server.requests("/api/gear/catalog/products").last?.query)
        XCTAssertEqual(query["brand_id"], "3")
        XCTAssertEqual(query["keyword"], "A-1")
        XCTAssertEqual(query["house"], "true")
        XCTAssertEqual(query["page"], "2")
    }

    func testMineDecodesItemsAndDefaults() async throws {
        let (api, server) = await makeAPI()
        server.stub("GET", "/api/gear/mine", .json(GearFixtures.mineJSON(
            items: [GearFixtures.itemJSON(5)],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 5)]
        )))

        let mine = try await api.mine()

        XCTAssertEqual(mine.maxItems, 30)
        XCTAssertEqual(mine.items.map(\.id), [5])
        XCTAssertEqual(mine.defaults, [GearFixtures.mainDefault(3, slot: .cue, itemId: 5)])
    }

    func testCreateSendsSnakeCaseBodyAndReturnsItem() async throws {
        let (api, server) = await makeAPI()
        server.stub("POST", "/api/gear/item/create", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(9))))

        let item = try await api.create(GearCreateRequest(category: "cue", productId: 101))

        XCTAssertEqual(item?.id, 9)
        let request = try XCTUnwrap(server.requests("/api/gear/item/create").first)
        XCTAssertEqual(request.bodyJSON["category"] as? String, "cue")
        XCTAssertEqual(request.bodyJSON["product_id"] as? Int64, 101)
        XCTAssertNil(request.bodyJSON["brand_name"])
    }

    func testUpdateAndSetDefaultSendOnlyRequestedFields() async throws {
        let (api, server) = await makeAPI()
        server.stub("POST", "/api/gear/item/update", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(9, status: 2))))
        server.stub("POST", "/api/gear/default/set", .json(GearFixtures.commonJSON()))

        let updated = try await api.update(GearUpdateRequest(itemId: 9, status: 2))
        try await api.setDefault(gameType: 3, slot: .breakSlot, itemId: 9)

        XCTAssertEqual(updated?.status, 2)
        let update = try XCTUnwrap(server.requests("/api/gear/item/update").first?.bodyJSON)
        XCTAssertEqual(update["item_id"] as? Int64, 9)
        XCTAssertEqual(update["status"] as? Int, 2)
        XCTAssertNil(update["update_tip"])

        let setDefault = try XCTUnwrap(server.requests("/api/gear/default/set").first?.bodyJSON)
        XCTAssertEqual(setDefault["game_type"] as? Int, 3)
        XCTAssertEqual(setDefault["slot"] as? String, "break")
        XCTAssertEqual(setDefault["item_id"] as? Int64, 9)
    }

    func testDeleteAndUserGear() async throws {
        let (api, server) = await makeAPI()
        server.stub("POST", "/api/gear/item/delete", .json(GearFixtures.commonJSON()))
        server.stub("GET", "/api/gear/user", .json(GearFixtures.userGearJSON(items: [GearFixtures.itemJSON(2, category: "tip")])))

        try await api.delete(itemId: 9)
        let user = try await api.userGear(userId: 42)

        XCTAssertEqual(server.requests("/api/gear/item/delete").first?.bodyJSON["item_id"] as? Int64, 9)
        XCTAssertEqual(user.items.map(\.id), [2])
        XCTAssertEqual(server.requests("/api/gear/user").first?.query["user_id"], "42")
    }

    func testNotFriendSurfacesReasonForReadOnlyPage() async {
        let (api, server) = await makeAPI()
        server.stub("GET", "/api/gear/user", .json(GearFixtures.userGearJSON(
            success: false,
            message: "仅好友可以查看",
            reason: "not_friend"
        )))

        do {
            _ = try await api.userGear(userId: 42)
            XCTFail("应抛出业务失败")
        } catch {
            XCTAssertEqual((error as? APIError)?.reason, "not_friend")
        }
    }
}
