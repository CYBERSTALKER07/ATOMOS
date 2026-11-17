import SwiftUI

struct SupplyRequestsView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var vm = SupplyRequestsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                FactoryLoadingState(
                    title: "Loading supply requests",
                    message: "Fetching the live warehouse demand queue for this factory."
                )
            } else if let error = vm.error {
                FactoryStateView(
                    kind: vm.realtimeStatus == .offline ? .offline : .error,
                    headline: vm.realtimeStatus == .offline ? "Supply queue unavailable offline" : "Unable to load supply requests",
                    message: error,
                    actionTitle: "Retry",
                    action: { Task { await vm.load() } }
                )
            } else if vm.filteredRequests.isEmpty {
                FactoryStateView(
                    kind: vm.selectedFilter == "ALL" ? .empty : .noResults,
                    headline: vm.selectedFilter == "ALL"
                        ? "No supply requests"
                        : "No \(vm.selectedFilter.replacingOccurrences(of: "_", with: " ").lowercased()) requests",
                    message: vm.selectedFilter == "ALL"
                        ? "Warehouse demand will appear here as soon as requests reach this factory queue."
                        : "Adjust the active filter or wait for the next queue refresh.",
                    actionTitle: vm.selectedFilter == "ALL" ? nil : "Clear Filter",
                    action: vm.selectedFilter == "ALL" ? nil : {
                        PegasusHaptics.selection()
                        vm.selectedFilter = "ALL"
                    }
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        SupplySummaryCard(
                            total: vm.requests.count,
                            visible: vm.filteredRequests.count,
                            runtimeStatus: vm.runtimeStatus,
                            runtimeTone: vm.runtimeTone
                        )
                        SupplyFilterAndModeBar(
                            selectedFilter: $vm.selectedFilter,
                            viewMode: $vm.viewMode
                        )

                        if vm.viewMode == "BOARD" {
                            SupplyBoard(
                                requests: vm.filteredRequests,
                                transitioningID: vm.transitioningID,
                                qcById: vm.qcById,
                                onAction: { request, action in
                                    Task { await vm.handleAction(request: request, action: action) }
                                },
                                onQC: { request, result in
                                    Task { await vm.handleQC(request: request, result: result) }
                                }
                            )
                        } else {
                            LazyVStack(spacing: LabTheme.spacingSM) {
                                ForEach(Array(vm.filteredRequests.enumerated()), id: \.element.id) { index, request in
                                    SupplyRequestCard(
                                        request: request,
                                        transitioning: vm.transitioningID == request.id,
                                        qcResult: vm.qcById[request.id] ?? "",
                                        onAction: { action in
                                            Task { await vm.handleAction(request: request, action: action) }
                                        },
                                        onQC: { result in
                                            Task { await vm.handleQC(request: request, result: result) }
                                        }
                                    )
                                    .staggeredAppear(index: index)
                                }
                            }
                        }
                    }
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.supply_requests")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load(background: !vm.requests.isEmpty) }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh supply requests")
            }
        }
        .task {
            await vm.runPollingLoop()
        }
        .onChange(of: scenePhase) { _, newPhase in
            if newPhase == .active {
                Task { await vm.load(background: !vm.requests.isEmpty) }
            }
        }
        .onDisappear {
            vm.stopRealtime()
        }
        .sheet(item: $vm.fulfillModalItem) { item in
            SupplyFulfillSheet(
                item: item,
                onCancel: {
                    PegasusHaptics.selection()
                    vm.fulfillModalItem = nil
                },
                onConfirm: {
                    Task { await vm.confirmFulfill(item) }
                }
            )
            .presentationDetents([.medium])
        }
    }
}

struct SupplyFulfillSheet: View {
    let item: SupplyFulfillModalItem
    let onCancel: () -> Void
    let onConfirm: () -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("Fulfill decision") {
                    Text(L10n.format("mobile_factory.ui.warehousename_transfermode", "\(item.options.warehouseName)", "\(item.options.transferMode)"))
                    Text(item.options.outcomeInternal)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    Text(item.options.outcomeTruck)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    if let eta = item.options.linkedDriverETA {
                        Text(L10n.format("mobile_factory.ui.driver_eta_eta_2", "\(eta)"))
                            .font(.footnote)
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("factory_portal.supply_requests.text.confirm_fulfill")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(action: onCancel) {
                        Text("common.action.cancel")
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Cancel fulfill")
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: onConfirm) {
                        Text("mobile_factory.ui.confirm")
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Confirm fulfill")
                }
            }
        }
    }
}
