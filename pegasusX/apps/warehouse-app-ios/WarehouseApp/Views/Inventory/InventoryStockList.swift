import SwiftUI

/// Inventory stock list rendering product cards with quantity,
/// low-stock badges, OOS policy pickers, and adjust buttons.
struct InventoryStockList: View {
    let items: [InventoryItem]
    let policySavingId: String?
    let policies: [String]
    var onAdjust: (InventoryItem) -> Void
    var onPolicyChange: (InventoryItem, String) -> Void

    var body: some View {
        if items.isEmpty {
            WarehouseEmptyView(title: "No Inventory Items", message: "There are no matching items.")
        } else {
            ResponsiveGridContentWrapper {
                ForEach(items) { item in
                    VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(item.productName)
                                    .font(.headline)
                                Text("Qty: \(item.quantity) · Reorder: \(item.reorderThreshold)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            if item.quantity <= item.reorderThreshold {
                                Text("LOW")
                                    .font(.caption.bold())
                                    .padding(.horizontal, LabTheme.spacingSM)
                                    .padding(.vertical, LabTheme.spacingXS)
                                    .foregroundStyle(.white)
                                    .background(.red, in: Capsule())
                            }
                            Button("Adjust") { onAdjust(item) }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                        }
                        Picker("Out-of-stock policy", selection: Binding(
                            get: { item.outOfStockPolicy?.isEmpty == false ? item.outOfStockPolicy! : "INHERIT" },
                            set: { newValue in onPolicyChange(item, newValue) }
                        )) {
                            ForEach(policies, id: \.self) { policy in
                                Text(policy).tag(policy)
                            }
                        }
                        .pickerStyle(.menu)
                        .disabled(policySavingId == item.productId)
                    }
                    .labCard()
                }
            }
        }
    }
}
