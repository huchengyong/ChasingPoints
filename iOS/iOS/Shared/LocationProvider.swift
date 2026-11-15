import CoreLocation
import Foundation

/// GCJ-02 坐标；后端与小程序统一使用该坐标系。
struct Coordinate: Equatable {
    let latitude: Double
    let longitude: Double
}

enum LocationError: Error, Equatable {
    case denied
    case unavailable
}

/// 一次性定位。页面通过它取当前位置，测试里注入替身。
@MainActor
protocol LocationProviding {
    func currentLocation() async throws -> Coordinate
}

/// WGS-84 → GCJ-02 的公开算法实现；境外坐标不转换。
enum GCJ02 {
    static func transform(_ coordinate: Coordinate) -> Coordinate {
        guard !isOutOfChina(coordinate) else { return coordinate }

        let a = 6378245.0
        let ee = 0.00669342162296594323
        let latitude = coordinate.latitude
        let longitude = coordinate.longitude
        var dLat = latitudeOffset(longitude - 105, latitude - 35)
        var dLon = longitudeOffset(longitude - 105, latitude - 35)
        let radLat = latitude / 180 * .pi
        var magic = sin(radLat)
        magic = 1 - ee * magic * magic
        let sqrtMagic = sqrt(magic)
        dLat = dLat * 180 / ((a * (1 - ee)) / (magic * sqrtMagic) * .pi)
        dLon = dLon * 180 / (a / sqrtMagic * cos(radLat) * .pi)
        return Coordinate(latitude: latitude + dLat, longitude: longitude + dLon)
    }

    static func isOutOfChina(_ coordinate: Coordinate) -> Bool {
        !(coordinate.longitude > 73.66 && coordinate.longitude < 135.05
            && coordinate.latitude > 3.86 && coordinate.latitude < 53.55)
    }

    private static func latitudeOffset(_ x: Double, _ y: Double) -> Double {
        var result = -100 + 2 * x + 3 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * sqrt(abs(x))
        result += (20 * sin(6 * x * .pi) + 20 * sin(2 * x * .pi)) * 2 / 3
        result += (20 * sin(y * .pi) + 40 * sin(y / 3 * .pi)) * 2 / 3
        result += (160 * sin(y / 12 * .pi) + 320 * sin(y * .pi / 30)) * 2 / 3
        return result
    }

    private static func longitudeOffset(_ x: Double, _ y: Double) -> Double {
        var result = 300 + x + 2 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * sqrt(abs(x))
        result += (20 * sin(6 * x * .pi) + 20 * sin(2 * x * .pi)) * 2 / 3
        result += (20 * sin(x * .pi) + 40 * sin(x / 3 * .pi)) * 2 / 3
        result += (150 * sin(x / 12 * .pi) + 300 * sin(x / 30 * .pi)) * 2 / 3
        return result
    }
}

/// 使用期间授权的一次性定位；定位结果转换为 GCJ-02 后返回。
@MainActor
final class SystemLocationProvider: NSObject, LocationProviding {
    private let manager = CLLocationManager()
    private var continuation: CheckedContinuation<Coordinate, Error>?

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
    }

    func currentLocation() async throws -> Coordinate {
        guard continuation == nil else { throw LocationError.unavailable }
        // 缺少用途说明时请求定位会被系统强制终止，先按不可用处理。
        guard Bundle.main.object(forInfoDictionaryKey: "NSLocationWhenInUseUsageDescription") != nil else {
            throw LocationError.unavailable
        }

        switch manager.authorizationStatus {
        case .denied, .restricted:
            throw LocationError.denied
        case .notDetermined, .authorizedWhenInUse, .authorizedAlways:
            break
        @unknown default:
            throw LocationError.unavailable
        }

        return try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
            if manager.authorizationStatus == .notDetermined {
                manager.requestWhenInUseAuthorization()
            } else {
                manager.requestLocation()
            }
        }
    }

    private func handleAuthorizationChange() {
        guard continuation != nil else { return }
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            manager.requestLocation()
        case .denied, .restricted:
            finish(.failure(LocationError.denied))
        case .notDetermined:
            break
        @unknown default:
            finish(.failure(LocationError.unavailable))
        }
    }

    private func finish(_ result: Result<Coordinate, Error>) {
        guard let continuation else { return }
        self.continuation = nil
        continuation.resume(with: result)
    }
}

extension SystemLocationProvider: CLLocationManagerDelegate {
    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        MainActor.assumeIsolated {
            handleAuthorizationChange()
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        MainActor.assumeIsolated {
            guard let location = locations.last else { return }
            let coordinate = Coordinate(
                latitude: location.coordinate.latitude,
                longitude: location.coordinate.longitude
            )
            finish(.success(GCJ02.transform(coordinate)))
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        MainActor.assumeIsolated {
            let isDenied = (error as? CLError)?.code == .denied
            finish(.failure(isDenied ? LocationError.denied : LocationError.unavailable))
        }
    }
}
