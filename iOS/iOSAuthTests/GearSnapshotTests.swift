import SwiftUI
import XCTest
@testable import iOS

/// 装备四个页面的渲染快照：有数据、空态与失败各一份，亮暗各一张。
/// 页面模型在渲染前预读，不依赖页面的 `.task` 完成请求。
@MainActor
final class GearSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)
    private let minePath = "/api/gear/mine"
    private let userPath = "/api/gear/user"

    private func makeContext(_ server: MockHTTPServer) async -> (SessionStore, GearStore) {
        let session = await makeRestoredAccountSession(server)
        return (session, GearStore(session: session))
    }

    private func snapshot(
        _ name: String,
        session: SessionStore,
        store: GearStore,
        @ViewBuilder content: () -> some View
    ) throws {
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: pageSize) {
                page(session: session, store: store, content: content)
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    private func page<Content: View>(
        session: SessionStore,
        store: GearStore,
        @ViewBuilder content: () -> Content
    ) -> some View {
        NavigationStack {
            content()
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(store)
        .environmentObject(AppNavigator())
        .environmentObject(ToastCenter())
        .environmentObject(ActivityStore(session: session))
    }

    func testGearIndexSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [
                GearFixtures.itemJSON(1, product: GearFixtures.productJSON(lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9)),
                GearFixtures.itemJSON(3, category: "break_jump", roles: ["break", "jump"]),
                GearFixtures.itemJSON(5, category: "cue", status: 2, unlisted: true, brandName: "其他品牌", modelName: "未收录型号"),
            ],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let model = GearIndexModel(session: session, store: store)
        await model.onAppear()
        XCTAssertEqual(model.items.count, 3)

        try snapshot("gear-index", session: session, store: store) {
            GearIndexView(session: session, store: store, model: model)
        }
    }

    func testGearIndexEmptyAndErrorSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", minePath, .json(GearFixtures.mineJSON()))
        let empty = GearIndexModel(session: session, store: store)
        await empty.onAppear()
        XCTAssertTrue(empty.isEmpty)

        try snapshot("gear-index-empty", session: session, store: store) {
            GearIndexView(session: session, store: store, model: empty)
        }

        server.stub("GET", minePath, .raw("service down", status: 503))
        let failedSession = await makeRestoredAccountSession(server)
        let failedStore = GearStore(session: failedSession)
        let failed = GearIndexModel(session: failedSession, store: failedStore)
        await failed.onAppear()
        XCTAssertNotNil(failed.failure)

        try snapshot("gear-index-error", session: failedSession, store: failedStore) {
            GearIndexView(session: failedSession, store: failedStore, model: failed)
        }
    }

    func testGearAddSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON([GearFixtures.brandJSON(1), GearFixtures.brandJSON(2, name: "其他")])))
        server.stub("GET", "/api/gear/catalog/products") { request in
            if request.query["house"] == "true" {
                return .json(GearFixtures.productsJSON([GearFixtures.productJSON(200, brandName: "", name: "球房公杆", house: true)], total: 1))
            }
            return .json(GearFixtures.productsJSON([
                GearFixtures.productJSON(200, brandName: "", name: "球房公杆", house: true),
                GearFixtures.productJSON(101, lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9),
            ], total: 2))
        }
        let model = GearAddModel(session: session, store: store, requestedCategory: "cue", isPick: false, pickToken: nil)
        await model.start()
        XCTAssertEqual(model.products.map(\.id), [101])

        try snapshot("gear-add", session: session, store: store) {
            GearAddView(session: session, store: store, category: "cue", pickToken: nil, model: model)
        }

        model.toggleUnlisted()
        model.form.brandName = "自家小厂"
        model.form.modelName = "手工杆 1"
        try snapshot("gear-add-unlisted", session: session, store: store) {
            GearAddView(session: session, store: store, category: "cue", pickToken: nil, model: model)
        }
    }

    func testGearAddEmptyAndErrorSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", "/api/gear/catalog/brands", .json(GearFixtures.brandsJSON()))
        server.stub("GET", "/api/gear/catalog/products", .json(GearFixtures.productsJSON()))
        let empty = GearAddModel(session: session, store: store, requestedCategory: "cue", isPick: false, pickToken: nil)
        await empty.start()

        try snapshot("gear-add-empty", session: session, store: store) {
            GearAddView(session: session, store: store, category: "cue", pickToken: nil, model: empty)
        }

        let failedServer = MockHTTPServer()
        let (failedSession, failedStore) = await makeContext(failedServer)
        failedServer.stub("GET", "/api/gear/catalog/brands", .raw("service down", status: 503))
        failedServer.stub("GET", "/api/gear/catalog/products", .raw("service down", status: 503))
        let failed = GearAddModel(session: failedSession, store: failedStore, requestedCategory: "cue", isPick: false, pickToken: nil)
        await failed.start()
        XCTAssertNotNil(failed.listError)

        try snapshot("gear-add-error", session: failedSession, store: failedStore) {
            GearAddView(session: failedSession, store: failedStore, category: "cue", pickToken: nil, model: failed)
        }
    }

    func testGearItemSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", minePath, .json(GearFixtures.mineJSON(
            items: [GearFixtures.itemJSON(
                1,
                product: GearFixtures.productJSON(lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9),
                tip: GearFixtures.productJSON(201, category: "tip", brandName: "KAMUI", name: "M"),
                chalk: GearFixtures.productJSON(301, category: "chalk", brandName: "", name: "球房巧克")
            )],
            defaults: [GearFixtures.defaultJSON(gameType: 3, itemId: 1)]
        )))
        let model = GearItemModel(session: session, store: store, itemId: 1)
        await model.onAppear()
        XCTAssertNotNil(model.item)

        try snapshot("gear-item", session: session, store: store) {
            GearItemView(session: session, store: store, itemId: 1, model: model)
        }
    }

    func testGearItemErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", minePath, .raw("service down", status: 503))
        let model = GearItemModel(session: session, store: store, itemId: 1)
        await model.onAppear()
        XCTAssertNotNil(model.failure)

        try snapshot("gear-item-error", session: session, store: store) {
            GearItemView(session: session, store: store, itemId: 1, model: model)
        }
    }

    func testGearFriendSnapshots() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON(
            items: [
                GearFixtures.itemJSON(1, product: GearFixtures.productJSON()),
                GearFixtures.itemJSON(3, category: "break_jump", roles: ["break"]),
            ],
            defaults: [
                GearFixtures.defaultJSON(gameType: 3, itemId: 1),
                GearFixtures.defaultJSON(gameType: 3, slot: "break", itemId: 3),
            ]
        )))
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")
        await model.load()
        XCTAssertEqual(model.items.count, 2)

        try snapshot("gear-friend", session: session, store: store) {
            GearFriendView(session: session, userId: 42, name: "", model: model)
        }

        server.stub("GET", userPath, .json(GearFixtures.userGearJSON()))
        let empty = GearFriendModel(session: session, userId: 42, ownerName: "TA")
        await empty.load()
        try snapshot("gear-friend-empty", session: session, store: store) {
            GearFriendView(session: session, userId: 42, name: "", model: empty)
        }
    }

    func testGearFriendNotFriendSnapshot() async throws {
        let server = MockHTTPServer()
        let (session, store) = await makeContext(server)
        server.stub("GET", userPath, .json(GearFixtures.userGearJSON(
            success: false,
            message: "仅好友可以查看",
            reason: "not_friend"
        )))
        let model = GearFriendModel(session: session, userId: 42, ownerName: "TA")
        await model.load()
        XCTAssertEqual(model.failure?.retryable, false)

        try snapshot("gear-friend-not-friend", session: session, store: store) {
            GearFriendView(session: session, userId: 42, name: "", model: model)
        }
    }
}
