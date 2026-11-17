import SwiftUI

struct AdjustStockSheet: View {
    let locationId: String
    let initialSku: String
    let allSkus: [String]
    var onComplete: () -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var vm = StoreStockActionViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section("Target SKU") {
                    if allSkus.isEmpty {
                        TextField("SKU Identifier", text: $vm.sku)
                    } else {
                        Picker("Product SKU", selection: $vm.sku) {
                            Text("Select SKU...").tag("")
                            ForEach(allSkus, id: \.self) { s in
                                Text(s).tag(s)
                            }
                        }
                    }
                }

                Section("Adjustment (Delta)") {
                    TextField("Qty (+ to add, - for loss)", text: $vm.qtyDelta)
                        .keyboardType(.numbersAndPunctuation)
                }

                if let errorMessage = vm.errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }

                Section {
                    Button {
                        Task {
                            if await vm.submitAdjust(locationId: locationId) {
                                onComplete()
                                dismiss()
                            }
                        }
                    } label: {
                        if vm.busy {
                            ProgressView()
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        } else {
                            Text("Apply Count Adjustment")
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        }
                    }
                    .disabled(vm.busy || vm.sku.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("Adjust Count")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
            .onAppear {
                vm.configureInitialSku(initialSku: initialSku, candidates: allSkus)
            }
        }
    }
}
