import SwiftUI

struct TreasuryHubView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = TreasuryViewModel()

    var body: some View {
        ScrollView {
            Group {
                if vm.loading && vm.earnings == nil {
                    SupplierLoadingView(title: "Loading treasury…")
                } else if let error = vm.error {
                    SupplierErrorView(message: error) { Task { await vm.load() } }
                } else {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                        SupplierSectionHeader(
                            title: "Treasury",
                            subtitle: "Payments, settlement authority, and reconciliation"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 160), spacing: SupplierTheme.spacingMD)],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            KpiTile(title: "Month earnings", value: vm.monthLabel, systemImage: "chart.line.uptrend.xyaxis", tint: SupplierTheme.success)
                            KpiTile(title: "Ledger rows", value: "\(vm.ledgerCount)", systemImage: "banknote", tint: .accentColor)
                            KpiTile(title: "Settlement groups", value: "\(vm.settlementRows)", systemImage: "building.columns", tint: SupplierTheme.warning)
                            KpiTile(title: "Mismatches", value: "\(vm.mismatchCount)", systemImage: "scalemass", tint: vm.mismatchCount > 0 ? SupplierTheme.destructive : SupplierTheme.success)
                        }

                        List {
                            Section("Treasury surfaces") {
                                NavigationLink { LedgerView() } label: {
                                    Label("Payment ledger", systemImage: "banknote")
                                }
                                NavigationLink { PaymentsView() } label: {
                                    Label("Payments", systemImage: "creditcard")
                                }
                                NavigationLink { ChargebacksView() } label: {
                                    Label("Chargebacks", systemImage: "exclamationmark.bubble")
                                }
                                NavigationLink { ReconciliationView() } label: {
                                    Label("Reconciliation", systemImage: "scalemass")
                                }
                                NavigationLink { EarningsView() } label: {
                                    Label("Earnings", systemImage: "chart.line.uptrend.xyaxis")
                                }
                            }
                        }
                        .frame(minHeight: 320)
                    }
                }
            }
            .supplierReadableWidth()
            .padding()
        }
        .background(SupplierTheme.background)
        .navigationTitle("Treasury")
        .task { await vm.load() }
        .refreshable { await vm.load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
    }
}
