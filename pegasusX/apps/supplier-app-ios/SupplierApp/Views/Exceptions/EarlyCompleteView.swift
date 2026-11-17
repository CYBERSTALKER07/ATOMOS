import SwiftUI

struct EarlyCompleteView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = EarlyCompleteViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            Section {
                Text("supplier_portal.residual.text.approve_a_driver_request_to_finish_the_route_before_all_stops_ar")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Section("mobile_supplier.chrome.driver") {
                TextField("supplier_portal.exceptions.early_complete.text.driver_id", text: $vm.driverId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
            }
            Section {
                Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.approving" : "mobile_supplier.chrome.approve_early_complete")) {
                    Task { await viewModel.approve() }
                }
                .frame(minHeight: 44)
                .disabled(viewModel.busy || viewModel.driverId.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            if let error = viewModel.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            if let success = viewModel.success {
                Section { Text(success).foregroundStyle(SupplierTheme.success) }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("supplier_portal.exceptions.early_complete.text.early_route_complete")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            viewModel.handleReconnect()
        }
    }
}
