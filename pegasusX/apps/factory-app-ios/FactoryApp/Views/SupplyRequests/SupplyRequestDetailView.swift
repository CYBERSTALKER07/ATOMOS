import SwiftUI

struct SupplyRequestDetailView: View {
    @State private var vm: SupplyRequestDetailViewModel

    init(requestId: String) {
        _vm = State(initialValue: SupplyRequestDetailViewModel(requestId: requestId))
    }

    var body: some View {
        Group {
            if vm.loading {
                FactoryLoadingView(
                    title: "Loading request",
                    message: "Fetching details for \(vm.requestId.prefix(8))"
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if let request = vm.request {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        headerSection(request)

                        if let actionError = vm.actionError {
                            Text(actionError)
                                .font(.caption)
                                .foregroundStyle(.red)
                                .padding()
                                .background(Color.red.opacity(0.1))
                                .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
                                .accessibilityLabel("Action error: \(actionError)")
                        }

                        actionSection(request)

                        detailsSection(request)
                    }
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("mobile_factory.ui.request_details")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .disabled(vm.isProcessing)
                .accessibilityLabel("Refresh request details")
            }
        }
        .task { await vm.load() }
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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Request \(request.id), state \(request.state), \(Int(request.totalVolumeVU)) volume units, priority \(request.priority)")
    }

    @ViewBuilder
    private func actionSection(_ request: SupplyRequest) -> some View {
        let actions = vm.availableActions(for: request.state)

        if !actions.isEmpty {
            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                Text("retailer_desktop.stock.text.actions")
                    .font(.headline)

                HStack(spacing: LabTheme.spacingMD) {
                    ForEach(actions, id: \.self) { action in
                        Button {
                            Task { await vm.performAction(action) }
                        } label: {
                            if vm.isProcessing {
                                ProgressView()
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .padding(.horizontal)
                            } else {
                                Text(action)
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .contentShape(Rectangle())
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(vm.isProcessing)
                        .accessibilityLabel("Perform action \(action)")
                    }
                }
            }
            .padding()
            .background(Color(uiColor: .secondarySystemBackground))
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusLG, style: .continuous))
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
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusLG, style: .continuous))

            if !request.notes.isEmpty {
                Text("factory_portal.transfers._id_.text.notes")
                    .font(.headline)
                    .padding(.top, LabTheme.spacingSM)

                Text(request.notes)
                    .font(.body)
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color(uiColor: .secondarySystemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusLG, style: .continuous))
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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
