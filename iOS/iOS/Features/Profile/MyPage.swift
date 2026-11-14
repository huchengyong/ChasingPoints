import SwiftUI

/// 「我的」主页，对应小程序 `pages/user/index`。
struct MyPage: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: MyPageModel
    @StateObject private var rolloverPresenter: SeasonRolloverPresenter
    @State private var isQRCodePresented = false

    init(session: SessionStore, model: MyPageModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MyPageModel(session: session))
        _rolloverPresenter = StateObject(wrappedValue: SeasonRolloverPresenter(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                if session.isRestoring {
                    ProgressView("正在恢复登录状态")
                        .frame(maxWidth: .infinity)
                        .padding(24)
                } else if session.identity != nil {
                    memberContent
                } else {
                    guestContent
                }

                if let restoreError = session.restoreError {
                    HStack {
                        Text(restoreError)
                            .font(.system(size: 14))
                            .foregroundStyle(palette.textSecondary)
                        Spacer(minLength: 8)
                        Button("重试") {
                            Task { await session.retryRestore() }
                        }
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, 12)
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 12)
            .padding(.bottom, 24)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("我的")
        .task {
            await model.onAppear()
            await presentSeasonRollover()
        }
        .refreshable { await model.refresh() }
        .onChange(of: session.identity) { _, _ in
            Task {
                await model.onIdentityChanged()
                await presentSeasonRollover()
            }
        }
        .seasonRolloverAlert(rolloverPresenter)
        .onDisappear { rolloverPresenter.respond(confirmed: false) }
        .sheet(isPresented: $isQRCodePresented) {
            MatchQRCodeSheet(session: session)
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            if model.floatRewardVisible {
                MyFloatRewardBar(
                    entry: model.compactRewardEntry,
                    onOpen: { navigator.open(.venueSubmit) },
                    onSnooze: { model.snoozeFloatReward() }
                )
                .padding(.horizontal, 12)
                .padding(.bottom, 12)
            }
        }
    }

    /// 换季提醒：优先用概览里缓存的未读通知，处理完清空缓存避免另一页重复弹出。
    private func presentSeasonRollover() async {
        await rolloverPresenter.present(
            notification: activity.latestSeasonRollover,
            onRead: {
                activity.clearLatestSeasonRollover()
                activity.decrementUnread()
            },
            onNavigate: { navigator.open($0) }
        )
    }

    private var guestContent: some View {
        VStack(spacing: 12) {
            MyGuestCard()
            MyGuestLinksCard()
            MyGuestSettingsCard()
        }
    }

    private var memberContent: some View {
        VStack(spacing: 0) {
            MyProfileHeaderSection(model: model) { isQRCodePresented = true }
            MyRankCardSection(model: model)
            MyCareerSection(model: model)
                .padding(.top, 20)
            MyShortcutSection(model: model)
                .padding(.top, 12)
            if model.compactRewardEntry.inline {
                MyRewardEntryCard(entry: model.compactRewardEntry)
                    .padding(.top, 12)
            }
            MyServiceListSection(model: model)
                .padding(.top, 12)
        }
    }
}
