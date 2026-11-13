import MapKit
import SwiftUI
import UIKit

/// 球馆详情：档案、图片、基础信息、签到动态与底部操作栏。
struct VenueDetailView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: VenueDetailModel

    /// 快照测试关闭地图，避免渲染瓦片。
    private let showsMap: Bool

    init(
        session: SessionStore,
        venueId: Int64,
        model: VenueDetailModel? = nil,
        showsMap: Bool = true
    ) {
        _model = StateObject(wrappedValue: model ?? VenueDetailModel(session: session, venueId: venueId))
        self.showsMap = showsMap
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        LoadableContent(
            loader: model.loader,
            requiresLogin: false,
            resource: "球馆详情"
        ) { venue in
            detail(venue)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("球馆详情")
        .navigationBarTitleDisplayMode(.inline)
        .onChange(of: session.identity) { _, _ in
            model.syncIdentity()
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
    }

    private func detail(_ venue: VenueInfo) -> some View {
        ScrollView {
            VStack(spacing: 12) {
                heroCard(venue)
                imageSection(venue)
                infoCard(venue)
                mapSection(venue)
                checkinCard(venue)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .safeAreaInset(edge: .bottom) {
            actionBar(venue)
        }
    }

    private func heroCard(_ venue: VenueInfo) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            VStack(alignment: .leading, spacing: 6) {
                Text("球馆档案")
                    .font(.system(size: 12))
                    .foregroundStyle(.white.opacity(0.72))
                Text(venue.name)
                    .font(.system(size: 22, weight: .bold))
                    .foregroundStyle(.white)
                Text(venue.description.isEmpty ? "该球馆暂未填写简介，可先查看地址、营业时间和签到情况。" : venue.description)
                    .font(.system(size: 13))
                    .foregroundStyle(.white.opacity(0.8))
                    .lineSpacing(4)
            }

            HStack(spacing: 0) {
                heroStat("营业时间", venue.businessHours.isEmpty ? "暂未填写" : venue.businessHours)
                heroStat("球桌数量", venue.tableCount > 0 ? "\(venue.tableCount) 台" : "暂未填写")
                heroStat("签到人数", "\(venue.checkinCount)")
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            LinearGradient(colors: palette.heroColors, startPoint: .topLeading, endPoint: .bottomTrailing),
            in: RoundedRectangle(cornerRadius: 16)
        )
    }

    private func heroStat(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(.white.opacity(0.65))
            Text(value)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(.white)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func imageSection(_ venue: VenueInfo) -> some View {
        let urls = venue.images.compactMap(CDNAsset.url(for:))
        if urls.isEmpty {
            VStack(spacing: 6) {
                Image(systemName: "location")
                    .font(.system(size: 24))
                    .foregroundStyle(palette.brandStrong)
                    .frame(width: 48, height: 48)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 12))
                Text("暂无球馆图片")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text("先通过地址、营业信息和签到数据了解这个球馆。")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 24)
            .appCard()
        } else {
            TabView {
                ForEach(Array(urls.enumerated()), id: \.offset) { _, url in
                    RemoteImage(url: url) {
                        palette.subtle
                    }
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .always))
            .frame(height: 200)
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }

    private func infoCard(_ venue: VenueInfo) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("基础信息")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Text(venue.city.isEmpty ? "未标注城市" : venue.city)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            infoRow(icon: "location", label: "详细地址", value: venue.address.isEmpty ? "暂无地址" : venue.address)
            if !venue.phone.isEmpty {
                infoRow(icon: "phone", label: "联系电话", value: venue.phone)
            }
            if !venue.businessHours.isEmpty {
                infoRow(icon: "calendar", label: "营业时间", value: venue.businessHours)
            }

            if venue.tableCount > 0 || !venue.priceRange.isEmpty || !venue.city.isEmpty {
                HStack(spacing: 8) {
                    if venue.tableCount > 0 {
                        infoTag("\(venue.tableCount)台球桌")
                    }
                    if !venue.priceRange.isEmpty {
                        infoTag(venue.priceRange)
                    }
                    if !venue.city.isEmpty {
                        infoTag(venue.city)
                    }
                }
            }
        }
        .appCard()
    }

    private func infoRow(icon: String, label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
                .frame(width: 18)
            VStack(alignment: .leading, spacing: 2) {
                Text(label)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                Text(value)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textPrimary)
            }
            Spacer(minLength: 0)
        }
    }

    private func infoTag(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(palette.textSecondary)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 4))
    }

    @ViewBuilder
    private func mapSection(_ venue: VenueInfo) -> some View {
        if venue.latitude != 0 && venue.longitude != 0 {
            if showsMap {
                Map(
                    initialPosition: .region(MKCoordinateRegion(
                        center: CLLocationCoordinate2D(latitude: venue.latitude, longitude: venue.longitude),
                        span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                    )),
                    interactionModes: []
                ) {
                    Marker(venue.name, coordinate: CLLocationCoordinate2D(latitude: venue.latitude, longitude: venue.longitude))
                }
                .frame(height: 140)
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .allowsHitTesting(false)
                .accessibilityLabel("球馆位置")
            } else {
                RoundedRectangle(cornerRadius: 12)
                    .fill(palette.subtle)
                    .frame(height: 140)
            }
        }
    }

    private func checkinCard(_ venue: VenueInfo) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("签到动态")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                if venue.checkinCount > 0 {
                    Text("\(venue.checkinCount)人到访")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
            }
            // 后端列表与详情都不返回签到用户，原生只呈现空态。
            Text("还没有人来签到，来当第一个打卡的人吧。")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
        }
        .appCard()
    }

    private func actionBar(_ venue: VenueInfo) -> some View {
        HStack(spacing: 10) {
            Button {
                openNavigation(venue)
            } label: {
                actionLabel(icon: "location.north.fill", title: "去导航", color: palette.brandStrong)
            }
            .buttonStyle(.plain)

            if !venue.phone.isEmpty {
                Button {
                    call(venue.phone)
                } label: {
                    actionLabel(icon: "phone.fill", title: "打电话", color: palette.textSecondary)
                }
                .buttonStyle(.plain)
            }

            Button {
                Task {
                    let result = await model.checkin()
                    if result == .requiresLogin {
                        navigator.openLogin()
                    }
                }
            } label: {
                Text(model.hasCheckedIn ? "今日已签到" : "签到打卡")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 44)
                    .background(
                        model.hasCheckedIn ? AppTheme.primary.opacity(0.5) : AppTheme.primary,
                        in: RoundedRectangle(cornerRadius: 10)
                    )
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card)
        .overlay(alignment: .top) {
            Rectangle()
                .fill(palette.border)
                .frame(height: 1)
        }
    }

    private func actionLabel(icon: String, title: String, color: Color) -> some View {
        VStack(spacing: 3) {
            Image(systemName: icon)
                .font(.system(size: 18))
            Text(title)
                .font(.system(size: 12))
        }
        .foregroundStyle(color)
        .frame(minWidth: 52)
        .frame(height: 44)
    }

    private func openNavigation(_ venue: VenueInfo) {
        guard venue.latitude != 0, venue.longitude != 0 else {
            toast.show("打开导航失败")
            return
        }
        let item = MKMapItem(placemark: MKPlacemark(coordinate: CLLocationCoordinate2D(
            latitude: venue.latitude,
            longitude: venue.longitude
        )))
        item.name = venue.name
        // 后端坐标已是 GCJ-02，地图直接使用，不再做二次转换。
        guard item.openInMaps(launchOptions: nil) else {
            toast.show("打开导航失败")
            return
        }
    }

    private func call(_ phone: String) {
        let digits = phone.filter { $0.isNumber || $0 == "+" }
        guard !digits.isEmpty, let url = URL(string: "tel:\(digits)") else { return }
        UIApplication.shared.open(url)
    }
}
