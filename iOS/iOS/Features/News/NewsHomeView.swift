import SwiftUI

/// 赛讯 Tab：状态筛选、年份 / 球种选择、分区赛事卡片与分页，逐条对应 `pages/tournament/index.vue`。
struct NewsHomeView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: SaiXunListModel

    init(session: SessionStore, model: SaiXunListModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SaiXunListModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            header
            content
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("赛讯")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.reloadForIdentityChange() }
        }
    }

    // MARK: - 顶部筛选

    private var header: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 6) {
                ForEach(SaiXunListModel.statusTabs) { tab in
                    statusTab(tab)
                }
            }
            filterRow
        }
        .padding(.horizontal, 12)
        .padding(.top, 10)
        .padding(.bottom, 2)
    }

    private func statusTab(_ tab: SaiXunListModel.StatusTab) -> some View {
        let isActive = model.filter.status == tab.value

        return Button {
            Task { await model.selectStatus(tab.value) }
        } label: {
            Text(tab.label)
                .font(.system(size: 13, weight: isActive ? .semibold : .medium))
                .foregroundStyle(isActive ? .white : palette.textSecondary)
                .lineLimit(1)
                .padding(.horizontal, 13)
                .frame(height: 30)
                .background(isActive ? AppTheme.primary : palette.card, in: Capsule())
                .overlay {
                    if !isActive {
                        Capsule().stroke(palette.border)
                    }
                }
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isActive ? .isSelected : [])
    }

    private var filterRow: some View {
        HStack(spacing: 16) {
            Menu {
                ForEach(model.yearOptions, id: \.self) { year in
                    Button("\(String(year))年") {
                        Task { await model.selectYear(year) }
                    }
                }
            } label: {
                HStack(spacing: 3) {
                    Text(verbatim: "\(String(model.filter.year ?? 0))年")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(palette.textSecondary)
                    Image(systemName: "chevron.down")
                        .font(.system(size: 8, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
            }

            Menu {
                ForEach(SaiXunListModel.gameTypeOptions) { option in
                    Button(option.label) {
                        Task { await model.selectGameType(option.value) }
                    }
                }
            } label: {
                HStack(spacing: 3) {
                    Text(model.selectedGameTypeLabel)
                        .font(.system(size: 12, weight: model.filter.gameType == 0 ? .medium : .semibold))
                        .foregroundStyle(model.filter.gameType == 0 ? palette.textSecondary : palette.brandStrong)
                    Image(systemName: "chevron.down")
                        .font(.system(size: 8, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
            }

            Spacer(minLength: 8)

            if model.total > 0 {
                Text(verbatim: "共 \(String(model.total)) 项赛事")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
        }
        .frame(minHeight: 22)
    }

    // MARK: - 列表

    private var content: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                switch model.phase {
                case .idle, .loading:
                    skeletonList
                case .failed(let failure):
                    failureState(failure)
                case .empty:
                    emptyState()
                case .loaded:
                    feed
                }
            }
        }
        .refreshable { await model.refresh() }
    }

    private var feed: some View {
        VStack(spacing: 0) {
            if let refreshError = model.refreshError {
                refreshBanner(refreshError)
                    .padding(.horizontal, 12)
            }

            ForEach(model.sections, id: \.key) { section in
                sectionHeader(section)
                ForEach(section.items) { item in
                    cardView(item)
                        .padding(.horizontal, 12)
                        .padding(.top, 8)
                }
            }

            loadMoreFooter
        }
        .padding(.bottom, 12)
    }

    private func sectionHeader(_ section: SaiXunListSection) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 6) {
            Text(section.title)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            if !section.caption.isEmpty {
                Text(section.caption)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 14)
        .padding(.top, 14)
        .padding(.bottom, 2)
    }

    private func cardView(_ item: EventNewsInfo) -> some View {
        let card = model.card(for: item)
        let isLast = model.sections.last?.items.last?.id == item.id

        return NavigationLink(value: AppRoute.tournamentDetail(id: card.id)) {
            if card.isFeatured {
                featuredCard(card)
            } else {
                eventCard(card)
            }
        }
        .buttonStyle(.plain)
        .task {
            // 滚动到底部附近时加载下一页。
            guard isLast else { return }
            await model.loadMore()
        }
    }

    private func featuredCard(_ card: SaiXunCard) -> some View {
        VStack(spacing: 0) {
            ZStack(alignment: .topLeading) {
                LinearGradient(
                    colors: palette.heroColors,
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                if card.hasCover {
                    RemoteImage(url: coverURL(card.coverImage)) {
                        Color.clear
                    }
                }

                LinearGradient(
                    colors: [.clear, Color.black.opacity(0.8)],
                    startPoint: UnitPoint(x: 0.5, y: 0.28),
                    endPoint: .bottom
                )

                featureBadge(card)

                VStack(alignment: .leading, spacing: 4) {
                    Text(card.title)
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(.white)
                        .lineLimit(2)
                    Text(card.metaText)
                        .font(.system(size: 11))
                        .foregroundStyle(.white.opacity(0.82))
                        .lineLimit(1)
                }
                .padding(.horizontal, 12)
                .padding(.bottom, 11)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomLeading)
            }
            .frame(height: 150)
            .clipped()

            HStack(spacing: 7) {
                if let highlight = card.highlight, !highlight.label.isEmpty {
                    Text(highlight.label)
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 6))
                }
                Text(card.highlight?.primary ?? card.matchCountText)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Spacer(minLength: 4)
                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 11)
        }
        .background(palette.card)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    private func featureBadge(_ card: SaiXunCard) -> some View {
        HStack(spacing: 4) {
            if card.statusTone == .live {
                Circle()
                    .fill(palette.success)
                    .frame(width: 5, height: 5)
            }
            Text(card.statusLabel)
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(.white)
        }
        .padding(.horizontal, 8)
        .frame(height: 22)
        .background(Color.black.opacity(0.6), in: Capsule())
        .padding(10)
    }

    private func eventCard(_ card: SaiXunCard) -> some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack(spacing: 8) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(card.title)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)
                    Text(card.metaText)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                if card.statusTone == .canceled {
                    SaiXunStatusBadge(tone: card.statusTone, label: card.statusLabel)
                } else {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
            }

            if let highlight = card.highlight {
                HStack(spacing: 6) {
                    HStack(spacing: 2) {
                        Image(systemName: "medal.fill")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.brandStrong)
                        Text(highlight.label)
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                    }
                    Text(highlight.primary)
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(highlight.secondary)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(1)
                    Spacer(minLength: 0)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 7)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
            }
        }
        .appCard(padding: 12)
    }

    private func coverURL(_ source: SaiXunImageSource) -> URL? {
        if case .remote(let url) = source { return url }
        return nil
    }

    // MARK: - 底部与状态

    private var loadMoreFooter: some View {
        Group {
            if model.hasMore {
                Text(model.isLoadingMore ? "正在加载..." : "上拉加载更多")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            } else {
                HStack(spacing: 8) {
                    Rectangle()
                        .fill(palette.border)
                        .frame(width: 24, height: 1)
                    Text("已展示全部赛事")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                    Rectangle()
                        .fill(palette.border)
                        .frame(width: 24, height: 1)
                }
            }
        }
        .padding(.top, 18)
    }

    private var skeletonList: some View {
        VStack(spacing: 8) {
            ForEach(0..<3, id: \.self) { _ in
                VStack(alignment: .leading, spacing: 8) {
                    skeletonLine(width: 0.56, height: 16)
                    skeletonLine(width: 0.72, height: 11)
                    skeletonLine(width: 0.4, height: 11)
                }
                .appCard(padding: 14)
            }
        }
        .padding(12)
    }

    private func skeletonLine(width: CGFloat, height: CGFloat) -> some View {
        GeometryReader { proxy in
            RoundedRectangle(cornerRadius: 6)
                .fill(palette.border)
                .frame(width: proxy.size.width * width, height: height)
        }
        .frame(height: height)
    }

    private func emptyState() -> some View {
        VStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(palette.brandTint)
                    .frame(width: 56, height: 56)
                Image(systemName: "calendar")
                    .font(.system(size: 24, weight: .medium))
                    .foregroundStyle(palette.brandStrong)
            }
            Text(model.emptyTitle)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(model.emptyDescription)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
            if model.hasNarrowFilter {
                Button("查看全部赛事") {
                    Task { await model.resetNarrowFilter() }
                }
                .buttonStyle(.appPill)
                .padding(.top, 4)
            }
        }
        .padding(.horizontal, 24)
        .padding(.top, 60)
        .padding(.bottom, 24)
        .frame(maxWidth: .infinity)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "赛讯")

        return VStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(palette.brandTint)
                    .frame(width: 56, height: 56)
                Image(systemName: "info.circle")
                    .font(.system(size: 24, weight: .medium))
                    .foregroundStyle(palette.textMuted)
            }
            Text(feedback.title)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
            if feedback.retryable {
                Button(feedback.actionText) {
                    Task { await model.retry() }
                }
                .buttonStyle(.appPill)
                .padding(.top, 4)
            }
        }
        .padding(.horizontal, 24)
        .padding(.top, 60)
        .padding(.bottom, 24)
        .frame(maxWidth: .infinity)
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9).stroke(palette.border)
        }
        .padding(.top, 8)
    }
}
