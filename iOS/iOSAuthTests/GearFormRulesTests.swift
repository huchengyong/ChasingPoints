import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/gear-form.test.mjs`。
@MainActor
final class GearFormRulesTests: XCTestCase {
    private func form(
        category: String = "cue",
        brandId: Int64 = 0,
        brandName: String = "品牌",
        modelName: String = "型号",
        cueStyle: String = "pool",
        roles: [String] = []
    ) -> GearUnlistedForm {
        GearUnlistedForm(
            category: category,
            brandId: brandId,
            brandName: brandName,
            modelName: modelName,
            cueStyle: cueStyle,
            roles: roles
        )
    }

    func testTypedBrandBuildsPayloadAndTrimsNames() {
        let result = GearFormRules.buildUnlistedPayload(form(
            brandName: "  自家小厂 ",
            modelName: " 手工杆 1 ",
            cueStyle: "snooker"
        ))
        XCTAssertEqual(result, .success(GearUnlistedPayload(
            category: "cue",
            modelName: "手工杆 1",
            brandId: nil,
            brandName: "自家小厂",
            cueStyle: "snooker"
        )))
    }

    func testCatalogBrandWinsOverTypedName() {
        let result = GearFormRules.buildUnlistedPayload(form(
            brandId: 12,
            brandName: "残留的自填内容",
            modelName: "Pro 1"
        ))
        guard case .success(let payload) = result else { return XCTFail("应通过校验") }
        XCTAssertEqual(payload.brandId, 12)
        XCTAssertNil(payload.brandName)
    }

    func testBreakJumpNeedsAtLeastOneRoleInStableOrder() {
        let none = GearFormRules.buildUnlistedPayload(form(
            category: "break_jump",
            brandId: 3,
            modelName: "X",
            roles: []
        ))
        XCTAssertEqual(none, .failure("请选择冲杆或跳杆"))

        let both = GearFormRules.buildUnlistedPayload(form(
            category: "break_jump",
            brandId: 3,
            modelName: "X",
            roles: ["jump", "break"]
        ))
        guard case .success(let payload) = both else { return XCTFail("应通过校验") }
        XCTAssertEqual(payload.roles, ["break", "jump"])
        XCTAssertNil(payload.cueStyle)
    }

    func testEveryRequiredFieldHasItsOwnMessage() {
        XCTAssertEqual(GearFormRules.buildUnlistedPayload(form(modelName: "   ")), .failure("请填写型号名称"))
        XCTAssertEqual(GearFormRules.buildUnlistedPayload(form(brandName: " ")), .failure("请选择品牌或填写品牌名称"))
        XCTAssertEqual(GearFormRules.buildUnlistedPayload(form(cueStyle: "")), .failure("请选择杆型"))
        XCTAssertEqual(GearFormRules.buildUnlistedPayload(form(cueStyle: "carom")), .failure("请选择杆型"))
        XCTAssertEqual(GearFormRules.buildUnlistedPayload(form(category: "tip")), .failure("装备类别无效"))
    }

    func testLengthLimitsCountUnicodeScalars() {
        let model = String(repeating: "型", count: GearFormRules.modelNameMax)
        XCTAssertNotNil(success(GearFormRules.buildUnlistedPayload(form(modelName: model))))
        XCTAssertEqual(
            GearFormRules.buildUnlistedPayload(form(modelName: model + "型")),
            .failure("型号名称不能超过 64 个字")
        )

        let brand = String(repeating: "牌", count: GearFormRules.brandNameMax)
        XCTAssertNotNil(success(GearFormRules.buildUnlistedPayload(form(brandName: brand))))
        XCTAssertEqual(
            GearFormRules.buildUnlistedPayload(form(brandName: brand + "牌")),
            .failure("品牌名称不能超过 32 个字")
        )

        // 一个 emoji 在 JS 里占两个 UTF-16 单元，但仍只算一个字。
        XCTAssertNotNil(success(GearFormRules.buildUnlistedPayload(
            form(modelName: String(repeating: "🎱", count: GearFormRules.modelNameMax))
        )))
    }

    func testToggleRoleKeepsBreakBeforeJump() {
        XCTAssertEqual(GearFormRules.toggleRole([], "jump"), ["jump"])
        XCTAssertEqual(GearFormRules.toggleRole(["jump"], "break"), ["break", "jump"])
        XCTAssertEqual(GearFormRules.toggleRole(["break", "jump"], "break"), ["jump"])
    }

    private func success(_ result: GearFormResult) -> GearUnlistedPayload? {
        guard case .success(let payload) = result else { return nil }
        return payload
    }
}
