import SwiftUI

struct InboundReturnsList: View {
    var rows: [InboundReturnRow]
    var selectable: Bool
    @Binding var selected: Set<String>

    var body: some View {
        ScrollView {
            LazyVStack(spacing: TermTheme.s12) {
                ForEach(rows) { row in
                    inboundCard(row, selectable: selectable)
                }
            }
            .padding(TermTheme.s16)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func inboundCard(_ row: InboundReturnRow, selectable: Bool) -> some View {
        let isSelected = selected.contains(row.returnId)
        return Button {
            guard selectable else { return }
            PegasusHaptics.selection()
            if isSelected { selected.remove(row.returnId) }
            else { selected.insert(row.returnId) }
        } label: {
            VStack(alignment: .leading, spacing: 4) {
                Text(row.productName)
                    .font(.headline)
                    .foregroundStyle(TermTheme.accent)
                Text("\(row.driverName.isEmpty ? "Driver" : row.driverName) · \(row.reason) · \(row.receivedQty)/\(row.expectedQty)")
                    .font(.subheadline)
                    .foregroundStyle(TermTheme.secondary)
                Text(L10n.format("mobile_payload.ui.prefix_suggest_suggesteddisposition", "\(row.returnId.prefix(8))", "\(row.suggestedDisposition)"))
                    .font(.caption.monospaced())
                    .foregroundStyle(TermTheme.tertiary)
                if let barcode = row.barcode, !barcode.isEmpty {
                    Text(L10n.format("mobile_payload.ui.ean_barcode_3", "\(barcode)"))
                        .font(.caption2.monospaced())
                        .foregroundStyle(TermTheme.secondary)
                }
            }
            .frame(maxWidth: .infinity, minHeight: 48, alignment: .leading)
            .padding(TermTheme.s16)
            .background(isSelected ? TermTheme.accent.opacity(0.12) : TermTheme.card)
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(isSelected ? TermTheme.accent : TermTheme.separator, lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(row.productName), from \(row.driverName.isEmpty ? "Driver" : row.driverName), reason: \(row.reason), \(row.receivedQty) of \(row.expectedQty) items")
        .accessibilityAddTraits(isSelected ? [.isSelected] : [])
        .buttonStyle(.plain)
    }
}
