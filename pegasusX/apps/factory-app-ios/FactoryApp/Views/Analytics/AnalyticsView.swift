import SwiftUI

struct AnalyticsView: View {
    @State private var vm = AnalyticsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                FactoryLoadingState(
                    title: "Loading analytics",
                    message: "Fetching factory throughput, manifest pressure, and exception queue."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        FactorySectionHeader(
                            title: "Analytics overview",
                            subtitle: "Factory throughput, manifest pressure, and exception queue"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 150), spacing: LabTheme.spacingSM)],
                            spacing: LabTheme.spacingSM
                        ) {
                            KpiTile(
                                title: "Transfers Total",
                                value: "\(vm.overview.transfersTotal)",
                                systemImage: "arrow.left.arrow.right",
                                tint: .accentColor,
                                showChevron: false
                            )
                            KpiTile(
                                title: "Active Manifests",
                                value: "\(vm.overview.manifestsActive)",
                                systemImage: "list.clipboard",
                                tint: LabTheme.warning,
                                showChevron: false
                            )
                            KpiTile(
                                title: "Exception Queue",
                                value: "\(vm.overview.exceptionQueue)",
                                systemImage: "exclamationmark.triangle",
                                tint: LabTheme.destructive,
                                chip: vm.overview.exceptionQueue > 0 ? ("ALERT", LabTheme.destructive) : nil,
                                showChevron: false
                            )
                            KpiTile(
                                title: "Avg Lead Time (min)",
                                value: String(format: "%.1f", vm.overview.avgLeadTimeMins),
                                systemImage: "clock",
                                tint: LabTheme.secondaryLabel,
                                showChevron: false
                            )
                        }

                        if !vm.overview.dailyActivity.isEmpty {
                            FactorySectionHeader(
                                title: "7-day transfer activity",
                                subtitle: "Daily transfer volume"
                            )
                            ForEach(vm.overview.dailyActivity, id: \.date) { day in
                                HStack {
                                    Text(day.date)
                                    Spacer()
                                    Text(L10n.format("mobile_factory.ui.transfers_transfers", "\(day.transfers)"))
                                        .font(.subheadline.monospacedDigit())
                                }
                                .frame(minHeight: 44)
                                .labCard()
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(day.date): \(day.transfers) transfers")
                            }
                        }
                    }
                    .labReadableWidth()
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("mobile_factory.ui.analytics_overview")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh analytics")
            }
        }
        .task { await vm.load() }
    }
}
