import XCTest
@testable import iOS

@MainActor
final class GearAddModelTests: XCTestCase {
    private func makeModel(
        _ server: MockHTTPServer,
        category: String? = "cue",
        isPick: Bool = false,
        pickToken: String? = nil
    ) async -> (GearAddModel, GearStore) {
        let session = await makeRestoredAccountSession(server)
        let store = GearStore(session: session)
        return (GearAddModel(
            session: session,
            store: store,
            requestedCategory: category,
            isPick: isPick,
            pickToken: pickToken
        ), store)
    }

    func testPickModeOnlyAllowsTipAndChalk() async {
        let server = MockHTTPServer()
        let (picked, _) = await makeModel(server, category: "cue", isPick: true, pickToken: "t1")
        XCTAssertEqual(picked.category, .tip, "非法类别取允许的第一个")
        XCTAssertTrue(picked.isPick)

        let (chalk, _) = await makeModel(server, category: "chalk", isPick: true, pickToken: "t1")
        XCTAssertEqual(chalk.category, .chalk)

        let (added, _) = await makeModel(server, category: "tip")
        XCTAssertEqual(added.category, .cue, "添加页只允许球杆与冲跳杆")
    }

    func testHouseHintFollowsCategory() async {
        let server = MockHTTPServer()
        let (cue, _) = await makeModel(server)
        XCTAssertEqual(cue.houseHint, "用球房的公杆打球也能记录")
        XCTAssertTrue(cue.worksWithHouse)

        let (chalk, _) = await makeModel(server, category: "chalk", isPick: true, pickToken: "t1")
        XCTAssertEqual(chalk.houseHint, "用球房的巧克也能记录")

        let (breakJump, _) = await makeModel(server, category: "break_jump")
        XCTAssertFalse(breakJump.worksWithHouse)
    }

    func testStartLoadsBrandsHouseProductAndFiltersDuplicateHouseEntries() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON([GearFixtures.brandJSON()])))
        server.stub("GET", "/api/gear/catalog/products") { request in
            if request.query["house"] == "true" {
                return .json(GearFixtures.productsJSON([
                    GearFixtures.productJSON(200, brandName: "", name: "球房公杆", house: true),
                ], total: 1))
            }
            return .json(GearFixtures.productsJSON([
                GearFixtures.productJSON(200, brandName: "", name: "球房公杆", house: true),
                GearFixtures.productJSON(101),
            ], total: 2))
        }
        let (model, _) = await makeModel(server)

        await model.start()

        XCTAssertEqual(model.brands, [GearFixtures.brand()])
        XCTAssertEqual(model.houseProduct?.id, 200)
        XCTAssertEqual(model.products.map(\.id), [101], "没有筛选时不重复展示球房公用条目")
        XCTAssertFalse(model.hasMore)
    }

    func testKeywordAndBrandKeepHouseEntries() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON()))
        server.stub("GET", "/api/gear/catalog/products") { request in
            if request.query["house"] == "true" { return .json(GearFixtures.productsJSON([], total: 0)) }
            return .json(GearFixtures.productsJSON([
                GearFixtures.productJSON(200, brandName: "", name: "球房公杆", house: true),
            ], total: 1))
        }
        let (model, _) = await makeModel(server)
        await model.start()

        model.updateKeyword("公杆")
        model.applyKeyword()
        await waitUntil { model.products.map(\.id) == [200] }

        model.selectBrand(3)
        await waitUntil { server.requests("/api/gear/catalog/products").last?.query["brand_id"] == "3" }
    }

    func testSwitchCategoryReloadsList() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON()))
        server.stub("GET", "/api/gear/catalog/products") { request in
            if request.query["house"] == "true" { return .json(GearFixtures.productsJSON([], total: 0)) }
            if request.query["category"] == "break_jump" {
                return .json(GearFixtures.productsJSON([GearFixtures.productJSON(301, category: "break_jump")], total: 1))
            }
            return .json(GearFixtures.productsJSON([GearFixtures.productJSON(101)], total: 1))
        }
        let (model, _) = await makeModel(server)
        await model.start()
        XCTAssertEqual(model.products.map(\.id), [101])

        model.switchCategory(.breakJump)
        await waitUntil { model.products.map(\.id) == [301] }
        XCTAssertEqual(model.form.category, "break_jump")
    }

    func testCreateAddsToStoreAndMarksJustAdded() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/gear/item/create", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(9))))
        let (model, store) = await makeModel(server)

        let outcome = await model.create(from: GearFixtures.product())

        XCTAssertEqual(outcome, .created(9))
        XCTAssertEqual(server.requests("/api/gear/item/create").first?.bodyJSON["product_id"] as? Int64, 101)
        XCTAssertTrue(store.consumeJustAdded(9))
    }

    func testCreateFailureReturnsMessage() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/gear/item/create", .json(GearFixtures.commonJSON(success: false, message: "装备数量已达上限")))
        let (model, store) = await makeModel(server)

        let outcome = await model.create(from: GearFixtures.product())

        XCTAssertEqual(outcome, .failed("装备数量已达上限"))
        XCTAssertNil(store.peek()?.items.first)
    }

    func testSubmitUnlistedValidatesTrimsAndSendsPayload() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/gear/item/create", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(11, brandName: "自家小厂", modelName: "手工杆 1"))))
        let (model, store) = await makeModel(server)

        model.form.brandName = "  自家小厂 "
        model.form.modelName = " 手工杆 1 "
        model.form.cueStyle = "snooker"
        let outcome = await model.submitUnlisted()

        XCTAssertEqual(outcome, .created(11))
        let body = server.requests("/api/gear/item/create").first?.bodyJSON
        XCTAssertEqual(body?["brand_name"] as? String, "自家小厂")
        XCTAssertEqual(body?["model_name"] as? String, "手工杆 1")
        XCTAssertEqual(body?["cue_style"] as? String, "snooker")
        XCTAssertNil(body?["product_id"])
        XCTAssertEqual(store.justAddedItemId, 11)
    }

    func testSubmitUnlistedStopsOnValidationFailure() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        model.form.modelName = "   "

        let outcome = await model.submitUnlisted()

        XCTAssertEqual(outcome, .failed("请填写型号名称"))
        XCTAssertEqual(server.requestCount("/api/gear/item/create"), 0)
    }

    func testToggleUnlistedShowsForm() async {
        let server = MockHTTPServer()
        let (model, _) = await makeModel(server)
        XCTAssertFalse(model.showUnlisted)
        model.toggleUnlisted()
        XCTAssertTrue(model.showUnlisted)
    }
}
