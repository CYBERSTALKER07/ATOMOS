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
                            ReturnItemRowView(
                                item: item,
                                selectable: isQueueTab,
                                selected: $selected
                            )
                        }
                    }
                }
            }
        }
    }
}

private struct ReturnItemRowView: View {
    let item: InboundReturnRow
    let selectable: Bool
    @Binding var selected: Set<String>

    @State private var isSelected: Bool

    init(
        item: InboundReturnRow,
        selectable: Bool,
        selected: Binding<Set<String>>
    ) {
        self.item = item
        self.selectable = selectable
        self._selected = selected
        self._isSelected = State(initialValue: selected.wrappedValue.contains(item.returnId))
    }

    var body: some View {
        let displayTitle = item.productName.isEmpty ? item.returnId : item.productName
        HStack {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                HStack {
                    Text(displayTitle)
                        .font(.headline)
                    if item.isClaimTicket {
                        Text("mobile_warehouse.ui.claim_ticket")
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .foregroundStyle(PegasusMonochromeTheme.warning)
                            .background(PegasusMonochromeTheme.warning.opacity(0.15))
                            .clipShape(Capsule())
                    }
                }
                Text(L10n.format("mobile_warehouse.ui.qty_receivedqty_expectedqty_reason", "\(item.receivedQty)", "\(item.expectedQty)", "\(item.reason)"))
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Text(item.driverName.isEmpty ? (item.isClaimTicket ? "store return" : "—") : "Driver: \(item.driverName)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if !item.suggestedDisposition.isEmpty {
                    Text(L10n.format("mobile_warehouse.ui.suggested_suggesteddisposition", "\(item.suggestedDisposition)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if !item.driverNotes.isEmpty {
                    Text(item.driverNotes)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if let code = item.barcode, !code.isEmpty {
                    Text(L10n.format("mobile_warehouse.ui.ean_code_2", "\(code)"))
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
            }
            Spacer()
            if selectable {
                Toggle("Select \(displayTitle)", isOn: $isSelected)
                    .labelsHidden()
                    .frame(minWidth: 44, minHeight: 44)
                    .accessibilityLabel("Select return \(displayTitle)")
                    .onChange(of: isSelected) { _, newValue in
                        guard selected.contains(item.returnId) != newValue else { return }
                        PegasusHaptics.selection()
                        if newValue {
                            selected.insert(item.returnId)
                        } else {
                            selected.remove(item.returnId)
                        }
                    }
                    .onChange(of: selected) { _, newSet in
                        let contains = newSet.contains(item.returnId)
                        if isSelected != contains {
                            isSelected = contains
                        }
                    }
            }
        }
        .accessibilityElement(children: .contain)
    }
}
