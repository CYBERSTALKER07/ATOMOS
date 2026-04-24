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

    @State private var settings: AutoOrderSettings?
    @State private var forecasts: [DemandForecast] = []
    @State private var isLoading = true
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
                        headerCard.slideIn(delay: 0)
                        globalToggleCard.slideIn(delay: 0.05)

                        if let s = settings {
                            if !s.supplierOverrides.isEmpty {
                                overridesSection(
                                    title: "Supplier Overrides",
                                    icon: "building.2",
                                    items: s.supplierOverrides.map { OverrideItem(id: $0.supplierID, label: $0.supplierName ?? $0.supplierID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .supplier) }
                                )
                                .slideIn(delay: 0.1)
                            }

                            if !s.categoryOverrides.isEmpty {
                                overridesSection(
                                    title: "Category Overrides",
                                    icon: "square.grid.2x2",
                                    items: s.categoryOverrides.map { OverrideItem(id: $0.categoryID, label: $0.categoryID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .category) }
                                )
                                .slideIn(delay: 0.125)
                            }

                            if !s.productOverrides.isEmpty {
                                overridesSection(
                                    title: "Product Overrides",
                                    icon: "leaf",
                                    items: s.productOverrides.map { OverrideItem(id: $0.productID, label: $0.productName ?? $0.productID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .product) }
                                )
                                .slideIn(delay: 0.15)
                            }

                            if !s.variantOverrides.isEmpty {
                                overridesSection(
                                    title: "Variant / SKU Overrides",
                                    icon: "cube",
                                    items: s.variantOverrides.map { OverrideItem(id: $0.skuID, label: $0.skuLabel ?? $0.skuID, enabled: $0.enabled, hasHistory: $0.hasHistory, level: .variant) }
                                )
                                .slideIn(delay: 0.2)
                            }
                        }

                        if !forecasts.isEmpty {
                            predictionsSection.slideIn(delay: 0.25)
                        }

                        engineExplainerCard.slideIn(delay: 0.3)
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

    // MARK: - Header

    private var headerCard: some View {
        GradientHeaderCard(title: "Empathy Engine", subtitle: "Auto-order intelligence with 5-level control", icon: "wand.and.stars") {
            HStack(spacing: AppTheme.spacingXL) {
                miniStat(value: "\(settings?.supplierOverrides.count ?? 0)", label: "Suppliers")
                miniStat(value: "\(settings?.categoryOverrides.count ?? 0)", label: "Categories")
                miniStat(value: "\(settings?.productOverrides.count ?? 0)", label: "Products")
                miniStat(value: "\(forecasts.count)", label: "Predictions")
            }
        }
    }

    private func miniStat(value: String, label: String) -> some View {
        VStack(spacing: 3) {
            Text(value)
                .font(.system(.headline, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(label)
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
    }

    // MARK: - Global Toggle

    private var globalToggleCard: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(globalAutoOrder ? AppTheme.accent.opacity(0.15) : AppTheme.surfaceElevated)
                            .frame(width: 40, height: 40)
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(globalAutoOrder ? AppTheme.accent : AppTheme.textSecondary)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text("Global Auto-Order")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Auto-order everything from all suppliers")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    Toggle("", isOn: Binding(
                        get: { globalAutoOrder },
                        set: { newVal in
                            globalAutoOrder = newVal
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
                    ))
                        .tint(AppTheme.accent)
                        .labelsHidden()
                }
                .padding(AppTheme.spacingLG)

                if globalAutoOrder {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.success)
                        Text("Global auto-order active. Overrides all supplier/product settings.")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }

                if let dateStr = settings?.analyticsStartDate {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "calendar.badge.clock")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.accent)
                        Text("Analytics since: \(dateStr)")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                }
            }
        }
        .animation(AnimationConstants.express, value: globalAutoOrder)
    }

    // MARK: - Override Section

    private func overridesSection(title: String, icon: String, items: [OverrideItem]) -> some View {
        LabCardWithHeader(title: title, icon: icon) {
            VStack(spacing: 0) {
                ForEach(items) { item in
                    HStack(spacing: AppTheme.spacingMD) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(item.label)
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textPrimary)
                                .lineLimit(1)
                            Text(item.level.subtitle)
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        Spacer()

                        Toggle("", isOn: Binding(
                            get: { localToggleStates[item.id] ?? item.enabled },
                            set: { newVal in
                                localToggleStates[item.id] = newVal
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
                        ))
                        .tint(AppTheme.accent)
                        .labelsHidden()
                        .scaleEffect(0.85)
                    }
                    .padding(.vertical, AppTheme.spacingSM)
                    .padding(.horizontal, AppTheme.spacingXS)

                    if item.id != items.last?.id {
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.2))
                            .frame(height: AppTheme.separatorHeight)
                    }
                }
            }
        }
    }

    // MARK: - Predictions

    private var predictionsSection: some View {
        LabCardWithHeader(title: "Active Predictions", icon: "sparkles") {
            VStack(spacing: AppTheme.spacingMD) {
                ForEach(forecasts) { forecast in
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            Circle()
                                .stroke(AppTheme.separator.opacity(0.3), lineWidth: 2)
                                .frame(width: 36, height: 36)
                            Circle()
                                .trim(from: 0, to: forecast.confidence)
                                .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 2, lineCap: .round))
                                .frame(width: 36, height: 36)
                                .rotationEffect(.degrees(-90))
                            Text(forecast.confidencePercent)
                                .font(.system(size: 9, weight: .bold, design: .rounded))
                                .foregroundStyle(confidenceColor(forecast.confidence))
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(forecast.productName)
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textPrimary)
                                .lineLimit(1)
                            Text("Order by \(forecast.suggestedOrderDate)")
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        Spacer()

                        VStack(spacing: 1) {
                            Text("\(forecast.predictedQuantity)")
                                .font(.system(.headline, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.accent)
                            Text("units")
                                .font(.system(size: 8, weight: .medium, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }

                    if forecast.id != forecasts.last?.id {
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.15))
                            .frame(height: AppTheme.separatorHeight)
                    }
                }
            }
        }
    }

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }

    // MARK: - Engine Explainer

    private var engineExplainerCard: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "info.circle.fill")
                        .font(.system(size: 14))
                        .foregroundStyle(AppTheme.accent)
                    Text("How It Works")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                }

                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    explainerRow(num: "1", text: "The AI analyzes your purchase patterns even when auto-order is off")
                    explainerRow(num: "2", text: "When you enable, choose to use your history or start fresh")
                    explainerRow(num: "3", text: "Starting fresh requires at least 2 orders per product")
                    explainerRow(num: "4", text: "Overrides: Variant > Product > Category > Supplier > Global")
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func explainerRow(num: String, text: String) -> some View {
        HStack(alignment: .top, spacing: AppTheme.spacingSM) {
            Text(num)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(.white)
                .frame(width: 20, height: 20)
                .background(AppTheme.accent)
                .clipShape(.circle)
            Text(text)
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
        }
    }

    // MARK: - API

    private func loadAll() async {
        if settings == nil { isLoading = true }
        async let settingsReq: AutoOrderSettings? = loadSettings()
        async let forecastsReq: [DemandForecast] = loadForecasts()
        
        let fetchedSettings = await settingsReq
        let fetchedForecasts = await forecastsReq
        
        withAnimation(AnimationConstants.fluid) {
            settings = fetchedSettings
            forecasts = fetchedForecasts
            globalAutoOrder = fetchedSettings?.globalEnabled ?? false
            localToggleStates.removeAll()
            isLoading = false
        }
    }

    private func loadSettings() async -> AutoOrderSettings? {
        do {
            return try await api.get(path: "/v1/retailer/settings/auto-order")
        } catch {
            return nil
        }
    }

    private func loadForecasts() async -> [DemandForecast] {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            return try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
        } catch {
            return []
        }
    }

    private func enableGlobal(useHistory: Bool) async {
        do {
            let body: [String: Any] = ["global_auto_order_enabled": true, "use_history": useHistory]
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/global",
                body: AnyCodable(body)
            )
            await loadAll()
        } catch {
            globalAutoOrder = false
        }
    }

    private func disableGlobal() async {
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/global",
                body: ["global_auto_order_enabled": false]
            )
            await loadAll()
        } catch {
            globalAutoOrder = true
        }
    }

    private func toggleOverride(item: OverrideItem, enabled: Bool, useHistory: Bool) async {
        let path: String
        switch item.level {
        case .supplier:
            path = "/v1/retailer/settings/auto-order/supplier/\(item.id)"
        case .category:
            path = "/v1/retailer/settings/auto-order/category/\(item.id)"
        case .product:
            path = "/v1/retailer/settings/auto-order/product/\(item.id)"
        case .variant:
            path = "/v1/retailer/settings/auto-order/variant/\(item.id)"
        }
        do {
            var body: [String: Any] = ["auto_order_enabled": enabled]
            if enabled { body["use_history"] = useHistory }
            let _: [String: Bool] = try await api.patch(
                path: path,
                body: AnyCodable(body)
            )
            await loadAll()
        } catch {
            await loadAll() // revert to server state on failure
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

// MARK: - Override Item Model

private struct OverrideItem: Identifiable {
    let id: String
    let label: String
    let enabled: Bool
    let hasHistory: Bool
    let level: OverrideLevel
}

private enum OverrideLevel {
    case supplier, category, product, variant

    var subtitle: String {
        switch self {
        case .supplier: "Supplier-level override"
        case .category: "Category-level override"
        case .product: "Product-level override"
        case .variant: "Variant / SKU override"
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
