import SwiftUI

struct InsightsView: View {
    @State private var vm = InsightsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.insights.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
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
            } else if vm.insights.isEmpty {
                ContentUnavailableView(
                    "No Insights",
                    systemImage: "chart.bar.xaxis",
                    description: Text("warehouse_portal.residual.text.no_replenishment_insights")
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(Array(vm.insights.enumerated()), id: \.element.id) { index, insight in
                        InsightRow(insight: insight)
                            .staggeredAppear(index: index)
                    }
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("mobile_factory.ui.replenishment_insights")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel("Refresh replenishment insights")
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
