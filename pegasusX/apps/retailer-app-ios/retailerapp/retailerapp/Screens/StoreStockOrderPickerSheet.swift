import SwiftUI

struct StoreStockOrderPickerSheet: View {
    @Bindable var vm: StoreStockViewModel

    var body: some View {
        List {
            Section {
                Text("Pick a completed delivery to file a return or damage claim.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if let preferredSku = vm.preferredSku {
                    Text("Pre-selected SKU: \(preferredSku)")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.blue)
                }
                TextField("Search by Order ID...", text: $vm.pickerQuery)
            }
            if let pickerError = vm.pickerError {
                Section {
                    Text(pickerError)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }
            Section("Completed Orders") {
                if vm.pickerLoading {
                    ProgressView()
                } else if vm.filteredOrders.isEmpty {
                    Text("No completed or delivered orders found.")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(vm.filteredOrders) { order in
                        Button {
                            Task { await vm.pickOrder(order) }
                        } label: {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(order.id.suffix(8)) — \(order.status.rawValue.replacingOccurrences(of: "_", with: " "))")
                                    .font(.subheadline.weight(.semibold))
                                Text(order.id)
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget, alignment: .leading)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Order \(order.id.suffix(8)), status \(order.status.rawValue)")
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("Select Order for Claim")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Close") { vm.showOrderPicker = false }
            }
        }
    }
}
