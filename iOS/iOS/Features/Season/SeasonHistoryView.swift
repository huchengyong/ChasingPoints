import SwiftUI

/// 历届赛季，对应小程序 `subPages/season/history.vue`。
struct SeasonHistoryView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: SeasonHistoryModel

    init(session: SessionStore, gameType: Int, model: SeasonHistoryModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SeasonHistoryModel(session: session, gameType: gameType))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看历届赛季",
                    actionTitle: "去登录"
                ) {
                    dismiss()
                }
            } else {
                ScrollView {
                    VStack(spacing: 10) {
                        gameTypeTabs
                        listSection
                    }
                    .padding(.horizontal, 12)
                    .padding(.top, 10)
                    .padding(.bottom, 28)
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("历届赛季")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.loadIfNeeded() }
        }
    }

    private var gameTypeTabs: some View {
        HStack(spacing: 6) {
            ForEach(GameTypeCatalog.ordered) { option in
                let isActive = model.currentGameType == option.value
                Button {
                    Task { await model.selectGameType(option.value) }
                } label: {
                    Text(option.label)
                        .font(.system(size: 12, weight: isActive ? .semibold : .regular))
                        .foregroundStyle(isActive ? palette.brandStrong : palette.textSecondary)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                        .frame(height: 34)
                        .background(isActive ? palette.brandTint : palette.card, in: Capsule())
                        .overlay {
                            Capsule().stroke(isActive ? AppTheme.primary.opacity(0.22) : palette.border)
                        }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    @ViewBuilder
    private var listSection: some View {
        if model.isSwitchingGameType {
            Text("加载中...")
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 24)
                .appCard()
        } else {
            switch model.phase {
            case .idle, .loading:
                loadingState
            case .failed(let failure):
                failureState(failure)
            case .empty:
                StateMessageView(
                    icon: "trophy",
                    title: "暂无已结束的赛季",
                    detail: "赛季结算后，这里会留下你的排名与赛季报告"
                )
                .padding(.top, 40)
            case .ready:
                ForEach(model.items) { row in
                    historyItem(row)
                }
                footer
            }
        }
    }

    private var loadingState: some View {
        VStack(spacing: 12) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 40)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "历届赛季")
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
        .padding(.top, 40)
    }

    @ViewBuilder
    private func historyItem(_ row: SeasonHistoryRow) -> some View {
        if row.hasRecord {
            NavigationLink(value: AppRoute.seasonReport(seasonId: row.id, gameType: model.currentGameType)) {
                rowContent(row)
            }
            .buttonStyle(.plain)
        } else {
            rowContent(row).opacity(0.72)
        }
    }

    private func rowContent(_ row: SeasonHistoryRow) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(row.name)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Spacer(minLength: 4)
                if !row.dateRange.isEmpty {
                    Text(row.dateRange)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
            }
            if row.hasRecord {
                HStack(spacing: 10) {
                    Text(row.rankText)
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.brandStrong)
                    Text("\(String(row.matches)) 场")
                    Text("\(String(row.wins)) 胜")
                    Text("胜率 \(String(row.winRate))%")
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            } else {
                Text("未参加该球种排位")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 14)
        .padding(.horizontal, 16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border)
        }
    }

    @ViewBuilder
    private var footer: some View {
        if let failure = model.refreshFailure {
            HStack(spacing: 10) {
                Text(failure.feedback(resource: "历届赛季").description)
                    .font(.system(size: 12))
                    .foregroundStyle(AppTheme.danger)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Button("重试") {
                    Task { await model.loadMore() }
                }
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.danger)
            }
            .padding(10)
            .background(AppTheme.danger.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
            .overlay {
                RoundedRectangle(cornerRadius: 10).stroke(AppTheme.danger)
            }
        } else if model.hasMore {
            Button {
                Task { await model.loadMore() }
            } label: {
                HStack(spacing: 6) {
                    if model.isLoadingMore {
                        ProgressView().controlSize(.small)
                    }
                    Text(model.isLoadingMore ? "加载中..." : "加载更多")
                        .font(.system(size: 13))
                }
                .foregroundStyle(palette.brandStrong)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
            }
            .buttonStyle(.plain)
            .disabled(model.isLoadingMore)
        } else {
            Text("没有更多了")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
        }
    }
}
