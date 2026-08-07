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

                        ResponsiveGridContentWrapper {
                            Section("Treasury surfaces") {
                                NavigationLink { LedgerView() } label: {
                                    Label("mobile_supplier.ui.payment_ledger", systemImage: "banknote")
                                }
                                NavigationLink { PaymentsView() } label: {
                                    Label("portal.nav.payments", systemImage: "creditcard")
                                }
                                NavigationLink { ChargebacksView() } label: {
                                    Label("portal.nav.chargebacks", systemImage: "exclamationmark.bubble")
                                }
                                NavigationLink { ClaimChargebacksView() } label: {
                                    Label("portal.nav.claim_chargebacks", systemImage: "list.bullet.rectangle")
                                }
                                NavigationLink { ClaimsView() } label: {
                                    Label("supplier_portal.exceptions.claims.text.claims_queue", systemImage: "exclamationmark.triangle.fill")
                                }
                                NavigationLink { ReconciliationView() } label: {
                                    Label("portal.nav.reconciliation", systemImage: "scalemass")
                                }
                                NavigationLink { CashReconciliationsView() } label: {
                                    Label("portal.nav.cash_reconciliations", systemImage: "dollarsign.circle")
                                }
                                NavigationLink { CreditNotesListView() } label: {
                                    Label("portal.nav.credit_notes", systemImage: "doc.text")
                                }
                                NavigationLink { CreditProfilesView() } label: {
                                    Label("mobile_supplier.ui.credit_profiles", systemImage: "creditcard.and.123")
                                }
                                NavigationLink { RoutePerformanceListView() } label: {
                                    Label("portal.nav.route_performance", systemImage: "map")
                                }
                                NavigationLink { EarningsView() } label: {
                                    Label("portal.nav.earnings", systemImage: "chart.line.uptrend.xyaxis")
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
        .navigationTitle("portal.nav.treasury")
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
