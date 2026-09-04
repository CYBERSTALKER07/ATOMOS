import SwiftUI

struct PromotionsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var promotions: [SupplierPromotion] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var showEdit = false
    @State private var editingPromotion: SupplierPromotion?
    @State private var name = ""
    @State private var discountBps = "500"
    @State private var simResults: [String: PromoSimulateResult] = [:]
    @State private var simulatingId: String?

    var body: some View {
        Group {
            if loading && promotions.isEmpty {
                SupplierLoadingView(title: "Loading promotions…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if promotions.isEmpty {
                SupplierEmptyView(
                    title: "No promotions",
                    message: "Create a sale for products, categories, or your full catalog."
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(promotions) { promo in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(promo.name)
                                .font(.headline)
                            Text(promoSummary(promo))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if let sim = simResults[promo.promotionId] {
                                Text(L10n.format("mobile_supplier.ui.p_l_sandbox_projectedvolume_units_margin_projectedmarginminor_100_margin", "\(sim.projectedVolume)", "\(sim.projectedMarginMinor / 100)", "\(Int(sim.marginDeltaPct))"))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .swipeActions(edge: .leading, allowsFullSwipe: false) {
                            if promo.isActive {
                                Button {
                                    Task { await simulate(promo) }
                                } label: {
                                    Text(simulatingId == promo.promotionId ? "…" : "P&L")
                                }
                                .tint(.purple)
                            }
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if promo.isActive {
                                Button {
                                    editingPromotion = promo
                                    name = promo.name
                                    discountBps = String(promo.discountBps)
                                    showEdit = true
                                } label: {
                                    Text("supplier_portal.demand.signals.text.edit")
                                }
                                .tint(.blue)
                                Button(role: .destructive) {
                                    Task { await deactivate(promo.promotionId) }
                                } label: {
                                    Text("supplier_portal.demand.signals.text.deactivate")
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.promotions")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("mobile_supplier.ui.new") { showCreate = true }
            }
        }
        .task { await load() }
        .refreshable { await load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await load(silent: silent) }
        }
        .sheet(isPresented: $showCreate) {
            NavigationStack {
                Form {
                    TextField("retailer_desktop.pos.text.name", text: $name)
                    TextField("supplier_portal.pricing._product_id_.text.discount_bps", text: $discountBps)
                        .keyboardType(.numberPad)
                    Text("mobile_supplier.ui.default_scope_all_products_all_retailers")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .navigationTitle("mobile_supplier.ui.new_promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { showCreate = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_supplier.ui.create") { Task { await create() } }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .sheet(isPresented: $showEdit) {
            NavigationStack {
                Form {
                    TextField("retailer_desktop.pos.text.name", text: $name)
                    TextField("supplier_portal.pricing._product_id_.text.discount_bps", text: $discountBps)
                        .keyboardType(.numberPad)
                }
                .navigationTitle("mobile_supplier.ui.edit_promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") {
                            showEdit = false
                            editingPromotion = nil
                        }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("common.action.save") { Task { await saveEdit() } }
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }

    private func promoSummary(_ promo: SupplierPromotion) -> String {
        let pct = Double(promo.discountBps) / 100.0
        let status = promo.isActive ? "" : " · inactive"
        return String(format: "%.1f%% · %@ · %@", pct, promo.scopeType, promo.retailerScope) + status
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            promotions = try await SupplierService.promotions()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }

    @MainActor
    private func create() async {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, let bps = Int64(discountBps.filter(\.isNumber)), bps > 0 else { return }
        do {
            _ = try await SupplierService.createPromotion(
                SupplierPromotionUpsertRequest(
                    name: trimmed,
                    description: "",
                    discountBps: bps,
                    scopeType: "ALL_PRODUCTS",
                    retailerScope: "ALL",
                    scopeProductId: nil
                )
            )
            name = ""
            discountBps = "500"
            showCreate = false
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func saveEdit() async {
        guard let promo = editingPromotion else { return }
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, let bps = Int64(discountBps.filter(\.isNumber)), bps > 0 else { return }
        do {
            _ = try await SupplierService.updatePromotion(
                promotionId: promo.promotionId,
                SupplierPromotionUpsertRequest(
                    name: trimmed,
                    description: "",
                    discountBps: bps,
                    scopeType: promo.scopeType,
                    retailerScope: promo.retailerScope,
                    scopeProductId: promo.scopeProductId
                )
            )
            showEdit = false
            editingPromotion = nil
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func simulate(_ promo: SupplierPromotion) async {
        simulatingId = promo.promotionId
        defer { simulatingId = nil }
        do {
            let result = try await SupplierOperationsService.simulatePromotionPandL(
                PromoSimulateInput(
                    promotionId: promo.promotionId,
                    discountPct: Double(promo.discountBps) / 100.0,
                    expectedUnits: 500,
                    avgUnitMarginMinor: 1000
                )
            )
            simResults[promo.promotionId] = result
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func deactivate(_ promotionId: String) async {
        do {
            try await SupplierService.deactivatePromotion(promotionId: promotionId)
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }
}
