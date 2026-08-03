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
    }

    private func inboundCard(_ row: InboundReturnRow, selectable: Bool) -> some View {
        let isSelected = selected.contains(row.returnId)
        return Button {
            guard selectable else { return }
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
                Text("\(row.returnId.prefix(8)) · suggest \(row.suggestedDisposition)")
                    .font(.caption.monospaced())
                    .foregroundStyle(TermTheme.tertiary)
                if let barcode = row.barcode, !barcode.isEmpty {
                    Text("EAN \(barcode)")
                        .font(.caption2.monospaced())
                        .foregroundStyle(TermTheme.secondary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(TermTheme.s16)
            .background(isSelected ? TermTheme.accent.opacity(0.12) : TermTheme.card)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? TermTheme.accent : TermTheme.separator, lineWidth: 1)
            )
            .clipShape(.rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
    }
}
