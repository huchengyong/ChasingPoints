import Foundation
import XCTest
@testable import iOS

@MainActor
final class VenueSubmitModelTests: XCTestCase {
    func testValidationReportsFirstMissingFieldInSourceOrder() {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let model = makeModel(session)

        XCTAssertEqual(model.validate(), "请输入球馆名称")
        XCTAssertTrue(model.showValidation)

        model.form.name = " 星轨台球 "
        XCTAssertEqual(model.validate(), "请选择地区")

        model.form.regionText = "广东省 深圳市 南山区"
        XCTAssertEqual(model.validate(), "请输入详细地址")

        model.form.address = "科苑路 8 号"
        XCTAssertNil(model.validate())
        XCTAssertEqual(model.barCopy.title, "确认常玩球馆后提交")
    }

    func testSubmitRequiresLoginWithoutSendingRequest() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let model = makeModel(session)
        fill(model)

        await model.submit()

        XCTAssertEqual(model.notice, "请先登录")
        XCTAssertFalse(model.didSubmit)
        XCTAssertEqual(server.requestCount("/api/venue/create"), 0)
    }

    func testSuccessfulSubmitSendsBasicFieldsAndLocation() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        server.stub("POST", "/api/venue/create", .json([
            "success": true,
            "venue_id": 42,
            "message": "提交成功，等待审核",
        ]))
        let model = makeModel(session)
        fill(model)

        await model.submit()

        XCTAssertTrue(model.didSubmit)
        XCTAssertFalse(model.isSubmitting)
        XCTAssertEqual(model.notice, "提交成功，等待审核")
        let request = server.requests("/api/venue/create").first
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(request?.bodyJSON.keys.sorted(), ["address", "city", "district", "latitude", "longitude", "name"])
        XCTAssertEqual(request?.bodyJSON["city"] as? String, "深圳市")
        XCTAssertEqual(request?.bodyJSON["latitude"] as? Double, 22.54)
    }

    func testFailedSubmitKeepsFormAndCanRetry() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        let attempts = LockedCounter()
        server.stub("POST", "/api/venue/create") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(["success": true, "venue_id": 42, "message": ""])
        }
        let model = makeModel(session)
        fill(model)

        await model.submit()

        XCTAssertFalse(model.didSubmit)
        XCTAssertFalse(model.isSubmitting)
        XCTAssertEqual(model.notice, "提交失败")
        XCTAssertEqual(model.form.name, "星轨台球")
        XCTAssertEqual(model.form.address, "科苑路 8 号")

        await model.submit()

        XCTAssertTrue(model.didSubmit)
        XCTAssertEqual(model.notice, "已提交，审核通过后会展示给附近球友")
        XCTAssertEqual(server.requestCount("/api/venue/create"), 2)
    }

    func testSubmittingBlocksDuplicateRequests() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/venue/create") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["success": true, "venue_id": 42, "message": "已提交"])
        }
        let model = makeModel(session)
        fill(model)

        let first = Task { await model.submit() }
        await waitUntil { server.requestCount("/api/venue/create") == 1 }

        await model.submit()
        XCTAssertEqual(server.requestCount("/api/venue/create"), 1)

        gate.signal()
        await first.value
        XCTAssertTrue(model.didSubmit)
    }

    func testAreasLoadAndConfirmSelection() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        stubAreas(server)
        let model = makeModel(session)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isAreaReady)
        XCTAssertEqual(model.columns.map(\.count), [1, 1, 1])
        let confirmed = await model.confirmAreaSelection()
        XCTAssertTrue(confirmed)
        XCTAssertEqual(model.form.regionText, "广东省 深圳市 南山区")
        XCTAssertEqual(model.form.city, "深圳市")
        XCTAssertEqual(model.form.district, "南山区")
        XCTAssertEqual(model.form.areaIds, [19, 321, 2723])
    }

    func testConfirmWithoutCityShowsIncompleteRegionCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/areas", .json(["success": true, "list": []]))
        let model = makeModel(session)

        await model.loadIfNeeded()
        let confirmed = await model.confirmAreaSelection()

        XCTAssertFalse(confirmed)
        XCTAssertFalse(model.isAreaReady)
        XCTAssertEqual(model.notice, "请选择完整地区")
        XCTAssertEqual(model.form.regionText, "")
    }

    func testAreaLoadFailureShowsCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        server.stub("GET", "/api/venue/areas", .raw("down", status: 503))
        let model = makeModel(session)

        await model.loadIfNeeded()

        XCTAssertFalse(model.isAreaReady)
        XCTAssertEqual(model.notice, "地区数据加载失败")
    }

    func testRewardStatusDrivesIntroCopy() async {
        let server = MockHTTPServer()
        let session = makeGuestVenueSession(server)
        let status = FavoriteVenueRewardStatus(
            success: true,
            enabled: true,
            popupEnabled: true,
            rewardDays: 30,
            newUserWindowDays: 7,
            status: "not_started",
            memberExpiresAt: nil,
            submittedVenueId: nil,
            submittedVenueName: nil,
            rejectReason: nil
        )
        let model = VenueSubmitModel(session: session, rewardStatusLoader: { status })
        stubAreas(server)

        await model.loadIfNeeded()

        XCTAssertEqual(model.intro.title, "免费领取 1 个月VIP")
        fill(model)
        XCTAssertEqual(model.barCopy.tip, "首次有效补充并审核通过，送 1 个月VIP。")
    }

    func testAccountChangeClearsDraft() async {
        let server = MockHTTPServer()
        let session = await makeSignedInVenueSession(server)
        let model = makeModel(session)
        fill(model)
        XCTAssertFalse(model.form.name.isEmpty)

        session.logout()
        model.syncIdentity()

        XCTAssertEqual(model.form, VenueSubmitForm())
        XCTAssertFalse(model.showValidation)
        XCTAssertFalse(model.didSubmit)
    }

    private func makeModel(_ session: SessionStore) -> VenueSubmitModel {
        VenueSubmitModel(session: session, rewardStatusLoader: { nil })
    }

    private func fill(_ model: VenueSubmitModel) {
        model.form.name = "星轨台球"
        model.form.regionText = "广东省 深圳市 南山区"
        model.form.areaIds = [19, 321, 2723]
        model.form.city = "深圳市"
        model.form.district = "南山区"
        model.form.address = "科苑路 8 号"
        model.form.location = PickedLocation(latitude: 22.54, longitude: 113.91, name: "", address: "")
    }

    private func stubAreas(_ server: MockHTTPServer) {
        server.stub("GET", "/api/venue/areas") { request in
            switch request.query["parent_id"] {
            case "0":
                return .json(VenueFixtures.areaJSON([(id: 19, parentId: 0, name: "广东省")]))
            case "19":
                return .json(VenueFixtures.areaJSON([(id: 321, parentId: 19, name: "深圳市")]))
            default:
                return .json(VenueFixtures.areaJSON([(id: 2723, parentId: 321, name: "南山区")]))
            }
        }
    }
}
