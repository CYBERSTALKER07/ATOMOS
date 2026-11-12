import SwiftUI

struct TransferListView: View {
    var initialFilter: String? = nil
    @State private var vm = TransferListViewModel()

    var body: some View {
        @Bindable var vm = vm

        VStack(spacing: 0) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: LabTheme.spacingSM) {
                    ForEach(TransferListViewModel.filters, id: \.self) { filter in
                        let selected = vm.selectedFilter == filter
                        Button {
                            PegasusHaptics.selection()
                            vm.selectedFilter = filter
                        } label: {
                            Text(filter)
                                .font(.footnote.bold())
                                .padding(.horizontal, 12)
                                .padding(.vertical, 6)
                                .frame(minWidth: 44, minHeight: 44)
                                .background(selected ? LabTheme.label : Color.clear, in: Capsule())
                                .foregroundStyle(selected ? Color(.systemBackground) : LabTheme.label)
                                .overlay(Capsule().stroke(.quaternary))
                        }
                        .buttonStyle(PressableButtonStyle())
                        .accessibilityLabel("Filter transfers by \(filter)")
                        .accessibilityAddTraits(selected ? .isSelected : [])
                    }
                }
                .padding(.horizontal)
                .padding(.vertical, LabTheme.spacingSM)
            }

            Divider()

            if vm.loading && vm.transfers.isEmpty {
                FactoryLoadingView(
                    title: "Loading transfers",
                    message: "Fetching the factory transfer queue and lifecycle states."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.transfers.isEmpty {
                FactoryStateView(
                    kind: vm.selectedFilter == "ALL" ? .empty : .noResults,
                    headline: vm.selectedFilter == "ALL" ? "No transfers" : "No \(vm.selectedFilter) transfers",
                    message: vm.selectedFilter == "ALL"
                        ? "There are no transfers in the factory queue right now."
                        : "Adjust the filter or wait for the next queue refresh.",
                    actionTitle: vm.selectedFilter == "ALL" ? nil : "Show All",
                    action: vm.selectedFilter == "ALL" ? nil : { vm.selectedFilter = "ALL" }
                )
            } else {
                List {
                    Section {
                        TransferListSummary(count: vm.transfers.count, selectedFilter: vm.selectedFilter)
                            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                            .listRowBackground(Color.clear)
                    }

                    Section {
                        ForEach(vm.transfers) { transfer in
                            NavigationLink {
                                TransferDetailView(transferId: transfer.id)
                            } label: {
                                TransferRow(transfer: transfer)
                            }
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.transfers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_factory.ui.create", systemImage: "plus") {
                    PegasusHaptics.selection()
                    vm.showCreateTransfer = true
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Create transfer")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh transfers")
            }
        }
        .sheet(isPresented: $vm.showCreateTransfer) {
            CreateTransferView { transferId in
                vm.selectedTransferID = transferId
                Task { await vm.load() }
            }
        }
        .task(id: initialFilter) {
            vm.applyInitialFilter(initialFilter)
        }
        .task(id: vm.selectedFilter) {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}
