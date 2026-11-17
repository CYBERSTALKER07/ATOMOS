import SwiftUI

struct TreasuryHubView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = TreasuryViewModel()

    var body: some View {
        ScrollView {
            Group {
                if vm.loading && vm.earnings == nil {
                    SupplierLoadingView(title: "mobile_supplier.loading.treasury")
                } else if let error = vm.error, vm.earnings == nil {
                    SupplierErrorView(message: error) { Task { await vm.load() } }
                } else {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                        SupplierSectionHeader(
                            title: "mobile_supplier.chrome.treasury",
                            subtitle: "mobile_supplier.chrome.treasury_subtitle"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 160), spacing: SupplierTheme.spacingMD)],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            KpiTile(title: "mobile_supplier.chrome.month_earnings", value: vm.monthLabel, systemImage: "chart.line.uptrend.xyaxis", tint: SupplierTheme.success)
                            KpiTile(title: "mobile_supplier.chrome.ledger_rows", value: "\(vm.ledgerCount)", systemImage: "banknote", tint: SupplierTheme.primaryAction)
                            KpiTile(title: "mobile_supplier.chrome.settlement_groups", value: "\(vm.settlementRows)", systemImage: "building.columns", tint: SupplierTheme.warning)
                            KpiTile(title: "mobile_supplier.chrome.mismatches", value: "\(vm.mismatchCount)", systemImage: "scalemass", tint: vm.mismatchCount > 0 ? SupplierTheme.destructive : SupplierTheme.success)
                        }

                        if !vm.migratedOpeningInvoices.isEmpty {
                            let ackCount = vm.migratedOpeningInvoices.filter { $0.ackState == "ACKNOWLEDGED" }.count
                            let disputeCount = vm.migratedOpeningInvoices.filter { $0.ackState == "DISPUTED" }.count
                            let pendingCount = vm.migratedOpeningInvoices.filter { $0.ackState == "PENDING_ACK" }.count

                            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                SupplierSectionHeader(title: "supplier_portal.migrations.pack_nasiya")

                                VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                    HStack(spacing: SupplierTheme.spacingSM) {
                                        SupplierStatusBadge(text: "MIGRATION")
                                        Text("Ack \(ackCount) · Pending \(pendingCount) · Disputed \(disputeCount)")
                                            .font(.caption.monospacedDigit())
                                            .foregroundStyle(SupplierTheme.secondaryLabel)
                                    }

                                    ForEach(vm.migratedOpeningInvoices.prefix(8)) { invoice in
                                        Divider()
                                        HStack(alignment: .top) {
                                            VStack(alignment: .leading, spacing: 2) {
                                                Text(invoice.externalId?.isEmpty == false ? invoice.externalId! : invoice.orderId)
                                                    .font(.subheadline.weight(.semibold))
                                                Text(invoice.retailerId)
                                                    .font(.caption)
                                                    .foregroundStyle(SupplierTheme.secondaryLabel)
                                            }
                                            Spacer()
                                            VStack(alignment: .trailing, spacing: 2) {
                                                Text(MoneyFormat.minor(invoice.principalMinor, currency: invoice.currency))
                                                    .font(.subheadline.bold().monospacedDigit())
                                                SupplierStatusBadge(text: invoice.ackState)
                                            }
                                        }
                                    }
                                }
                                .supplierCard()
                            }
                        }

                        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                            Text("mobile_supplier.chrome.treasury_surfaces")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                                .padding(.horizontal, SupplierTheme.spacingXS)

                            VStack(spacing: 0) {
                                TreasuryLinkRow(title: "mobile_supplier.ui.payment_ledger", systemImage: "banknote") { LedgerView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.payments", systemImage: "creditcard") { PaymentsView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.chargebacks", systemImage: "exclamationmark.bubble") { ChargebacksView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.claim_chargebacks", systemImage: "list.bullet.rectangle") { ClaimChargebacksView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "supplier_portal.exceptions.claims.text.claims_queue", systemImage: "exclamationmark.triangle.fill") { ClaimsView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.reconciliation", systemImage: "scalemass") { ReconciliationView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.cash_reconciliations", systemImage: "dollarsign.circle") { CashReconciliationsView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.credit_notes", systemImage: "doc.text") { CreditNotesListView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "mobile_supplier.ui.credit_profiles", systemImage: "creditcard.and.123") { CreditProfilesView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.route_performance", systemImage: "map") { RoutePerformanceListView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.earnings", systemImage: "chart.line.uptrend.xyaxis") { EarningsView() }
                                Divider().padding(.leading, 44)
                                TreasuryLinkRow(title: "portal.nav.payouts", systemImage: "banknote") { PayoutsView() }
                            }
                            .supplierCard()
                        }
                    }
                }
            }
            .supplierReadableWidth()
            .padding()
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.treasury")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
    }
}

private struct TreasuryLinkRow<Destination: View>: View {
    let title: LocalizedStringKey
    let systemImage: String
    @ViewBuilder let destination: () -> Destination

    var body: some View {
        NavigationLink(destination: destination) {
            HStack(spacing: SupplierTheme.spacingMD) {
                Image(systemName: systemImage)
                    .font(.body)
                    .foregroundStyle(SupplierTheme.primaryAction)
                    .frame(width: 24)
                Text(title)
                    .font(.body)
                    .foregroundStyle(SupplierTheme.label)
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(SupplierTheme.tertiaryLabel)
            }
            .padding(.vertical, SupplierTheme.spacingMD)
            .padding(.horizontal, SupplierTheme.spacingSM)
            .frame(minHeight: 44)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
