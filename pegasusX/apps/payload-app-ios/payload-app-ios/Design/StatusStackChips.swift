import SwiftUI

struct SourceChip: View {
    let source: String

    var body: some View {
        Text(source.uppercased())
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .overlay(Capsule().strokeBorder(.secondary, lineWidth: 1))
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Source: \(source)")
            .accessibilityIdentifier("gs-u-source-chip")
    }
}

struct FlexibleStatusChips: View {
    let rows: [StatusStackRow]
    var onSelect: ((String) -> Void)? = nil

    var body: some View {
        FlowLayout {
            ForEach(rows, id: \.key) { row in
                chip(row)
            }
        }
    }

    @ViewBuilder
    private func chip(_ row: StatusStackRow) -> some View {
        let readableStatus = row.key.replacingOccurrences(of: "_", with: " ")
        let countText = row.count.map(String.init) ?? "unavailable"
        let label = HStack(spacing: 6) {
            Text(readableStatus)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(row.count.map(String.init) ?? "—")
                .font(.caption.monospacedDigit())
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .frame(minWidth: 44, minHeight: 44)
        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(.secondary.opacity(0.4)))
        .accessibilityIdentifier("gs-u-chip-\(row.key)")

        if let onSelect {
            Button {
                PegasusHaptics.selection()
                onSelect(row.key)
            } label: {
                label
            }
            .buttonStyle(.plain)
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(readableStatus): \(countText)")
        } else {
            label
                .accessibilityElement(children: .combine)
                .accessibilityLabel("\(readableStatus): \(countText)")
        }
    }
}

struct FlowLayout: Layout {
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
