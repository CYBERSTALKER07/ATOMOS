import SwiftUI

struct WarehousesView: View {
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showAdd = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading warehouses…")
            } else if let error, warehouses.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty {
                TopologyCenteredEmptyState(
                    title: "No warehouses",
                    message: "Add your first distribution node to start fulfilling orders.",
                    actionLabel: "Add first warehouse"
                ) {
                    showAdd = true
                }
            } else {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Pin stores and city coverage on the supplier desktop portal by 2026-09-16. Mobile stays view-only.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal)
                    WarehouseList(warehouses: warehouses)
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showAdd = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showAdd) {
            AddWarehouseSheet {
                Task { await load(silent: true) }
            }
        }
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let topology = try await SupplierOperationsService.topology()
            warehouses = topology.warehouses
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}

struct CRMView: View {
    @State private var retailers: [SupplierCRMRetailer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selected: SupplierCRMRetailerDetail?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading CRM…")
            } else if let error, retailers.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if retailers.isEmpty {
                Text("No retailers")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(retailers) { row in
                        Button {
                            Task { await loadDetail(row.retailerId) }
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(row.retailerName.isEmpty ? row.retailerId : row.retailerName)
                                    .font(.headline)
                                Text("\(row.status) · lifetime \(row.lifetime) · \(row.orderCount) orders")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !row.phone.isEmpty {
                                    Text(row.phone)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                if !row.email.isEmpty {
                                    Text(row.email)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                    if let selected {
                        Section("Detail") {
                            Text(selected.retailerName)
                            if !selected.email.isEmpty {
                                Text(selected.email)
                                    .font(.caption)
                            }
                            Text("Lifetime \(selected.lifetime) (minor, not divided)")
                                .font(.caption)
                            ForEach(selected.orders) { order in
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(order.orderId.prefix(8)) · \(order.state) · \(order.amount)")
                                        .font(.caption)
                                    ForEach(order.lines) { line in
                                        Text("\(line.productName.isEmpty ? line.sku : line.productName) × \(line.qty) · \(line.amountMinor)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.crm")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            retailers = try await SupplierOperationsService.crmRetailers()
        } catch let err as APIError {
            if case .httpError(503) = err {
                if !silent { error = "CRM unavailable" }
            } else if !silent {
                error = err.localizedDescription
            }
            retailers = []
        } catch {
            if !silent { self.error = error.localizedDescription }
            retailers = []
        }
    }

    @MainActor
    private func loadDetail(_ id: String) async {
        do {
            selected = try await SupplierOperationsService.crmRetailer(id)
        } catch {
            selected = nil
        }
    }
}

struct LoyaltyProgramView: View {
    @State private var program: LoyaltyProgram?
    @State private var earnBps = "100"
    @State private var reason = ""
    @State private var status: String?
    @State private var error: String?
    @State private var loading = true
    @State private var busy = false

    var body: some View {
        Form {
            if loading {
                ProgressView("Loading loyalty program…")
            } else if let error {
                Text(error).foregroundStyle(.red)
            } else {
                Text("Earn on paid orders. Burn is out of scope. Retailers without a program see enrolled=false, not a fake Bronze.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                Text("Source: \(program?.source ?? "unconfigured")")
                    .font(.caption)
                TextField("Earn bps", text: $earnBps)
                    .keyboardType(.numberPad)
                TextField("Reason (required)", text: $reason)
                Button(busy ? "Saving…" : "Save program") { Task { await save() } }
                    .disabled(busy)
                if let status {
                    Text(status).font(.footnote)
                }
                ForEach(program?.tiers ?? [], id: \.name) { tier in
                    Text("\(tier.name) from \(tier.minPoints) lifetime points")
                        .font(.caption)
                }
            }
        }
        .navigationTitle("portal.nav.loyalty")
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.loyaltyProgram()
            program = resp
            earnBps = String(resp.earnBps)
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func save() async {
        let why = reason.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !why.isEmpty else {
            status = "Typed reason required"
            return
        }
        busy = true
        defer { busy = false }
        do {
            let bps = Int64(earnBps) ?? 100
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            var body = try await SupplierOperationsService.loyaltyProgram()
            body.earnBps = bps > 0 ? bps : 100
            body.reason = why
            let resp = try await SupplierOperationsService.patchLoyaltyProgram(
                body,
                idempotencyKey: SupplierIdempotencyKeys.loyaltyProgramPatch(scopeId: scope, reason: why)
            )
            program = resp
            reason = ""
            status = "Saved (\(resp.source ?? "program"))"
        } catch {
            status = error.localizedDescription
        }
    }
}

struct EntityResolutionView: View {
    private let types = ["ANY", "ORDER", "RETAILER", "WAREHOUSE", "FACTORY", "DRIVER", "VEHICLE", "SUPPLIER"]
    @State private var entityType = "ANY"
    @State private var query = ""
    @State private var entityId = ""
    @State private var busy = false
    @State private var error: String?
    @State private var resolved: EntityResolutionResolveResponse?
    @State private var explain: EntityResolutionExplainResponse?

    var body: some View {
        Form {
            Picker("Type", selection: $entityType) {
                ForEach(types, id: \.self) { Text($0).tag($0) }
            }
            TextField("Query", text: $query)
            TextField("Entity ID", text: $entityId)
            Button(busy ? "Resolving…" : "Resolve") { Task { await runResolve() } }
                .disabled(busy)
            if let error {
                Text(error).foregroundStyle(.red)
            }
            if let resolved {
                Text("Requested \(resolved.requestedType) · \(resolved.candidates.count) candidates")
                    .font(.footnote)
                if let top = resolved.resolved {
                    Text("Top: \(top.label) (\(top.entityId)) score \(top.score)")
                        .font(.caption)
                } else {
                    Text("No deterministic match").font(.caption)
                }
                ForEach(resolved.candidates) { c in
                    Button("\(c.label) · \(c.entityType)/\(c.entityId) · \(c.confidenceClass)") {
                        Task { await runExplain(type: c.entityType, id: c.entityId) }
                    }
                }
            }
            if let explain {
                Text("Lineage for \(explain.source.label)").font(.headline)
                ForEach(Array(explain.projection.edges.enumerated()), id: \.offset) { _, e in
                    Text("\(e.relation): \(e.from) → \(e.to)").font(.caption)
                }
            }
        }
        .navigationTitle("portal.nav.entity_resolution")
    }

    @MainActor
    private func runResolve() async {
        busy = true
        error = nil
        explain = nil
        defer { busy = false }
        do {
            resolved = try await SupplierOperationsService.resolveEntity(
                EntityResolutionResolveRequest(
                    entityType: entityType,
                    query: query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : query,
                    entityId: entityId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : entityId
                )
            )
        } catch {
            self.error = error.localizedDescription
            resolved = nil
        }
    }

    @MainActor
    private func runExplain(type: String, id: String) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            explain = try await SupplierOperationsService.explainEntity(
                EntityResolutionExplainRequest(entityType: type, entityId: id)
            )
        } catch {
            self.error = error.localizedDescription
        }
    }
}
