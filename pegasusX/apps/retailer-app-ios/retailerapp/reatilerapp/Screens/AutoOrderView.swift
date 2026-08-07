import SwiftUI

// MARK: - Response Models

struct AutoOrderShadowStats: Codable {
    let proposalCount: Int64
    let matchedOrders: Int64
    let wape: Double
    let unmodifiedAcceptRate: Double
    let windowDays: Int

    enum CodingKeys: String, CodingKey {
        case proposalCount = "proposal_count"
        case matchedOrders = "matched_orders"
        case wape
        case unmodifiedAcceptRate = "unmodified_accept_rate"
        case windowDays = "window_days"
    }
}

struct AutoOrderShadowProposal: Codable, Identifiable {
    var id: String { proposalId }
    let proposalId: String
    let retailerId: String
    let sku: String
    let supplierId: String?
    let proposedQty: Int64
    let ip: Double
    let reorderPoint: Double
    let orderUpTo: Double
    let bucketDate: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case proposalId = "proposal_id"
        case retailerId = "retailer_id"
        case sku
        case supplierId = "supplier_id"
        case proposedQty = "proposed_qty"
        case ip
        case reorderPoint = "reorder_point"
        case orderUpTo = "order_up_to"
        case bucketDate = "bucket_date"
        case status
    }
}

struct AutoOrderShadowProposalsResponse: Codable {
    let items: [AutoOrderShadowProposal]
}

struct AutoOrderSettings: Codable {
    let globalEnabled: Bool
    let executionMode: String?
    let hasAnyHistory: Bool
    let analyticsStartDate: String?
    let supplierOverrides: [SupplierOverride]
    let categoryOverrides: [CategoryOverride]
    let productOverrides: [ProductOverride]
    let variantOverrides: [VariantOverride]
    let shadowStats: AutoOrderShadowStats?

    enum CodingKeys: String, CodingKey {
        case globalEnabled = "global_enabled"
        case executionMode = "execution_mode"
        case hasAnyHistory = "has_any_history"
        case analyticsStartDate = "analytics_start_date"
        case supplierOverrides = "supplier_overrides"
        case categoryOverrides = "category_overrides"
        case productOverrides = "product_overrides"
        case variantOverrides = "variant_overrides"
        case shadowStats = "shadow_stats"
    }
}

struct SupplierOverride: Codable, Identifiable, Hashable {
    var id: String { supplierID }
    let supplierID: String
    let enabled: Bool
    let hasHistory: Bool
    let supplierName: String?
    let analyticsStartDate: String?

    enum CodingKeys: String, CodingKey {
        case supplierID = "supplier_id"
        case enabled
        case hasHistory = "has_history"
        case supplierName = "supplier_name"
        case analyticsStartDate = "analytics_start_date"
    }
}

struct CategoryOverride: Codable, Identifiable, Hashable {
    var id: String { categoryID }
    let categoryID: String
    let enabled: Bool
    let hasHistory: Bool
    let analyticsStartDate: String?

    enum CodingKeys: String, CodingKey {
        case categoryID = "category_id"
        case enabled
        case hasHistory = "has_history"
        case analyticsStartDate = "analytics_start_date"
    }
}

struct ProductOverride: Codable, Identifiable, Hashable {
    var id: String { productID }
    let productID: String
    let supplierID: String
    let enabled: Bool
    let hasHistory: Bool
    let productName: String?
    let analyticsStartDate: String?

    enum CodingKeys: String, CodingKey {
        case productID = "product_id"
        case supplierID = "supplier_id"
        case enabled
        case hasHistory = "has_history"
        case productName = "product_name"
        case analyticsStartDate = "analytics_start_date"
    }
}

struct VariantOverride: Codable, Identifiable, Hashable {
    var id: String { skuID }
    let skuID: String
    let productID: String?
    let enabled: Bool
    let hasHistory: Bool
    let skuLabel: String?
    let analyticsStartDate: String?

    enum CodingKeys: String, CodingKey {
        case skuID = "variant_id"
        case skuIDAlt = "sku_id"
        case productID = "product_id"
        case enabled
        case hasHistory = "has_history"
        case skuLabel = "sku_label"
        case analyticsStartDate = "analytics_start_date"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        if let v = try c.decodeIfPresent(String.self, forKey: .skuID) {
            skuID = v
        } else {
            skuID = try c.decodeIfPresent(String.self, forKey: .skuIDAlt) ?? ""
        }
        productID = try c.decodeIfPresent(String.self, forKey: .productID)
        enabled = try c.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        hasHistory = try c.decodeIfPresent(Bool.self, forKey: .hasHistory) ?? false
        skuLabel = try c.decodeIfPresent(String.self, forKey: .skuLabel)
        analyticsStartDate = try c.decodeIfPresent(String.self, forKey: .analyticsStartDate)
    }
}

// MARK: - Auto-Order View

struct AutoOrderView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var refreshCenter = RetailerRefreshCenter.shared

    @State private var settings: AutoOrderSettings?
    @State private var forecasts: [DemandForecast] = []
    @State private var isLoading = true
    @State private var syncMessage: String? = nil
    @State private var globalAutoOrder = false
    @State private var executionMode = "draft"
    @State private var pendingTarget: EnableTarget?
    @State private var localToggleStates: [String: Bool] = [:]
    @State private var runs: [AutoOrderRun] = []
    @State private var lastRun: AutoOrderRun?
    @State private var running = false
    @State private var runningMode: String?
    @State private var placeConfirmOpen = false
    @State private var reorderSuggestions: [RetailerReorderSuggestion] = []
    @State private var shadowProposals: [AutoOrderShadowProposal] = []

    private enum EnableTarget {
        case global
        case supplier(String)
        case category(String)
        case product(String)
        case variant(String)
    }

    private var alertEntityLabel: String {
        switch pendingTarget {
        case .global: return "global auto-order"
        case .supplier: return "this supplier's auto-order"
        case .category: return "this category's auto-order"
        case .product: return "this product's auto-order"
        case .variant: return "this variant's auto-order"
        case nil: return "auto-order"
        }
    }

    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            ScrollView {
                if isLoading {
                    loadingState
                } else {
                    VStack(spacing: AppTheme.spacingLG) {
                        if let syncMessage, !syncMessage.isEmpty {
                            syncStatusBanner(syncMessage)
                                .slideIn(delay: 0)
                        }

                        AutoOrderHeaderCard(
                            supplierCount: settings?.supplierOverrides.count ?? 0,
                            categoryCount: settings?.categoryOverrides.count ?? 0,
                            productCount: settings?.productOverrides.count ?? 0,
                            predictionCount: forecasts.count
                        ).slideIn(delay: 0)

                        executionModeCard.slideIn(delay: 0.01)
                        shadowInboxCard.slideIn(delay: 0.015)
                        autoOrderWorkerCard.slideIn(delay: 0.02)

                        reorderSuggestionsCard.slideIn(delay: 0.03)
                        
                        AutoOrderGlobalToggleCard(
                            globalAutoOrder: $globalAutoOrder,
                            analyticsStartDate: settings?.analyticsStartDate,
                            onToggle: { newVal in
                                if newVal {
                                    if settings?.hasAnyHistory == true {
                                        pendingTarget = .global
                                    } else {
                                        Task { await enableGlobal(useHistory: false) }
                                    }
                                } else {
                                    Task { await disableGlobal() }
                                }
                            }
                        ).slideIn(delay: 0.05)

                        if let s = settings {
                            if !s.supplierOverrides.isEmpty {
                                AutoOrderOverridesSection(
                                    title: "Supplier Overrides",
                                    icon: "building.2",
                                    items: s.supplierOverrides.map { OverrideItem(id: $0.supplierID, label: $0.supplierName ?? $0.supplierID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .supplier) },
                                    localToggleStates: $localToggleStates,
                                    onToggle: handleOverrideToggle
                                )
                                .slideIn(delay: 0.1)
                            }

                            if !s.categoryOverrides.isEmpty {
                                AutoOrderOverridesSection(
                                    title: "Category Overrides",
                                    icon: "square.grid.2x2",
                                    items: s.categoryOverrides.map { OverrideItem(id: $0.categoryID, label: $0.categoryID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .category) },
                                    localToggleStates: $localToggleStates,
                                    onToggle: handleOverrideToggle
                                )
                                .slideIn(delay: 0.125)
                            }

                            if !s.productOverrides.isEmpty {
                                AutoOrderOverridesSection(
                                    title: "Product Overrides",
                                    icon: "leaf",
                                    items: s.productOverrides.map { OverrideItem(id: $0.productID, label: $0.productName ?? $0.productID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .product) },
                                    localToggleStates: $localToggleStates,
                                    onToggle: handleOverrideToggle
                                )
                                .slideIn(delay: 0.15)
                            }

                            if !s.variantOverrides.isEmpty {
                                AutoOrderOverridesSection(
                                    title: "Size / variant Overrides",
                                    icon: "cube",
                                    items: s.variantOverrides.map { OverrideItem(id: $0.skuID, label: $0.skuLabel ?? $0.skuID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .variant) },
                                    localToggleStates: $localToggleStates,
                                    onToggle: handleOverrideToggle
                                )
                                .slideIn(delay: 0.2)
                            }
                        }

                        if !forecasts.isEmpty {
                            AutoOrderPredictionsSection(forecasts: forecasts).slideIn(delay: 0.25)
                        }

                        AutoOrderExplainerCard().slideIn(delay: 0.3)
                    }
                    .padding(AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
            }
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .navigationTitle("portal.nav.auto_order")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(AppTheme.textSecondary)
                            .frame(width: 30, height: 30)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.circle)
                    }
                    .accessibilityLabel("Close")
                }
            }
            .task { await loadAll() }
            .task(id: refreshCenter.refreshToken) { await loadAll() }
            .refreshable { await loadAll() }
            .alert("Use Previous Analytics?", isPresented: Binding(
                get: { pendingTarget != nil },
                set: { val in
                    if !val {
                        if let target = pendingTarget {
                            switch target {
                            case .global: globalAutoOrder = false
                            case .supplier(let id), .category(let id), .product(let id), .variant(let id):
                                localToggleStates[id] = false
                            }
                        }
                        pendingTarget = nil
                    }
                }
            ), actions: {
                Button("mobile_retailer.ui.use_history") {
                    Task { await confirmEnable(useHistory: true) }
                }
                Button("mobile_retailer.ui.start_fresh", role: .destructive) {
                    Task { await confirmEnable(useHistory: false) }
                }
                Button("common.action.cancel", role: .cancel) {
                    if let target = pendingTarget {
                        switch target {
                        case .global: globalAutoOrder = false
                        case .supplier(let id), .category(let id), .product(let id), .variant(let id):
                            localToggleStates[id] = false
                        }
                    }
                    pendingTarget = nil
                }
            }, message: {
                Text(L10n.format("mobile_retailer.ui.enable_alertentitylabel_using_your_existing_order_history_or_start_fresh", "\(alertEntityLabel)"))
            })
        }
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
                    Button(label) {
                        Task { await setExecutionMode(mode) }
                    }
                    .buttonStyle(.bordered)
                    .tint(executionMode == mode ? AppTheme.accent : AppTheme.textSecondary)
                    .disabled(running)
                }
            }
            if let stats = settings?.shadowStats {
                Text(String(format: "30d WAPE %.0f%% · accept %.0f%% (%lld proposals)", stats.wape * 100, stats.unmodifiedAcceptRate * 100, stats.proposalCount))
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private var shadowInboxCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            Text("mobile_retailer.ui.shadow_inbox")
                .font(.system(.headline, design: .rounded))
            if shadowProposals.isEmpty {
                Text("mobile_retailer.ui.no_shadow_proposals_yet_set_mode_to_shadow_and_run_shadow_now")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                ForEach(shadowProposals.prefix(8)) { p in
                    Text(L10n.format("mobile_retailer.ui.sku_qty_proposedqty_ip_ip_rop_reorderpoint", "\(p.sku)", "\(p.proposedQty)", "\(Int(p.ip))", "\(Int(p.reorderPoint))"))
                        .font(.system(.caption, design: .rounded))
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private var autoOrderWorkerCard: some View {
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
                    Task { await runAutoOrder(mode: "shadow") }
                } label: {
                    Text(running && runningMode == "shadow" ? "…" : "Shadow")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(running || executionMode == "off")

                Button {
                    Task { await runAutoOrder(mode: "draft") }
                } label: {
                    HStack {
                        if running && runningMode == "draft" {
                            ProgressView()
                        } else {
                            Image(systemName: "play.fill")
                        }
                        Text(running && runningMode == "draft" ? "Drafting…" : "Draft")
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(running || executionMode == "off")

                Button {
                    placeConfirmOpen = true
                } label: {
                    HStack {
                        if running && runningMode == "place" {
                            ProgressView().tint(.white)
                        } else {
                            Image(systemName: "cart.fill")
                        }
                        Text(running && runningMode == "place" ? "Placing…" : "Place")
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .disabled(running || executionMode == "off")
            }
            .confirmationDialog(
                "Create real supplier orders?",
                isPresented: $placeConfirmOpen,
                titleVisibility: .visible
            ) {
                Button("mobile_retailer.ui.confirm_place", role: .destructive) {
                    Task { await runAutoOrder(mode: "place") }
                }
                Button("common.action.cancel", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.place_mode_creates_real_procurement_orders_auto_order_requires_p")
            }

            if let lastRun {
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

            if runs.isEmpty {
                Text("mobile_retailer.ui.no_runs_yet_enable_auto_order_and_use_draft_or_place")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                Text("mobile_retailer.ui.last_runs")
                    .font(.system(.caption, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
                ForEach(runs.prefix(8)) { run in
                    let p = run.placedLines ?? 0
                    HStack {
                        Text(L10n.format("mobile_retailer.ui.schedulebucket_string_mode_ddraftlinesp_0_p", "\(run.scheduleBucket ?? String(run.startedAt.prefix(10)))", "\(run.mode)", "\(run.draftLines)", "\(p > 0 ? " p\(p)" : "")"))
                            .font(.system(.caption2, design: .rounded))
                        Spacer()
                        Text(run.status)
                            .font(.system(.caption2, design: .rounded, weight: .semibold))
                            .foregroundStyle((run.status == "OK" || run.status == "PARTIAL") ? AppTheme.accent : AppTheme.warning)
                    }
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surface)
        .clipShape(.rect(cornerRadius: AppTheme.radiusLG))
    }

    private var reorderSuggestionsCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            Text("supplier_portal.replenishment.suggestions.text.reorder_suggestions")
                .font(.system(.headline, design: .rounded))
            Text("mobile_retailer.ui.sell_through_aware_open_suggestions_store_pos_wholesale")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            if reorderSuggestions.isEmpty {
                Text("mobile_retailer.ui.no_open_suggestions_yet_pos_sell_through_and_demand_batch_popula")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                ForEach(reorderSuggestions.prefix(12)) { row in
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
                    Divider()
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surface)
        .clipShape(.rect(cornerRadius: AppTheme.radiusLG))
    }

    private func syncStatusBanner(_ message: String) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.warning)
            Text(message)
                .font(.system(.caption, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(3)
            Spacer()
            Button("common.action.retry") {
                Task { await loadAll() }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(AppTheme.warning.opacity(0.12))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
    }

    private func handleOverrideToggle(item: OverrideItem, newVal: Bool) {
        if newVal && item.hasHistory {
            switch item.level {
            case .supplier: pendingTarget = .supplier(item.id)
            case .category: pendingTarget = .category(item.id)
            case .product:  pendingTarget = .product(item.id)
            case .variant:  pendingTarget = .variant(item.id)
            }
        } else {
            Task { await toggleOverride(item: item, enabled: newVal, useHistory: false) }
        }
    }

    // MARK: - API

    private func loadAll() async {
        if settings == nil { isLoading = true }
        async let settingsReq: (AutoOrderSettings?, String?) = loadSettings()
        async let forecastsReq: ([DemandForecast], String?) = loadForecasts()
        async let runsReq: [AutoOrderRun] = loadRuns()
        async let suggestionsReq: [RetailerReorderSuggestion] = loadSuggestions()
        async let shadowReq: [AutoOrderShadowProposal] = loadShadowProposals()

        let (fetchedSettings, settingsSyncMessage) = await settingsReq
        let (fetchedForecasts, forecastSyncMessage) = await forecastsReq
        let fetchedRuns = await runsReq
        let fetchedSuggestions = await suggestionsReq
        let fetchedShadow = await shadowReq

        withAnimation(AnimationConstants.fluid) {
            settings = fetchedSettings
            forecasts = fetchedForecasts
            runs = fetchedRuns
            reorderSuggestions = fetchedSuggestions
            shadowProposals = fetchedShadow
            globalAutoOrder = fetchedSettings?.globalEnabled ?? false
            executionMode = normalizeMode(fetchedSettings?.executionMode, global: fetchedSettings?.globalEnabled ?? false)
            localToggleStates.removeAll()
            syncMessage = settingsSyncMessage ?? forecastSyncMessage
            isLoading = false
        }
    }

    private func normalizeMode(_ raw: String?, global: Bool) -> String {
        switch raw?.lowercased() {
        case "off", "shadow", "draft", "place": return raw!.lowercased()
        default: return global ? "draft" : "off"
        }
    }

    private func loadShadowProposals() async -> [AutoOrderShadowProposal] {
        do {
            return try await api.getAutoOrderShadowProposals().items
        } catch {
            return shadowProposals
        }
    }

    private func setExecutionMode(_ mode: String) async {
        do {
            _ = try await api.setAutoOrderExecutionMode(mode: mode)
            await loadAll()
            syncMessage = {
                switch mode {
                case "off": return "Auto-order off."
                case "shadow": return "Mode: Shadow (proposals only — recommended)."
                case "draft": return "Mode: Draft cart lines."
                case "place": return "Mode: Place (still requires Place now + server flag)."
                default: return "Mode updated."
                }
            }()
        } catch {
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Cannot change execution mode for this account.",
                offline: "Offline mode active. Reconnect and retry.",
                fallback: "Execution mode update failed.",
            )
        }
    }

    private func loadRuns() async -> [AutoOrderRun] {
        do {
            return try await api.getAutoOrderRuns().items
        } catch {
            return runs
        }
    }

    private func loadSuggestions() async -> [RetailerReorderSuggestion] {
        do {
            return try await api.getReorderSuggestions().items
        } catch {
            return reorderSuggestions
        }
    }

    private func runAutoOrder(mode: String) async {
        running = true
        runningMode = mode
        do {
            let run = try await api.runAutoOrder(mode: mode)
            lastRun = run
            let placed = run.placedLines ?? 0
            if mode == "place" {
                if placed > 0 {
                    let orders = run.placedOrders?.count ?? 0
                    syncMessage = "Place run: \(placed) line(s) in \(orders) order(s)" + (run.message.map { " — \($0)" } ?? "")
                } else {
                    syncMessage = "Place run \(run.status)\(run.message.map { ": \($0)" } ?? "")"
                }
            } else if mode == "shadow" && (run.status == "OK" || run.status == "PARTIAL") {
                syncMessage = "Shadow run: \(run.draftLines) proposal(s)" + (run.message.map { " — \($0)" } ?? "")
            } else if run.status == "OK" || run.status == "PARTIAL" {
                syncMessage = "Draft run complete: \(run.draftLines) line(s)" + (run.message.map { " — \($0)" } ?? "")
            } else {
                syncMessage = "Run \(run.status)\(run.message.map { ": \($0)" } ?? "")"
            }
            runs = await loadRuns()
            reorderSuggestions = await loadSuggestions()
            shadowProposals = await loadShadowProposals()
        } catch {
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Auto-order requires order.place (and manager role for place).",
                offline: "Offline mode active. Reconnect and retry auto-order run.",
                fallback: "Auto-order run failed. Check geo, flag, and permissions.",
            )
        }
        running = false
        runningMode = nil
    }

    private func loadSettings() async -> (AutoOrderSettings?, String?) {
        do {
            return (try await api.getAutoOrderSettings(), nil)
        } catch {
            return (
                nil,
                RetailerErrorSupport.message(
                    for: error,
                    restricted: "Auto-order settings access is restricted for this account.",
                    offline: "Offline mode active. Showing latest auto-order settings.",
                    fallback: "Auto-order settings sync is degraded. Retry is available.",
                )
            )
        }
    }

    private func loadForecasts() async -> ([DemandForecast], String?) {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            return (try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)"), nil)
        } catch {
            return (
                [],
                RetailerErrorSupport.message(
                    for: error,
                    restricted: "Prediction access is restricted for this account.",
                    offline: "Offline mode active. Predictions will resume after reconnect.",
                    fallback: "Predictions sync is degraded. Retry is available.",
                )
            )
        }
    }

    private func enableGlobal(useHistory: Bool) async {
        do {
            let _: [String: Bool] = try await api.setGlobalAutoOrder(enabled: true, useHistory: useHistory)
            syncMessage = nil
            await loadAll()
        } catch {
            globalAutoOrder = false
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Global auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry global auto-order update.",
                fallback: "Global auto-order update failed. Please try again.",
            )
        }
    }

    private func disableGlobal() async {
        do {
            let _: [String: Bool] = try await api.setGlobalAutoOrder(enabled: false)
            syncMessage = nil
            await loadAll()
        } catch {
            globalAutoOrder = true
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Global auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry global auto-order update.",
                fallback: "Global auto-order update failed. Please try again.",
            )
        }
    }

    private func toggleOverride(item: OverrideItem, enabled: Bool, useHistory: Bool) async {
        do {
            let historyFlag = enabled ? useHistory : nil
            switch item.level {
            case .supplier:
                _ = try await api.setSupplierAutoOrder(supplierId: item.id, enabled: enabled, useHistory: historyFlag)
            case .category:
                _ = try await api.setCategoryAutoOrder(categoryId: item.id, enabled: enabled, useHistory: historyFlag)
            case .product:
                _ = try await api.setProductAutoOrder(productId: item.id, enabled: enabled, useHistory: historyFlag)
            case .variant:
                _ = try await api.setVariantAutoOrder(skuId: item.id, enabled: enabled, useHistory: historyFlag)
            }
            syncMessage = nil
            await loadAll()
        } catch {
            let message = RetailerErrorSupport.message(
                for: error,
                restricted: "Scoped auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry scoped auto-order update.",
                fallback: "Scoped auto-order update failed. Please try again.",
            )
            await loadAll() // revert to server state on failure
            syncMessage = message
        }
    }

    private func confirmEnable(useHistory: Bool) async {
        guard let target = pendingTarget else { return }
        let captured = target
        pendingTarget = nil
        switch captured {
        case .global:
            await enableGlobal(useHistory: useHistory)
        case .supplier(let id):
            let item = OverrideItem(id: id, label: "", enabled: false, hasHistory: false, level: .supplier)
            await toggleOverride(item: item, enabled: true, useHistory: useHistory)
        case .category(let id):
            let item = OverrideItem(id: id, label: "", enabled: false, hasHistory: false, level: .category)
            await toggleOverride(item: item, enabled: true, useHistory: useHistory)
        case .product(let id):
            let item = OverrideItem(id: id, label: "", enabled: false, hasHistory: false, level: .product)
            await toggleOverride(item: item, enabled: true, useHistory: useHistory)
        case .variant(let id):
            let item = OverrideItem(id: id, label: "", enabled: false, hasHistory: false, level: .variant)
            await toggleOverride(item: item, enabled: true, useHistory: useHistory)
        }
    }
}


// MARK: - AnyCodable Helper

struct AnyCodable: Encodable {
    private let value: [String: Any]

    init(_ value: [String: Any]) {
        self.value = value
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: DynamicCodingKey.self)
        for (key, val) in value {
            let codingKey = DynamicCodingKey(stringValue: key)
            if let boolVal = val as? Bool {
                try container.encode(boolVal, forKey: codingKey)
            } else if let stringVal = val as? String {
                try container.encode(stringVal, forKey: codingKey)
            } else if let intVal = val as? Int {
                try container.encode(intVal, forKey: codingKey)
            } else if let doubleVal = val as? Double {
                try container.encode(doubleVal, forKey: codingKey)
            }
        }
    }
}

private struct DynamicCodingKey: CodingKey {
    var stringValue: String
    var intValue: Int?

    init(stringValue: String) {
        self.stringValue = stringValue
        self.intValue = nil
    }

    init?(intValue: Int) {
        self.stringValue = "\(intValue)"
        self.intValue = intValue
    }
}

#Preview {
    AutoOrderView()
}
