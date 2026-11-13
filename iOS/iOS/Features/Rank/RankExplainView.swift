import SwiftUI

/// 段位说明：当前段位英雄卡、段位之路与计分规则。
struct RankExplainView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: RankExplainModel
    @State private var isRuleDetailsOpen = false

    init(session: SessionStore, gameType: Int, model: RankExplainModel? = nil) {
        _model = StateObject(wrappedValue: model ?? RankExplainModel(session: session, gameType: gameType))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            gameTypeTabs
            content
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("段位说明")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.reloadForIdentityChange() }
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
    }

    private var gameTypeTabs: some View {
        HStack(spacing: 0) {
            ForEach(GameTypeCatalog.ordered) { option in
                gameTypeTab(option)
            }
        }
        .padding(4)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
        .overlay {
            RoundedRectangle(cornerRadius: 14).stroke(palette.border)
        }
        .padding(.horizontal, 12)
        .padding(.top, 12)
        .padding(.bottom, 4)
    }

    private func gameTypeTab(_ option: GameTypeOption) -> some View {
        let isActive = model.currentGameType == option.value
        let isPending = model.isRefreshing && isActive

        return Button {
            Task { await model.selectGameType(option.value) }
        } label: {
            Text(option.label)
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(isActive ? .white : palette.textSecondary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
                .frame(maxWidth: .infinity)
                .frame(height: 34)
                .background {
                    if isActive {
                        RoundedRectangle(cornerRadius: 10)
                            .fill(
                                LinearGradient(
                                    colors: [palette.gradientStart, palette.gradientEnd],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                )
                            )
                    }
                }
        }
        .buttonStyle(.plain)
        .opacity(isPending ? 0.72 : 1)
        .accessibilityAddTraits(isActive ? .isSelected : [])
    }

    @ViewBuilder
    private var content: some View {
        if session.identity == nil {
            StateMessageView(icon: "lock.fill", title: "登录后查看段位", actionTitle: "去登录") {
                navigator.openLogin()
            }
        } else if model.isInitialLoading {
            loadingState
        } else if model.isInitialError {
            errorState
        } else if let hero = model.heroModel {
            ScrollView {
                VStack(spacing: 12) {
                    heroCard(hero)
                    growthSection
                    rulesSection
                }
                .padding(.horizontal, 12)
                .padding(.top, 6)
                .padding(.bottom, 24)
            }
        } else {
            loadingState
        }
    }

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var errorState: some View {
        VStack(spacing: 0) {
            ZStack {
                Circle()
                    .fill(palette.brandTint)
                    .frame(width: 56, height: 56)
                Image(systemName: "info.circle.fill")
                    .font(.system(size: 14))
                    .foregroundStyle(RankHeroPalette.warningText)
            }
            Text("段位数据加载失败")
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
                .padding(.top, 14)
            Text("请检查网络连接后重新加载")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .padding(.top, 6)
            Button {
                Task { await model.retry() }
            } label: {
                Text("重新加载")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 126, height: 40)
                    .background(
                        LinearGradient(
                            colors: [palette.gradientStart, palette.gradientEnd],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: Capsule()
                    )
            }
            .buttonStyle(.plain)
            .padding(.top, 15)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func heroCard(_ hero: RankHeroModel) -> some View {
        ZStack(alignment: .topTrailing) {
            LinearGradient(
                colors: RankHeroPalette.gradient(level: hero.level),
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            Circle()
                .fill(RankHeroPalette.accent(level: hero.level))
                .frame(width: 180, height: 180)
                .blur(radius: 46)
                .offset(x: 48, y: -54)

            Circle()
                .stroke(Color.white.opacity(0.1), lineWidth: 1)
                .frame(width: 195, height: 195)
                .offset(x: 63, y: -72)

            VStack(alignment: .leading, spacing: 13) {
                HStack(alignment: .center, spacing: 6) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(verbatim: "\(model.currentGameTypeLabel) · 当前段位")
                            .font(.system(size: 10, weight: .semibold))
                            .tracking(1)
                            .foregroundStyle(Color.white.opacity(0.7))
                        Text(hero.name)
                            .font(.system(size: 24, weight: .bold))
                            .foregroundStyle(.white)
                        HStack(alignment: .firstTextBaseline, spacing: 4) {
                            Text(String(hero.rankScore))
                                .font(.system(size: 18, weight: .bold))
                                .foregroundStyle(.white)
                            Text("排位分")
                                .font(.system(size: 12))
                                .foregroundStyle(Color.white.opacity(0.82))
                        }
                        if hero.placementRemaining > 0 {
                            Text(verbatim: "定级赛 · 还剩 \(String(hero.placementRemaining)) 场")
                                .font(.system(size: 11, weight: .semibold))
                                .foregroundStyle(RankHeroPalette.placementText)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(RankHeroPalette.placementText.opacity(0.14), in: Capsule())
                                .overlay {
                                    Capsule().stroke(RankHeroPalette.placementText.opacity(0.4))
                                }
                                .padding(.top, 3)
                        }
                    }

                    Spacer(minLength: 6)

                    RemoteImage(url: CDNAsset.url(for: hero.icon), contentMode: .fit) {
                        Color.clear
                    }
                    .frame(width: 104, height: 104)
                }

                if hero.isMax {
                    progressBlock(
                        title: "最高段位成就",
                        value: "MAX",
                        percent: 100,
                        hint: "已解锁全部段位成长节点",
                        pill: "已达最高段位"
                    )
                } else {
                    progressBlock(
                        title: "晋级\(hero.nextName ?? "")",
                        value: "\(String(hero.earned ?? 0)) / \(String(hero.span ?? 0))",
                        percent: hero.progressPercent,
                        hint: "本段位进度 \(String(hero.progressPercent))%",
                        pill: "还差 \(String(hero.remaining ?? 0)) 分"
                    )
                }
            }
            .padding(17)
        }
        .clipShape(RoundedRectangle(cornerRadius: 19))
        .shadow(color: .black.opacity(0.18), radius: 16, y: 10)
    }

    private func progressBlock(
        title: String,
        value: String,
        percent: Int,
        hint: String,
        pill: String
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Rectangle()
                .fill(Color.white.opacity(0.14))
                .frame(height: 1)

            HStack(spacing: 8) {
                Text(title)
                    .font(.system(size: 11))
                    .foregroundStyle(Color.white.opacity(0.78))
                Spacer(minLength: 4)
                Text(value)
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.white)
            }

            GeometryReader { proxy in
                Capsule()
                    .fill(Color.white.opacity(0.16))
                    .overlay(alignment: .leading) {
                        Capsule()
                            .fill(
                                LinearGradient(
                                    colors: [RankHeroPalette.progressStart, .white],
                                    startPoint: .leading,
                                    endPoint: .trailing
                                )
                            )
                            .frame(width: proxy.size.width * CGFloat(percent) / 100)
                    }
            }
            .frame(height: 7)

            HStack(spacing: 8) {
                Text(hint)
                    .font(.system(size: 10))
                    .foregroundStyle(Color.white.opacity(0.72))
                Spacer(minLength: 4)
                Text(pill)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(RankHeroPalette.placementText)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 4)
                    .background(Color.white.opacity(0.1), in: Capsule())
                    .overlay {
                        Capsule().stroke(Color.white.opacity(0.16))
                    }
            }
        }
        .padding(.top, 11)
    }

    private var growthSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .bottom, spacing: 8) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("段位之路")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text("达到对应排位分即可晋级")
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 8)
                HStack(spacing: 5) {
                    Circle()
                        .fill(AppTheme.primary)
                        .frame(width: 7, height: 7)
                        .shadow(color: AppTheme.primary.opacity(0.14), radius: 3)
                    Text("我的段位")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                }
            }

            VStack(spacing: 0) {
                ForEach(model.growthPath) { step in
                    growthStep(step)
                }
            }
            .background(alignment: .leading) {
                Rectangle()
                    .fill(palette.border)
                    .frame(width: 2)
                    .padding(.leading, 14)
                    .padding(.vertical, 25)
            }
        }
        .padding(15)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 17))
        .overlay {
            RoundedRectangle(cornerRadius: 17).stroke(palette.border)
        }
    }

    private func growthStep(_ step: RankGrowthStep) -> some View {
        let isCurrent = step.state == .current
        let isReached = step.state == .reached

        return HStack(spacing: 9) {
            ZStack {
                Circle()
                    .fill(isCurrent || isReached ? AppTheme.primary : palette.card)
                    .frame(width: isCurrent ? 11 : 8, height: isCurrent ? 11 : 8)
                    .overlay {
                        Circle()
                            .stroke(isCurrent || isReached ? AppTheme.primary : palette.border, lineWidth: 2)
                    }
                if isReached {
                    Image(systemName: "checkmark")
                        .font(.system(size: 5, weight: .bold))
                        .foregroundStyle(.white)
                }
            }
            .frame(width: 14)

            RemoteImage(url: CDNAsset.url(for: step.icon), contentMode: .fit) {
                Color.clear
            }
            .frame(width: 42, height: 42)
            .opacity(isCurrent || isReached ? 1 : 0.58)

            VStack(alignment: .leading, spacing: 3) {
                Text(step.name)
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(isDark ? palette.textPrimary : palette.textSecondary)
                Text(step.rangeText)
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
            }

            Spacer(minLength: 4)

            Text(step.stateText)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(isCurrent ? palette.brandStrong : palette.textMuted)
                .frame(minWidth: 40, alignment: .trailing)
        }
        .padding(.horizontal, 7)
        .padding(.vertical, 6)
        .frame(minHeight: 52)
        .background(isCurrent ? palette.brandTint : .clear, in: RoundedRectangle(cornerRadius: 13))
        .overlay {
            if isCurrent {
                RoundedRectangle(cornerRadius: 13).stroke(palette.brandStrong.opacity(0.32))
            }
        }
    }

    private var rulesSection: some View {
        let rules = model.rankRules

        return VStack(alignment: .leading, spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                Text("段位分怎么算？")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text("先看关键规则，再按需查看细节")
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
            }

            if !rules.notice.isEmpty {
                Text(rules.notice)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .lineSpacing(4)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 9)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
                    .overlay {
                        RoundedRectangle(cornerRadius: 9).stroke(palette.brandStrong.opacity(0.36))
                    }
            }

            HStack(spacing: 7) {
                ForEach(Array(rules.summaryStats.enumerated()), id: \.offset) { _, stat in
                    VStack(spacing: 3) {
                        Text(stat.value)
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(palette.brandStrong)
                        Text(stat.label)
                            .font(.system(size: 9))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                            .minimumScaleFactor(0.8)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .padding(.horizontal, 2)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 11))
                }
            }

            HStack(spacing: 0) {
                Rectangle()
                    .fill(AppTheme.primary)
                    .frame(width: 3)
                Text(rules.note)
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textSecondary)
                    .lineSpacing(4)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 9)
            }
            .background(palette.brandTint)

            Button {
                isRuleDetailsOpen.toggle()
            } label: {
                HStack(spacing: 5) {
                    Text(isRuleDetailsOpen ? "收起完整计分规则" : "查看完整计分规则")
                        .font(.system(size: 11, weight: .bold))
                    Image(systemName: isRuleDetailsOpen ? "chevron.up" : "chevron.down")
                        .font(.system(size: 10, weight: .semibold))
                }
                .foregroundStyle(palette.brandStrong)
                .frame(maxWidth: .infinity)
                .frame(height: 38)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 11))
                .overlay {
                    RoundedRectangle(cornerRadius: 11)
                        .stroke(palette.brandStrong.opacity(0.28))
                }
            }
            .buttonStyle(.plain)
            .accessibilityValue(isRuleDetailsOpen ? "已展开" : "已收起")

            if isRuleDetailsOpen {
                VStack(spacing: 0) {
                    ForEach(Array(rules.detailRows.enumerated()), id: \.offset) { _, row in
                        HStack(alignment: .top, spacing: 10) {
                            Text(row.title)
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(palette.textSecondary)
                                .frame(width: 71, alignment: .leading)
                            Text(row.description)
                                .font(.system(size: 10))
                                .foregroundStyle(palette.textSecondary)
                                .lineSpacing(4)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .padding(.vertical, 9)
                        .overlay(alignment: .bottom) {
                            Rectangle().fill(palette.border).frame(height: 1)
                        }
                    }

                    VStack(alignment: .leading, spacing: 5) {
                        Text(rules.example.title)
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(RankHeroPalette.exampleTitle)
                        Text(rules.example.formula)
                            .font(.system(size: 11))
                            .foregroundStyle(.white)
                            .lineSpacing(4)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(11)
                    .background(
                        LinearGradient(
                            colors: [RankHeroPalette.exampleStart, RankHeroPalette.exampleEnd],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: RoundedRectangle(cornerRadius: 11)
                    )
                    .padding(.top, 10)
                }
                .padding(.top, 12)
                .overlay(alignment: .top) {
                    Rectangle().fill(palette.border).frame(height: 1)
                }
            }
        }
        .padding(15)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 17))
        .overlay {
            RoundedRectangle(cornerRadius: 17).stroke(palette.border)
        }
    }

    private var isDark: Bool { colorScheme == .dark }
}

/// 段位英雄卡随段位变化的渐变与深色文字色，取值照搬 `subPages/user/rankExplain.scss`。
private enum RankHeroPalette {
    static let placementText = Color(hex: 0xFFF1BD)
    static let progressStart = Color(hex: 0xFFE28A)
    static let warningText = Color(hex: 0xB77908)
    static let exampleTitle = Color(hex: 0xFFE49A)
    static let exampleStart = Color(hex: 0x2B230F)
    static let exampleEnd = Color(hex: 0x664909)

    static func gradient(level: Int) -> [Color] {
        switch level {
        case 1: [Color(hex: 0x17140D), Color(hex: 0x4A291A), Color(hex: 0x8A4D2A)]
        case 2: [Color(hex: 0x111827), Color(hex: 0x243A55), Color(hex: 0x3D6485)]
        case 3: [Color(hex: 0x17140D), Color(hex: 0x4C3608), Color(hex: 0x8F6507)]
        case 4: [Color(hex: 0x101827), Color(hex: 0x153963), Color(hex: 0x236998)]
        case 5: [Color(hex: 0x141327), Color(hex: 0x24234F), Color(hex: 0x433B82)]
        default: [Color(hex: 0x1B120B), Color(hex: 0x4C230C), Color(hex: 0x8F5207)]
        }
    }

    static func accent(level: Int) -> Color {
        switch level {
        case 1: Color(hex: 0xDE8F5B, opacity: 0.3)
        case 2: Color(hex: 0xCFE6FF, opacity: 0.28)
        case 3: Color(hex: 0xFFD75B, opacity: 0.3)
        case 4: Color(hex: 0x9EDCFF, opacity: 0.28)
        case 5: Color(hex: 0xAEB9FF, opacity: 0.28)
        default: Color(hex: 0xFFCC66, opacity: 0.32)
        }
    }
}
