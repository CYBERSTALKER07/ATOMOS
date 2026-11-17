import SwiftUI

struct PromotionsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = PromotionsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.promotions.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.promotions")
            } else if let error = viewModel.error, viewModel.promotions.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.promotions.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_promotions",
                    message: "mobile_supplier.empty.promotions_message"
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.promotions) { promo in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(promo.name)
                                .font(.headline)
                            Text(Self.formatPromoSummary(promo))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if let sim = viewModel.simResults[promo.promotionId] {
                                Text(simSummary(sim))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .accessibilityElement(children: .combine)
                        .swipeActions(edge: .leading, allowsFullSwipe: false) {
                            if promo.isActive {
                                Button {
                                    Task { await viewModel.simulate(promo) }
                                } label: {
                                    Text(viewModel.simulatingId == promo.promotionId ? "…" : "P&L")
                                }
                                .tint(.purple)
                            }
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if promo.isActive {
                                Button {
                                    viewModel.beginEdit(promo)
                                } label: {
                                    Text("supplier_portal.demand.signals.text.edit")
                                }
                                .tint(.blue)
                                Button(role: .destructive) {
                                    Task { await viewModel.deactivate(promo.promotionId) }
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
                Button("mobile_supplier.ui.new") { viewModel.showCreate = true }
            }
        }
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await viewModel.load(silent: silent) }
        }
        .sheet(isPresented: $vm.showCreate) {
            NavigationStack {
                Form {
                    TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    TextField("supplier_portal.pricing._product_id_.text.discount_bps", text: $vm.discountBps)
                        .keyboardType(.numberPad)
                    Text("mobile_supplier.ui.default_scope_all_products_all_retailers")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("mobile_supplier.ui.new_promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { viewModel.showCreate = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_supplier.ui.create") { Task { await viewModel.create() } }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .sheet(isPresented: $vm.showEdit) {
            NavigationStack {
                Form {
                    TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    TextField("supplier_portal.pricing._product_id_.text.discount_bps", text: $vm.discountBps)
                        .keyboardType(.numberPad)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("mobile_supplier.ui.edit_promotion")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") {
                            viewModel.cancelEdit()
                        }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("common.action.save") { Task { await viewModel.saveEdit() } }
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }

    static func formatPromoSummary(_ promo: SupplierPromotion) -> String {
        let pct = formatMinorUnits(promo.discountBps, fractionDigits: 2, trimTrailingZeros: true)
        let status = promo.isActive ? "" : L10n.format("mobile_supplier.chrome.inactive_suffix")
        return "\(pct)% · \(promo.scopeType) · \(promo.retailerScope)" + status
    }

    private func simSummary(_ sim: PromoSimulateResult) -> String {
        let margin = formatMinorUnits(sim.projectedMarginMinor, fractionDigits: 2, trimTrailingZeros: true)
        return L10n.format(
            "mobile_supplier.ui.p_l_sandbox_projectedvolume_units_margin_projectedmarginminor_100_margin",
            "\(sim.projectedVolume)",
            margin,
            "\(Int(sim.marginDeltaPct))"
        )
    }
}
