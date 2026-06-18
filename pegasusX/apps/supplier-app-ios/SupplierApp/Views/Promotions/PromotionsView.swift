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
                List(promotions) { promo in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(promo.name)
                            .font(.headline)
                        Text(promoSummary(promo))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        if promo.isActive {
                            Button {
                                editingPromotion = promo
                                name = promo.name
                                discountBps = String(promo.discountBps)
                                showEdit = true
                            } label: {
                                Text("Edit")
                            }
                            .tint(.blue)
                            Button(role: .destructive) {
                                Task { await deactivate(promo.promotionId) }
                            } label: {
                                Text("Deactivate")
                            }
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Promotions")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("New") { showCreate = true }
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
                    TextField("Name", text: $name)
                    TextField("Discount (bps)", text: $discountBps)
                        .keyboardType(.numberPad)
                    Text("Default scope: all products, all retailers.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .navigationTitle("New promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showCreate = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Create") { Task { await create() } }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .sheet(isPresented: $showEdit) {
            NavigationStack {
                Form {
                    TextField("Name", text: $name)
                    TextField("Discount (bps)", text: $discountBps)
                        .keyboardType(.numberPad)
                }
                .navigationTitle("Edit promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") {
                            showEdit = false
                            editingPromotion = nil
                        }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Save") { Task { await saveEdit() } }
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
    private func deactivate(_ promotionId: String) async {
        do {
            try await SupplierService.deactivatePromotion(promotionId: promotionId)
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }
}
