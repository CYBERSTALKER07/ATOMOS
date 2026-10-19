import SwiftUI

struct SupplyRequestDetailView: View {
    let requestId: String

    @State private var request: SupplyRequest?
    @State private var loading = true
    @State private var error: String?
    @State private var actionError: String?
    @State private var isProcessing = false

    var body: some View {
        Group {
            if loading {
                FactoryLoadingView(
                    title: "Loading request",
                    message: "Fetching details for \(requestId.prefix(8))"
                )
            } else if let error {
                FactoryErrorView(message: error) {
                    Task { await load() }
                }
            } else if let request {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        headerSection(request)

                        if let actionError {
                            Text(actionError)
                                .font(.caption)
                                .foregroundStyle(.red)
                                .padding()
                                .background(Color.red.opacity(0.1))
                                .cornerRadius(LabTheme.radiusMD)
                        }

                        actionSection(request)

                        detailsSection(request)
                    }
                    .padding()
                }
            }
        }
        .navigationTitle("mobile_factory.ui.request_details")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    Task { await load() }
                }
                .disabled(isProcessing)
            }
        }
        .task { await load() }
    }

    @ViewBuilder
    private func headerSection(_ request: SupplyRequest) -> some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            HStack {
                Text(request.id)
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                Spacer()
                FactoryStatusBadge(text: request.state)
            }
            Text(L10n.format("mobile_factory.ui.totalvolumevu_vu", "\(Int(request.totalVolumeVU))"))
                .font(.largeTitle.bold())
            Text(L10n.format("mobile_factory.ui.priority_priority", "\(request.priority)"))
                .font(.headline)
                .foregroundStyle(request.priority == "URGENT" ? .red : .primary)
        }
    }

    @ViewBuilder
    private func actionSection(_ request: SupplyRequest) -> some View {
        let actions = availableActions(for: request.state)
        
        if !actions.isEmpty {
            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                Text("retailer_desktop.stock.text.actions")
                    .font(.headline)
                
                HStack(spacing: LabTheme.spacingMD) {
                    ForEach(actions, id: \.self) { action in
                        Button {
                            Task { await performAction(action) }
                        } label: {
                            if isProcessing {
                                ProgressView()
                                    .padding(.horizontal)
                            } else {
                                Text(action)
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(isProcessing)
                    }
                }
            }
            .padding()
            .background(Color(uiColor: .secondarySystemBackground))
            .cornerRadius(LabTheme.radiusLG)
        }
    }

    @ViewBuilder
    private func detailsSection(_ request: SupplyRequest) -> some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            Text("mobile_factory.ui.details")
                .font(.headline)

            VStack(spacing: LabTheme.spacingSM) {
                detailRow("Warehouse ID", value: request.warehouseId)
                detailRow("Supplier ID", value: request.supplierId)
                if let due = request.requestedDeliveryDate {
                    detailRow("Requested Delivery", value: due)
                }
                detailRow("Created At", value: request.createdAt)
                if let updated = request.updatedAt {
                    detailRow("Updated At", value: updated)
                }
            }
            .padding()
            .background(Color(uiColor: .secondarySystemBackground))
            .cornerRadius(LabTheme.radiusLG)

            if !request.notes.isEmpty {
                Text("factory_portal.transfers._id_.text.notes")
                    .font(.headline)
                    .padding(.top, LabTheme.spacingSM)
                
                Text(request.notes)
                    .font(.body)
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color(uiColor: .secondarySystemBackground))
                    .cornerRadius(LabTheme.radiusLG)
            }
        }
    }

    private func detailRow(_ label: String, value: String) -> some View {
        HStack {
            Text(label)
                .foregroundStyle(.secondary)
            Spacer()
            Text(value)
                .font(.subheadline.monospaced())
        }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        do {
            let requests = try await FactoryService.supplyRequests()
            self.request = requests.first { $0.id == requestId }
            if self.request == nil {
                self.error = "Request not found."
            }
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    @MainActor
    private func performAction(_ action: String) async {
        isProcessing = true
        actionError = nil
        do {
            _ = try await FactoryService.transitionSupplyRequest(id: requestId, action: action)
            await load()
        } catch {
            self.actionError = error.localizedDescription
        }
        isProcessing = false
    }

    private func availableActions(for state: String) -> [String] {
        switch state.uppercased() {
        case "OPEN": return ["ACKNOWLEDGE"]
        case "ACKNOWLEDGED": return ["IN_PRODUCTION"]
        case "PRODUCTION": return ["READY"]
        case "READY": return ["FULFILL"]
        default: return []
        }
    }
}
