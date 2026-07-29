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
        
        let (fetchedSettings, settingsSyncMessage) = await settingsReq
        let (fetchedForecasts, forecastSyncMessage) = await forecastsReq
        
        withAnimation(AnimationConstants.fluid) {
            settings = fetchedSettings
            forecasts = fetchedForecasts
            globalAutoOrder = fetchedSettings?.globalEnabled ?? false
            localToggleStates.removeAll()
            syncMessage = settingsSyncMessage ?? forecastSyncMessage
            isLoading = false
        }
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
