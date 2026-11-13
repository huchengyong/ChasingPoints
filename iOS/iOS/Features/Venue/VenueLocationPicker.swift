import MapKit
import SwiftUI

/// 原生地图选点：图钉固定在屏幕中心，确定时回填中心坐标。
/// 小程序用 uni.chooseLocation 带逆地理编码；这里只取坐标，名称与地址留空。
struct VenueLocationPicker: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    private let onConfirm: (Coordinate) -> Void
    @State private var position: MapCameraPosition
    @State private var center: Coordinate

    /// 没有已选点时从上海市中心开始；shortcut: 不做定位居中，等原生接入定位精度后再改。
    private static let fallback = Coordinate(latitude: 31.2304, longitude: 121.4737)

    init(initial: Coordinate?, onConfirm: @escaping (Coordinate) -> Void) {
        let start = initial ?? Self.fallback
        self.onConfirm = onConfirm
        _center = State(initialValue: start)
        _position = State(initialValue: .region(MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: start.latitude, longitude: start.longitude),
            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
        )))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        NavigationStack {
            ZStack {
                Map(position: $position, interactionModes: [.pan, .zoom])
                    .onMapCameraChange(frequency: .continuous) { context in
                        center = Coordinate(
                            latitude: context.region.center.latitude,
                            longitude: context.region.center.longitude
                        )
                    }

                // 屏幕中心固定图钉：地图移动后确认的仍是图钉对准的位置。
                VStack(spacing: 2) {
                    Image(systemName: "mappin.circle.fill")
                        .font(.system(size: 34))
                        .foregroundStyle(AppTheme.primary)
                        .shadow(radius: 2)
                    Spacer()
                }
                .padding(.bottom, 34)
                .allowsHitTesting(false)
                .accessibilityHidden(true)
            }
            .overlay(alignment: .bottom) {
                Text("拖动地图，让图钉对准球馆位置")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(palette.card.opacity(0.92), in: Capsule())
                    .padding(.bottom, 16)
            }
            .navigationTitle("选择位置")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("确定") {
                        onConfirm(center)
                        dismiss()
                    }
                    .fontWeight(.semibold)
                }
            }
        }
    }
}
