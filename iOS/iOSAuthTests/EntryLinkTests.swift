import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/entry-link.test.mjs`。
@MainActor
final class EntryLinkTests: XCTestCase {
    private let code = "Ab3dEf6hIj9k"

    func testEntryLinkBuildsAndParsesAllThreeTypes() {
        let invite = EntryLink.buildLink(.invite(code: code))
        let match = EntryLink.buildLink(.match(matchId: 123))
        let home = EntryLink.buildLink(.home)

        XCTAssertEqual(invite, "https://www.zhuifen.cn/go/?j=\(code)")
        XCTAssertEqual(match, "https://www.zhuifen.cn/go/?m=123")
        XCTAssertEqual(home, "https://www.zhuifen.cn/go/")

        XCTAssertEqual(EntryLink.parse(invite), .invite(inviteCode: code))
        XCTAssertEqual(EntryLink.parse(match), .match(matchId: 123))
        XCTAssertEqual(EntryLink.parse(home), .home)
        XCTAssertEqual(EntryLink.spectateDestination(matchId: 88), .spectate(matchId: 88))
    }

    func testEntryLinkAcceptsPathWithOrWithoutTrailingSlash() {
        XCTAssertEqual(
            EntryLink.parse("https://www.zhuifen.cn/go?j=\(code)"),
            .invite(inviteCode: code)
        )
        XCTAssertEqual(
            EntryLink.parse("https://www.zhuifen.cn/go/?j=\(code)#frag"),
            .invite(inviteCode: code)
        )
        XCTAssertEqual(EntryLink.parse("https://www.zhuifen.cn/go"), .home)
    }

    func testEntryLinkRejectsOtherHostsProtocolsAndLookalikePaths() {
        let rejected = [
            "http://www.zhuifen.cn/go/?j=\(code)",
            "https://evil.example.com/go/?j=\(code)",
            "https://zhuifen.cn/go/?j=\(code)",
            "https://sub.www.zhuifen.cn/go/?j=\(code)",
            "https://www.zhuifen.cn/gox?j=\(code)",
            "https://www.zhuifen.cn/go/extra?j=\(code)",
            "https://www.zhuifen.cn.evil.com/go/?j=\(code)",
            "not a link",
            "",
        ]
        for raw in rejected {
            XCTAssertEqual(EntryLink.parse(raw), .other, raw)
        }
    }

    func testEntryLinkFallsBackToHomeWhenParamsAreInvalidOrAmbiguous() {
        let invalid = [
            "https://www.zhuifen.cn/go/?j=\(code)&m=1",
            "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9",
            "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9kk",
            "https://www.zhuifen.cn/go/?j=Ab3d-Ef6hIj9",
            "https://www.zhuifen.cn/go/?m=abc",
            "https://www.zhuifen.cn/go/?m=0",
            "https://www.zhuifen.cn/go/?m=-1",
            "https://www.zhuifen.cn/go/?m=1.5",
            "https://www.zhuifen.cn/go/?m=",
            "https://www.zhuifen.cn/go/?unknown=1",
        ]
        for raw in invalid {
            XCTAssertEqual(EntryLink.parse(raw), .home, raw)
        }
    }

    func testLaunchQueryDecodesOnceOrTwiceAndSurvivesBadInput() {
        let invite = "https://www.zhuifen.cn/go/?j=\(code)"
        XCTAssertEqual(EntryLink.decodeLaunchQuery(encodeURIComponent(invite)), invite)
        XCTAssertEqual(EntryLink.decodeLaunchQuery(encodeURIComponent(encodeURIComponent(invite))), invite)
        XCTAssertEqual(EntryLink.decodeLaunchQuery(invite), invite)
        XCTAssertEqual(EntryLink.decodeLaunchQuery(""), "")
        XCTAssertEqual(EntryLink.decodeLaunchQuery(nil), "")
        XCTAssertEqual(EntryLink.decodeLaunchQuery("%E0%A4%A"), "%E0%A4%A")
    }

    /// 与 JS 的 encodeURIComponent 一样保留 `-_.!~*'()`。
    private func encodeURIComponent(_ value: String) -> String {
        var allowed = CharacterSet.alphanumerics
        allowed.insert(charactersIn: "-_.!~*'()")
        return value.addingPercentEncoding(withAllowedCharacters: allowed) ?? value
    }
}
