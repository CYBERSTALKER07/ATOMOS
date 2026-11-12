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
                    InventoryStockRowView(
                        item: item,
                        isSavingPolicy: policySavingId == item.productId,
                        policies: policies,
                        onAdjust: { onAdjust(item) },
                        onPolicyChange: { policy in onPolicyChange(item, policy) }
                    )
                }
            }
        }
    }
}

private struct InventoryStockRowView: View {
    let item: InventoryItem
    let isSavingPolicy: Bool
    let policies: [String]
    let onAdjust: () -> Void
    let onPolicyChange: (String) -> Void

    @State private var selectedPolicy: String

    private static func resolvedPolicy(for item: InventoryItem) -> String {
        if let policy = item.outOfStockPolicy, !policy.isEmpty {
            return policy
        }
        return "INHERIT"
    }

    init(
        item: InventoryItem,
        isSavingPolicy: Bool,
        policies: [String],
        onAdjust: @escaping () -> Void,
        onPolicyChange: @escaping (String) -> Void
    ) {
        self.item = item
        self.isSavingPolicy = isSavingPolicy
        self.policies = policies
        self.onAdjust = onAdjust
        self.onPolicyChange = onPolicyChange
        _selectedPolicy = State(initialValue: Self.resolvedPolicy(for: item))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(item.productName)
                        .font(.headline)
                    Text(L10n.format("mobile_warehouse.ui.qty_quantity_reorder_reorderthreshold", "\(item.quantity)", "\(item.reorderThreshold)"))
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
                Button("retailer_desktop.stock.text.adjust") {
                    PegasusHaptics.light()
                    onAdjust()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Adjust stock for \(item.productName)")
            }
            Picker("Out-of-stock policy", selection: $selectedPolicy) {
                ForEach(policies, id: \.self) { policy in
                    Text(policy).tag(policy)
                }
            }
            .pickerStyle(.menu)
            .frame(minHeight: 44)
            .disabled(isSavingPolicy)
            .accessibilityLabel("Out-of-stock policy for \(item.productName)")
            .onChange(of: item.outOfStockPolicy) { _, _ in
                let resolved = Self.resolvedPolicy(for: item)
                if selectedPolicy != resolved {
                    selectedPolicy = resolved
                }
            }
            .onChange(of: selectedPolicy) { _, newValue in
                let resolved = Self.resolvedPolicy(for: item)
                guard newValue != resolved else { return }
                PegasusHaptics.selection()
                onPolicyChange(newValue)
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(item.productName), \(item.quantity) units in stock, reorder threshold \(item.reorderThreshold)")
        .labCard()
    }
}
