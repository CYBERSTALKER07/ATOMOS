import SwiftUI

private let broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"]

struct OperationsView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var loading = true
    @State private var error: String?
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
    @State private var templateDate = ""

    var body: some View {
        NavigationStack {
            Group {
                if loading && empathy == nil && error == nil {
                    SupplierLoadingView(
                        title: "Loading operations",
                        message: "Fetching empathy adoption and operator tools."
                    )
                } else if let error, empathy == nil {
                    SupplierErrorView(message: error) {
                        Task { await loadEmpathy() }
                    }
                } else {
                    operationsForm
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Operations")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await loadEmpathy() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await loadEmpathy() }
            .refreshable { await loadEmpathy(silent: true) }
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
    }

    private var operationsForm: some View {
        Form {
            if let empathy {
                Section {
                    SupplierSectionHeader(
                        title: "Empathy adoption",
                        subtitle: "Prediction lifecycle for your network"
                    )
                }
                Section {
                    LabeledContent("Predictions", value: "\(empathy.totalPredictions)")
                    LabeledContent("Waiting", value: "\(empathy.predictionsWaiting)")
                    LabeledContent("Fired", value: "\(empathy.predictionsFired)")
                    LabeledContent("Dormant", value: "\(empathy.predictionsDormant)")
                    LabeledContent("Rejected", value: "\(empathy.predictionsRejected)")
                }
            }

            Section {
                SupplierSectionHeader(title: "Operator broadcast")
            }
            Section {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: SupplierTheme.spacingSM) {
                        ForEach(supplierBroadcastTemplates) { template in
                            Button(template.title) {
                                applyTemplate(template)
                            }
                            .buttonStyle(.bordered)
                            .font(.caption)
                        }
                    }
                }
                TextField("Closure / effective date (optional)", text: $templateDate)
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

            Section {
                SupplierSectionHeader(
                    title: "Replenishment",
                    subtitle: "Warehouse supply request against your primary node"
                )
            }
            Section {
                Button(replenishing ? "Triggering…" : "Trigger replenishment") {
                    runReplenishment()
                }
                .disabled(replenishing)
            }

            Section {
                SupplierSectionHeader(title: "Payment bypass")
            }
            Section {
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
    }

    @MainActor
    private func loadEmpathy(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            empathy = try await SupplierOperationsService.empathyAdoption()
        } catch {
            if !silent || empathy == nil {
                self.error = error.localizedDescription
            }
        }
    }

    private func applyTemplate(_ template: SupplierBroadcastTemplate) {
        title = template.title
        broadcastRole = template.defaultRole
        let date = templateDate.trimmingCharacters(in: .whitespacesAndNewlines)
        if template.body.contains("{date}") {
            bodyText = template.body.replacingOccurrences(of: "{date}", with: date.isEmpty ? "the selected date" : date)
        } else {
            bodyText = template.body
        }
    }

    @MainActor
    private func sendBroadcast() async {
        broadcasting = true
        statusMessage = nil
        defer { broadcasting = false }
        do {
            let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
            let trimmedBody = bodyText.trimmingCharacters(in: .whitespacesAndNewlines)
            let scopeId = tokenStore.supplierId ?? "supplier"
            let key = SupplierIdempotency.broadcast(
                scopeId: scopeId,
                role: broadcastRole,
                title: trimmedTitle,
                body: trimmedBody
            )
            let response = try await SupplierOperationsService.broadcast(
                SupplierBroadcastRequest(
                    title: trimmedTitle,
                    body: trimmedBody,
                    role: broadcastRole
                ),
                idempotencyKey: key
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
            let trimmedOrderId = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
            let reason = bypassReason.trimmingCharacters(in: .whitespacesAndNewlines)
            let key = SupplierIdempotency.paymentBypass(orderId: trimmedOrderId, reason: reason)
            let response = try await SupplierOperationsService.issuePaymentBypass(
                PaymentBypassRequest(
                    orderId: trimmedOrderId,
                    reason: reason
                ),
                idempotencyKey: key
            )
            bypassToken = response.bypassToken
            statusMessage = "Bypass issued for \(response.orderId)"
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
