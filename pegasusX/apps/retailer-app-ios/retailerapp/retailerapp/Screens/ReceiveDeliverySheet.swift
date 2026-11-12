import SwiftUI

struct ReceiveDeliverySheet: View {
    let locationId: String
    var onComplete: () -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var vm = StoreStockActionViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section("Incoming Shipment") {
                    TextField("Delivery Order ID (e.g. ord_...)", text: $vm.orderId)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
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
                            if await vm.submitReceive(locationId: locationId) {
                                onComplete()
                                dismiss()
                            }
                        }
                    } label: {
                        if vm.busy {
                            ProgressView()
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        } else {
                            Text("Receive into Backroom")
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                        }
                    }
                    .disabled(vm.busy || vm.orderId.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("Receive Delivery")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}
