import SwiftUI

/// 附近 / 全部球馆列表：定位栏、模式切换、分页与上传入口。
struct VenueListView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: VenueListModel

    init(session: SessionStore, model: VenueListModel? = nil) {
        _model = StateObject(wrappedValue: model ?? VenueListModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            locationBar
            modeTabs
            content
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("附近球馆")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            model.syncIdentity()
            Task { await model.loadIfNeeded() }
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
    }

    private var locationBar: some View {
        HStack(spacing: 12) {
            Button {
                Task { await model.requestCurrentLocation() }
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                        .font(.system(size: 15))
                    Text(model.locationText.isEmpty ? "定位中..." : model.locationText)
                        .font(.system(size: 14, weight: .medium))
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                .foregroundStyle(palette.textPrimary)
            }
            .buttonStyle(.plain)

            Spacer(minLength: 8)

            NavigationLink(value: AppRoute.venueSubmit) {
                HStack(spacing: 3) {
                    Image(systemName: "plus")
                        .font(.system(size: 12, weight: .bold))
                    Text("添加常玩球馆")
                        .font(.system(size: 13))
                }
                .foregroundStyle(.white)
                .padding(.horizontal, 12)
                .frame(height: 28)
                .background(AppTheme.primary, in: Capsule())
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card)
    }

    private var modeTabs: some View {
        HStack(spacing: 8) {
            ForEach(VenueListModel.Mode.allCases, id: \.self) { tab in
                let isActive = model.mode == tab
                Button {
                    Task { await model.switchMode(tab) }
                } label: {
                    Text(tab.title)
                        .font(.system(size: 13, weight: isActive ? .medium : .regular))
                        .foregroundStyle(isActive ? palette.brandStrong : palette.textSecondary)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 6)
                        .background(isActive ? palette.brandTint : palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                }
                .buttonStyle(.plain)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 12)
        .padding(.bottom, 10)
        .background(palette.card)
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .idle:
            loadingState
        case .loading where model.items.isEmpty:
            loadingState
        case .failed(let failure):
            failureState(failure)
        case .empty:
            emptyState
        default:
            list
        }
    }

    private var loadingState: some View {
        VStack(spacing: 8) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var list: some View {
        ScrollView {
            LazyVStack(spacing: 8) {
                if let refreshError = model.refreshError {
                    refreshBanner(refreshError)
                }
                ForEach(model.items) { item in
                    NavigationLink(value: AppRoute.venueDetail(id: item.id)) {
                        card(item)
                    }
                    .buttonStyle(.plain)
                    .task {
                        guard item.id == model.items.last?.id else { return }
                        await model.loadMore()
                    }
                }
                if model.isLoadingMore {
                    ProgressView()
                        .padding(.vertical, 8)
                }
                if let loadMoreError = model.loadMoreError {
                    Text(loadMoreError)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                if !model.items.isEmpty && !model.hasMore {
                    Text("没有更多了")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                        .padding(.vertical, 12)
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        .refreshable { await model.refresh() }
    }

    @MainActor
    private func card(_ item: VenueInfo) -> some View {
        HStack(alignment: .top, spacing: 10) {
            RemoteImage(url: item.images.first.flatMap(CDNAsset.url(for:))) {
                Text("🎱")
                    .font(.system(size: 24))
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(palette.subtle)
            }
            .frame(width: 80, height: 80)
            .clipShape(RoundedRectangle(cornerRadius: 6))

            VStack(alignment: .leading, spacing: 4) {
                Text(item.name)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text(item.address)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                    .lineLimit(1)
                if item.tableCount > 0 || !item.priceRange.isEmpty {
                    HStack(spacing: 8) {
                        if item.tableCount > 0 {
                            metaTag("\(item.tableCount)台")
                        }
                        if !item.priceRange.isEmpty {
                            metaTag(item.priceRange)
                        }
                    }
                }
                HStack(spacing: 10) {
                    if VenueDistance.isVisible(item.distance), let distance = item.distance {
                        Text(VenueDistance.format(distance))
                            .font(.system(size: 12))
                            .foregroundStyle(palette.brandStrong)
                    }
                    if item.checkinCount > 0 {
                        Text("\(item.checkinCount)人签到")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                    }
                    Spacer(minLength: 0)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 8))
    }

    private func metaTag(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(palette.textSecondary)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 3))
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Text("🎱")
                .font(.system(size: 48))
            Text(model.emptyTitle)
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
            Button("添加常玩球馆") {
                navigator.open(.venueSubmit)
            }
            .buttonStyle(.appPill)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "球馆列表")
        return StateMessageView(
            icon: "exclamationmark.triangle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task { await model.retry() }
            } else {
                dismiss()
            }
        }
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 8) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
    }
}
