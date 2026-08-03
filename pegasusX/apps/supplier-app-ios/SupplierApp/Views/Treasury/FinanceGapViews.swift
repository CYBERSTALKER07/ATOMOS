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
                Button("Retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("No open cash discrepancies.")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.reconciliationId).font(.caption.monospaced())
                        Text("Driver \(row.driverId) · \(row.status)")
                        Text("Diff \(row.differenceMinor) minor")
                        let open = row.status.uppercased() == "PENDING" || row.status.uppercased() == "DISPUTED"
                        if open {
                            Button("Accept") {
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
        .navigationTitle("Cash reconciliations")
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
                Button("Retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("No draft credit notes.")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.creditNoteId).font(.caption.monospaced())
                        Text("Order \(row.orderId) · \(row.status)")
                        Text("\(row.totalGrossMinor) minor")
                        if row.status.uppercased() == "DRAFT" {
                            Button("Issue") {
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
        .navigationTitle("Credit notes")
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
                Button("Retry") { Task { await load() } }
            } else if rows.isEmpty {
                Text("No credit profiles for this supplier.")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.retailerId).font(.caption.monospaced())
                        Text("\(row.status) · risk \(row.riskTier.isEmpty ? "—" : row.riskTier)")
                        Text("Limit \(row.creditLimitMinor) · bal \(row.currentBalanceMinor) · avail \(row.availableCreditMinor)")
                        HStack {
                            if row.status.uppercased() == "ACTIVE" {
                                Button("Freeze") {
                                    Task { await setStatus(row, status: "FROZEN") }
                                }
                                .disabled(busyId == row.retailerId)
                            } else if row.status.uppercased() == "FROZEN" {
                                Button("Unfreeze") {
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
        .navigationTitle("Credit profiles")
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
            Text("Route \(row.routeId) · \(row.ordersCompleted) orders")
        }
        .navigationTitle("Route performance")
        .task {
            if let resp = try? await SupplierOperationsService.routePerformance() {
                rows = resp.routes
            }
        }
    }
}
