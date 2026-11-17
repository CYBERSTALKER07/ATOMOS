import SwiftUI

struct AutoOrderView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = AutoOrderViewModel()

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            if vm.isLoading {
                loadingState
            } else {
                VStack(spacing: AppTheme.spacingLG) {
                    if let syncMessage = vm.syncMessage, !syncMessage.isEmpty {
                        syncStatusBanner(syncMessage)
                            .slideIn(delay: 0)
                    }

                    AutoOrderHeaderCard(
                        supplierCount: vm.settings?.supplierOverrides.count ?? 0,
                        categoryCount: vm.settings?.categoryOverrides.count ?? 0,
                        productCount: vm.settings?.productOverrides.count ?? 0,
                        predictionCount: vm.forecasts.count
                    ).slideIn(delay: 0)

                    executionModeCard.slideIn(delay: 0.01)
                    shadowInboxCard.slideIn(delay: 0.015)
                    autoOrderWorkerCard(placeConfirmOpenBinding: $vm.placeConfirmOpen).slideIn(delay: 0.02)
                    reorderSuggestionsCard.slideIn(delay: 0.03)

                    AutoOrderGlobalToggleCard(
                        globalAutoOrder: $vm.globalAutoOrder,
                        analyticsStartDate: vm.settings?.analyticsStartDate,
                        onToggle: { newVal in
                            vm.handleGlobalToggle(newVal)
                        }
                    ).slideIn(delay: 0.05)

                    if let s = vm.settings {
                        if !s.supplierOverrides.isEmpty {
                            AutoOrderOverridesSection(
                                title: "Supplier Overrides",
                                icon: "building.2",
                                items: s.supplierOverrides.map {
                                    OverrideItem(
                                        id: $0.supplierID,
                                        label: $0.supplierName ?? $0.supplierID,
                                        enabled: $0.enabled,
                                        hasHistory: $0.hasHistory,
                                        level: .supplier
                                    )
                                },
                                localToggleStates: $vm.localToggleStates,
                                onToggle: { item, newVal in vm.handleOverrideToggle(item: item, newVal: newVal) }
                            )
                            .slideIn(delay: 0.1)
                        }

                        if !s.categoryOverrides.isEmpty {
                            AutoOrderOverridesSection(
                                title: "Category Overrides",
                                icon: "square.grid.2x2",
                                items: s.categoryOverrides.map {
                                    OverrideItem(
                                        id: $0.categoryID,
                                        label: $0.categoryID,
                                        enabled: $0.enabled,
                                        hasHistory: $0.hasHistory,
                                        level: .category
                                    )
                                },
                                localToggleStates: $vm.localToggleStates,
                                onToggle: { item, newVal in vm.handleOverrideToggle(item: item, newVal: newVal) }
                            )
                            .slideIn(delay: 0.125)
                        }

                        if !s.productOverrides.isEmpty {
                            AutoOrderOverridesSection(
                                title: "Product Overrides",
                                icon: "leaf",
                                items: s.productOverrides.map {
                                    OverrideItem(
                                        id: $0.productID,
                                        label: $0.productName ?? $0.productID,
                                        enabled: $0.enabled,
                                        hasHistory: $0.hasHistory,
                                        level: .product
                                    )
                                },
                                localToggleStates: $vm.localToggleStates,
                                onToggle: { item, newVal in vm.handleOverrideToggle(item: item, newVal: newVal) }
                            )
                            .slideIn(delay: 0.15)
                        }

                        if !s.variantOverrides.isEmpty {
                            AutoOrderOverridesSection(
                                title: "Size / variant Overrides",
                                icon: "cube",
                                items: s.variantOverrides.map {
                                    OverrideItem(
                                        id: $0.skuID,
                                        label: $0.skuLabel ?? $0.skuID,
                                        enabled: $0.enabled,
                                        hasHistory: $0.hasHistory,
                                        level: .variant
                                    )
                                },
                                localToggleStates: $vm.localToggleStates,
                                onToggle: { item, newVal in vm.handleOverrideToggle(item: item, newVal: newVal) }
                            )
                            .slideIn(delay: 0.2)
                        }
                    }

                    if !vm.forecasts.isEmpty {
                        AutoOrderPredictionsSection(forecasts: vm.forecasts).slideIn(delay: 0.25)
                    }

                    AutoOrderExplainerCard().slideIn(delay: 0.3)
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle("portal.nav.auto_order")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button {
                    Haptics.light()
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .frame(width: 30, height: 30)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.circle)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Close")
            }
        }
        .task { await vm.loadAll() }
        .task(id: refreshCenter.refreshToken) { await vm.loadAll() }
        .refreshable { await vm.loadAll() }
        .alert("Use Previous Analytics?", isPresented: $vm.showHistoryAlert, actions: {
            Button("mobile_retailer.ui.use_history") {
                Task { await vm.confirmEnable(useHistory: true) }
            }
            Button("mobile_retailer.ui.start_fresh", role: .destructive) {
                Task { await vm.confirmEnable(useHistory: false) }
            }
            Button("common.action.cancel", role: .cancel) {
                vm.cancelPendingEnable()
            }
        }, message: {
            Text(L10n.format("mobile_retailer.ui.enable_alertentitylabel_using_your_existing_order_history_or_start_fresh", "\(vm.alertEntityLabel)"))
        })
    }

    // MARK: - Loading

    private var loadingState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ProgressView()
                .tint(AppTheme.accent)
            Text("mobile_retailer.ui.loading_settings")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    private var executionModeCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            Text("mobile_retailer.ui.execution_mode")
                .font(.system(.headline, design: .rounded))
            Text("mobile_retailer.ui.off_shadow_recommended_draft_cart_place_scope_toggles_below_choo")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            HStack(spacing: 6) {
                ForEach([("off", "Off"), ("shadow", "Shadow"), ("draft", "Draft"), ("place", "Place")], id: \.0) { mode, label in
                    Button {
                        Task { await vm.setExecutionMode(mode) }
                    } label: {
                        Text(label)
                            .pegasusMinimumHitTarget()
                    }
                    .buttonStyle(.bordered)
                    .tint(vm.executionMode == mode ? AppTheme.accent : AppTheme.textSecondary)
                    .disabled(vm.running)
                    .accessibilityLabel("Execution mode \(label)\(vm.executionMode == mode ? ", active" : "")")
                }
            }
            if let stats = vm.settings?.shadowStats {
                Text(String(format: "30d WAPE %.0f%% · accept %.0f%% (%lld proposals)", stats.wape * 100, stats.unmodifiedAcceptRate * 100, stats.proposalCount))
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            if let gate = vm.soakGate {
                let allowed = gate.decision?.allowed == true
                Text(allowed ? "Place readiness: passed" : "Place readiness: blocked")
                    .font(.system(.caption, design: .rounded).weight(.semibold))
                    .foregroundStyle(allowed ? AppTheme.accent : AppTheme.destructive.opacity(0.85))
                if let stats = gate.decision?.stats {
                    Text(String(
                        format: "%lld proposals · %lld matched · WAPE %.0f%% · unmodified %.0f%%",
                        stats.proposalCount,
                        stats.matchedOrders,
                        stats.wape * 100,
                        stats.unmodifiedAcceptRate * 100
                    ))
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                }
                if gate.placeFlagEnabled == false {
                    Text("Server place flag is off (dual-control required).")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                if let reasons = gate.decision?.reasons, !reasons.isEmpty {
                    Text(reasons.joined(separator: " · "))
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                Text("Download soak evidence from desktop or scripts/generate_auto_order_soak_artifact.sh")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private var shadowInboxCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            Text("mobile_retailer.ui.shadow_inbox")
                .font(.system(.headline, design: .rounded))
            if vm.shadowProposals.isEmpty {
                Text("mobile_retailer.ui.no_shadow_proposals_yet_set_mode_to_shadow_and_run_shadow_now")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                ForEach(vm.shadowProposals.prefix(8)) { p in
                    Text(L10n.format("mobile_retailer.ui.sku_qty_proposedqty_ip_ip_rop_reorderpoint", "\(p.sku)", "\(p.proposedQty)", "\(Int(p.ip))", "\(Int(p.reorderPoint))"))
                        .font(.system(.caption, design: .rounded))
                        .accessibilityLabel("Proposal SKU \(p.sku), quantity \(p.proposedQty)")
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private func autoOrderWorkerCard(placeConfirmOpenBinding: Binding<Bool>) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Image(systemName: "play.circle.fill")
                    .foregroundStyle(AppTheme.accent)
                Text("mobile_retailer.ui.auto_order_worker")
                    .font(.system(.headline, design: .rounded))
                Spacer()
            }
            Text("mobile_retailer.ui.shadow_records_proposals_only_draft_stages_cart_lines_place_crea")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            HStack(spacing: AppTheme.spacingSM) {
                Button {
                    Task { await vm.runAutoOrder(mode: "shadow") }
                } label: {
                    Text(vm.running && vm.runningMode == "shadow" ? "…" : "Shadow")
                        .frame(maxWidth: .infinity)
                        .pegasusMinimumHitTarget()
                }
                .buttonStyle(.bordered)
                .disabled(vm.running || vm.executionMode == "off")
                .accessibilityLabel("Run shadow auto-order")

                Button {
                    Task { await vm.runAutoOrder(mode: "draft") }
                } label: {
                    HStack {
                        if vm.running && vm.runningMode == "draft" {
                            ProgressView()
                        } else {
                            Image(systemName: "play.fill")
                        }
                        Text(vm.running && vm.runningMode == "draft" ? "Drafting…" : "Draft")
                    }
                    .frame(maxWidth: .infinity)
                    .pegasusMinimumHitTarget()
                }
                .buttonStyle(.bordered)
                .disabled(vm.running || vm.executionMode == "off")
                .accessibilityLabel("Run draft auto-order")

                Button {
                    Haptics.warning()
                    vm.placeConfirmOpen = true
                } label: {
                    HStack {
                        if vm.running && vm.runningMode == "place" {
                            ProgressView().tint(.white)
                        } else {
                            Image(systemName: "cart.fill")
                        }
                        Text(vm.running && vm.runningMode == "place" ? "Placing…" : "Place")
                    }
                    .frame(maxWidth: .infinity)
                    .pegasusMinimumHitTarget()
                }
                .buttonStyle(.borderedProminent)
                .disabled(vm.running || vm.executionMode == "off")
                .accessibilityLabel("Run place auto-order")
            }
            .confirmationDialog(
                "Create real supplier orders?",
                isPresented: placeConfirmOpenBinding,
                titleVisibility: .visible
            ) {
                Button("mobile_retailer.ui.confirm_place", role: .destructive) {
                    Task { await vm.runAutoOrder(mode: "place") }
                }
                Button("common.action.cancel", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.place_mode_creates_real_procurement_orders_auto_order_requires_p")
            }

            if let lastRun = vm.lastRun {
                let placed = lastRun.placedLines ?? 0
                let via = lastRun.candidateSource.map { " · via \($0)" } ?? ""
                Text("Latest: \(lastRun.mode) · draft \(lastRun.draftLines)\(placed > 0 ? " · placed \(placed)" : "") · \(lastRun.status)\(via)")
                    .font(.system(.caption, design: .rounded, weight: .medium))
                if let message = lastRun.message {
                    Text(message)
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                ForEach(lastRun.placedOrders ?? []) { po in
                    Text(L10n.format("mobile_retailer.ui.orderidmap_linecount_lines", "\(po.orderId)", "\(po.supplierId.map { " · \($0)" } ?? "")", "\(po.lineCount)"))
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.accent)
                }
            }

            if vm.runs.isEmpty {
                Text("mobile_retailer.ui.no_runs_yet_enable_auto_order_and_use_draft_or_place")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                Text("mobile_retailer.ui.last_runs")
                    .font(.system(.caption, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
                ForEach(vm.runs.prefix(8)) { run in
                    let p = run.placedLines ?? 0
                    HStack {
                        Text(L10n.format("mobile_retailer.ui.schedulebucket_string_mode_ddraftlinesp_0_p", "\(run.scheduleBucket ?? String(run.startedAt.prefix(10)))", "\(run.mode)", "\(run.draftLines)", "\(p > 0 ? " p\(p)" : "")"))
                            .font(.system(.caption2, design: .rounded))
                        Spacer()
                        Text(run.status)
                            .font(.system(.caption2, design: .rounded, weight: .semibold))
                            .foregroundStyle((run.status == "OK" || run.status == "PARTIAL") ? AppTheme.accent : AppTheme.warning)
                    }
                    .accessibilityElement(children: .combine)
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusLG))
    }

    private var reorderSuggestionsCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            Text("supplier_portal.replenishment.suggestions.text.reorder_suggestions")
                .font(.system(.headline, design: .rounded))
            Text("mobile_retailer.ui.sell_through_aware_open_suggestions_store_pos_wholesale")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            if vm.reorderSuggestions.isEmpty {
                Text("mobile_retailer.ui.no_open_suggestions_yet_pos_sell_through_and_demand_batch_popula")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                ForEach(vm.reorderSuggestions.prefix(12)) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text("\(row.sku) · qty \(row.suggestedQty)\(row.currentStock.map { " · stock \($0)" } ?? "")")
                            .font(.system(.subheadline, design: .rounded))
                        DemandSourceChips(sources: row.sources)
                        if let vel = row.sellThroughVelocity, vel > 0 {
                            Text(String(format: "POS vel %.1f/d", vel))
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    Divider()
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusLG))
    }

    private func syncStatusBanner(_ message: String) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(AppTheme.warning)
            Text(message)
                .font(.system(.caption, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(3)
            Spacer()
            Button("common.action.retry") {
                Haptics.light()
                Task { await vm.loadAll() }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
            .pegasusMinimumHitTarget()
            .accessibilityLabel("Retry loading auto-order settings")
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(AppTheme.warning.opacity(0.12))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
    }
}

#Preview {
    AutoOrderView()
}
