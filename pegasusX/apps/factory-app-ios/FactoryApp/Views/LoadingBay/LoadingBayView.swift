import SwiftUI

struct LoadingBayView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var transfers: [Transfer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var dispatching = false
    @State private var handoffEvents: [FactoryPulseEvent] = []
    @State private var handoffLoading = true
    @State private var handoffError: String?

    private var approved: [Transfer] { transfers.filter { $0.state == "APPROVED" } }
    private var loadingState: [Transfer] { transfers.filter { $0.state == "LOADING" } }
    private var dispatched: [Transfer] { transfers.filter { $0.state == "DISPATCHED" } }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    FactoryLoadingState(
                        title: "Loading loading bay",
                        message: "Fetching approved, loading, and dispatched transfers at the bay."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if transfers.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No transfers at the bay",
                        message: "No transfers are active in the loading bay right now."
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            LoadingBayOverviewCard(
                                readyCount: approved.count,
                                loadingCount: loadingState.count,
                                dispatchedCount: dispatched.count
                            )

                            FactoryHandoffTimelineSection(events: handoffEvents, loading: handoffLoading, error: handoffError)

                            BaySection(
                                title: "Ready for Loading",
                                count: approved.count,
                                transfers: approved,
                                emptyMessage: "No approved transfers are waiting at the bay."
                            )
                            BaySection(
                                title: "Now Loading",
                                count: loadingState.count,
                                transfers: loadingState,
                                emptyMessage: "Nothing is actively loading right now."
                            )
                            BaySection(
                                title: "Dispatched",
                                count: dispatched.count,
                                transfers: dispatched,
                                emptyMessage: "No transfers have been dispatched in the current view."
                            )
                        }
                        .labReadableWidth()
                        .padding()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.loading_bay")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }

                if !loadingState.isEmpty {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button(dispatching ? "Dispatching" : "Batch Dispatch", systemImage: "truck.box") {
                            Task { await batchDispatch() }
                        }
                        .disabled(dispatching)
                    }
                }
            }
            .task { await load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard let eventType = event.eventType else { return }
                        guard eventType == .transferUpdate || eventType == .manifestUpdate else { return }
                        Task { await load() }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil

        do {
            let response = try await FactoryService.loadingBayTransfers()
            transfers = response.transfers
        } catch {
            self.error = error.localizedDescription
        }

        handoffLoading = true
        handoffError = nil
        do {
            let pulse = try await FactoryService.pulse()
            let result = PulseHonesty.apply(
                ok: true,
                incoming: FactoryHandoffPulseSupport.filter(pulse.events),
                previous: handoffEvents
            )
            handoffEvents = result.events
            handoffError = result.error
        } catch {
            let result = PulseHonesty.apply(ok: false, incoming: nil, previous: handoffEvents)
            handoffEvents = result.events
            handoffError = result.error
        }
        handoffLoading = false

        loading = false
    }

    @MainActor
    private func batchDispatch() async {
        dispatching = true

        do {
            let ids = loadingState.map(\.id)
            let result = try await FactoryService.dispatch(transferIds: ids)
            if result.createdManifestCount == 0 {
                self.error = "No transfers to dispatch"
            }
            await load()
        } catch {
            self.error = error.localizedDescription
        }

        dispatching = false
    }
}

