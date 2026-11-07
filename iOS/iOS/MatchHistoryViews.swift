import SwiftUI

struct MatchReportRoute: Hashable {
    let matchId: Int64
}

private enum HistorySection: Hashable {
    case participant
    case referee
}

/// 对局记录：参赛与执裁两个分栏，各自持有独立的分页状态；战报为只读页面。
struct MatchHistoryView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var participant: PagedHistoryList<MatchListItem>
    @StateObject private var referee: PagedHistoryList<RefereeHistoryItem>
    @State private var section: HistorySection = .participant

    init(session: SessionStore) {
        let api = MatchHistoryAPI(session: session)
        _participant = StateObject(wrappedValue: PagedHistoryList(session: session) { page in
            try await api.participantPage(page: page)
        })
        _referee = StateObject(wrappedValue: PagedHistoryList(session: session) { page in
            try await api.refereePage(page: page)
        })
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                MatchHistoryStateMessage(icon: "lock.fill", title: "请登录后查看对局记录", detail: nil)
            } else {
                VStack(spacing: 0) {
                    Picker("记录类型", selection: $section) {
                        Text("参赛").tag(HistorySection.participant)
                        Text("执裁").tag(HistorySection.referee)
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal, 20)
                    .padding(.vertical, 12)

                    switch section {
                    case .participant:
                        PagedHistoryListContent(
                            list: participant,
                            emptyTitle: "暂无对局记录",
                            emptyHint: "完成一场对局后会出现在这里"
                        ) { item in
                            NavigationLink(value: MatchReportRoute(matchId: item.id)) {
                                ParticipantHistoryRow(item: item)
                            }
                        }
                    case .referee:
                        PagedHistoryListContent(
                            list: referee,
                            emptyTitle: "暂无执裁记录",
                            emptyHint: "担任裁判并完成结算后会出现在这里"
                        ) { item in
                            // 已取消的执裁记录没有可读战报，只展示状态。
                            if item.status == 2 {
                                NavigationLink(value: MatchReportRoute(matchId: item.id)) {
                                    RefereeHistoryRow(item: item, isNavigable: true)
                                }
                            } else {
                                RefereeHistoryRow(item: item, isNavigable: false)
                            }
                        }
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("对局记录")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: MatchReportRoute.self) { route in
            MatchReportView(session: session, matchId: route.matchId)
        }
        .onChange(of: session.identity) { _, _ in
            // 退出、切号或会话失效后清空两个分栏，旧账号的在途结果不再采纳。
            participant.syncIdentity()
            referee.syncIdentity()
        }
    }
}

private struct PagedHistoryListContent<Item: Identifiable, Row: View>: View where Item.ID: Hashable {
    @ObservedObject var list: PagedHistoryList<Item>
    let emptyTitle: String
    let emptyHint: String
    @ViewBuilder let row: (Item) -> Row

    @Environment(\.colorScheme) private var colorScheme

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        content
            .task { await list.loadIfNeeded() }
    }

    @ViewBuilder
    private var content: some View {
        if list.items.isEmpty {
            switch list.phase {
            case .failed(let message):
                MatchHistoryStateMessage(
                    icon: "exclamationmark.triangle",
                    title: "加载失败",
                    detail: message,
                    actionTitle: "重试",
                    action: { Task { await list.retry() } }
                )
            case .empty:
                MatchHistoryStateMessage(icon: "list.bullet", title: emptyTitle, detail: emptyHint)
            case .idle, .loading, .loaded:
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        } else {
            List {
                if let refreshError = list.refreshError {
                    HStack {
                        Text(refreshError)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Spacer()
                        Button("重试") { Task { await list.refresh() } }
                            .font(.system(size: 13, weight: .semibold))
                    }
                    .listRowBackground(Color.clear)
                    .listRowSeparator(.hidden)
                }

                ForEach(list.items) { item in
                    row(item)
                        .listRowBackground(Color.clear)
                        .listRowSeparator(.hidden)
                        .onAppear {
                            // 滚动到最后一条时请求下一页；失败与“没有更多”由页脚表达。
                            if item.id == list.items.last?.id {
                                Task { await list.loadMore() }
                            }
                        }
                }

                footer
                    .listRowBackground(Color.clear)
                    .listRowSeparator(.hidden)
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .refreshable { await list.refresh() }
        }
    }

    @ViewBuilder
    private var footer: some View {
        if list.isLoadingMore {
            ProgressView()
                .frame(maxWidth: .infinity)
        } else if let loadMoreError = list.loadMoreError {
            VStack(spacing: 8) {
                Text(loadMoreError)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                Button("重试") { Task { await list.loadMore() } }
                    .font(.system(size: 13, weight: .semibold))
            }
            .frame(maxWidth: .infinity)
        } else if list.loadedPage > 0, !list.hasMore {
            Text("没有更多了")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
        }
    }
}

private struct ParticipantHistoryRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let item: MatchListItem

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let outcome = MatchOutcome(result: item.result)
        let timeText = MatchTimeFormatting.relative(item.matchTime)

        HStack(spacing: 12) {
            MatchHistoryAvatar(urlString: item.opponentAvatar, size: 44)
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(item.opponentName)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    GameTypeTag(text: item.gameTypeName)
                }
                Text(timeText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                OutcomeBadge(outcome: outcome)
                Text("\(item.myScore) - \(item.opponentScore)")
                    .font(.system(size: 15, weight: .semibold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
            }
            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
        }
        .padding(16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "对手\(item.opponentName)，\(item.gameTypeName)，\(timeText)，\(outcome?.title ?? "结果未知")，比分\(item.myScore)比\(item.opponentScore)"
        )
    }
}

private struct RefereeHistoryRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let item: RefereeHistoryItem
    let isNavigable: Bool

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let isCancelled = item.status == 3
        let statusText = isCancelled ? "已取消" : "已完成"
        let timeText = MatchTimeFormatting.relative(item.endTime)
        let matchup = "\(item.player1Name) VS \(item.player2Name)"

        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(matchup)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    GameTypeTag(text: item.gameTypeName)
                }
                Text(timeText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                Text(statusText)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                Text("\(item.player1Score) - \(item.player2Score)")
                    .font(.system(size: 15, weight: .semibold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
            }
            if isNavigable {
                Image(systemName: "chevron.right")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
            }
        }
        .padding(16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .opacity(isCancelled ? 0.6 : 1)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(matchup)，\(item.gameTypeName)，\(timeText)，\(statusText)，比分\(item.player1Score)比\(item.player2Score)")
    }
}

private struct OutcomeBadge: View {
    @Environment(\.colorScheme) private var colorScheme
    let outcome: MatchOutcome?

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let isWin = outcome == .win
        Text(outcome?.title ?? "结果未知")
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(isWin ? palette.textPrimary : palette.textSecondary)
            .padding(.horizontal, 10)
            .padding(.vertical, 4)
            .background(
                isWin ? AppTheme.primary.opacity(0.22) : palette.subtle,
                in: Capsule()
            )
    }
}

private struct GameTypeTag: View {
    @Environment(\.colorScheme) private var colorScheme
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 12))
            .foregroundStyle(AppTheme.palette(for: colorScheme).textSecondary)
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .background(AppTheme.palette(for: colorScheme).subtle, in: RoundedRectangle(cornerRadius: 6))
    }
}

/// 头像加载失败、地址为空或无效时，底层占位图保持可见。
struct MatchHistoryAvatar: View {
    let urlString: String
    let size: CGFloat

    var body: some View {
        ZStack {
            Circle().fill(AppTheme.primary.opacity(0.14))
            Image(systemName: "person.crop.circle.fill")
                .resizable()
                .scaledToFit()
                .foregroundStyle(AppTheme.primary)
                .padding(size * 0.12)
            if !urlString.isEmpty, let url = URL(string: urlString) {
                AsyncImage(url: url) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Color.clear
                }
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
        .accessibilityHidden(true)
    }
}

struct MatchHistoryStateMessage: View {
    @Environment(\.colorScheme) private var colorScheme
    let icon: String
    let title: String
    let detail: String?
    var actionTitle: String?
    var action: (() -> Void)?

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundStyle(palette.textSecondary)
                .accessibilityHidden(true)
            Text(title)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            if let detail {
                Text(detail)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
            if let actionTitle, let action {
                Button(action: action) {
                    Text(actionTitle)
                        .font(.system(size: 15, weight: .semibold))
                        .padding(.horizontal, 24)
                        .padding(.vertical, 10)
                        .foregroundStyle(.white)
                        .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 10))
                }
                .padding(.top, 4)
            }
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
