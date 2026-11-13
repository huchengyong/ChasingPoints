import Foundation

struct VenuePage: Equatable {
    let items: [VenueInfo]
    let total: Int
}

struct VenueCreateResult: Equatable {
    let venueId: Int64
    let message: String
}

/// 球馆模块的请求门面。公开读取走 optionalAuthRequest，写入走 authorizedPost。
@MainActor
struct VenueAPI {
    static let listPageSize = 10
    static let nearbyRadius = 10000
    static let nearbyLimit = 20

    let session: SessionStore

    func list(page: Int, coordinate: Coordinate?) async throws -> VenuePage {
        let path = APIQuery.path("/api/venue/list", [
            "page": page,
            "page_size": Self.listPageSize,
            "latitude": coordinate?.latitude,
            "longitude": coordinate?.longitude,
        ])
        let response: VenueListResponse = try await session.optionalAuthRequest(path)
        return VenuePage(items: response.list, total: Int(response.total))
    }

    /// 附近球馆：小程序固定 radius 10000、limit 20，返回后不再分页。
    func nearby(_ coordinate: Coordinate) async throws -> VenuePage {
        let path = APIQuery.path("/api/venue/nearby", [
            "latitude": coordinate.latitude,
            "longitude": coordinate.longitude,
            "radius": Self.nearbyRadius,
            "limit": Self.nearbyLimit,
        ])
        let response: VenueListResponse = try await session.optionalAuthRequest(path)
        return VenuePage(items: response.list, total: Int(response.total))
    }

    func detail(id: Int64) async throws -> VenueInfo {
        guard id > 0 else {
            throw APIError(message: "未找到球馆信息，请返回球馆列表后重试。", statusCode: 404)
        }
        let path = APIQuery.path("/api/venue/detail", ["venue_id": id])
        let response: VenueDetailResponse = try await session.optionalAuthRequest(path)
        guard let venue = response.venue else {
            throw APIError(message: "球馆不存在或已下线", statusCode: 404)
        }
        return venue
    }

    func areas(parentId: Int64) async throws -> [VenueAreaOption] {
        let path = APIQuery.path("/api/venue/areas", ["parent_id": parentId])
        let response: VenueAreaResponse = try await session.optionalAuthRequest(path)
        return response.list
    }

    func create(_ payload: VenueSubmitPayload) async throws -> VenueCreateResult {
        let response: VenueCreateResponse = try await session.authorizedPost("/api/venue/create", body: payload)
        return VenueCreateResult(venueId: response.venueId, message: response.message)
    }

    func checkin(venueId: Int64) async throws -> String {
        let response: VenueCheckinResponse = try await session.authorizedPost(
            "/api/venue/checkin",
            body: VenueIdBody(venueId: venueId)
        )
        return response.message
    }
}

private struct VenueListResponse: Decodable {
    let total: Int64
    let list: [VenueInfo]
}

private struct VenueDetailResponse: Decodable {
    let venue: VenueInfo?
}

private struct VenueAreaResponse: Decodable {
    let list: [VenueAreaOption]
}

private struct VenueCreateResponse: Decodable {
    let venueId: Int64
    let message: String
}

private struct VenueCheckinResponse: Decodable {
    let message: String
}

private struct VenueIdBody: Encodable {
    let venueId: Int64
}
