import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/gear-display.test.mjs`；默认图文件是否存在由小程序侧保证，
/// 原生端只能校验路径形态与 CDN 转换。
@MainActor
final class GearDisplayRulesTests: XCTestCase {
    func testDefaultImageFollowsCategory() {
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "cue", house: true), GearDisplayRules.houseCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "cue", cueStyle: "pool"), GearDisplayRules.poolCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "cue", cueStyle: "snooker"), GearDisplayRules.snookerCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "cue"), GearDisplayRules.snookerCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "break_jump", roles: ["break"]), GearDisplayRules.breakCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "break_jump", roles: ["break", "jump"]), GearDisplayRules.breakCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "break_jump", roles: ["jump"]), GearDisplayRules.jumpCueImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "tip"), GearDisplayRules.tipImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "chalk", house: true), GearDisplayRules.chalkImage)
        XCTAssertEqual(GearDisplayRules.defaultImagePath(category: "unknown"), GearDisplayRules.snookerCueImage)
    }

    func testCatalogImagePathsOnlyAcceptGearDirectory() {
        XCTAssertTrue(GearDisplayRules.isGearImagePath("/static/images/gear/a-100.png"))
        XCTAssertTrue(GearDisplayRules.isGearImagePath("/static/images/gear/A_1.v2.webp"))
        let rejected: [String?] = [
            nil,
            "",
            "https://evil.example/a.png",
            "/static/images/gear/../ranks/a.png",
            "/static/images/gear/sub/a.png",
            "/static/images/ranks/a.png",
            "/static/images/gear/a.gif",
            "/static/images/gear/.png",
            "/static/images/gear/a.png?x=1",
        ]
        for path in rejected {
            XCTAssertFalse(GearDisplayRules.isGearImagePath(path), "\(path ?? "nil") 不应被接受")
        }
    }

    func testResolveImageFallsBackToCategoryDefault() {
        let pool = GearDisplayRules.image(for: GearImageSource(
            category: "cue",
            cueStyle: "pool",
            imagePath: "/static/images/gear/a-100.png"
        ))
        XCTAssertEqual(pool.src?.absoluteString, "\(GearFixtures.cdn)/static/images/gear/a-100.png")
        XCTAssertEqual(pool.fallback?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.poolCueImage)")

        let noImage = GearDisplayRules.image(for: GearImageSource(category: "cue", cueStyle: "pool", imagePath: ""))
        XCTAssertEqual(noImage.src?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.poolCueImage)")

        let hostile = GearDisplayRules.image(for: GearImageSource(
            category: "tip",
            imagePath: "https://evil.example/a.png"
        ))
        XCTAssertEqual(hostile.src?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.tipImage)")

        let product = GearDisplayRules.image(for: GearImageSource(
            product: GearFixtures.product(category: "break_jump", roles: ["jump"], imagePath: "/static/images/gear/jump-1.webp")
        ))
        XCTAssertEqual(product.src?.absoluteString, "\(GearFixtures.cdn)/static/images/gear/jump-1.webp")
        XCTAssertEqual(product.fallback?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.jumpCueImage)")

        let unlisted = GearDisplayRules.image(for: GearImageSource(category: "cue", cueStyle: "snooker"))
        XCTAssertEqual(unlisted.src?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.snookerCueImage)")
        let empty = GearDisplayRules.image(for: GearImageSource(category: "cue"))
        XCTAssertEqual(empty.src?.absoluteString, "\(GearFixtures.cdn)\(GearDisplayRules.snookerCueImage)")
    }

    func testSpecsHideMissingValuesAndShowBothUnits() {
        let full = GearFixtures.product(lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9)
        XCTAssertEqual(GearDisplayRules.specs(full), [
            GearSpec(key: "length", label: "长度", value: "1470 mm"),
            GearSpec(key: "weight", label: "重量", value: "540 g（19.0 oz）"),
            GearSpec(key: "tipDiameter", label: "皮头直径", value: "12.9 mm"),
        ])
        XCTAssertEqual(GearDisplayRules.specs(GearFixtures.product(lengthMm: 0, weightG: 0, tipDiameterMm: 0)), [])
        XCTAssertEqual(GearDisplayRules.specs(GearFixtures.product(weightG: 510)).map(\.key), ["weight"])
        XCTAssertEqual(GearDisplayRules.specs(GearFixtures.product(tipDiameterMm: 13)).first?.value, "13 mm")
        XCTAssertEqual(GearDisplayRules.specs(GearFixtures.product(tipDiameterMm: 11.75)).first?.value, "11.75 mm")
        XCTAssertEqual(GearDisplayRules.specs(nil), [])
        XCTAssertEqual(GearDisplayRules.specs(GearFixtures.product(weightG: -3)), [])
    }

    func testTitlesCombineBrandAndModel() {
        XCTAssertEqual(GearDisplayRules.itemTitle(GearFixtures.item(brandName: "Alpha", modelName: "A-100")), "Alpha A-100")
        XCTAssertEqual(
            GearDisplayRules.itemTitle(GearFixtures.item(house: true, brandName: "", modelName: "球房公杆")),
            "球房公杆"
        )
        XCTAssertEqual(
            GearDisplayRules.itemTitle(GearFixtures.item(brandName: " 其他品牌 ", modelName: "未收录型号")),
            "其他品牌 未收录型号"
        )
        XCTAssertEqual(GearDisplayRules.itemTitle(GearFixtures.item(brandName: "", modelName: "")), "未命名装备")
        XCTAssertEqual(GearDisplayRules.itemTitle(nil), "")
        XCTAssertEqual(
            GearDisplayRules.productTitle(GearFixtures.product(brandName: "Alpha", name: "A-Tip")),
            "Alpha A-Tip"
        )
        XCTAssertEqual(
            GearDisplayRules.productTitle(GearFixtures.product(brandName: "", name: "球房巧克")),
            "球房巧克"
        )
    }

    func testSubtitleShowsStyleRolesAndFlagsUnlisted() {
        XCTAssertEqual(GearDisplayRules.itemSubtitle(GearFixtures.item(cueStyle: "pool")), "美式杆")
        XCTAssertEqual(
            GearDisplayRules.itemSubtitle(GearFixtures.item(unlisted: true, cueStyle: "snooker")),
            "斯诺克 / 中式杆 · 未收录"
        )
        XCTAssertEqual(GearDisplayRules.itemSubtitle(GearFixtures.item(house: true)), "球房公用")
        XCTAssertEqual(
            GearDisplayRules.itemSubtitle(GearFixtures.item(category: "break_jump", roles: ["break", "jump"])),
            "冲杆 · 跳杆"
        )
        XCTAssertEqual(GearDisplayRules.roles(["jump"]), "跳杆")
        XCTAssertEqual(GearDisplayRules.roles([]), "")
        XCTAssertEqual(GearDisplayRules.cueStyle("carom"), "")
    }

    func testProductSubtitleAddsSpecsWithoutOunces() {
        let product = GearFixtures.product(lengthMm: 1470, weightG: 540, tipDiameterMm: 12.9)
        XCTAssertEqual(
            GearDisplayRules.productSubtitle(product),
            "美式杆 · 1470 mm · 540 g · 12.9 mm"
        )
        XCTAssertEqual(
            GearDisplayRules.productSubtitle(GearFixtures.product(category: "break_jump", roles: ["break", "jump"])),
            "冲杆 · 跳杆"
        )
        XCTAssertEqual(GearDisplayRules.productSubtitle(nil), "")
    }

    func testGroupsSplitCuesBreakJumpAndRetiredDeletingDeleted() {
        let items = [
            GearFixtures.item(1, category: "cue"),
            GearFixtures.item(2, category: "cue", status: .retired),
            GearFixtures.item(3, category: "break_jump", roles: ["break"]),
            GearFixtures.item(4, category: "break_jump", roles: ["jump"]),
            GearFixtures.item(5, category: "break_jump", status: .retired, roles: ["break", "jump"]),
            GearFixtures.item(6, category: "cue", status: .deleted),
            GearFixtures.item(7, category: "break_jump", roles: ["jump", "break"]),
        ]
        let groups = GearDisplayRules.groups(items)
        XCTAssertEqual(groups.cues.map(\.id), [1])
        XCTAssertEqual(groups.breakJump.map(\.id), [3, 4, 7])
        XCTAssertEqual(groups.retired.map(\.id), [2, 5])
        XCTAssertEqual(GearDisplayRules.groups([]), GearGroups())
    }

    func testMainCandidatesOnlyActiveMatchingItems() {
        let items = [
            GearFixtures.item(1, category: "cue"),
            GearFixtures.item(2, category: "cue", status: .retired),
            GearFixtures.item(3, category: "break_jump", roles: ["break"]),
            GearFixtures.item(4, category: "break_jump", roles: ["jump"]),
            GearFixtures.item(7, category: "break_jump", roles: ["jump", "break"]),
        ]
        XCTAssertEqual(GearDisplayRules.mainCandidates(items, slot: .cue).map(\.id), [1])
        XCTAssertEqual(GearDisplayRules.mainCandidates(items, slot: .breakSlot).map(\.id), [3, 7])
        XCTAssertFalse(GearDisplayRules.canBeMainCue(GearFixtures.item(3, category: "cue", status: .retired)))
        XCTAssertFalse(GearDisplayRules.canBeMainBreakCue(GearFixtures.item(4, category: "break_jump", roles: ["jump"])))
        XCTAssertFalse(GearDisplayRules.canBeMainBreakCue(nil))
    }

    func testDefaultsViewUsesFixedGameOrderAndSnookerHasNoBreakSlot() {
        let items = [
            GearFixtures.item(1, category: "cue"),
            GearFixtures.item(3, category: "break_jump", roles: ["break"]),
        ]
        let defaults = [
            GearFixtures.mainDefault(3, slot: .cue, itemId: 1),
            GearFixtures.mainDefault(3, slot: .breakSlot, itemId: 3),
            GearFixtures.mainDefault(1, slot: .cue, itemId: 1),
            GearFixtures.mainDefault(1, slot: .breakSlot, itemId: 3),
            GearFixtures.mainDefault(4, slot: .cue, itemId: 999),
        ]
        let view = GearDisplayRules.defaultsView(defaults, items: items)
        XCTAssertEqual(view.map(\.gameType), [3, 2, 1, 4])
        XCTAssertEqual(view[0].label, "中式八球")
        XCTAssertEqual(view[0].cue?.id, 1)
        XCTAssertEqual(view[0].breakCue?.id, 3)
        XCTAssertFalse(view[2].supportsBreak)
        XCTAssertNil(view[2].breakCue, "斯诺克没有冲杆槽位，即使数据里有也不展示")
        XCTAssertNil(view[1].cue)
        XCTAssertNil(view[3].cue, "指向不存在装备的设置按未设置展示")
        XCTAssertEqual(GearDisplayRules.defaultsView([], items: []).count, 4)
    }

    func testMainGameLabelsReadAsGamePlusMain() {
        let defaults = [
            GearFixtures.mainDefault(4, slot: .cue, itemId: 1),
            GearFixtures.mainDefault(3, slot: .cue, itemId: 1),
            GearFixtures.mainDefault(2, slot: .breakSlot, itemId: 1),
            GearFixtures.mainDefault(3, slot: .cue, itemId: 2),
        ]
        XCTAssertEqual(
            GearDisplayRules.mainGameLabels(defaults, itemId: 1),
            ["中式八球 主力", "九球追分 冲杆主力", "美式九球 主力"]
        )
        XCTAssertEqual(GearDisplayRules.mainGameLabels(defaults, itemId: 2), ["中式八球 主力"])
        XCTAssertEqual(GearDisplayRules.mainGameLabels(defaults, itemId: 99), [])
        XCTAssertEqual(GearDisplayRules.mainGameLabels([], itemId: 1), [])
    }
}
