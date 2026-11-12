import SwiftUI

struct TransferDetailView: View {
    let transferId: String
    @State private var vm: TransferDetailViewModel

    init(transferId: String) {
        self.transferId = transferId
        _vm = State(initialValue: TransferDetailViewModel(transferId: transferId))
    }

    var body: some View {
        Group {
            if vm.loading && vm.transfer == nil {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.transfer == nil {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.selection()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            } else if let transfer = vm.transfer {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        if let error = vm.error {
                            FactoryRuntimeBanner(tone: .warning, message: error)
                        }

                        TransferOverviewCard(transfer: transfer)

                        LazyVGrid(
                            columns: [
                                GridItem(.flexible(), spacing: LabTheme.spacingMD),
                                GridItem(.flexible(), spacing: LabTheme.spacingMD),
                            ],
                            spacing: LabTheme.spacingMD
                        ) {
                            TransferSummaryCard(label: "Items", value: "\(transfer.totalItems)")
                            TransferSummaryCard(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
                        }

                        if transfer.state == "APPROVED" || transfer.state == "LOADING" {
                            HStack(spacing: LabTheme.spacingMD) {
                                if transfer.state == "APPROVED" {
                                    Button("mobile_factory.ui.start_loading") {
                                        Task { await vm.transition(to: "LOADING") }
                                    }
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .buttonStyle(.borderedProminent)
                                    .accessibilityLabel("Start loading transfer")
                                }

                                if transfer.state == "LOADING" {
                                    Button("mobile_factory.ui.mark_dispatched") {
                                        Task { await vm.transition(to: "DISPATCHED") }
                                    }
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .buttonStyle(.borderedProminent)
                                    .accessibilityLabel("Mark transfer dispatched")
                                }
                            }
                            .disabled(vm.transitioning)
                        } else {
                            Text("mobile_factory.ui.no_manual_transition_is_available_for_the_current_state")
                                .font(.body)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(LabTheme.spacingLG)
                                .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
                        }

                        Divider()

                        Text("warehouse_portal.supply_requests._id_.text.items")
                            .font(.headline)

                        ForEach(Array(transfer.items.enumerated()), id: \.element.id) { index, item in
                            let itemTitle = item.productName.isEmpty ? String(item.productId.prefix(8)) : item.productName
                            VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(itemTitle)
                                        .font(.subheadline.bold())
                                    Text(String(item.productId.prefix(8)))
                                        .font(.footnote)
                                        .foregroundStyle(.secondary)
                                }

                                HStack(spacing: LabTheme.spacingSM) {
                                    TransferSummaryCard(label: "Qty", value: "\(item.quantity)")
                                    TransferSummaryCard(label: "Available", value: "\(item.quantityAvailable)")
                                    TransferSummaryCard(label: "Volume", value: String(format: "%.1fL", item.unitVolumeL))
                                }
                            }
                            .labCard()
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(itemTitle), quantity \(item.quantity), available \(item.quantityAvailable)")
                            .staggeredAppear(index: index)
                        }

                        if !transfer.notes.isEmpty {
                            Divider()
                            Text("factory_portal.transfers._id_.text.notes")
                                .font(.headline)
                            Text(transfer.notes)
                                .font(.body)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(LabTheme.spacingLG)
                                .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
                        }
                    }
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("factory_portal.payload_override.payload_list.text.transfer")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh transfer details")
            }
        }
        .task(id: transferId) {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}

struct TransferDetailTag: View {
    let text: String
    var emphasized = true

    var body: some View {
        Text(text)
            .font(.footnote.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .background(emphasized ? LabTheme.fill : LabTheme.tertiaryBackground, in: Capsule())
    }
}
