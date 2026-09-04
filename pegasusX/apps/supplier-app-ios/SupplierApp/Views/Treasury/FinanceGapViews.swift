import SwiftUI

struct CashReconciliationsView: View {
    @State private var rows: [CashReconciliationRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?

    var body: some View {
        List {
            if loading {
                ProgressView("Loading…")
            } else if let error {
                Text(error).foregroundStyle(.red)
                Button("common.action.retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("mobile_supplier.ui.no_open_cash_discrepancies")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.reconciliationId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.driver_driverid_status", "\(row.driverId)", "\(row.status)"))
                        Text(L10n.format("mobile_supplier.ui.diff_differenceminor_minor", "\(row.differenceMinor)"))
                        let open = row.status.uppercased() == "PENDING" || row.status.uppercased() == "DISPUTED"
                        if open {
                            Button("mobile_supplier.ui.accept") {
                                Task { await accept(row.reconciliationId) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(busyId == row.reconciliationId)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("portal.nav.cash_reconciliations")
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.cashReconciliations()
            rows = resp.reconciliations
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func accept(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            try await SupplierOperationsService.acceptCashReconciliation(
                id: id,
                idempotencyKey: "supplier-cash-recon-accept:\(id):\(UUID().uuidString)"
            )
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct CreditNotesListView: View {
    @State private var rows: [CreditNoteRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?

    var body: some View {
        List {
            if loading {
                ProgressView("Loading…")
            } else if let error {
                Text(error).foregroundStyle(.red)
                Button("common.action.retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("mobile_supplier.ui.no_draft_credit_notes")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.creditNoteId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.order_orderid_status", "\(row.orderId)", "\(row.status)"))
                        Text(L10n.format("mobile_supplier.ui.totalgrossminor_minor", "\(row.totalGrossMinor)"))
                        if row.status.uppercased() == "DRAFT" {
                            Button("mobile_supplier.ui.issue") {
                                Task { await issue(row.creditNoteId) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(busyId == row.creditNoteId)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("portal.nav.credit_notes")
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.creditNotes()
            rows = resp.creditNotes
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func issue(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            try await SupplierOperationsService.issueCreditNote(
                id: id,
                idempotencyKey: "supplier-credit-note-issue:\(id):\(UUID().uuidString)"
            )
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct CreditProfilesView: View {
    @State private var rows: [CreditProfileRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?

    var body: some View {
        List {
            if loading {
                ProgressView("Loading…")
            } else if let error {
                Text(error).foregroundStyle(.red)
                Button("common.action.retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("mobile_supplier.ui.no_credit_profiles_for_this_supplier")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.retailerId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.status_risk_risktier", "\(row.status)", "\(row.riskTier.isEmpty ? "—" : row.riskTier)"))
                        Text(L10n.format("mobile_supplier.ui.limit_creditlimitminor_bal_currentbalanceminor_avail_availablecreditmino", "\(row.creditLimitMinor)", "\(row.currentBalanceMinor)", "\(row.availableCreditMinor)"))
                        HStack {
                            if row.status.uppercased() == "ACTIVE" {
                                Button("mobile_supplier.ui.freeze") {
                                    Task { await setStatus(row, status: "FROZEN") }
                                }
                                .disabled(busyId == row.retailerId)
                            } else if row.status.uppercased() == "FROZEN" {
                                Button("mobile_supplier.ui.unfreeze") {
                                    Task { await setStatus(row, status: "ACTIVE") }
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(busyId == row.retailerId)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.credit_profiles")
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.creditProfiles()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func setStatus(_ row: CreditProfileRow, status: String) async {
        busyId = row.retailerId
        defer { busyId = nil }
        do {
            try await SupplierOperationsService.patchRetailerCreditProfile(
                RetailerCreditProfilePatchRequest(
                    retailerId: row.retailerId,
                    creditLimitMinor: row.creditLimitMinor,
                    status: status,
                    reason: "collections_desk"
                ),
                idempotencyKey: "supplier-credit-profile:\(status):\(row.retailerId):\(UUID().uuidString)"
            )
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct RoutePerformanceListView: View {
    @State private var rows: [RoutePerformanceRow] = []
    var body: some View {
        List(rows) { row in
            Text(L10n.format("mobile_supplier.ui.route_routeid_orderscompleted_orders", "\(row.routeId)", "\(row.ordersCompleted)"))
        }
        .navigationTitle("portal.nav.route_performance")
        .task {
            if let resp = try? await SupplierOperationsService.routePerformance() {
                rows = resp.routes
            }
        }
    }
}
