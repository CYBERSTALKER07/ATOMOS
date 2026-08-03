import SwiftUI

// MARK: - Response Models

struct AutoOrderSettings: Codable {
    let globalEnabled: Bool
    let hasAnyHistory: Bool
    let analyticsStartDate: String?
    let supplierOverrides: [SupplierOverride]
    let categoryOverrides: [CategoryOverride]
    let productOverrides: [ProductOverride]
    let variantOverrides: [VariantOverride]

    enum CodingKeys: String, CodingKey {
        case globalEnabled = "global_enabled"
        case hasAnyHistory = "has_any_history"
        case analyticsStartDate = "analytics_start_date"
        case supplierOverrides = "supplier_overrides"
        case categoryOverrides = "category_overrides"
        case productOverrides = "product_overrides"
        case variantOverrides = "variant_overrides"
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
    let productID: String
    let enabled: Bool
    let hasHistory: Bool
    let skuLabel: String?
    let analyticsStartDate: String?

    enum CodingKeys: String, CodingKey {
        case skuID = "sku_id"
        case productID = "product_id"
        case enabled
        case hasHistory = "has_history"
        case skuLabel = "sku_label"
        case analyticsStartDate = "analytics_start_date"
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
    @AppStorage("globalAutoOrder") private var globalAutoOrder = false
    @State private var pendingTarget: EnableTarget?
    @State private var localToggleStates: [String: Bool] = [:]
    @State private var runs: [AutoOrderRun] = []
    @State private var lastRun: AutoOrderRun?
    @State private var running = false
    @State private var runningMode: String?
    @State private var placeConfirmOpen = false
    @State private var reorderSuggestions: [RetailerReorderSuggestion] = []

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
                                    title: "Variant / SKU Overrides",
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
            .navigationTitle("Auto-Order")
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
                Button("Use History") {
                    Task { await confirmEnable(useHistory: true) }
                }
                Button("Start Fresh", role: .destructive) {
                    Task { await confirmEnable(useHistory: false) }
                }
                Button("Cancel", role: .cancel) {
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
                Text("Enable \(alertEntityLabel) using your existing order history, or start fresh? Starting fresh requires at least 2 orders before predictions begin.")
            })
        }
    }

    // MARK: - Loading

    private var loadingState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ProgressView()
                .tint(AppTheme.accent)
            Text("Loading settings…")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }

    private var autoOrderWorkerCard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Image(systemName: "play.circle.fill")
                    .foregroundStyle(AppTheme.accent)
                Text("Auto-order worker")
                    .font(.system(.headline, design: .rounded))
                Spacer()
            }
            Text("Draft stages cart lines (idempotent per SKU/day). Place creates real supplier orders when the server flag is on.")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            HStack(spacing: AppTheme.spacingSM) {
                Button {
                    Task { await runAutoOrder(mode: "draft") }
                } label: {
                    HStack {
                        if running && runningMode == "draft" {
                            ProgressView()
                        } else {
                            Image(systemName: "play.fill")
                        }
                        Text(running && runningMode == "draft" ? "Drafting…" : "Draft now")
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(running)

                Button {
                    placeConfirmOpen = true
                } label: {
                    HStack {
                        if running && runningMode == "place" {
                            ProgressView().tint(.white)
                        } else {
                            Image(systemName: "cart.fill")
                        }
                        Text(running && runningMode == "place" ? "Placing…" : "Place now")
                    }
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .disabled(running)
            }
            .confirmationDialog(
                "Create real supplier orders?",
                isPresented: $placeConfirmOpen,
                titleVisibility: .visible
            ) {
                Button("Confirm place", role: .destructive) {
                    Task { await runAutoOrder(mode: "place") }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Place mode creates real procurement orders (AUTO_ORDER). Requires primary location geo, place permission, and AUTO_ORDER_PLACE_ENABLED.")
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
                    Text("\(po.orderId)\(po.supplierId.map { " · \($0)" } ?? "") · \(po.lineCount) lines")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.accent)
                }
            }

            if runs.isEmpty {
                Text("No runs yet. Enable auto-order and use Draft or Place.")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else {
                Text("Last runs")
                    .font(.system(.caption, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
                ForEach(runs.prefix(8)) { run in
                    let p = run.placedLines ?? 0
                    HStack {
                        Text("\(run.scheduleBucket ?? String(run.startedAt.prefix(10))) · \(run.mode) · d\(run.draftLines)\(p > 0 ? " p\(p)" : "")")
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
            Text("Reorder suggestions")
                .font(.system(.headline, design: .rounded))
            Text("Sell-through aware OPEN suggestions (Store POS / Wholesale)")
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            if reorderSuggestions.isEmpty {
                Text("No OPEN suggestions yet. POS sell-through and demand batch populate this list.")
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
            Button("Retry") {
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

        let (fetchedSettings, settingsSyncMessage) = await settingsReq
        let (fetchedForecasts, forecastSyncMessage) = await forecastsReq
        let fetchedRuns = await runsReq
        let fetchedSuggestions = await suggestionsReq

        withAnimation(AnimationConstants.fluid) {
            settings = fetchedSettings
            forecasts = fetchedForecasts
            runs = fetchedRuns
            reorderSuggestions = fetchedSuggestions
            globalAutoOrder = fetchedSettings?.globalEnabled ?? false
            localToggleStates.removeAll()
            syncMessage = settingsSyncMessage ?? forecastSyncMessage
            isLoading = false
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
            } else if run.status == "OK" || run.status == "PARTIAL" {
                syncMessage = "Draft run complete: \(run.draftLines) line(s)" + (run.message.map { " — \($0)" } ?? "")
            } else {
                syncMessage = "Run \(run.status)\(run.message.map { ": \($0)" } ?? "")"
            }
            runs = await loadRuns()
            reorderSuggestions = await loadSuggestions()
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
