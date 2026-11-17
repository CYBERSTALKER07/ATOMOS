import Foundation
import Observation
import SwiftUI

@Observable
@MainActor
final class StoreStockViewModel {
    var rawRows: [StoreStockRow] = []
    var locationId: String = ""
    var banner: String?
    var loadError: String?
    var busy = false
    var searchQuery = ""
    var selectedFilter: StockFilterTab = .all

    // Modal Sheet Controls
    var showReceiveSheet = false
    var showPutawaySheet = false
    var showAdjustSheet = false
    var selectedSkuForAction: String?

    // Returns / Claim Picker
    var showOrderPicker = false
    var preferredSku: String?
    var claimableOrders: [Order] = []
    var pickerQuery = ""
    var pickerLoading = false
    var pickerError: String?
    var claimOrder: Order?
    var claimPreferredSku: String?

    private let api = APIClient.shared

    var aggregatedItems: [AggregatedStockItem] {
        var dict: [String: AggregatedStockItem] = [:]
        for row in rawRows {
            var item = dict[row.sku] ?? AggregatedStockItem(sku: row.sku)
            let upperBin = row.bin.uppercased()
            if upperBin.contains("FLOOR") || upperBin.contains("SHELF") {
                item.floorOnHand += row.onHand
                item.floorAvailable += row.available
            } else {
                item.backroomOnHand += row.onHand
                item.backroomAvailable += row.available
            }
            dict[row.sku] = item
        }
        return dict.values.sorted { $0.sku < $1.sku }
    }

    var filteredItems: [AggregatedStockItem] {
        let trimmed = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        return aggregatedItems.filter { item in
            let matchesSearch = trimmed.isEmpty || item.sku.localizedCaseInsensitiveContains(trimmed)
            let matchesFilter: Bool
            switch selectedFilter {
            case .all:
                matchesFilter = true
            case .lowShelf:
                matchesFilter = item.isLowShelf
            case .backroomReserve:
                matchesFilter = item.hasBackroom
            }
            return matchesSearch && matchesFilter
        }
    }

    var filteredOrders: [Order] {
        let q = pickerQuery.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if q.isEmpty { return claimableOrders }
        return claimableOrders.filter { $0.id.lowercased().contains(q) }
    }

    var totalStoreUnits: Int64 {
        aggregatedItems.reduce(0) { $0 + $1.totalOnHand }
    }

    var totalFloorOnHand: Int64 {
        aggregatedItems.reduce(0) { $0 + $1.floorOnHand }
    }

    var totalBackroomOnHand: Int64 {
        aggregatedItems.reduce(0) { $0 + $1.backroomOnHand }
    }

    var lowShelfCount: Int {
        aggregatedItems.filter(\.isLowShelf).count
    }

    func load() async {
        loadError = nil
        do {
            if locationId.isEmpty {
                let locs = try await api.getLocations()
                locationId = locs.activeLocationId ?? locs.items.first?.locationId ?? ""
            }
            let res = try await api.getStoreStock(locationId: locationId.isEmpty ? nil : locationId)
            rawRows = res.items.map {
                StoreStockRow(
                    id: "\($0.sku)-\($0.stockBin)",
                    sku: $0.sku,
                    bin: $0.stockBin,
                    onHand: $0.onHand,
                    available: $0.available ?? $0.onHand
                )
            }
        } catch {
            rawRows = []
            loadError = error.localizedDescription
        }
    }

    func quickPutaway(sku: String, qtyToMove: Int64) async {
        guard qtyToMove > 0 else { return }
        Haptics.medium()
        busy = true
        defer { busy = false }
        do {
            _ = try await api.transferStoreStock(
                locationId: locationId.isEmpty ? "default" : locationId,
                sku: sku,
                qty: qtyToMove
            )
            Haptics.success()
            banner = "Putaway completed: +\(qtyToMove) \(sku) moved to sales shelf."
            await load()
        } catch {
            Haptics.error()
            banner = "Putaway failed: \(error.localizedDescription)"
        }
    }

    func openOrderPicker(skuHint: String? = nil) {
        Haptics.selection()
        preferredSku = skuHint
        showOrderPicker = true
        Task { await loadClaimableOrders() }
    }

    func loadClaimableOrders() async {
        pickerLoading = true
        pickerError = nil
        let rid = AuthManager.shared.currentUser?.id ?? ""
        var orders: [Order] = []
        if !rid.isEmpty {
            let res: [Order] = (try? await api.get(path: "/v1/retailers/\(rid)/orders")) ?? []
            orders = res.filter { $0.status == .completed || $0.status == .deliveredOnCredit }
        }
        if orders.isEmpty {
            if let tracking = try? await api.getTracking() {
                let eligible = (tracking.orders + (tracking.recentReceipts ?? []))
                    .filter { $0.state.uppercased() == "COMPLETED" || $0.state.uppercased() == "DELIVERED_ON_CREDIT" }
                orders = eligible.map { $0.asClaimOrder() }
            }
        }
        claimableOrders = orders
        pickerLoading = false
    }

    func pickOrder(_ order: Order) async {
        Haptics.selection()
        pickerLoading = true
        defer { pickerLoading = false }
        var next = order
        if next.items.isEmpty {
            if let tracking = try? await api.getTracking() {
                let pool = tracking.orders + (tracking.recentReceipts ?? [])
                if let hit = pool.first(where: { $0.orderId == order.id }) {
                    next = hit.asClaimOrder()
                }
            }
        }
        claimPreferredSku = preferredSku
        claimOrder = next
        showOrderPicker = false
    }
}

@Observable
@MainActor
final class StoreStockActionViewModel {
    var orderId: String = ""
    var sku: String = ""
    var qty: String = "10"
    var qtyDelta: String = "0"
    var busy = false
    var errorMessage: String?

    private let api = APIClient.shared

    func configureInitialSku(initialSku: String, candidates: [String]) {
        if !initialSku.isEmpty {
            sku = initialSku
        } else if let first = candidates.first {
            sku = first
        }
    }

    func submitReceive(locationId: String) async -> Bool {
        let trimmed = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        busy = true
        errorMessage = nil
        defer { busy = false }
        do {
            _ = try await api.receiveStoreStock(orderId: trimmed, locationId: locationId)
            Haptics.success()
            return true
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            return false
        }
    }

    func submitPutaway(locationId: String) async -> Bool {
        let trimmed = sku.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        busy = true
        errorMessage = nil
        defer { busy = false }
        do {
            let amount = max(1, Int64(qty) ?? 1)
            _ = try await api.transferStoreStock(locationId: locationId, sku: trimmed, qty: amount)
            Haptics.success()
            return true
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            return false
        }
    }

    func submitAdjust(locationId: String) async -> Bool {
        let trimmed = sku.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        busy = true
        errorMessage = nil
        defer { busy = false }
        do {
            let delta = Int64(qtyDelta) ?? 0
            _ = try await api.adjustStoreStock(locationId: locationId, sku: trimmed, qtyDelta: delta)
            Haptics.success()
            return true
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            return false
        }
    }
}
