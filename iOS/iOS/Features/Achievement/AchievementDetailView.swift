import SwiftUI

/// 成就详情，对应小程序 `subPages/achievement/detail.vue`。
struct AchievementDetailView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: AchievementDetailModel

    init(session: SessionStore, achievementId: Int64, model: AchievementDetailModel? = nil) {
        _model = StateObject(wrappedValue: model ?? AchievementDetailModel(session: session, achievementId: achievementId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看成就详情",
                    actionTitle: "返回"
                ) {
                    dismiss()
                }
            } else if model.isInitialLoading {
                loadingState
            } else if model.isInitialError {
                errorState
            } else if let achievement = model.achievement {
                page(achievement)
            } else {
                VStack {
                    Spacer()
                    Text("成就不存在")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textMuted)
                    Spacer()
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("成就详情")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.loadIfNeeded() }
        }
    }

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    @ViewBuilder
    private var errorState: some View {
        let feedback = model.failureFeedback
            ?? LoadFailure(category: .business, message: "").feedback(resource: "成就详情")

        StateMessageView(
            icon: "info.circle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if model.dismissesOnError || !feedback.retryable {
                dismiss()
            } else {
                Task { await model.retry() }
            }
        }
    }

    private func page(_ achievement: AchievementDef) -> some View {
        ScrollView {
            VStack(spacing: 12) {
                hero(achievement)
                infoCard(achievement)
                progressCard(achievement)
                if achievement.unlocked, let unlockedAt = achievement.unlockedAt, !unlockedAt.isEmpty {
                    unlockCard(unlockedAt)
                }
            }
            .padding(.bottom, 32)
        }
    }

    private func hero(_ achievement: AchievementDef) -> some View {
        VStack(spacing: 10) {
            RemoteImage(url: CDNAsset.url(for: achievement.icon), contentMode: .fit) {
                Text(model.fallbackEmoji)
                    .font(.system(size: 54))
            }
            .frame(width: 104, height: 104)
            .background(palette.card, in: Circle())
            .overlay {
                Circle().stroke(palette.border)
            }

            Text(achievement.name)
                .font(.system(size: 20, weight: .bold))
                .foregroundStyle(palette.textPrimary)
                .multilineTextAlignment(.center)
                .fixedSize(horizontal: false, vertical: true)

            if achievement.unlocked {
                HStack(spacing: 4) {
                    Image(systemName: "checkmark")
                        .font(.system(size: 11, weight: .bold))
                    Text("已解锁")
                        .font(.system(size: 12, weight: .semibold))
                }
                .foregroundStyle(.white)
                .padding(.horizontal, 12)
                .padding(.vertical, 4)
                .background(AppTheme.primary, in: Capsule())
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 24)
        .background(
            LinearGradient(
                colors: [
                    achievement.unlocked ? AppTheme.primary.opacity(0.18) : palette.border,
                    palette.subtle,
                ],
                startPoint: .top,
                endPoint: .bottom
            )
        )
    }

    private func infoCard(_ achievement: AchievementDef) -> some View {
        VStack(spacing: 0) {
            infoRow(
                "解锁条件",
                achievement.description.isEmpty ? "完成指定目标解锁此成就" : achievement.description,
                isLast: false
            )
            infoRow("分类", model.categoryLabel, isLast: false)
            infoRow("球种", model.gameTypeLabel, isLast: achievement.rewardTitleName?.isEmpty ?? true)
            if let reward = achievement.rewardTitleName, !reward.isEmpty {
                infoRow("奖励称号", reward, isLast: true, highlighted: true)
            }
        }
        .appCard(padding: 0)
        .padding(.horizontal, 12)
    }

    private func infoRow(_ label: String, _ value: String, isLast: Bool, highlighted: Bool = false) -> some View {
        VStack(spacing: 0) {
            HStack(alignment: .top, spacing: 12) {
                Text(label)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                Spacer(minLength: 8)
                Text(value)
                    .font(.system(size: 14, weight: highlighted ? .bold : .regular))
                    .foregroundStyle(highlighted ? palette.brandStrong : palette.textPrimary)
                    .multilineTextAlignment(.trailing)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 13)

            if !isLast {
                Rectangle()
                    .fill(palette.border)
                    .frame(height: 1)
                    .padding(.horizontal, 16)
            }
        }
    }

    private func progressCard(_ achievement: AchievementDef) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("当前进度")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            progressBar(percent: model.progressPercent, complete: achievement.unlocked)
            HStack {
                Text(model.progressText)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                Spacer(minLength: 8)
                Text("\(String(model.progressPercent))%")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 16)
        .padding(.horizontal, 12)
    }

    private func progressBar(percent: Int, complete: Bool) -> some View {
        GeometryReader { proxy in
            ZStack(alignment: .leading) {
                Capsule().fill(palette.border)
                Capsule()
                    .fill(complete ? AppTheme.primary : palette.textMuted)
                    .frame(width: proxy.size.width * CGFloat(min(100, max(0, percent))) / 100)
            }
        }
        .frame(height: 10)
        .accessibilityLabel("进度 \(String(percent))%")
    }

    private func unlockCard(_ unlockedAt: String) -> some View {
        HStack(spacing: 8) {
            Image(systemName: "calendar")
                .font(.system(size: 16))
                .foregroundStyle(palette.brandStrong)
            Text("解锁于 \(unlockedAt)")
                .font(.system(size: 13))
                .foregroundStyle(palette.brandStrong)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 12)
    }
}
