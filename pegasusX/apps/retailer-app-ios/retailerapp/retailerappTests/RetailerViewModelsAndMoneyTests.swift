import Foundation
import SwiftUI
import Testing
@testable import retailerapp

@MainActor
struct RetailerViewModelsAndMoneyTests {

    // MARK: - 1. Pure Int64 Money Formatting & Lossless Decoding

    @Test func retailerMoney_formatTiyin_handlesZeroPositiveNegativeAndSubUnitNegative() {
        #expect(RetailerMoney.formatTiyin(0) == "0.00")
        #expect(RetailerMoney.formatTiyin(12_500) == "125.00")
        #expect(RetailerMoney.formatTiyin(12_505) == "125.05")
        #expect(RetailerMoney.formatTiyin(-5) == "-0.05")
        #expect(RetailerMoney.formatTiyin(-105) == "-1.05")
        #expect(RetailerMoney.formatTiyin(999_999_999) == "9999999.99")
    }

    @Test func variantDecoding_decodesIntegerDecimalAndMinorPriceIntoInt64() throws {
        let intJson = Data(#"{"id":"v-int","size":"1L","pack":"Box","pack_count":6,"weight_per_unit":"1kg","price":18500}"#.utf8)
        let vInt = try JSONDecoder().decode(Variant.self, from: intJson)
        #expect(vInt.priceMinor == 18_500)
        #expect(RetailerMoney.formatTiyin(vInt.priceMinor) == "185.00")

        let decJson = Data(#"{"id":"v-dec","size":"2L","pack":"Single","pack_count":1,"weight_per_unit":"2kg","price":24000.0}"#.utf8)
        let vDec = try JSONDecoder().decode(Variant.self, from: decJson)
        #expect(vDec.priceMinor == 24_000)

        let minorJson = Data(#"{"id":"v-min","size":"500ml","pack":"Case","pack_count":12,"weight_per_unit":"500g","price_minor":9900}"#.utf8)
        let vMin = try JSONDecoder().decode(Variant.self, from: minorJson)
        #expect(vMin.priceMinor == 9_900)

        let encoded = try JSONEncoder().encode(vMin)
        let roundTrip = try JSONDecoder().decode(Variant.self, from: encoded)
        #expect(roundTrip.priceMinor == 9_900)
        #expect(roundTrip.id == "v-min")
    }

    @Test func productDisplayPrice_usesInt64MinorUnitsAndSalePrice() {
        let variant = Variant(id: "v-1", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1kg", priceMinor: 15_000)
        let offer = ProductOffer(
            productID: "prod-1",
            listPriceMinor: 20_000,
            salePriceMinor: 15_000,
            discountBps: 2500,
            promotionID: "promo-1",
            promotionName: "Flash Sale",
            promotionLabel: "-25%",
            promotionEndsAt: nil
        )
        let product = Product(
            id: "prod-1",
            name: "Milk",
            description: "Whole milk",
            nutrition: "",
            imageURL: nil,
            variants: [variant],
            priceMinor: 15_000,
            offer: offer
        )
        #expect(product.hasSaleOffer == true)
        #expect(product.displayPrice.contains("15"))
        #expect(product.displayListPrice?.contains("20") == true)
    }

    // MARK: - 2. CartManager Int64 Checkout & Preview Cap Arithmetic

    @Test func cartManager_checkoutTotalsAndPreviewCapsUsePureInt64() throws {
        let cart = CartManager()
        cart.clear()
        let v1 = Variant(id: "sku-1", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1kg", priceMinor: 12_000)
        let p1 = Product(id: "p-1", name: "Ayran", description: "", nutrition: "", imageURL: nil, variants: [v1])
        cart.add(product: p1, variant: v1, quantity: 5)

        #expect(cart.totalPriceMinor == 60_000)
        #expect(cart.checkoutSubtotalMinor == 60_000)
        cart.quotedDiscountMinor = 15_000
        #expect(cart.checkoutTotalMinor == 45_000)
        // Clamps at 0 when discount exceeds subtotal
        cart.quotedDiscountMinor = 100_000
        #expect(cart.checkoutTotalMinor == 0)
        cart.quotedDiscountMinor = 0

        let previewJson = Data(#"{"ok":true,"stock_warnings":[],"orderable_quantities":{"sku-1":3},"default_out_of_stock_policy":"REJECT"}"#.utf8)
        let preview = try JSONDecoder().decode(CheckoutPreviewResponse.self, from: previewJson)
        #expect(cart.maxQuantity(for: cart.items[0], preview: preview) == 3)
        cart.applyPreviewCaps(preview)
        #expect(cart.items[0].quantity == 3)
        #expect(cart.totalPriceMinor == 36_000)
        cart.clear()
    }

    // MARK: - 3. Invariant I1 (UUIDv7 Idempotency & Keychain POS Queue) + Invariant I4 (Monotonic Clock)

    @Test func retailerIdempotency_uuidV7_conformsToRFC9562AndOrdersMonotonically() {
        let idEarly = RetailerIdempotency.uuidV7(timestampMs: 1_700_000_000_000)
        let idLate = RetailerIdempotency.uuidV7(timestampMs: 1_700_000_005_000)

        #expect(idEarly.count == 36)
        #expect(idLate.count == 36)

        let parts = idEarly.split(separator: "-")
        #expect(parts.count == 5)
        #expect(parts[2].first == "7")
        let variantChar = parts[3].first ?? "0"
        #expect(["8", "9", "a", "b"].contains(String(variantChar)))
        #expect(String(idEarly.prefix(13)) < String(idLate.prefix(13)))
    }

    @Test func pendingPosStore_persistsUUIDv7EntriesAndSupportsLifecycle() throws {
        let store = PendingPosStore.shared
        store.clearAll()

        let entry = try store.enqueue(
            sessionId: "sess-100",
            lines: [PosSaleLineWire(sku: "SKU-A", name: "Bread", qty: 2, unitPriceMinor: 5_000)],
            totalMinor: 10_000
        )
        #expect(entry.clientSaleId.count == 36)
        #expect(entry.clientSaleId.split(separator: "-")[2].first == "7")
        #expect(store.countActive(sessionId: "sess-100") == 1)

        var updated = entry
        updated.status = "FAILED"
        updated.retryCount = 1
        updated.lastError = "Network down"
        store.update(updated)

        let fetched = store.active().first { $0.clientSaleId == entry.clientSaleId }
        #expect(fetched?.status == "FAILED")
        #expect(fetched?.retryCount == 1)

        store.remove(clientSaleId: entry.clientSaleId)
        #expect(store.countActive(sessionId: "sess-100") == 0)
        store.clearAll()
    }

    @Test func countdownTimer_monotonicClockIgnoresWallClockRollback() {
        let timer = CountdownTimer()
        let anchorWall = Date(timeIntervalSince1970: 1_700_000_000)
        let targetWall = anchorWall.addingTimeInterval(3600) // 1 hour from anchor
        timer.start(from: targetWall, anchorNow: anchorWall, anchorUptime: 1000.0)

        // Advance monotonic uptime by 600s (10 minutes), regardless of wall-clock
        timer.updateMonotonic(currentUptime: 1600.0)
        #expect(timer.remainingString == "00:50:00")
        #expect(timer.isExpired == false)
    }

    // MARK: - 4. ViewModels Unit Tests (@MainActor)

    @Test @MainActor func storeStockViewModel_computesTotalsLowShelfAndFilters() {
        let vm = StoreStockViewModel()
        vm.rawRows = [
            StoreStockRow(id: "1", sku: "MILK-1L", bin: "FLOOR", onHand: 3, available: 3),
            StoreStockRow(id: "2", sku: "MILK-1L", bin: "BACKROOM", onHand: 20, available: 20),
            StoreStockRow(id: "3", sku: "BREAD-1", bin: "FLOOR", onHand: 12, available: 12)
        ]

        #expect(vm.totalFloorOnHand == 15)
        #expect(vm.totalBackroomOnHand == 20)
        #expect(vm.totalStoreUnits == 35)
        #expect(vm.lowShelfCount == 1)

        vm.selectedFilter = .backroomReserve
        #expect(vm.filteredItems.count == 1)
        #expect(vm.filteredItems.first?.sku == "MILK-1L")

        vm.selectedFilter = .all
        vm.searchQuery = "bread"
        #expect(vm.filteredItems.count == 1)
        #expect(vm.filteredItems.first?.sku == "BREAD-1")
    }

    @Test @MainActor func appShellViewModel_managesNavigationAndFloatingBarVisibility() {
        let vm = AppShellViewModel()
        #expect(vm.showFloatingBar(horizontalSizeClass: .compact) == true)
        vm.selectedTab = .more
        #expect(vm.showFloatingBar(horizontalSizeClass: .compact) == false)

        vm.deliveriesHubInitialTab = .dock
        vm.handleTabChange(from: .home, to: .deliveries)
        #expect(vm.deliveriesHubInitialTab == .map)

        vm.navigateToSuppliersTab(horizontalSizeClass: .regular)
        #expect(vm.sideMenuSelection == .suppliers)
    }

    @Test @MainActor func checkoutViewModel_mapsGatewayCodesAndDetectsOosItems() {
        let vm = CheckoutViewModel()
        vm.catalogCodes = ["CASH", "GLOBAL_PAY"]
        #expect(vm.gatewayCode(for: "GlobalPay") == "GLOBAL_PAY")
        #expect(vm.gatewayCode(for: "Cash") == "CASH")

        let v = Variant(id: "sku-oos", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1kg", priceMinor: 10_000)
        let p = Product(id: "prod-oos", name: "Juice", description: "", nutrition: "", imageURL: nil, variants: [v])
        let cartItem = CartItem(id: "prod-oos-sku-oos", product: p, variant: v, quantity: 2)

        vm.oosItems = ["sku-oos"]
        #expect(vm.isOos(cartItem) == true)
    }

    @Test @MainActor func productDetailViewModel_selectsVariantAddsToCartAndCancelsPendingEnable() {
        let vm = ProductDetailViewModel()
        let v1 = Variant(id: "v-1", size: "500ml", pack: "Single", packCount: 1, weightPerUnit: "500g", priceMinor: 8_000)
        let v2 = Variant(id: "v-2", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1kg", priceMinor: 14_000)
        let product = Product(id: "p-10", name: "Water", description: "", nutrition: "", imageURL: nil, variants: [v1, v2])

        vm.configure(for: product)
        #expect(vm.selectedVariant?.id == "v-1")

        vm.quantity = 4
        vm.selectVariant(v2)
        #expect(vm.selectedVariant?.id == "v-2")
        #expect(vm.quantity == 1)

        let cart = CartManager()
        cart.clear()
        vm.quantity = 3
        vm.addToCart(product: product, cart: cart)
        #expect(cart.totalItems == 3)
        #expect(cart.totalPriceMinor == 42_000)
        cart.clear()

        vm.pendingProductEnable = true
        vm.productAutoOrderEnabled = true
        vm.pendingVariantTarget = "v-2"
        vm.variantAutoOrder["v-2"] = true
        vm.showHistoryAlert = true
        vm.cancelPendingEnable()
        #expect(vm.productAutoOrderEnabled == false)
        #expect(vm.variantAutoOrder["v-2"] == false)
        #expect(vm.showHistoryAlert == false)
    }
}
