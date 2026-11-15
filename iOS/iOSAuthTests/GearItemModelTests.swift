import XCTest
@testable import iOS

@MainActor
final class GearItemModelTests: XCTestCase {
    private let minePath = "/api/gear/mine"

    private func loadModel(
        _ server: MockHTTPServer,
        itemId: Int64 = 1
    ) async -> (GearItemModel, GearStore, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        let store = GearStore(session: session)
        let model = GearItemModel(session: session, store: store, itemId: itemId)
        await model.onAppear()
        return (model, store, session)
    }

    private func cueJSON() -> [String: Any] {
        GearFixtures.itemJSON(
            1,
            product: GearFixtures.productJSON(lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9),
            tip: GearFixtures.productJSON(201, category: "tip", brandName: "KAMUI", name: "M", cueStyle: nil),
            chalk: nil
        )
    }

    func testCueShowsSpecsPartsAndMainRows() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [cueJSON(), GearFixtures.itemJSON(3, category: "break_jump", roles: ["break"])],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let (model, _, _) = await loadModel(server)

        XCTAssertEqual(model.title, "Alpha A-100")
        XCTAssertEqual(model.subtitle, "斯诺克 / 中式杆")
        XCTAssertEqual(model.specs.map(\.key), ["length", "weight", "tipDiameter"])
        XCTAssertEqual(model.parts.map(\.text), ["KAMUI M", "未设置"])
        XCTAssertEqual(model.mainRows.map(\.gameType), [3, 2, 1, 4])
        XCTAssertEqual(model.mainRows.first { $0.gameType == 3 }?.active, true)
        XCTAssertFalse(model.retired)
    }

    func testBreakCueMainRowsSkipSnookerAndRequireBreakRole() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(3, category: "break_jump", roles: ["break"])])))
        let (model, _, _) = await loadModel(server, itemId: 3)

        XCTAssertEqual(model.mainRows.map(\.gameType), [3, 2, 4])

        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [GearFixtures.itemJSON(4, category: "break_jump", roles: ["jump"])])))
        let (jump, _, _) = await loadModel(server, itemId: 4)
        XCTAssertTrue(jump.mainRows.isEmpty, "只有带冲杆身份的冲跳装备能设为主力冲杆")
    }

    func testSetPartUpdatesItemAndStore() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [cueJSON()])))
        let (model, store, _) = await loadModel(server)

        let updated = GearFixtures.itemJSON(1, chalk: GearFixtures.productJSON(301, category: "chalk", brandName: "", name: "球房巧克", cueStyle: nil))
        server.stub("POST", "/api/gear/item/update", .json(GearFixtures.itemResponseJSON(updated)))
        let outcome = await model.setPart(.chalk, product: GearFixtures.product(301, category: "chalk"))

        XCTAssertTrue(outcome.succeeded)
        XCTAssertEqual(outcome.message, "已保存")
        XCTAssertEqual(model.parts.last?.text, "球房巧克")
        XCTAssertEqual(store.peek()?.items.first?.chalkProduct?.id, 301)
        XCTAssertEqual(server.requests("/api/gear/item/update").first?.bodyJSON["chalk_product_id"] as? Int64, 301)
        XCTAssertEqual(server.requests("/api/gear/item/update").first?.bodyJSON["update_chalk"] as? Bool, true)

        server.stub("POST", "/api/gear/item/update", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(1))))
        let cleared = await model.setPart(.chalk, product: nil)
        XCTAssertTrue(cleared.succeeded)
        XCTAssertEqual(server.requests("/api/gear/item/update").last?.bodyJSON["chalk_product_id"] as? Int64, 0)
    }

    func testToggleMainSetsAndClearsDefault() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [cueJSON()])))
        server.stub("POST", "/api/gear/default/set", .json(GearFixtures.commonJSON()))
        let (model, store, _) = await loadModel(server)

        guard let row = model.mainRows.first(where: { $0.gameType == 3 }) else { return XCTFail("缺少中式八球槽位") }
        let set = await model.toggleMain(row)

        XCTAssertEqual(set.message, "已设为中式八球主力")
        XCTAssertEqual(store.peek()?.defaults, [GearFixtures.mainDefault(3, slot: .cue, itemId: 1)])
        XCTAssertEqual(model.mainRows.first { $0.gameType == 3 }?.active, true)

        guard let activeRow = model.mainRows.first(where: { $0.gameType == 3 }) else { return XCTFail("缺少中式八球槽位") }
        let cleared = await model.toggleMain(activeRow)

        XCTAssertEqual(cleared.message, "已取消主力")
        XCTAssertTrue(store.peek()?.defaults.isEmpty == true)
        XCTAssertEqual(server.requests("/api/gear/default/set").last?.bodyJSON["item_id"] as? Int64, 0)
    }

    func testRetireAndRestoreUpdateLocalDefaults() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [cueJSON()],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        server.stub("POST", "/api/gear/item/update", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(1, status: 2))))
        let (model, store, _) = await loadModel(server)

        let retired = await model.toggleRetire()

        XCTAssertEqual(retired.message, "已退役")
        XCTAssertTrue(model.retired)
        XCTAssertTrue(store.peek()?.defaults.isEmpty == true, "退役后取消主力")
        XCTAssertTrue(model.mainRows.isEmpty)

        server.stub("POST", "/api/gear/item/update", .json(GearFixtures.itemResponseJSON(GearFixtures.itemJSON(1, status: 1))))
        let restored = await model.toggleRetire()
        XCTAssertEqual(restored.message, "已恢复使用")
        XCTAssertFalse(model.retired)
    }

    func testDeleteRemovesItemFromStore() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [cueJSON()],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        server.stub("POST", "/api/gear/item/delete", .json(GearFixtures.commonJSON()))
        let (model, store, _) = await loadModel(server)

        let outcome = await model.delete()

        XCTAssertEqual(outcome.message, "已删除")
        XCTAssertNil(model.item)
        XCTAssertTrue(store.peek()?.items.isEmpty == true)
        XCTAssertTrue(store.peek()?.defaults.isEmpty == true)
    }

    func testJustAddedHintIsConsumedOnce() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(items: [cueJSON()])))
        let session = await makeRestoredAccountSession(server)
        let store = GearStore(session: session)
        store.markJustAdded(1)

        let model = GearItemModel(session: session, store: store, itemId: 1)
        await model.onAppear()
        XCTAssertTrue(model.justAdded)

        let again = GearItemModel(session: session, store: store, itemId: 1)
        await again.onAppear()
        XCTAssertFalse(again.justAdded)
    }

    func testFailureWithoutCacheShowsFeedback() async {
        let server = MockHTTPServer()
        server.stub("GET", minePath, .raw("service down", status: 503))
        let (model, _, _) = await loadModel(server)

        XCTAssertNil(model.item)
        XCTAssertNotNil(model.failure)
        XCTAssertFalse(model.isInitialLoading)
    }
}
