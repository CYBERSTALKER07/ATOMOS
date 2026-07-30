import SwiftUI

enum OrderMutationAction: String, Identifiable {
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
    @State private var recommendations: [TruckRecommendation] = []
    @State private var showReassignSheet = false
    @State private var selectedDriverId: String?
    @State private var selectedManifestId: String?
    @State private var isPartialReassign = false

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
                ResponsiveGridContentWrapper {
                    Section("Summary") {
                        LabeledContent("State", value: order.state)
                        LabeledContent("Total", value: "\(order.totalUzs.formatted()) UZS")
                        LabeledContent("Retailer", value: order.retailerName.isEmpty ? "—" : order.retailerName)
                        LabeledContent("Order ID", value: order.orderId)
                            .font(.caption.monospaced())
                    }
                    OrderOpsActions(
                        state: order.state,
                        proposeDate: $proposeDate,
                        reasonInput: $reasonInput,
                        showProposeSheet: $showProposeSheet,
                        pendingAction: $pendingAction,
                        mutating: mutating,
                        onLoadRecommendations: { loadRecommendations() }
                    )
                    OrderLineItems(lineItems: order.lineItems)
                }
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
        .sheet(isPresented: $showReassignSheet) {
            NavigationStack {
                Form {
                    Section("Select Vehicle") {
                        ForEach(recommendations, id: \.driverId) { rec in
                            Button {
                                selectedDriverId = rec.driverId
                                selectedManifestId = rec.toRoute.isEmpty ? nil : rec.toRoute
                            } label: {
                                HStack {
                                    VStack(alignment: .leading) {
                                        Text(rec.driverName).foregroundColor(.primary)
                                        Text("\(rec.vehicleClass) • \(rec.licensePlate)").font(.caption).foregroundColor(.secondary)
                                    }
                                    Spacer()
                                    if selectedDriverId == rec.driverId {
                                        Image(systemName: "checkmark").foregroundColor(.blue)
                                    }
                                }
                            }
                        }
                    }
                    Section("Options") {
                        Toggle("Partial Reassignment", isOn: $isPartialReassign)
                        TextField("Reason (optional)", text: $reasonInput, axis: .vertical)
                            .lineLimit(2...4)
                    }
                }
                .navigationTitle("Reassign Order")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showReassignSheet = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Confirm") { submitReassign() }
                            .disabled(selectedDriverId == nil || mutating)
                    }
                }
            }
            .presentationDetents([.medium, .large])
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

    private func loadRecommendations() {
        mutating = true
        statusMessage = "Loading recommendations..."
        Task {
            defer { mutating = false }
            do {
                let response = try await WarehouseOperationsService.recommendReassign(orderId: orderId)
                recommendations = response.recommendations
                if recommendations.isEmpty {
                    statusMessage = "No available vehicles found for reassignment"
                } else {
                    statusMessage = nil
                    showReassignSheet = true
                }
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }

    private func submitReassign() {
        guard let driverId = selectedDriverId else { return }
        mutating = true
        statusMessage = "Reassigning..."
        Task {
            defer { mutating = false }
            do {
                let request = ReassignOrderRequest(
                    orderId: orderId,
                    toDriverId: driverId,
                    reason: reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "warehouse-reassign" : reasonInput,
                    toManifestId: selectedManifestId,
                    isPartial: isPartialReassign
                )
                try await WarehouseOperationsService.reassignOrder(request, idempotencyKey: UUID().uuidString)
                statusMessage = "Order reassigned successfully"
                showReassignSheet = false
                reasonInput = ""
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
