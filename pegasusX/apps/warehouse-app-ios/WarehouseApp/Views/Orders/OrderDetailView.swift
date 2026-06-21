import SwiftUI

private enum OrderMutationAction: String, Identifiable {
    case reject
    case overflow

    var id: String { rawValue }
}

struct OrderDetailView: View {
    let orderId: String
    @State private var order: Order?
    @State private var loading = true
    @State private var error: String?
    @State private var mutating = false
    @State private var pendingAction: OrderMutationAction?
    @State private var reasonInput = ""
    @State private var proposeDate = Date()
    @State private var showProposeSheet = false
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else if let order {
                List {
                    Section("Summary") {
                        LabeledContent("State", value: order.state)
                        LabeledContent("Total", value: "\(order.totalUzs.formatted()) UZS")
                        LabeledContent("Retailer", value: order.retailerName.isEmpty ? "—" : order.retailerName)
                        LabeledContent("Order ID", value: order.orderId)
                            .font(.caption.monospaced())
                    }
                    if showOps(for: order.state) {
                        Section("Quick actions") {
                            if canProposeDate(order.state) {
                                DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                                Button("Propose new date") { showProposeSheet = true }
                                    .disabled(mutating || reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                            }
                            TextField("Reason (required for propose and cancel)", text: $reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                            if canOverflow(order.state) {
                                Button("Return to dispatch pool") { pendingAction = .overflow }
                                    .disabled(mutating)
                            }
                            if canReject(order.state) {
                                Button("Cancel order", role: .destructive) { pendingAction = .reject }
                                    .disabled(mutating)
                            }
                        }
                    }
                    Section("Line Items (\(order.lineItems.count))") {
                        ForEach(order.lineItems) { item in
                            HStack {
                                VStack(alignment: .leading) {
                                    Text(item.productName.isEmpty ? "Product" : item.productName)
                                        .font(.headline)
                                    Text("Qty: \(item.quantity)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text("\(item.unitPrice.formatted()) UZS")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Order detail")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
        .sheet(isPresented: $showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                    Section {
                        TextField("Reason", text: $reasonInput, axis: .vertical)
                            .lineLimit(3...5)
                    } footer: {
                        Text("The retailer is notified and can accept or reject the new date.")
                    }
                }
                .navigationTitle("Propose new date")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showProposeSheet = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Send") { submitPropose() }
                            .disabled(mutating || reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert(item: $pendingAction) { action in
            Alert(
                title: Text(alertTitle(for: action)),
                message: Text(reasonPrompt(for: action)),
                primaryButton: .default(Text(confirmLabel(for: action))) {
                    runMutation(action)
                },
                secondaryButton: .cancel()
            )
        }
        .safeAreaInset(edge: .bottom) {
            if let statusMessage {
                Text(statusMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                order = try await WarehouseService.order(id: orderId)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func submitPropose() {
        let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else {
            statusMessage = "Reason is required"
            return
        }
        mutating = true
        statusMessage = nil
        Task {
            defer { mutating = false }
            do {
                let response = try await WarehouseOperationsService.proposeOrderDelivery(
                    orderId: orderId,
                    proposedDeliveryDate: isoDeliveryDate(from: proposeDate),
                    reason: reason
                )
                statusMessage = "New delivery date proposed · retailer notified · \(response.status)"
                showProposeSheet = false
                reasonInput = ""
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }

    private func runMutation(_ action: OrderMutationAction) {
        mutating = true
        statusMessage = nil
        Task {
            defer { mutating = false }
            do {
                let response: WarehouseOrderMutationResponse
                switch action {
                case .reject:
                    let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !reason.isEmpty else {
                        statusMessage = "Reason is required to cancel"
                        return
                    }
                    response = try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reason)
                case .overflow:
                    response = try await WarehouseOperationsService.overflowOrder(orderId: orderId, reason: reasonInput.isEmpty ? nil : reasonInput)
                }
                statusMessage = "Order updated · \(response.status)"
                reasonInput = ""
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }

    private func showOps(for state: String) -> Bool {
        canProposeDate(state) || canReject(state) || canOverflow(state)
    }

    private func canProposeDate(_ state: String) -> Bool {
        let s = state.uppercased()
        let terminal = s == "COMPLETED" || s == "CANCELLED"
        let inFlight = s == "LOADED" || s == "IN_TRANSIT"
        return !terminal && !inFlight
    }

    private func canReject(_ state: String) -> Bool {
        let s = state.uppercased()
        return ["PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED"].contains(s)
    }

    private func canOverflow(_ state: String) -> Bool {
        ["LOADED", "IN_TRANSIT"].contains(state.uppercased())
    }

    private func alertTitle(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Cancel order?"
        case .overflow: return "Return to dispatch pool?"
        }
    }

    private func reasonPrompt(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Enter a cancellation reason in the field above before confirming."
        case .overflow: return "Optional reason can be entered above."
        }
    }

    private func confirmLabel(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Cancel order"
        case .overflow: return "Return to pool"
        }
    }

    private func isoDeliveryDate(from date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        let components = calendar.dateComponents([.year, .month, .day], from: date)
        var merged = DateComponents()
        merged.year = components.year
        merged.month = components.month
        merged.day = components.day
        merged.hour = 12
        merged.minute = 0
        merged.second = 0
        merged.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        let normalized = calendar.date(from: merged) ?? date
        return formatter.string(from: normalized)
    }
}
