import SwiftUI
import UIKit

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
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if let order {
                let showReceipt = ["COMPLETED", "FISCALIZING", "FISCAL_FAILED"].contains(order.state.uppercased())
                ResponsiveGridContentWrapper {
                    Section("Summary") {
                        LabeledContent("State", value: order.state)
                        LabeledContent("Total", value: "\(order.totalUzs.formatted()) UZS")
                        LabeledContent("Retailer", value: order.retailerName.isEmpty ? "—" : order.retailerName)
                        LabeledContent("Order ID", value: order.orderId)
                            .font(.caption.monospaced())
                        if showReceipt {
                            Button("mobile_warehouse.ui.view_pegasus_receipt") {
                                Task { await openReceipt() }
                            }
                        }
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
        .navigationTitle("mobile_warehouse.ui.order_detail")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
        .sheet(isPresented: $showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                    Section {
                        TextField("supplier_portal.admin.control_center.field.reason", text: $reasonInput, axis: .vertical)
                            .lineLimit(3...5)
                    } footer: {
                        Text("mobile_warehouse.ui.the_retailer_is_notified_and_can_accept_or_reject_the_new_date")
                    }
                }
                .navigationTitle("mobile_warehouse.ui.propose_new_date")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { showProposeSheet = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_warehouse.ui.send") { submitPropose() }
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
                                        Text(L10n.format("mobile_warehouse.ui.vehicleclass_licenseplate", "\(rec.vehicleClass)", "\(rec.licensePlate)")).font(.caption).foregroundColor(.secondary)
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
                        TextField("warehouse_portal.inventory.text.reason_optional", text: $reasonInput, axis: .vertical)
                            .lineLimit(2...4)
                    }
                }
                .navigationTitle("supplier_portal.orders.re_dispatch_dialog.text.reassign_order")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { showReassignSheet = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_warehouse.ui.confirm") { submitReassign() }
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

    private func openReceipt() async {
        do {
            let meta: OrderReceiptMeta = try await APIClient.shared.get(
                "v1/warehouse/orders/\(orderId)/receipt",
                query: ["format": "json"]
            )
            let raw = [meta.htmlUrl, meta.qrUrl, meta.pdfUrl]
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .first { !$0.isEmpty }
            if let raw, let url = URL(string: raw) {
                await MainActor.run { UIApplication.shared.open(url) }
            } else {
                await MainActor.run { statusMessage = "Receipt unavailable" }
            }
        } catch {
            await MainActor.run { statusMessage = error.localizedDescription }
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
