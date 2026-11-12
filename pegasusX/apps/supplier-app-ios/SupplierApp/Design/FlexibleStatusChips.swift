import SwiftUI

struct FlexibleStatusChips: View {
    let rows: [StatusStackRow]
    var onSelect: ((String) -> Void)? = nil

    var body: some View {
        StatusChipsFlowLayout {
            ForEach(rows, id: \.key) { row in
                chip(row)
            }
        }
    }

    @ViewBuilder
    private func chip(_ row: StatusStackRow) -> some View {
        let label = HStack(spacing: 6) {
            Text(row.key.replacingOccurrences(of: "_", with: " "))
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(row.count.map(String.init) ?? "—")
                .font(.caption.monospacedDigit())
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .frame(minHeight: 44)
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(.secondary.opacity(0.4)))
        .accessibilityIdentifier("gs-u-chip-\(row.key)")

        if let onSelect {
            Button {
                PegasusHaptics.selection()
                onSelect(row.key)
            } label: {
                label
            }
            .buttonStyle(.plain)
        } else {
            label
        }
    }
}

struct StatusChipsFlowLayout: Layout {
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let maxWidth = proposal.width ?? .infinity
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowH: CGFloat = 0
        var height: CGFloat = 0
        for view in subviews {
            let size = view.sizeThatFits(.unspecified)
            if x + size.width > maxWidth && x > 0 {
                x = 0
                y += rowH + 8
                rowH = 0
            }
            rowH = max(rowH, size.height)
            x += size.width + 8
            height = y + rowH
        }
        return CGSize(width: maxWidth.isFinite ? maxWidth : x, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX
        var y = bounds.minY
        var rowH: CGFloat = 0
        for view in subviews {
            let size = view.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX && x > bounds.minX {
                x = bounds.minX
                y += rowH + 8
                rowH = 0
            }
            view.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
            x += size.width + 8
            rowH = max(rowH, size.height)
        }
    }
}
