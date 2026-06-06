import SwiftUI

private enum OrderMutationAction: String, Identifiable {
    case delay
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
                    }
                    if showOps(for: order.state) {
                        Section("Warehouse actions") {
                            TextField("Reason (required for reject)", text: $reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                            if canDelay(order.state) {
                                Button("Delay order") { pendingAction = .delay }
                                    .disabled(mutating)
                            }
                            if canOverflow(order.state) {
                                Button("Return to dispatch pool") { pendingAction = .overflow }
                                    .disabled(mutating)
                            }
                            if canReject(order.state) {
                                Button("Reject order", role: .destructive) { pendingAction = .reject }
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
        .navigationTitle("Order Detail")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
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

    private func runMutation(_ action: OrderMutationAction) {
        mutating = true
        statusMessage = nil
        Task {
            defer { mutating = false }
            do {
                let response: WarehouseOrderMutationResponse
                switch action {
                case .delay:
                    response = try await WarehouseOperationsService.delayOrder(orderId: orderId, reason: reasonInput.isEmpty ? nil : reasonInput)
                case .reject:
                    let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !reason.isEmpty else {
                        statusMessage = "Reason is required to reject"
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
        canDelay(state) || canReject(state) || canOverflow(state)
    }

    private func canDelay(_ state: String) -> Bool {
        ["PENDING", "LOADED"].contains(state.uppercased())
    }

    private func canReject(_ state: String) -> Bool {
        ["PENDING", "LOADED", "IN_TRANSIT"].contains(state.uppercased())
    }

    private func canOverflow(_ state: String) -> Bool {
        ["LOADED", "IN_TRANSIT"].contains(state.uppercased())
    }

    private func alertTitle(for action: OrderMutationAction) -> String {
        switch action {
        case .delay: return "Mark order delayed?"
        case .reject: return "Reject order?"
        case .overflow: return "Return to dispatch pool?"
        }
    }

    private func reasonPrompt(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Enter a rejection reason in Notes before confirming."
        default: return "Optional reason can be entered from the detail screen after this ships."
        }
    }

    private func confirmLabel(for action: OrderMutationAction) -> String {
        switch action {
        case .delay: return "Delay"
        case .reject: return "Reject"
        case .overflow: return "Overflow"
        }
    }
}
