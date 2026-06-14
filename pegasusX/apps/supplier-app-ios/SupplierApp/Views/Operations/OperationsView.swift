import SwiftUI

private let broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"]

struct OperationsView: View {
    @State private var loading = true
    @State private var empathy: SupplierEmpathyAdoption?
    @State private var title = ""
    @State private var bodyText = ""
    @State private var broadcastRole = "ALL"
    @State private var orderId = ""
    @State private var bypassReason = ""
    @State private var bypassToken: String?
    @State private var showBypassConfirm = false
    @State private var broadcasting = false
    @State private var replenishing = false
    @State private var bypassing = false
    @State private var statusMessage: String?

    var body: some View {
        Form {
            if loading && empathy == nil {
                Section {
                    SupplierLoadingView(title: "Loading operations…")
                }
            }

            if let empathy {
                Section("Empathy adoption") {
                    LabeledContent("Predictions", value: "\(empathy.totalPredictions)")
                    LabeledContent("Waiting", value: "\(empathy.predictionsWaiting)")
                    LabeledContent("Fired", value: "\(empathy.predictionsFired)")
                    LabeledContent("Dormant", value: "\(empathy.predictionsDormant)")
                    LabeledContent("Rejected", value: "\(empathy.predictionsRejected)")
                }
            }

            Section("Operator broadcast") {
                TextField("Title", text: $title)
                TextField("Message", text: $bodyText, axis: .vertical)
                    .lineLimit(3...6)
                Picker("Target role", selection: $broadcastRole) {
                    ForEach(broadcastRoles, id: \.self) { role in
                        Text(role).tag(role)
                    }
                }
                Button(broadcasting ? "Sending…" : "Send broadcast") {
                    Task { await sendBroadcast() }
                }
                .disabled(broadcasting || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    || bodyText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }

            Section("Replenishment") {
                Text("Opens a warehouse supply request against your primary active warehouse.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                Button(replenishing ? "Triggering…" : "Trigger replenishment") {
                    runReplenishment()
                }
                .disabled(replenishing)
            }

            Section("Payment bypass") {
                TextField("Order ID (AWAITING_PAYMENT)", text: $orderId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                TextField("Reason (optional)", text: $bypassReason)
                Button(bypassing ? "Issuing…" : "Issue bypass token") {
                    showBypassConfirm = true
                }
                .disabled(bypassing || orderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if let bypassToken {
                    Text("Driver token: \(bypassToken)")
                        .font(.footnote.monospaced())
                }
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
        .task { await loadEmpathy() }
        .confirmationDialog(
            "Issue payment bypass?",
            isPresented: $showBypassConfirm,
            titleVisibility: .visible
        ) {
            Button("Issue token", role: .destructive) {
                Task { await issueBypass() }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Order must be AWAITING_PAYMENT. Driver receives a one-time bypass token.")
        }
    }

    @MainActor
    private func loadEmpathy() async {
        loading = true
        defer { loading = false }
        do {
            empathy = try await SupplierOperationsService.empathyAdoption()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    @MainActor
    private func sendBroadcast() async {
        broadcasting = true
        statusMessage = nil
        defer { broadcasting = false }
        do {
            let response = try await SupplierOperationsService.broadcast(
                SupplierBroadcastRequest(
                    title: title.trimmingCharacters(in: .whitespacesAndNewlines),
                    body: bodyText.trimmingCharacters(in: .whitespacesAndNewlines),
                    role: broadcastRole
                )
            )
            statusMessage = "Broadcast · \(response.status)"
            title = ""
            bodyText = ""
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func runReplenishment() {
        replenishing = true
        statusMessage = nil
        Task {
            defer { replenishing = false }
            do {
                let response = try await SupplierOperationsService.triggerReplenishment()
                await MainActor.run { statusMessage = "Replenishment · \(response.status)" }
            } catch {
                await MainActor.run { statusMessage = error.localizedDescription }
            }
        }
    }

    @MainActor
    private func issueBypass() async {
        bypassing = true
        bypassToken = nil
        statusMessage = nil
        defer { bypassing = false }
        do {
            let response = try await SupplierOperationsService.issuePaymentBypass(
                PaymentBypassRequest(
                    orderId: orderId.trimmingCharacters(in: .whitespacesAndNewlines),
                    reason: bypassReason.trimmingCharacters(in: .whitespacesAndNewlines)
                )
            )
            bypassToken = response.bypassToken
            statusMessage = "Bypass issued for \(response.orderId)"
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
