import Foundation
@testable import iOS

/// 球馆模块的虚构夹具：字段与 `VenueInfo` / `VenueAreaOption` 解码路径一致。
enum VenueFixtures {
    static func venueJSON(
        id: Int64 = 1,
        name: String = "星轨台球俱乐部",
        address: String = "科苑路 8 号",
        city: String = "深圳市",
        district: String = "南山区",
        latitude: Double = 22.540796,
        longitude: Double = 113.917176,
        phone: String = "0755-12345678",
        images: [String] = [],
        businessHours: String = "10:00-23:00",
        tableCount: Int = 18,
        priceRange: String = "30-60 元/小时",
        description: String = "社区球房",
        distance: Double? = nil,
        checkinCount: Int = 0
    ) -> [String: Any] {
        var payload: [String: Any] = [
            "id": id,
            "name": name,
            "address": address,
            "city": city,
            "district": district,
            "latitude": latitude,
            "longitude": longitude,
            "phone": phone,
            "images": images,
            "business_hours": businessHours,
            "table_count": tableCount,
            "price_range": priceRange,
            "description": description,
            "checkin_count": checkinCount,
            "status": 1,
        ]
        if let distance {
            payload["distance"] = distance
        }
        return payload
    }

    static func listResponse(_ venues: [[String: Any]], total: Int? = nil) -> [String: Any] {
        [
            "success": true,
            "total": total ?? venues.count,
            "list": venues,
        ]
    }

    static func areaJSON(_ areas: [(id: Int64, parentId: Int64, name: String)]) -> [String: Any] {
        [
            "success": true,
            "list": areas.map { ["area_id": $0.id, "parent_id": $0.parentId, "name": $0.name] },
        ]
    }
}

@MainActor
func makeGuestVenueSession(_ server: MockHTTPServer) -> SessionStore {
    makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
}

@MainActor
func makeSignedInVenueSession(_ server: MockHTTPServer) async -> SessionStore {
    server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
    let session = makeSessionStore(
        server,
        credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
    )
    await session.restore()
    return session
}

@MainActor
final class StubLocationProvider: LocationProviding {
    var result: Result<Coordinate, LocationError>
    private(set) var callCount = 0

    init(result: Result<Coordinate, LocationError>) {
        self.result = result
    }

    func currentLocation() async throws -> Coordinate {
        callCount += 1
        return try result.get()
    }
}
