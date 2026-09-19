import SwiftUI

struct EarlyCompleteView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var driverId = ""
    @State private var busy = false
    @State private var error: String?
    @State private var success: String?

    var body: some View {
        Form {
            Section {
                Text("supplier_portal.residual.text.approve_a_driver_request_to_finish_the_route_before_all_stops_ar")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Section("Driver") {
                TextField("supplier_portal.exceptions.early_complete.text.driver_id", text: $driverId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
            }
            Section {
                Button(busy ? "Approving…" : "Approve early complete") {
                    Task { await approve() }
                }
                .disabled(busy || driverId.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            if let error {
                Section { Text(error).foregroundStyle(.red) }
            }
            if let success {
                Section { Text(success).foregroundStyle(.green) }
            }
        }
        .navigationTitle("supplier_portal.exceptions.early_complete.text.early_route_complete")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busy {
                busy = false
                error = "Connection restored — verify approval status before retrying."
            }
        }
    }

    private func approve() async {
        let trimmed = driverId.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else {
            error = "Driver ID is required."
            return
        }
        busy = true
        error = nil
        success = nil
        defer { busy = false }
        do {
            try await SupplierOperationsService.approveEarlyComplete(driverId: trimmed)
            success = "Early route complete approved for driver \(trimmed.prefix(12))…"
            driverId = ""
        } catch {
            self.error = error.localizedDescription
        }
    }
}
