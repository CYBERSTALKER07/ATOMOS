import SwiftUI

struct ReturnsList: View {
    let items: [InboundReturnRow]
    let isQueueTab: Bool
    @Binding var selected: Set<String>

    var body: some View {
        Group {
            if items.isEmpty {
                ContentUnavailableView(
                    isQueueTab ? "No inbound returns" : "No history",
                    systemImage: isQueueTab ? "arrow.uturn.backward.circle" : "clock.arrow.circlepath"
                )
            } else {
                ResponsiveGridContentWrapper {
                    Section(isQueueTab ? "Open at gate" : "Completed receives") {
                        ForEach(items) { item in
                            returnRow(item, selectable: isQueueTab)
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func returnRow(_ item: InboundReturnRow, selectable: Bool) -> some View {
        HStack {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                HStack {
                    Text(item.productName.isEmpty ? item.returnId : item.productName)
                        .font(.headline)
                    if item.isClaimTicket {
                        Text("Claim ticket")
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.orange.opacity(0.2))
                            .clipShape(Capsule())
                    }
                }
                Text("Qty \(item.receivedQty)/\(item.expectedQty) · \(item.reason)")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Text(item.driverName.isEmpty ? (item.isClaimTicket ? "store return" : "—") : "Driver: \(item.driverName)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if !item.suggestedDisposition.isEmpty {
                    Text("Suggested: \(item.suggestedDisposition)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if !item.driverNotes.isEmpty {
                    Text(item.driverNotes)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if let code = item.barcode, !code.isEmpty {
                    Text("EAN \(code)")
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
            }
            Spacer()
            if selectable {
                Toggle("", isOn: Binding(
                    get: { selected.contains(item.returnId) },
                    set: { on in
                        if on { selected.insert(item.returnId) }
                        else { selected.remove(item.returnId) }
                    }
                ))
                .labelsHidden()
            }
        }
    }
}
