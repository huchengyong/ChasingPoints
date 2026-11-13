import Foundation
import XCTest
@testable import iOS

@MainActor
final class VenueSubmitRulesTests: XCTestCase {
    func testBuildVenueSubmitPayloadOnlyKeepsBasicVenueFields() throws {
        var form = VenueSubmitForm()
        form.name = "  星轨台球  "
        form.city = " 深圳 "
        form.district = " 南山区 "
        form.address = " 科苑路 8 号 "

        let payload = VenueSubmitRules.payload(form)

        XCTAssertEqual(payload, VenueSubmitPayload(
            name: "星轨台球",
            city: "深圳",
            district: "南山区",
            address: "科苑路 8 号",
            latitude: nil,
            longitude: nil
        ))

        let object = try XCTUnwrap(
            JSONSerialization.jsonObject(with: JSONEncoder().encode(payload)) as? [String: Any]
        )
        XCTAssertEqual(Set(object.keys), ["name", "city", "district", "address"])
    }

    func testBuildVenueSubmitPayloadAddsPickedCoordinatesWhenLocationIsValid() {
        var form = VenueSubmitForm()
        form.name = "星轨台球"
        form.city = "上海市"
        form.district = "浦东新区"
        form.address = "世纪大道 100 号"
        form.location = PickedLocation(
            latitude: 31.234583,
            longitude: 121.507572,
            name: "星轨台球",
            address: "上海市浦东新区世纪大道100号"
        )

        XCTAssertEqual(VenueSubmitRules.payload(form), VenueSubmitPayload(
            name: "星轨台球",
            city: "上海市",
            district: "浦东新区",
            address: "世纪大道 100 号",
            latitude: 31.234583,
            longitude: 121.507572
        ))
    }

    func testBuildVenueSubmitPayloadDropsInvalidPickedCoordinates() {
        let base = VenueSubmitPayload(
            name: "星轨台球",
            city: "上海市",
            district: "浦东新区",
            address: "世纪大道 100 号",
            latitude: nil,
            longitude: nil
        )

        var form = VenueSubmitForm()
        form.name = "星轨台球"
        form.city = "上海市"
        form.district = "浦东新区"
        form.address = "世纪大道 100 号"

        XCTAssertEqual(VenueSubmitRules.payload(form), base)

        form.location = PickedLocation(latitude: 0, longitude: 0, name: "", address: "")
        XCTAssertEqual(VenueSubmitRules.payload(form), base)

        form.location = PickedLocation(latitude: .nan, longitude: 121.5, name: "", address: "")
        XCTAssertEqual(VenueSubmitRules.payload(form), base)

        form.location = PickedLocation(latitude: 31.2, longitude: .infinity, name: "", address: "")
        XCTAssertEqual(VenueSubmitRules.payload(form), base)
    }

    func testNormalizePickedLocationOnlyKeepsFiniteNonZeroCoordinates() {
        XCTAssertEqual(
            VenueSubmitRules.normalizePickedLocation(
                latitude: 31.234583,
                longitude: 121.507572,
                name: " 星轨台球 ",
                address: " 世纪大道100号 "
            ),
            PickedLocation(latitude: 31.234583, longitude: 121.507572, name: "星轨台球", address: "世纪大道100号")
        )

        XCTAssertNil(VenueSubmitRules.normalizePickedLocation(latitude: nil, longitude: nil))
        XCTAssertNil(VenueSubmitRules.normalizePickedLocation(latitude: 0, longitude: 121.5))
        XCTAssertNil(VenueSubmitRules.normalizePickedLocation(latitude: 31.2, longitude: 0))
        XCTAssertNil(VenueSubmitRules.normalizePickedLocation(latitude: .nan, longitude: 121.5))
    }

    func testStripRegionPrefixRemovesLeadingRegionNamesOnceEach() {
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("上海市浦东新区世纪大道100号", regionNames: ["上海市", "上海市", "浦东新区"]),
            "世纪大道100号"
        )
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("广东省深圳市南山区科苑路8号", regionNames: ["广东省", "深圳市", "南山区"]),
            "科苑路8号"
        )
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("科苑路8号", regionNames: ["广东省", "深圳市", "南山区"]),
            "科苑路8号"
        )
    }

    func testStripRegionPrefixPrefersLongerRegionNamesForMunicipalities() {
        let shanghai = ["上海", "上海市", "浦东新区"]
        let beijing = ["北京", "北京市", "朝阳区"]

        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("上海市浦东新区世纪大道100号", regionNames: shanghai),
            "世纪大道100号"
        )
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("北京市朝阳区建国路88号", regionNames: beijing),
            "建国路88号"
        )
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("浦东新区世纪大道100号", regionNames: shanghai),
            "世纪大道100号"
        )
        XCTAssertEqual(
            VenueSubmitRules.stripRegionPrefix("上海市浦东新区世纪大道100号", regionNames: []),
            "上海市浦东新区世纪大道100号"
        )
    }

    func testResolveBarCopyOnlyPromisesVIPWhileRewardIsClaimable() {
        XCTAssertEqual(
            VenueSubmitRules.resolveBarCopy(
                missingRequiredLabels: [],
                rewardStatus: rewardStatus(enabled: true, rewardDays: 30, status: "not_started")
            ),
            VenueSubmitBarCopy(title: "确认常玩球馆后提交", tip: "首次有效补充并审核通过，送 1 个月VIP。")
        )

        XCTAssertEqual(
            VenueSubmitRules.resolveBarCopy(missingRequiredLabels: [], rewardStatus: nil),
            VenueSubmitBarCopy(title: "确认常玩球馆后提交", tip: "审核通过后会展示给附近球友。")
        )

        XCTAssertEqual(
            VenueSubmitRules.resolveBarCopy(
                missingRequiredLabels: [],
                rewardStatus: rewardStatus(enabled: true, rewardDays: 30, status: "reward_granted")
            ),
            VenueSubmitBarCopy(title: "确认常玩球馆后提交", tip: "审核通过后会展示给附近球友。")
        )

        XCTAssertEqual(
            VenueSubmitRules.resolveBarCopy(
                missingRequiredLabels: ["球馆名称", "详细地址"],
                rewardStatus: rewardStatus(enabled: true, rewardDays: 30)
            ),
            VenueSubmitBarCopy(title: "还有 2 项未填写", tip: "请先填写：球馆名称、详细地址")
        )
    }

    func testResolveIntroUsesConfiguredRewardDurationOnlyWhenClaimable() {
        XCTAssertEqual(
            VenueSubmitRules.resolveIntro(rewardStatus(enabled: true, rewardDays: 14, status: "not_started")).title,
            "免费领取 14 天VIP"
        )
        XCTAssertEqual(
            VenueSubmitRules.resolveIntro(rewardStatus(enabled: false, rewardDays: 30, status: "not_started")).title,
            "提交常去球馆"
        )
        XCTAssertFalse(
            VenueSubmitRules.resolveIntro(rewardStatus(enabled: true, rewardDays: 30, status: "reward_granted"))
                .sectionTip
                .contains("VIP")
        )
    }

    func testBuildVenueRegionSelectionMapsPathIntoDisplayAndPayloadFields() {
        let selection = VenueSubmitRules.regionSelection([
            VenueAreaOption(areaId: 19, parentId: 0, name: "广东省"),
            VenueAreaOption(areaId: 321, parentId: 19, name: "深圳市"),
            VenueAreaOption(areaId: 2723, parentId: 321, name: "南山区"),
        ])

        XCTAssertEqual(selection, VenueRegionSelection(
            areaIds: [19, 321, 2723],
            regionText: "广东省 深圳市 南山区",
            city: "深圳市",
            district: "南山区"
        ))
    }

    func testBuildVenueRegionSelectionKeepsCityWhenDistrictIsAbsent() {
        let selection = VenueSubmitRules.regionSelection([
            VenueAreaOption(areaId: 1, parentId: 0, name: "北京"),
            VenueAreaOption(areaId: 36, parentId: 1, name: "北京市"),
        ])

        XCTAssertEqual(selection, VenueRegionSelection(
            areaIds: [1, 36],
            regionText: "北京 北京市",
            city: "北京市",
            district: ""
        ))
    }

    private func rewardStatus(
        enabled: Bool,
        rewardDays: Int,
        status: String = "not_started"
    ) -> FavoriteVenueRewardStatus {
        FavoriteVenueRewardStatus(
            success: true,
            enabled: enabled,
            popupEnabled: true,
            rewardDays: rewardDays,
            newUserWindowDays: 7,
            status: status,
            memberExpiresAt: nil,
            submittedVenueId: nil,
            submittedVenueName: nil,
            rejectReason: nil
        )
    }
}
