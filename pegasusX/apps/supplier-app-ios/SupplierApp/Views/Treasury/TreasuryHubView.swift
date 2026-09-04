import SwiftUI
#if canImport(UIKit)
import UIKit
#endif

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
                                NavigationLink { PayoutsView() } label: {
                                    Label("portal.nav.payouts", systemImage: "banknote")
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

struct PayoutsView: View {
    @State private var rail: PayoutRailInfo?
    @State private var policy: SupplierPayoutPolicy?
    @State private var draftMode = "HQ_SUPPLIER"
    @State private var policyReason = ""
    @State private var batches: [PayoutBatch] = []
    @State private var loading = true
    @State private var error: String?
    @State private var status: String?
    @State private var periodStart = ""
    @State private var periodEnd = ""
    @State private var busy = false

    var body: some View {
        Form {
            if loading {
                Section { ProgressView("Loading payouts…") }
            } else if let error, batches.isEmpty, rail == nil {
                Section {
                    SupplierErrorView(message: error) { Task { await load() } }
                }
            } else {
                Section("Rail") {
                    Text(rail?.message.isEmpty == false ? (rail?.message ?? "") : "Bank-file rail: generate → export CSV → mark-paid. Not a live bank.")
                        .font(.footnote)
                    Text("Live rail: \(rail?.isLive == true ? "yes" : "no")")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Section("Payout policy") {
                    Text("Mode \(policy?.payoutMode ?? "HQ_SUPPLIER") · source \(policy?.source ?? "DEFAULT"). Does not enable a live PSP.")
                        .font(.footnote)
                    Picker("Mode", selection: $draftMode) {
                        Text("HQ_SUPPLIER").tag("HQ_SUPPLIER")
                        Text("WAREHOUSE_LOCAL").tag("WAREHOUSE_LOCAL")
                    }
                    TextField("Reason (required)", text: $policyReason)
                    Button("Save mode") { Task { await savePolicy() } }
                        .disabled(busy || policyReason.isEmpty)
                }
                Section("Generate period") {
                    TextField("Period start (YYYY-MM-DD)", text: $periodStart)
                    TextField("Period end (YYYY-MM-DD)", text: $periodEnd)
                    Button("Generate batch") { Task { await generate() } }
                        .disabled(busy || periodStart.isEmpty || periodEnd.isEmpty)
                }
                Section("Batches") {
                    if batches.isEmpty {
                        Text("No batches")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(batches) { batch in
                            VStack(alignment: .leading, spacing: 4) {
                                Text("\(batch.status) · \(batch.netPayoutMinor) \(batch.currency)")
                                Text("\(batch.periodStart) → \(batch.periodEnd)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                HStack {
                                    Button("Export CSV") { Task { await exportCSV(batch.batchId) } }
                                        .disabled(busy)
                                    Button("Mark paid") { Task { await markPaid(batch.batchId) } }
                                        .disabled(busy)
                                    Button("Dispatch live") { Task { await dispatchLive(batch.batchId) } }
                                        .disabled(busy)
                                }
                                .font(.caption)
                            }
                        }
                    }
                }
                if let status {
                    Section { Text(status).font(.footnote) }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.payouts")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            async let railTask = SupplierOperationsService.payoutRail()
            async let listTask = SupplierOperationsService.payoutBatches()
            async let policyTask = SupplierOperationsService.payoutPolicy()
            rail = try await railTask
            batches = try await listTask
            if let loaded = try? await policyTask {
                policy = loaded
                draftMode = loaded.payoutMode.isEmpty ? "HQ_SUPPLIER" : loaded.payoutMode
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    @MainActor
    private func savePolicy() async {
        busy = true
        defer { busy = false }
        do {
            let next = try await SupplierOperationsService.patchPayoutPolicy(
                SupplierPayoutPolicyPatch(payoutMode: draftMode, feePolicyVersion: nil, reason: policyReason)
            )
            policy = next
            draftMode = next.payoutMode.isEmpty ? draftMode : next.payoutMode
            policyReason = ""
            status = "Payout mode saved. Bank-file rail is unchanged (no_live_rail)."
        } catch {
            status = error.localizedDescription
        }
    }

    @MainActor
    private func generate() async {
        busy = true
        defer { busy = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            _ = try await SupplierOperationsService.generatePayoutBatch(
                PayoutBatchGenerateRequest(periodStart: periodStart, periodEnd: periodEnd),
                idempotencyKey: SupplierIdempotencyKeys.payoutGenerate(
                    scopeId: scope,
                    periodStart: periodStart,
                    periodEnd: periodEnd
                )
            )
            status = "Batch generated"
            await load(silent: true)
        } catch {
            status = error.localizedDescription
        }
    }

    @MainActor
    private func exportCSV(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let csv = try await SupplierOperationsService.exportPayoutBatch(batchId)
            #if canImport(UIKit)
            UIPasteboard.general.string = csv
            status = "CSV copied (\(csv.count) chars). Process at bank, then mark-paid."
            #else
            status = "CSV exported (\(csv.count) chars). Process at bank, then mark-paid."
            #endif
            await load(silent: true)
        } catch {
            status = error.localizedDescription
        }
    }

    @MainActor
    private func markPaid(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let resp = try await SupplierOperationsService.markPayoutBatchPaid(batchId)
            status = resp.message.isEmpty ? "Marked paid" : resp.message
            await load(silent: true)
        } catch {
            status = error.localizedDescription
        }
    }

    @MainActor
    private func dispatchLive(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let resp = try await SupplierOperationsService.dispatchPayoutBatch(batchId, live: true)
            if resp.code == "no_live_rail" || resp.error == "no_live_rail" {
                status = resp.message.isEmpty ? "no_live_rail — export CSV, then mark-paid" : resp.message
            } else {
                status = resp.message.isEmpty ? "Dispatch attempted" : resp.message
            }
            await load(silent: true)
        } catch {
            let text = error.localizedDescription
            status = text
        }
    }
}
