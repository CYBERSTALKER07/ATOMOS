import SwiftUI

struct EarlyCompleteView: View {
    @State private var driverId = ""
    @State private var busy = false
    @State private var error: String?
    @State private var success: String?

    var body: some View {
        Form {
            Section {
                Text("Approve a driver request to finish the route before all stops are completed.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Section("Driver") {
                TextField("Driver ID", text: $driverId)
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
        .navigationTitle("Early route complete")
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
