import SwiftUI

struct PutawaySheet: View {
    let locationId: String
    let initialSku: String
    let availableSkus: [String]
    var onComplete: () -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var vm = StoreStockActionViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section("Select Item") {
                    if availableSkus.isEmpty {
                        TextField("SKU Identifier", text: $vm.sku)
                    } else {
                        Picker("Product SKU", selection: $vm.sku) {
                            Text("Select SKU...").tag("")
                            ForEach(availableSkus, id: \.self) { s in
                                Text(s).tag(s)
                            }
                        }
                    }
                }

                Section("Quantity to Shelf") {
                    TextField("Units to move", text: $vm.qty)
                        .keyboardType(.numberPad)
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
                            if await vm.submitPutaway(locationId: locationId) {
                                onComplete()
                                dismiss()
                            }
                        }
                    } label: {
                        if vm.busy {
                            ProgressView()
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        } else {
                            Text("Transfer to Sales Shelf")
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        }
                    }
                    .disabled(vm.busy || vm.sku.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("Putaway to Shelf")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
            .onAppear {
                vm.configureInitialSku(initialSku: initialSku, candidates: availableSkus)
            }
        }
    }
}
