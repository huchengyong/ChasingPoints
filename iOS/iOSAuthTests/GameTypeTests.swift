import Foundation
import XCTest
@testable import iOS

@MainActor
final class GameTypeCatalogTests: XCTestCase {
    func testOptionsUseAgreedDisplayOrder() {
        XCTAssertEqual(GameTypeCatalog.ordered.map(\.value), [3, 2, 1, 4])
        XCTAssertEqual(GameTypeCatalog.ordered.map(\.label), ["中式八球", "九球追分", "斯诺克", "美式九球"])
        XCTAssertEqual(GameTypeCatalog.ordered.map(\.key), ["chinese_eight", "nine_ball", "snooker", "american_nine"])
    }

    func testLabelKeyAndValueMappingsStayAligned() {
        XCTAssertEqual(GameTypeCatalog.label(for: 3), "中式八球")
        XCTAssertEqual(GameTypeCatalog.label(for: 2), "九球追分")
        XCTAssertEqual(GameTypeCatalog.label(for: 1), "斯诺克")
        XCTAssertEqual(GameTypeCatalog.label(for: 4), "美式九球")

        XCTAssertEqual(GameTypeCatalog.key(for: 3), "chinese_eight")
        XCTAssertEqual(GameTypeCatalog.key(for: 2), "nine_ball")
        XCTAssertEqual(GameTypeCatalog.key(for: 1), "snooker")
        XCTAssertEqual(GameTypeCatalog.key(for: 4), "american_nine")

        XCTAssertEqual(GameTypeCatalog.value(forKey: "chinese_eight"), 3)
        XCTAssertEqual(GameTypeCatalog.value(forKey: "nine_ball"), 2)
        XCTAssertEqual(GameTypeCatalog.value(forKey: "snooker"), 1)
        XCTAssertEqual(GameTypeCatalog.value(forKey: "american_nine"), 4)
    }

    func testHelpersReturnOrderedLabelsAndFallBackForUnknownTypes() {
        XCTAssertEqual(GameTypeCatalog.label(for: 99, fallback: "未知球种"), "未知球种")
        XCTAssertEqual(GameTypeCatalog.label(for: 0), "台球")
        XCTAssertNil(GameTypeCatalog.key(for: 99))
        XCTAssertNil(GameTypeCatalog.value(forKey: "unknown"))
        XCTAssertTrue(GameTypeCatalog.isSupported(3))
        XCTAssertFalse(GameTypeCatalog.isSupported(0))
    }
}

@MainActor
final class GameTypePreferenceTests: XCTestCase {
    private var suiteName = ""
    private var defaults: UserDefaults!

    override func setUp() {
        super.setUp()
        suiteName = "GameTypePreferenceTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
        defaults.removePersistentDomain(forName: suiteName)
    }

    override func tearDown() {
        defaults.removePersistentDomain(forName: suiteName)
        defaults = nil
        super.tearDown()
    }

    func testDefaultGameTypeAcceptsOnlySupportedValues() {
        XCTAssertEqual(GameTypePreference.normalize(1), 1)
        XCTAssertEqual(GameTypePreference.normalize(3), 3)
        XCTAssertEqual(GameTypePreference.normalize(0), 0)
        XCTAssertEqual(GameTypePreference.normalize(99), 0)
    }

    func testStorageIsIsolatedByUserId() {
        let preference = GameTypePreference(defaults: defaults)
        preference.save(userId: 101, gameType: 3)
        preference.save(userId: 202, gameType: 2)

        XCTAssertEqual(preference.read(userId: 101), 3)
        XCTAssertEqual(preference.read(userId: 202), 2)
        XCTAssertNotEqual(
            GameTypePreference.storageKey(userId: 101),
            GameTypePreference.storageKey(userId: 202)
        )
        XCTAssertEqual(GameTypePreference.storageKey(userId: 101), "default-game-type:101")
    }

    func testEveryTimeChoiceClearsStoredPreference() {
        let preference = GameTypePreference(defaults: defaults)
        preference.save(userId: 101, gameType: 1)
        XCTAssertEqual(preference.read(userId: 101), 1)

        XCTAssertEqual(preference.save(userId: 101, gameType: 0), 0)
        XCTAssertEqual(preference.read(userId: 101), 0)
        XCTAssertNil(defaults.object(forKey: "default-game-type:101"))
    }

    func testInvalidStoredValuesFallBackToEveryTime() {
        let preference = GameTypePreference(defaults: defaults)
        XCTAssertEqual(preference.read(userId: 101), 0, "没有保存过时按每次询问")
        defaults.set(88, forKey: "default-game-type:101")
        XCTAssertEqual(preference.read(userId: 101), 0)
        XCTAssertEqual(preference.read(userId: 0), 0)
        XCTAssertNil(GameTypePreference.storageKey(userId: 0))
    }

    func testSavedPreferenceSurvivesLaterReads() {
        let preference = GameTypePreference(defaults: defaults)
        preference.save(userId: 101, gameType: 3)

        // 扫码取消不写偏好，已明确保存的默认值应继续存在。
        XCTAssertEqual(preference.read(userId: 101), 3)
    }
}
