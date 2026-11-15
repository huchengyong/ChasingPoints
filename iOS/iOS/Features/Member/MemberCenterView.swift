import SwiftUI

/// VIP 中心，对应小程序 `subPages/user/memberCenter.vue`。
/// 合规模式下只展示获赠权益与成长，不出现订阅、支付或下单入口。
struct MemberCenterView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: MemberCenterModel

    init(session: SessionStore, model: MemberCenterModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MemberCenterModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var ownUserId: Int64 { session.identity?.userId ?? 0 }
    private var ownAvatar: String { session.user?.avatar ?? "" }
    private var ownName: String {
        let nickname = session.user?.nickname.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return nickname.isEmpty ? "我" : nickname
    }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看VIP中心",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("VIP中心")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.onIdentityChanged() }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if let message = model.overview.failureMessage, model.overview.value == nil {
                    loadFailureBanner(message)
                }
                heroCard
                identityCard
                analysisCard
                growthCard
                tipsCard
            }
            .padding(.horizontal, 12)
            .padding(.top, 12)
            .padding(.bottom, 12)
        }
        .refreshable { await model.refresh() }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            actionBar
        }
    }

    // MARK: - 头部

    private var heroCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(AppFlags.complianceMode ? "VIP权益" : "VIP中心")
                    .font(.system(size: 11, weight: .semibold))
                    .tracking(1)
                    .foregroundStyle(palette.brandStrong)
                Spacer(minLength: 8)
                Text(model.summary.statusText)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 3)
                    .background(palette.brandTint, in: Capsule())
            }
            Text(model.summary.title)
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text(model.summary.description)
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 18)
    }

    private func loadFailureBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(AppTheme.danger)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.overview.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(AppTheme.danger)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(AppTheme.danger.opacity(0.1), in: RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - 身份展示

    private var identityCard: some View {
        let preview = model.identityPreview
        return VStack(alignment: .leading, spacing: 12) {
            sectionHead("身份展示", tip: preview.active ? "\(preview.identity.tierLabel)外观" : nil)

            HStack(alignment: .center, spacing: 14) {
                MemberAvatarView(
                    avatar: ownAvatar,
                    userId: ownUserId,
                    vipLevel: preview.identity.vipLevel,
                    rankLevel: model.rankLevel,
                    size: 112,
                    reserveFrame: true
                )
                VStack(alignment: .leading, spacing: 6) {
                    Text(ownName)
                        .font(.system(size: 17, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(preview.hint)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }

            VStack(spacing: 6) {
                ForEach(preview.tiers) { tier in
                    tierRow(tier, active: preview.active)
                }
            }

            Text("成长等级解锁更高阶外观。外观只是展示身份，不影响排位分，也不影响对局功能。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func tierRow(_ tier: MemberTierPreview, active: Bool) -> some View {
        let highlighted = tier.current || (tier.reached && active)
        return HStack(spacing: 10) {
            MemberAvatarView(
                avatar: ownAvatar,
                userId: ownUserId,
                vipLevel: tier.sampleLevel,
                size: 64
            )
            VStack(alignment: .leading, spacing: 2) {
                Text(tier.label)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text("成长 \(tier.levelText)")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            if tier.current {
                Text("当前")
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(palette.brandTint, in: Capsule())
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(highlighted ? palette.brandTint : palette.subtle, in: RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - 深度分析

    private var analysisCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHead("深度分析", tip: "会员权益")
            analysisRow(
                title: "表现分析",
                description: "按对手实力反推近 90 天真实水平，附 6 个月实力曲线"
            ) {
                navigator.open(.performance(gameType: model.currentGameType))
            }
            Divider().overlay(palette.border)
            analysisRow(
                title: "单场复盘",
                description: "在对局记录里打开已完成的对局，回看每局走势与关键局"
            ) {
                navigator.open(.matchHistory)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func analysisRow(title: String, description: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 10) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(title)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text(description)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .multilineTextAlignment(.leading)
                        .fixedSize(horizontal: false, vertical: true)
                }
                Spacer(minLength: 8)
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    // MARK: - 成长

    private var growthCard: some View {
        let card = model.growthCard
        return VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("VIP成长")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text(card.statusText)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 8)
                Text(card.levelLabel)
                    .font(.system(size: 18, weight: .bold))
                    .foregroundStyle(palette.brandStrong)
            }

            HStack(spacing: 8) {
                growthMetric("当前进度", value: card.growthPointsText)
                growthMetric("今日成长", value: card.todayProgressText)
            }

            VStack(alignment: .leading, spacing: 6) {
                GeometryReader { proxy in
                    ZStack(alignment: .leading) {
                        Capsule().fill(palette.border)
                        Capsule()
                            .fill(AppTheme.primary)
                            .frame(width: proxy.size.width * CGFloat(min(100, max(0, card.progressPercent))) / 100)
                    }
                }
                .frame(height: 6)
                .accessibilityHidden(true)
                Text(card.nextLevelText)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }

            Text(card.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func growthMetric(_ label: String, value: String) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
            Text(value)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
    }

    // MARK: - 说明与操作

    private var tipsCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            sectionHead(AppFlags.complianceMode ? "权益说明" : "订阅说明", tip: nil)
            ForEach(tips, id: \.self) { tip in
                Text(tip)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var tips: [String] {
        var list: [String] = []
        if AppFlags.complianceMode {
            list.append("VIP权益由活动发放，到账后会自动同步到这里，无需订阅或支付。")
            if model.venueRewardClaimable {
                list.append("补充常玩球馆并审核通过后，可获得 \(model.venueRewardText)，无需订阅或支付。")
            }
        } else {
            list.append("支付成功后会自动更新VIP状态。")
            list.append("如果你当前VIP仍在有效期内，续费会在现有到期时间基础上顺延。")
        }
        list.append("VIP成长只统计真实完赛对局，每天最多计入 \(String(model.growthCard.dailyCap)) 场。")
        list.append("VIP到期后成长会冻结但保留，续开VIP后会从原进度继续成长。")
        list.append("VIP不影响排位分：排位分只看胜负、对手实力与赛制长度，会员与普通用户规则相同；高光仍会记录在战报和成就里。")
        list.append("对局的基本流程对所有用户完整开放，VIP只提供身份展示与深度分析。")
        list.append("所有展示时间统一按 UTC+8 显示。")
        return list
    }

    @ViewBuilder
    private var actionBar: some View {
        if AppFlags.complianceMode {
            Button {} label: {
                Text(model.summary.primaryActionText)
            }
            .buttonStyle(.appPrimary)
            .disabled(true)
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background(palette.page)
        }
    }

    private func sectionHead(_ title: String, tip: String?) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 8)
            if let tip {
                Text(tip)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
        }
    }
}
