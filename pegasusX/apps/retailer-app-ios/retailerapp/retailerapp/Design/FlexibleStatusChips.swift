import SwiftUI

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
        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(.secondary.opacity(0.4)))
        .accessibilityElement(children: .combine)
        .accessibilityIdentifier("gs-u-chip-\(row.key)")

        if let onSelect {
            Button {
                Haptics.selection()
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
