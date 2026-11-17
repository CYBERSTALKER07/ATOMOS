import SwiftUI

struct LoadingBayView: View {
    @State private var vm = LoadingBayViewModel()

    var body: some View {
        Group {
            if vm.loading {
                FactoryLoadingState(
                    title: "Loading loading bay",
                    message: "Fetching approved, loading, and dispatched transfers at the bay."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.transfers.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No transfers at the bay",
                    message: "No transfers are active in the loading bay right now."
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        LoadingBayOverviewCard(
                            readyCount: vm.approved.count,
                            loadingCount: vm.loadingState.count,
                            dispatchedCount: vm.dispatched.count
                        )

                        FactoryHandoffTimelineSection(
                            events: vm.handoffEvents,
                            loading: vm.handoffLoading,
                            error: vm.handoffError
                        )

                        BaySection(
                            title: "Ready for Loading",
                            count: vm.approved.count,
                            transfers: vm.approved,
                            emptyMessage: "No approved transfers are waiting at the bay."
                        )
                        BaySection(
                            title: "Now Loading",
                            count: vm.loadingState.count,
                            transfers: vm.loadingState,
                            emptyMessage: "Nothing is actively loading right now."
                        )
                        BaySection(
                            title: "Dispatched",
                            count: vm.dispatched.count,
                            transfers: vm.dispatched,
                            emptyMessage: "No transfers have been dispatched in the current view."
                        )
                    }
                    .labReadableWidth()
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.loading_bay")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh loading bay")
            }

            if !vm.loadingState.isEmpty {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(vm.dispatching ? "Dispatching" : "Batch Dispatch", systemImage: "truck.box") {
                        Task { await vm.batchDispatch() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(vm.dispatching)
                    .accessibilityLabel("Batch dispatch loading transfers")
                }
            }
        }
        .task {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}
