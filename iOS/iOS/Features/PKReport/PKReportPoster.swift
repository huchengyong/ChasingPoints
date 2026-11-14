import SwiftUI

/// PK 报表海报，布局与文案复刻 `uniapp/utils/posterGenerator.js` 的 `generatePkReportPoster`。
struct PKReportPoster: View {
    static let size = CGSize(width: 750, height: 1080)

    let payload: PKPosterPayload
    let qrImage: UIImage?

    private let titleColor = Color(hex: 0x94A3B8)
    private let hintColor = Color(hex: 0x64748B)

    var body: some View {
        ZStack(alignment: .topLeading) {
            LinearGradient(
                colors: [Color(hex: 0x0F172A), Color(hex: 0x1E3A5F)],
                startPoint: .top,
                endPoint: .bottom
            )

            Text("追分 · PK 报表")
                .font(.system(size: 28))
                .foregroundStyle(titleColor)
                .frame(width: Self.size.width)
                .position(x: Self.size.width / 2, y: 86)

            Text(payload.gameTypeLabel)
                .font(.system(size: 22))
                .foregroundStyle(hintColor)
                .frame(width: Self.size.width)
                .position(x: Self.size.width / 2, y: 132)

            if let qrImage {
                ZStack {
                    RoundedRectangle(cornerRadius: 8).fill(.white)
                    Image(uiImage: qrImage)
                        .interpolation(.none)
                        .resizable()
                        .scaledToFit()
                        .padding(4)
                }
                .frame(width: 108, height: 108)
                .position(x: 596 + 54, y: 28 + 54)

                Text("扫码打开追分")
                    .font(.system(size: 18))
                    .foregroundStyle(titleColor)
                    .frame(width: 108, alignment: .leading)
                    .position(x: 596 + 54, y: 162)
            }

            playerNames
            scoreRow

            Text("历史交锋胜场")
                .font(.system(size: 24))
                .foregroundStyle(titleColor)
                .frame(width: Self.size.width)
                .position(x: Self.size.width / 2, y: 430)

            metricsCard
            summaryCard

            Text("仅统计真实线下对局，不代表线上比赛结果")
                .font(.system(size: 22))
                .foregroundStyle(titleColor)
                .frame(width: Self.size.width)
                .position(x: Self.size.width / 2, y: 1000)
        }
        .frame(width: Self.size.width, height: Self.size.height)
        .clipped()
    }

    private var playerNames: some View {
        ZStack {
            Text(payload.myName)
                .font(.system(size: 28))
                .foregroundStyle(.white)
                .frame(width: 260)
                .position(x: 190, y: 250)
            Text(payload.opponentName)
                .font(.system(size: 28))
                .foregroundStyle(.white)
                .frame(width: 260)
                .position(x: 560, y: 250)
        }
        .frame(width: Self.size.width, height: Self.size.height, alignment: .topLeading)
    }

    private var scoreRow: some View {
        ZStack {
            Text(String(payload.myWins))
                .font(.system(size: 80))
                .foregroundStyle(Color(hex: 0x22C55E))
                .frame(width: 260)
                .position(x: 190, y: 380)
            Text(":")
                .font(.system(size: 42))
                .foregroundStyle(hintColor)
                .position(x: 375, y: 368)
            Text(String(payload.opponentWins))
                .font(.system(size: 80))
                .foregroundStyle(Color(hex: 0xF8FAFC))
                .frame(width: 260)
                .position(x: 560, y: 380)
        }
        .frame(width: Self.size.width, height: Self.size.height, alignment: .topLeading)
    }

    private var metricsCard: some View {
        VStack(spacing: 30) {
            HStack(spacing: 20) {
                metric("总场次", String(payload.totalMatches))
                metric("我的胜率", payload.winRateLabel)
            }
            HStack(spacing: 20) {
                metric("平均分差", payload.avgScoreDiffLabel)
                metric("最长连胜", String(payload.maxWinStreak))
            }
        }
        .padding(.horizontal, 50)
        .padding(.vertical, 22)
        .frame(width: 630, height: 250, alignment: .topLeading)
        .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 24))
        .position(x: 375, y: 625)
    }

    private func metric(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(label)
                .font(.system(size: 22))
                .foregroundStyle(titleColor)
            Text(value)
                .font(.system(size: 36))
                .foregroundStyle(.white)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var summaryCard: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("系统结论")
                .font(.system(size: 24))
                .foregroundStyle(Color(hex: 0xBFDBFE))
            Text(payload.summaryText)
                .font(.system(size: 28))
                .foregroundStyle(.white)
                .lineLimit(2)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(.horizontal, 40)
        .padding(.vertical, 30)
        .frame(width: 630, height: 180, alignment: .topLeading)
        .background(Color(hex: 0x3B82F6, opacity: 0.12), in: RoundedRectangle(cornerRadius: 24))
        .position(x: 375, y: 880)
    }
}
