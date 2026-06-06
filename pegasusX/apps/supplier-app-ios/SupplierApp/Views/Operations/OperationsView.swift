import SwiftUI

struct OperationsView: View {
    @State private var busy = false
    @State private var statusMessage: String?

    var body: some View {
        Form {
            Section {
                Text("Supplier operator actions. Broadcast and payment-bypass remain portal-only in v1.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            Section("Actions") {
                Button("Trigger replenishment") {
                    run(label: "Replenishment") {
                        try await SupplierOperationsService.triggerReplenishment()
                    }
                }
                .disabled(busy)
            }
            if let statusMessage {
                Section {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .navigationTitle("Operations")
    }

    private func run(label: String, _ block: @escaping () async throws -> SupplierReplenishmentTriggerResponse) {
        busy = true
        statusMessage = nil
        Task {
            defer { busy = false }
            do {
                let response = try await block()
                statusMessage = "\(label) · \(response.status)"
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
