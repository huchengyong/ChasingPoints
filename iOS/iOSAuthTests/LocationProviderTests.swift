import XCTest
@testable import iOS

final class LocationProviderTests: XCTestCase {
    func testTransformConvertsKnownMainlandCoordinates() {
        assertTransform(
            Coordinate(latitude: 31.1774276, longitude: 121.5272106),
            equals: Coordinate(latitude: 31.17530398364597, longitude: 121.531541859215)
        )
        assertTransform(
            Coordinate(latitude: 22.543847, longitude: 113.912316),
            equals: Coordinate(latitude: 22.540796131694766, longitude: 113.9171764808363)
        )
        assertTransform(
            Coordinate(latitude: 39.908823, longitude: 116.397470),
            equals: Coordinate(latitude: 39.91022649807321, longitude: 116.4037135824225)
        )
    }

    func testTransformLeavesOutOfChinaCoordinatesUntouched() {
        let tokyo = Coordinate(latitude: 35.6762, longitude: 139.6503)
        let newYork = Coordinate(latitude: 40.7128, longitude: -74.0060)
        let origin = Coordinate(latitude: 0, longitude: 0)

        XCTAssertEqual(GCJ02.transform(tokyo), tokyo)
        XCTAssertEqual(GCJ02.transform(newYork), newYork)
        XCTAssertEqual(GCJ02.transform(origin), origin)
    }

    private func assertTransform(
        _ source: Coordinate,
        equals expected: Coordinate,
        accuracy: Double = 1e-6,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        let result = GCJ02.transform(source)
        XCTAssertEqual(result.latitude, expected.latitude, accuracy: accuracy, file: file, line: line)
        XCTAssertEqual(result.longitude, expected.longitude, accuracy: accuracy, file: file, line: line)
    }
}
