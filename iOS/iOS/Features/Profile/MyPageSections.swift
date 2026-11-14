import SwiftUI

// 「我的」页的分段视图，对应 uniapp/pages/user/index.vue 的模板结构与 index.scss 的尺寸。

// MARK: - 资料头

struct MyProfileHeaderSection: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme

    @ObservedObject var model: MyPageModel
    let onShowQRCode: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        let user = session.user
        let nickname = (user?.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)

        HStack(spacing: 10) {
            HStack(spacing: 10) {
                Button {
                    navigator.open(.editProfile)
                } label: {
                    MemberAvatarView(
                        avatar: user?.avatar ?? "",
                        userId: user?.id ?? 0,
                        vipLevel: model.memberIdentity.vipLevel,
                        rankLevel: model.ranks.value?[model.currentRankGameType]?.level ?? 0,
                        size: 56,
                        reserveFrame: true,
                        showBadge: false
                    )
                }
                .buttonStyle(.plain)
                .accessibilityLabel("编辑资料")

                VStack(alignment: .leading, spacing: 6) {
                    Button {
                        navigator.open(.editProfile)
                    } label: {
                        Text(nickname.isEmpty ? "用户" : nickname)
                            .font(.system(size: 19, weight: .heavy))
                            .lineLimit(1)
                            .foregroundStyle(palette.textPrimary)
                    }
                    .buttonStyle(.plain)

                    chips
                }
            }

            Spacer(minLength: 8)

            HStack(spacing: 6) {
                circularButton(symbol: "qrcode", label: "出示我的二维码") { onShowQRCode() }
                circularButton(symbol: "gearshape", label: "设置") { navigator.open(.settings) }
            }
        }
        .padding(.horizontal, 2)
        .padding(.bottom, 14)
    }

    private var chips: some View {
        HStack(spacing: 5) {
            Button {
                navigator.open(.memberCenter)
            } label: {
                chipLabel(model.profileLevelText, vip: model.memberIdentity.showBadge)
            }
            .buttonStyle(.plain)

            Button {
                navigator.open(.reputation)
            } label: {
                chipLabel(model.reputationChip.text, restricted: model.reputationChip.restricted)
            }
            .buttonStyle(.plain)
        }
    }

    private func chipLabel(_ text: String, vip: Bool = false, restricted: Bool = false) -> some View {
        Text(text)
            .font(.system(size: 11, weight: .semibold))
            .lineLimit(1)
            .foregroundStyle(restricted ? AppTheme.danger : (vip ? palette.brandStrong : palette.textSecondary))
            .padding(.horizontal, 8)
            .frame(height: 20)
            .background(
                restricted
                    ? AppTheme.danger.opacity(0.12)
                    : (vip ? palette.brandTint : palette.card),
                in: Capsule()
            )
            .overlay {
                if !vip && !restricted {
                    Capsule().stroke(palette.border, lineWidth: 1)
                }
            }
    }

    private func circularButton(symbol: String, label: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: symbol)
                .font(.system(size: 16))
                .foregroundStyle(palette.textSecondary)
                .frame(width: 36, height: 36)
                .background(palette.card, in: Circle())
                .overlay { Circle().stroke(palette.border, lineWidth: 1) }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(label)
    }
}

// MARK: - 段位卡

struct MyRankCardSection: View {
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @ObservedObject var model: MyPageModel

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if !model.rankGameView.tabs.isEmpty {
                tabs(model.rankGameView.tabs)
            } else if model.ranks.value != nil {
                Text(model.rankGameView.primaryLabel)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(Color.white.opacity(0.8))
                    .padding(.top, 4)
            }

            if model.isRankSkeleton {
                skeleton
            } else {
                rankBody
            }

            if model.showRankGamePicker {
                picker
            }
        }
        .padding(.horizontal, 12)
        .padding(.top, 10)
        .padding(.bottom, 14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background {
            RoundedRectangle(cornerRadius: 16).fill(
                LinearGradient(
                    colors: palette.heroColors,
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
        }
    }

    private func tabs(_ tabs: [GameTypeOption]) -> some View {
        HStack(spacing: 2) {
            ForEach(tabs) { item in
                let active = model.currentRankGameType == item.value
                Button {
                    model.selectedRankGameType = item.value
                } label: {
                    Text(item.label)
                        .font(.system(size: 11, weight: .semibold))
                        .lineLimit(1)
                        .foregroundStyle(active ? Color(hex: 0x4F3707) : Color.white.opacity(0.72))
                        .frame(maxWidth: .infinity)
                        .frame(height: 28)
                        .background(active ? Color.white : Color.clear, in: Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(2)
        .background(Color.black.opacity(0.18), in: Capsule())
    }

    private var skeleton: some View {
        HStack(spacing: 10) {
            Circle()
                .fill(Color.white.opacity(0.18))
                .frame(width: 48, height: 48)
            VStack(alignment: .leading, spacing: 8) {
                Capsule().fill(Color.white.opacity(0.18)).frame(width: 140, height: 12)
                Capsule().fill(Color.white.opacity(0.18)).frame(width: 210, height: 12)
            }
            Spacer(minLength: 0)
        }
        .padding(.top, 14)
    }

    private var rankBody: some View {
        let card = model.rankCard

        return Button {
            navigator.open(.rankExplain(gameType: model.currentRankGameType))
        } label: {
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: 10) {
                    RemoteImage(url: card.iconURL, contentMode: .fit) {
                        Color.clear
                    }
                    .frame(width: 48, height: 48)

                    VStack(alignment: .leading, spacing: 3) {
                        Text(card.name)
                            .font(.system(size: 20, weight: .heavy))
                            .lineLimit(1)
                            .foregroundStyle(Color.white)
                        if card.ranked {
                            Text("排位分 \(String(card.score))")
                                .font(.system(size: 12))
                                .foregroundStyle(Color.white.opacity(0.8))
                        }
                    }

                    Spacer(minLength: 8)

                    HStack(spacing: 1) {
                        Text("段位说明")
                            .font(.system(size: 11))
                        Image(systemName: "chevron.right")
                            .font(.system(size: 9, weight: .semibold))
                    }
                    .foregroundStyle(Color.white.opacity(0.72))
                }

                if card.ranked {
                    GeometryReader { geometry in
                        Capsule()
                            .fill(Color.white.opacity(0.2))
                            .overlay(alignment: .leading) {
                                Capsule()
                                    .fill(Color.white.opacity(0.9))
                                    .frame(
                                        width: geometry.size.width
                                            * CGFloat(min(100, max(0, card.progress))) / 100
                                    )
                            }
                    }
                    .frame(height: 6)
                    .padding(.top, 13)
                }

                Text(card.hint)
                    .font(.system(size: 11))
                    .foregroundStyle(Color.white.opacity(0.8))
                    .padding(.top, 6)
            }
        }
        .buttonStyle(.plain)
        .padding(.top, 14)
    }

    private var picker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Rectangle()
                .fill(Color.white.opacity(0.16))
                .frame(height: 1)
            Text("你主要打哪种？选好后发起 PK 也会默认选中")
                .font(.system(size: 11))
                .foregroundStyle(Color.white.opacity(0.8))
            HStack(spacing: 6) {
                ForEach(GameTypeCatalog.ordered) { item in
                    Button {
                        model.pickPreferredGameType(item.value)
                        toast.show("已设为常用球种")
                    } label: {
                        Text(item.label)
                            .font(.system(size: 11, weight: .semibold))
                            .lineLimit(1)
                            .foregroundStyle(Color.white)
                            .frame(maxWidth: .infinity)
                            .frame(height: 28)
                            .background(Color.black.opacity(0.18), in: Capsule())
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .padding(.top, 12)
    }
}

// MARK: - 生涯数据

struct MyCareerSection: View {
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme

    @ObservedObject var model: MyPageModel

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "生涯数据", caption: "全部球种", moreTitle: "竞技分析") {
                navigator.open(.statsDetail(gameType: model.currentRankGameType))
            }

            if model.careerState == .empty {
                VStack(alignment: .leading, spacing: 5) {
                    Text("还没有对局记录")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("打完第一场，场次、胜率和连胜都会记在这里")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .appCard(padding: 12)
            } else {
                metricsCard
            }

            if model.careerState == .error {
                Button {
                    Task { await model.refresh() }
                } label: {
                    Text("生涯数据读取失败，点击重试")
                        .font(.system(size: 11))
                        .foregroundStyle(AppTheme.danger)
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 2)
            }
        }
    }

    private var metricsCard: some View {
        Button {
            navigator.open(.statsDetail(gameType: model.currentRankGameType))
        } label: {
            HStack(spacing: 0) {
                ForEach(model.coreMetrics) { metric in
                    VStack(spacing: 4) {
                        Text(metric.value)
                            .font(.system(size: 22, weight: .bold))
                            .lineLimit(1)
                            .foregroundStyle(
                                metric.key == "win-rate" ? palette.brandStrong : palette.textPrimary
                            )
                        Text(metric.label)
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                    .frame(maxWidth: .infinity)
                    .overlay(alignment: .leading) {
                        if metric.key != "total" {
                            Rectangle()
                                .fill(palette.border)
                                .frame(width: 1, height: 28)
                        }
                    }
                }
            }
            .padding(.vertical, 14)
        }
        .buttonStyle(.plain)
        .appCard(padding: 4)
    }
}

// MARK: - 常用入口

struct MyShortcutSection: View {
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme

    @ObservedObject var model: MyPageModel

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(alignment: .top, spacing: 4) {
            shortcut("对局记录", symbol: "list.bullet.rectangle") {
                navigator.open(.matchHistory)
            }
            shortcut("过往对手", symbol: "person.2") {
                navigator.open(.opponentRecord(target: nil))
            }
            shortcut("荣誉墙", symbol: "medal") {
                navigator.open(.achievementWall(gameType: model.currentRankGameType))
            }
            shortcut("好友", symbol: "person") {
                navigator.open(.friendList)
            }
        }
        .padding(.horizontal, 4)
        .padding(.top, 14)
        .padding(.bottom, 12)
        .appCard(padding: 0)
    }

    private func shortcut(_ title: String, symbol: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            VStack(spacing: 6) {
                Image(systemName: symbol)
                    .font(.system(size: 18))
                    .foregroundStyle(palette.brandStrong)
                    .frame(width: 40, height: 40)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 12))
                Text(title)
                    .font(.system(size: 11, weight: .semibold))
                    .lineLimit(1)
                    .foregroundStyle(palette.textPrimary)
            }
            .frame(maxWidth: .infinity)
        }
        .buttonStyle(.plain)
    }
}

// MARK: - 常去球馆奖励（审核中）

struct MyRewardEntryCard: View {
    @Environment(\.colorScheme) private var colorScheme

    let entry: CompactRewardEntry

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                Text(entry.title)
                    .font(.system(size: 14, weight: .semibold))
                    .lineLimit(1)
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Text(entry.statusText)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
            }
            Text(entry.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .appCard(padding: 12)
    }
}

// MARK: - 服务列表

struct MyServiceListSection: View {
    @Environment(\.colorScheme) private var colorScheme

    @ObservedObject var model: MyPageModel

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var items: [MyListRowItem] {
        [
            MyListRowItem(symbol: "list.bullet", title: "约球记录", route: .challenges(tab: .history)),
            MyListRowItem(
                symbol: "crown",
                title: "VIP中心",
                value: model.memberEntry.text,
                route: .memberCenter
            ),
            MyListRowItem(symbol: "slider.horizontal.3", title: "装备", route: .gearIndex),
            MyListRowItem(symbol: "calendar", title: "赛季", route: .seasonIndex),
            MyListRowItem(symbol: "questionmark.circle", title: "规则百科", route: .rulesIndex),
            MyListRowItem(symbol: "mappin.and.ellipse", title: "附近球馆", route: .venueList),
        ]
    }

    var body: some View {
        MyListCard(items: items)
    }
}

// MARK: - 访客

struct MyGuestCard: View {
    @Environment(\.colorScheme) private var colorScheme

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private let benefits = ["记录每一局真实比分", "四个球种分别积累排位分", "查看胜率、连胜和过往对手"]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "person.fill")
                    .font(.system(size: 26))
                    .foregroundStyle(palette.textMuted)
                    .frame(width: 56, height: 56)
                    .background(palette.subtle, in: Circle())
                    .overlay { Circle().stroke(palette.border, lineWidth: 1) }
                VStack(alignment: .leading, spacing: 4) {
                    Text("未登录")
                        .font(.system(size: 19, weight: .heavy))
                        .foregroundStyle(palette.textPrimary)
                    Text("登录后同步你的段位、战绩和好友")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 0)
            }

            VStack(alignment: .leading, spacing: 7) {
                ForEach(benefits, id: \.self) { item in
                    HStack(spacing: 6) {
                        Image(systemName: "checkmark")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                        Text(item)
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 12)
            .padding(.vertical, 11)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
            .padding(.top, 14)

            NavigationLink(value: AppRoute.login) {
                Text("登录 / 注册")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 44)
                    .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 9))
            }
            .buttonStyle(.plain)
            .padding(.top, 14)
        }
        .appCard(padding: 14)
    }
}

struct MyGuestLinksCard: View {
    var body: some View {
        MyListCard(items: [
            MyListRowItem(
                symbol: "medal",
                title: "排行榜",
                detail: "各球种高手的段位与积分",
                route: .ranking
            ),
            MyListRowItem(
                symbol: "questionmark.circle",
                title: "规则百科",
                detail: "常见台球规则与术语",
                route: .rulesIndex
            ),
            MyListRowItem(
                symbol: "bubble.left.and.bubble.right",
                title: "帮助、投诉与举报",
                detail: "无需登录即可提交问题与反馈",
                route: .feedback
            ),
        ])
    }
}

struct MyGuestSettingsCard: View {
    var body: some View {
        MyListCard(items: [MyListRowItem(symbol: "gearshape", title: "设置", route: .settings)])
    }
}

// MARK: - 悬浮奖励入口

struct MyFloatRewardBar: View {
    @Environment(\.colorScheme) private var colorScheme

    let entry: CompactRewardEntry
    let onOpen: () -> Void
    let onSnooze: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(spacing: 8) {
            Button(action: onOpen) {
                HStack(spacing: 8) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(entry.title)
                            .font(.system(size: 13, weight: .bold))
                            .lineLimit(1)
                            .foregroundStyle(palette.textPrimary)
                        Text(entry.description)
                            .font(.system(size: 11))
                            .lineLimit(2)
                            .foregroundStyle(palette.textSecondary)
                    }
                    Spacer(minLength: 0)
                    Text(entry.actionText)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                }
            }
            .buttonStyle(.plain)

            Button(action: onSnooze) {
                Image(systemName: "xmark")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
                    .frame(width: 22, height: 22)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("稍后提醒")
        }
        .padding(.leading, 12)
        .padding(.trailing, 10)
        .padding(.vertical, 10)
        .background {
            RoundedRectangle(cornerRadius: 12)
                .fill(palette.card)
                .overlay {
                    LinearGradient(
                        colors: [palette.brandTint, palette.brandTint.opacity(0)],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                }
                .clipShape(RoundedRectangle(cornerRadius: 12))
        }
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.brandStrong.opacity(0.32), lineWidth: 1)
        }
    }
}

// MARK: - 列表卡片

private struct MyListRowItem {
    let symbol: String
    let title: String
    var detail: String?
    var value: String?
    let route: AppRoute
}

private struct MyListCard: View {
    @Environment(\.colorScheme) private var colorScheme

    let items: [MyListRowItem]

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            ForEach(items.indices, id: \.self) { index in
                if index > 0 {
                    Rectangle()
                        .fill(palette.border)
                        .frame(height: 1)
                }
                MyListRow(item: items[index])
            }
        }
        .appCard(padding: 0)
    }
}

private struct MyListRow: View {
    @Environment(\.colorScheme) private var colorScheme

    let item: MyListRowItem

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        NavigationLink(value: item.route) {
            HStack(spacing: 10) {
                Image(systemName: item.symbol)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.brandStrong)
                    .frame(width: 28, height: 28)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 8))

                VStack(alignment: .leading, spacing: 3) {
                    Text(item.title)
                        .font(.system(size: 14, weight: .semibold))
                        .lineLimit(1)
                        .foregroundStyle(palette.textPrimary)
                    if let detail = item.detail {
                        Text(detail)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                    }
                }

                Spacer(minLength: 8)

                if let value = item.value, !value.isEmpty {
                    Text(value)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .frame(minHeight: 52)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
