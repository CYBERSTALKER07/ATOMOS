import Foundation
import SwiftUI
import Observation

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

struct AutoOrderSoakDecision: Codable {
    let allowed: Bool
    let reasons: [String]?
    let stats: AutoOrderShadowStats?
    let bypassSource: String?

    enum CodingKeys: String, CodingKey {
        case allowed
        case reasons
        case stats
        case bypassSource = "bypass_source"
    }
}

struct AutoOrderSoakThresholds: Codable {
    let minProposals: Int64?
    let maxWape: Double?
    let minUnmodified: Double?
    let gateDisabled: Bool?
    let bypassSource: String?

    enum CodingKeys: String, CodingKey {
        case minProposals = "min_proposals"
        case maxWape = "max_wape"
        case minUnmodified = "min_unmodified"
        case gateDisabled = "gate_disabled"
        case bypassSource = "bypass_source"
    }
}

struct AutoOrderSoakGate: Codable {
    let decision: AutoOrderSoakDecision?
    let thresholds: AutoOrderSoakThresholds?
    let placeFlagEnabled: Bool?

    enum CodingKeys: String, CodingKey {
        case decision
        case thresholds
        case placeFlagEnabled = "place_flag_enabled"
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

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(skuID, forKey: .skuID)
        try c.encodeIfPresent(productID, forKey: .productID)
        try c.encode(enabled, forKey: .enabled)
        try c.encode(hasHistory, forKey: .hasHistory)
        try c.encodeIfPresent(skuLabel, forKey: .skuLabel)
        try c.encodeIfPresent(analyticsStartDate, forKey: .analyticsStartDate)
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

// MARK: - ViewModel

@Observable
@MainActor
final class AutoOrderViewModel {
    enum EnableTarget {
        case global
        case supplier(String)
        case category(String)
        case product(String)
        case variant(String)
    }

    var settings: AutoOrderSettings?
    var forecasts: [DemandForecast] = []
    var isLoading = true
    var syncMessage: String?
    var globalAutoOrder = false
    var executionMode = "draft"
    var pendingTarget: EnableTarget?
    var showHistoryAlert = false
    var localToggleStates: [String: Bool] = [:]
    var runs: [AutoOrderRun] = []
    var lastRun: AutoOrderRun?
    var running = false
    var runningMode: String?
    var placeConfirmOpen = false
    var reorderSuggestions: [RetailerReorderSuggestion] = []
    var shadowProposals: [AutoOrderShadowProposal] = []
    var soakGate: AutoOrderSoakGate?

    private let api = APIClient.shared

    var alertEntityLabel: String {
        switch pendingTarget {
        case .global: return "global auto-order"
        case .supplier: return "this supplier's auto-order"
        case .category: return "this category's auto-order"
        case .product: return "this product's auto-order"
        case .variant: return "this variant's auto-order"
        case nil: return "auto-order"
        }
    }

    func handleGlobalToggle(_ newVal: Bool) {
        Haptics.selection()
        if newVal {
            if settings?.hasAnyHistory == true {
                pendingTarget = .global
                showHistoryAlert = true
            } else {
                Task { await enableGlobal(useHistory: false) }
            }
        } else {
            Task { await disableGlobal() }
        }
    }

    func handleOverrideToggle(item: OverrideItem, newVal: Bool) {
        Haptics.selection()
        if newVal && item.hasHistory {
            switch item.level {
            case .supplier: pendingTarget = .supplier(item.id)
            case .category: pendingTarget = .category(item.id)
            case .product: pendingTarget = .product(item.id)
            case .variant: pendingTarget = .variant(item.id)
            }
            showHistoryAlert = true
        } else {
            Task { await toggleOverride(item: item, enabled: newVal, useHistory: false) }
        }
    }

    func cancelPendingEnable() {
        if let target = pendingTarget {
            switch target {
            case .global:
                globalAutoOrder = false
            case .supplier(let id), .category(let id), .product(let id), .variant(let id):
                localToggleStates[id] = false
            }
        }
        pendingTarget = nil
        showHistoryAlert = false
    }

    func loadAll() async {
        if settings == nil { isLoading = true }
        async let settingsReq: (AutoOrderSettings?, String?) = loadSettings()
        async let forecastsReq: ([DemandForecast], String?) = loadForecasts()
        async let runsReq: [AutoOrderRun] = loadRuns()
        async let suggestionsReq: [RetailerReorderSuggestion] = loadSuggestions()
        async let shadowReq: [AutoOrderShadowProposal] = loadShadowProposals()
        async let soakReq: AutoOrderSoakGate? = loadSoakGate()

        let (fetchedSettings, settingsSyncMessage) = await settingsReq
        let (fetchedForecasts, forecastSyncMessage) = await forecastsReq
        let fetchedRuns = await runsReq
        let fetchedSuggestions = await suggestionsReq
        let fetchedShadow = await shadowReq
        let fetchedSoak = await soakReq

        withAnimation(AnimationConstants.fluid) {
            settings = fetchedSettings
            forecasts = fetchedForecasts
            runs = fetchedRuns
            reorderSuggestions = fetchedSuggestions
            shadowProposals = fetchedShadow
            soakGate = fetchedSoak
            globalAutoOrder = fetchedSettings?.globalEnabled ?? false
            executionMode = normalizeMode(fetchedSettings?.executionMode, global: fetchedSettings?.globalEnabled ?? false)
            localToggleStates.removeAll()
            syncMessage = settingsSyncMessage ?? forecastSyncMessage
            isLoading = false
        }
    }

    private func normalizeMode(_ raw: String?, global: Bool) -> String {
        guard let mode = raw?.lowercased() else {
            return global ? "draft" : "off"
        }
        switch mode {
        case "off", "shadow", "draft", "place": return mode
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

    private func loadSoakGate() async -> AutoOrderSoakGate? {
        do {
            return try await api.getAutoOrderSoakGate()
        } catch {
            return soakGate
        }
    }

    func setExecutionMode(_ mode: String) async {
        Haptics.selection()
        do {
            _ = try await api.setAutoOrderExecutionMode(mode: mode)
            Haptics.success()
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
            Haptics.error()
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Cannot change execution mode for this account.",
                offline: "Offline mode active. Reconnect and retry.",
                fallback: "Execution mode update failed."
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

    func runAutoOrder(mode: String) async {
        Haptics.medium()
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
            Haptics.success()
            runs = await loadRuns()
            reorderSuggestions = await loadSuggestions()
            shadowProposals = await loadShadowProposals()
        } catch {
            Haptics.error()
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Auto-order requires order.place (and manager role for place).",
                offline: "Offline mode active. Reconnect and retry auto-order run.",
                fallback: "Auto-order run failed. Check geo, flag, and permissions."
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
                AutoOrderSettings(
                    globalEnabled: false,
                    executionMode: "shadow",
                    hasAnyHistory: true,
                    analyticsStartDate: "2026-01-01T00:00:00Z",
                    supplierOverrides: [],
                    categoryOverrides: [],
                    productOverrides: [],
                    variantOverrides: [],
                    shadowStats: nil
                ),
                nil
            )
        }
    }

    private func loadForecasts() async -> ([DemandForecast], String?) {
        return ([], nil)
    }

    func enableGlobal(useHistory: Bool) async {
        do {
            let _: [String: Bool] = try await api.setGlobalAutoOrder(enabled: true, useHistory: useHistory)
            Haptics.success()
            syncMessage = nil
            await loadAll()
        } catch {
            Haptics.error()
            globalAutoOrder = false
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Global auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry global auto-order update.",
                fallback: "Global auto-order update failed. Please try again."
            )
        }
    }

    func disableGlobal() async {
        do {
            let _: [String: Bool] = try await api.setGlobalAutoOrder(enabled: false)
            Haptics.success()
            syncMessage = nil
            await loadAll()
        } catch {
            Haptics.error()
            globalAutoOrder = true
            syncMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Global auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry global auto-order update.",
                fallback: "Global auto-order update failed. Please try again."
            )
        }
    }

    func toggleOverride(item: OverrideItem, enabled: Bool, useHistory: Bool) async {
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
            Haptics.success()
            syncMessage = nil
            await loadAll()
        } catch {
            Haptics.error()
            let message = RetailerErrorSupport.message(
                for: error,
                restricted: "Scoped auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry scoped auto-order update.",
                fallback: "Scoped auto-order update failed. Please try again."
            )
            await loadAll()
            syncMessage = message
        }
    }

    func confirmEnable(useHistory: Bool) async {
        guard let target = pendingTarget else { return }
        let captured = target
        pendingTarget = nil
        showHistoryAlert = false
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
