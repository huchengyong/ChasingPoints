import SwiftUI

/// PK 报表，对应小程序 `subPages/social/pkReport.vue`。
struct PKReportView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: PKReportModel
    @State private var posterImage: UIImage?
    @State private var isGeneratingPoster = false

    init(session: SessionStore, target: FriendHomepageTarget, model: PKReportModel? = nil) {
        _model = StateObject(wrappedValue: model ?? PKReportModel(session: session, target: target))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.phase {
            case .loading:
                loadingState
            case .failed:
                failureState
            case .ready, .empty:
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("PK报表")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await model.onAppear()
            await generatePoster()
        }
        .onChange(of: session.identity) { _, _ in
            Task {
                await model.onAppear()
                await generatePoster()
            }
        }
    }

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("正在生成 PK 报表...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var failureState: some View {
        let feedback = model.failureFeedback ?? LoadFailure(category: .business, message: "")
            .feedback(resource: "PK 报表")

        return StateMessageView(
            icon: "exclamationmark.triangle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task {
                    await model.retry()
                    await generatePoster()
                }
            } else {
                dismiss()
            }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if let refreshError = model.refreshError {
                    refreshBanner(refreshError)
                }
                heroCard
                evidenceCard
                posterCard
                actionsCard
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .refreshable {
            await model.retry()
            await generatePoster()
        }
    }

    private func refreshBanner(_ failure: LoadFailure) -> some View {
        HStack(spacing: 8) {
            Text(failure.feedback(resource: "PK 报表").description)
                .font(.system(size: 12))
                .foregroundStyle(AppTheme.danger)
                .frame(maxWidth: .infinity, alignment: .leading)
            Button("重试") {
                Task {
                    await model.retry()
                    await generatePoster()
                }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(AppTheme.danger)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(AppTheme.danger.opacity(0.1), in: RoundedRectangle(cornerRadius: 10))
    }

    private var heroCard: some View {
        let hero = model.hero

        return VStack(alignment: .leading, spacing: 14) {
            Text("真实线下交锋")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(palette.brandStrong)
            Text(hero.title)
                .font(.system(size: 21, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text(hero.metaText)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)

            HStack(spacing: 10) {
                playerCard(name: "你", avatar: model.myAvatar, userId: session.user?.id ?? 0)
                VStack(spacing: 4) {
                    Text("VS")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(palette.textMuted)
                    Text("PK 报表")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
                playerCard(
                    name: model.opponentName,
                    avatar: model.opponentAvatar,
                    userId: model.opponent?.id ?? model.target.userId
                )
            }
            .frame(maxWidth: .infinity)

            VStack(spacing: 4) {
                Text(hero.scoreText)
                    .font(.system(size: 32, weight: .heavy).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
                Text("历史交锋胜场")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func playerCard(name: String, avatar: String, userId: Int64) -> some View {
        VStack(spacing: 6) {
            AvatarView(avatar: avatar, userId: userId, size: 56)
            Text(name)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
    }

    private var evidenceCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader("核心证据", tip: "一眼看懂这份 PK 报表为什么成立")

            if model.evidenceList.isEmpty {
                Text("还没有足够的交锋样本，先约一场见真章。")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.vertical, 12)
            } else {
                ForEach(Array(model.evidenceList.enumerated()), id: \.offset) { index, item in
                    HStack(alignment: .top, spacing: 10) {
                        Text("0\(String(index + 1))")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                        Text(item)
                            .font(.system(size: 14, weight: .medium))
                            .foregroundStyle(palette.textPrimary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                }
            }

            if let summary = model.latestMatchSummary {
                VStack(alignment: .leading, spacing: 4) {
                    Text("最近一次交锋")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                    Text(summary)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textPrimary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(12)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var posterCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader("分享海报", tip: "可保存到相册后发给好友")

            if let posterImage {
                Image(uiImage: posterImage)
                    .resizable()
                    .scaledToFit()
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            } else {
                Text(isGeneratingPoster ? "海报生成中..." : model.posterEmptyText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.vertical, 12)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var actionsCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader("下一步", tip: "先保存结论，再决定要不要继续约战")

            if let posterImage {
                ShareLink(
                    item: Image(uiImage: posterImage),
                    preview: SharePreview("PK 报表海报", image: Image(uiImage: posterImage))
                ) {
                    Text("保存海报")
                }
                .buttonStyle(.appPrimary)
            } else {
                Button(isGeneratingPoster ? "生成海报中" : "保存海报") {
                    Task { await savePoster() }
                }
                .buttonStyle(.appPrimary)
            }

            Button("复制分享文案") { copyShareSummary() }
                .buttonStyle(.appSecondary)
            Button("查看完整交锋") { goToH2H() }
                .buttonStyle(.appSecondary)
            Button("发 PK 邀约") { openPkInvite() }
                .buttonStyle(.appSecondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func sectionHeader(_ title: String, tip: String) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(tip)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    // MARK: - 海报与操作

    @MainActor
    private func generatePoster() async {
        guard model.phase == .ready, model.totalMatches > 0 else {
            posterImage = nil
            return
        }
        guard posterImage == nil else { return }
        isGeneratingPoster = true
        await Task.yield()
        let qr = QRCodeImage.make(content: EntryLink.base, size: 108)
        let renderer = ImageRenderer(content: PKReportPoster(payload: model.posterPayload, qrImage: qr))
        renderer.scale = 1
        posterImage = renderer.uiImage
        isGeneratingPoster = false
    }

    private func savePoster() async {
        guard model.totalMatches > 0 else {
            toast.show("至少完成一场交锋后再生成海报")
            return
        }
        guard !isGeneratingPoster else {
            toast.show("海报生成中，请稍后")
            return
        }
        if posterImage == nil {
            await generatePoster()
        }
        toast.show(posterImage == nil ? "海报生成失败" : "海报已生成，请再次点击保存")
    }

    private func copyShareSummary() {
        let content = model.shareSummary
        guard !content.isEmpty else {
            toast.show("还没有足够的交锋数据")
            return
        }
        UIPasteboard.general.string = content
        toast.show("已复制分享文案")
    }

    private func goToH2H() {
        navigator.open(.h2hRecord(
            opponentId: model.opponent?.id ?? model.target.userId,
            targetUserId: 0,
            targetName: "",
            targetAvatar: "",
            opponentName: model.opponentName
        ))
    }

    private func openPkInvite() {
        navigator.open(.challenges(tab: .received))
        toast.show("可在约球页继续发起邀约")
    }
}
