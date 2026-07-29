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
                    Section(isQueueTab ? "Arrived at gate" : "Completed receives") {
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
                Text(item.productName)
                    .font(.headline)
                Text("Qty \(item.receivedQty)/\(item.expectedQty) · \(item.reason)")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                if !item.driverName.isEmpty {
                    Text("Driver: \(item.driverName)")
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
