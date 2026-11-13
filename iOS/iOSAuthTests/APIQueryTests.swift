import Foundation
import XCTest
@testable import iOS

@MainActor
final class APIQueryTests: XCTestCase {
    func testNilValuesAreSkippedAndEmptyQueryHasNoQuestionMark() {
        XCTAssertEqual(APIQuery.path("/api/x", ["page": nil]), "/api/x")
        XCTAssertEqual(APIQuery.path("/api/x", [:]), "/api/x")
    }

    func testPairsAreSortedByKeyAndJoined() {
        XCTAssertEqual(
            APIQuery.path("/api/match/list", ["page": 1, "page_size": 20]),
            "/api/match/list?page=1&page_size=20"
        )
        XCTAssertEqual(APIQuery.path("/api/x", ["b": "2", "a": "1"]), "/api/x?a=1&b=2")
        XCTAssertEqual(APIQuery.path("/api/x", ["b": "2", "a": nil, "c": "3"]), "/api/x?b=2&c=3")
    }

    func testReservedCharactersArePercentEncoded() {
        let value = "a b&c=d+e#f?中"
        let path = APIQuery.path("/api/x", ["q": value])

        XCTAssertEqual(path, "/api/x?q=a%20b%26c%3Dd%2Be%23f%3F%E4%B8%AD")

        let components = try? XCTUnwrap(URLComponents(string: "https://example.test" + path))
        XCTAssertEqual(components?.queryItems?.first?.value, value, "解码后必须还原原值")
    }
}
