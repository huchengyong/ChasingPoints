import SwiftUI

/// 九球追分双方对照表与得分明细；`view` 已按查看者视角换边。
struct ChasingScoreCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let view: ChasingScoreView
    var showEvents = true

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("得分构成")
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(palette.textPrimary)

            HStack {
                Text(view.leftLabel)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Text("项目")
                    .frame(maxWidth: .infinity, alignment: .center)
                Text(view.rightLabel)
                    .frame(maxWidth: .infinity, alignment: .trailing)
            }
            .font(.system(size: 11))
            .foregroundStyle(palette.textMuted)

            ForEach(view.rows, id: \.key) { row in
                rowView(row)
            }

            if !view.notice.isEmpty {
                Text(view.notice)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 6))
            }

            if showEvents {
                Text("得分明细")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .padding(.top, 4)

                if view.events.isEmpty {
                    if view.available {
                        Text("暂无记分")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                    }
                } else {
                    VStack(spacing: 6) {
                        ForEach(view.events, id: \.seq) { event in
                            eventRow(event)
                        }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
    }

    private func rowView(_ row: ChasingScoreRow) -> some View {
        let isTotal = row.key == "score"
        return HStack {
            Text(row.left)
                .font(.system(size: isTotal ? 18 : 15, weight: isTotal ? .heavy : .semibold))
                .foregroundStyle(isTotal ? palette.brandStrong : palette.textPrimary)
                .frame(maxWidth: .infinity, alignment: .leading)
            Text(row.label)
                .font(.system(size: isTotal ? 12 : 12, weight: isTotal ? .semibold : .regular))
                .foregroundStyle(isTotal ? palette.textPrimary : palette.textSecondary)
                .frame(maxWidth: .infinity, alignment: .center)
            Text(row.right)
                .font(.system(size: isTotal ? 18 : 15, weight: isTotal ? .heavy : .semibold))
                .foregroundStyle(isTotal ? palette.brandStrong : palette.textPrimary)
                .frame(maxWidth: .infinity, alignment: .trailing)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(isTotal ? palette.brandTint : Color.clear, in: RoundedRectangle(cornerRadius: 6))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(row.label)：\(view.leftLabel) \(row.left)，\(view.rightLabel) \(row.right)")
    }

    private func eventRow(_ event: ChasingScoreEventView) -> some View {
        let accent: Color = event.kind == "foul"
            ? AppTheme.warning
            : (event.side == "left" ? palette.brandStrong : palette.textMuted)
        return HStack(spacing: 8) {
            HStack(spacing: 6) {
                Text(event.desc)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(event.pointsText)
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(event.kind == "foul"
                        ? AppTheme.warning
                        : (event.side == "left" ? palette.brandStrong : palette.textSecondary))
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 1) {
                Text(event.scoreText)
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                if !event.timeText.isEmpty {
                    Text(event.timeText)
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                }
            }
        }
        .padding(.horizontal, 9)
        .padding(.vertical, 8)
        .background(
            event.kind == "foul" ? AppTheme.warning.opacity(0.1) : palette.subtle,
            in: RoundedRectangle(cornerRadius: 6)
        )
        .overlay(alignment: .leading) {
            RoundedRectangle(cornerRadius: 3)
                .fill(accent)
                .frame(width: 3)
                .padding(.vertical, 6)
        }
    }
}
