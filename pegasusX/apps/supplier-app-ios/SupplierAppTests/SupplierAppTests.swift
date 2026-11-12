import XCTest
@testable import SupplierAppIOS
final class SupplierAppTests: XCTestCase {
    func testUnknownBuildIsNotTreatedAsZero() {
        XCTAssertNil(AppBuildVersion.code(nil))
        XCTAssertNil(AppBuildVersion.code("1.0.0"))
        XCTAssertNil(AppBuildVersion.code("  "))
        XCTAssertEqual(AppBuildVersion.code("42"), 42)
        XCTAssertFalse(AppBuildVersion.manifestIsNewer(5, than: nil))
        XCTAssertTrue(AppBuildVersion.manifestIsNewer(5, than: 4))
        XCTAssertFalse(AppBuildVersion.manifestIsNewer(5, than: 5))
        XCTAssertNil(AppBuildVersion.marketingVersion(nil))
        XCTAssertNil(AppBuildVersion.marketingVersion(" "))
        XCTAssertEqual(AppBuildVersion.marketingVersion(" 1.4 "), "1.4")
        XCTAssertFalse(AppBuildVersion.manifestIsNewer(9, than: nil))
    }

    func testFailedClientPolicyDoesNotLookUpToDate() {
        let first = AutoUpdater.failedPolicy(message: "offline", hadBanner: false)
        XCTAssertTrue(first.clearBanner)
        XCTAssertFalse(first.force)
        XCTAssertEqual(first.notice, "offline")

        let refresh = AutoUpdater.failedPolicy(message: "timeout", hadBanner: true)
        XCTAssertFalse(refresh.clearBanner)
        XCTAssertFalse(refresh.force)
        XCTAssertEqual(refresh.notice, "timeout")
    }

    func testLedgerSumIsNotMonthEarnings() {
        XCTAssertNil(EarningsHonesty.minor(9_900, authoritative: false))
        XCTAssertEqual(EarningsHonesty.minor(9_900, authoritative: true), 9_900)
    }

    func testEmptyActionResultIsNotOK() {
        XCTAssertNil(ActionResultText.nonEmpty(status: nil, error: nil))
        XCTAssertNil(ActionResultText.nonEmpty(status: "  ", error: ""))
        XCTAssertEqual(ActionResultText.nonEmpty(status: "disabled", error: nil), "disabled")
        XCTAssertEqual(ActionResultText.nonEmpty(status: nil, error: "forbidden"), "forbidden")
    }

    func testMissingNumberIsADashNotZero() {
        XCTAssertEqual(MissingNumber.label(nil as Int?), "—")
        XCTAssertEqual(MissingNumber.label(nil as Int64?), "—")
        XCTAssertEqual(MissingNumber.label(Optional(0)), "0")
        XCTAssertEqual(MissingNumber.label(Optional(Int64(0))), "0")
    }

    func testMoneyFormatKeepsDigitsPastTheFloatMantissa() {
        let amount: Int64 = 9_007_199_254_740_993
        let formatted = MoneyFormat.minor(amount, currency: "UZS")
        XCTAssertTrue(formatted.hasPrefix("90071992547409.93"))
        XCTAssertEqual(formatMinorUnits(-150, fractionDigits: 2, trimTrailingZeros: true), "-1.5")
        XCTAssertEqual(formatPackMoney(1_500, pack: nil), "15")
    }

    func testMoneyFormatMinorUnits() {
        let formatted = MoneyFormat.minor(150_000, currency: "UZS")
        XCTAssertTrue(formatted.contains("1500"))
        XCTAssertTrue(formatted.contains("UZS"))
        MarketPackStore.clear()
        let empty = MoneyFormat.minor(150_000, currency: "")
        XCTAssertFalse(empty.contains("UZS"))
    }

    func testWarehouseLineIdentityStaysStableWithoutAnId() {
        let line = WarehouseOrderLineItem(
            productId: nil,
            productName: "  ",
            quantity: 2,
            unitPrice: 1500
        )
        XCTAssertEqual(line.id, line.id)
        XCTAssertFalse(line.id.contains("-"))
        XCTAssertEqual(
            WarehouseOrderLineItem(productId: "sku_1", productName: nil, quantity: nil, unitPrice: nil).id,
            "sku_1"
        )
    }

    func testPricingViewFormatPriceUsesMinorUnits() {
        let product = CatalogProduct(
            productId: "prod_1",
            name: "Tea",
            categoryId: "cat_1",
            priceMinor: 48_500_000,
            currency: "UZS",
            unit: "BOX",
            unitVolumeVu: 1.0,
            imageUrl: nil,
            barcode: "1234567890128",
            isActive: true,
            version: 1
        )
        let formatted = PricingView.formatPrice(product)
        XCTAssertTrue(formatted.contains("485000") || formatted.contains("485 000"))
        XCTAssertTrue(formatted.contains("UZS"))
    }

    func testReconciliationSettlementNetMissingYieldsDash() {
        XCTAssertEqual(ReconciliationView.formatSettlementNet(netMinor: nil, currency: "UZS"), "—")
        let formatted = ReconciliationView.formatSettlementNet(netMinor: 150_000, currency: "UZS")
        XCTAssertTrue(formatted.contains("1500") || formatted.contains("1 500"))
        XCTAssertTrue(formatted.contains("UZS"))
        let zero = ReconciliationView.formatSettlementNet(netMinor: 0, currency: "UZS")
        XCTAssertTrue(zero.contains("0.00") || zero.contains("0"))
        XCTAssertTrue(zero.contains("UZS"))
    }

    func testPaymentBypassResponseDecoding() throws {
        let valid = Data("""
        {"status":"bypass_issued","bypass_token":"tok_123","order_id":"ord_1"}
        """.utf8)
        let resp = try JSONDecoder().decode(PaymentBypassResponse.self, from: valid)
        XCTAssertEqual(resp.status, "bypass_issued")
        XCTAssertEqual(resp.bypassToken, "tok_123")
        XCTAssertEqual(resp.orderId, "ord_1")

        let missingToken = Data("""
        {"status":"bypass_issued","order_id":"ord_1"}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(PaymentBypassResponse.self, from: missingToken))
    }

    @MainActor
    func testSupplierActionFailureSetsNoticeWhenHasContent() {
        struct CustomErr: LocalizedError {
            var errorDescription: String? { "payment gateway unavailable" }
        }
        var errorString: String? = nil
        SupplierStatusCenter.shared.clear()

        SupplierActionFailure.record(underlying: CustomErr(), hasContent: true, error: &errorString)
        XCTAssertNil(errorString)
        XCTAssertEqual(SupplierStatusCenter.shared.notice, "payment gateway unavailable")

        SupplierStatusCenter.shared.clear()
        SupplierActionFailure.record(underlying: CustomErr(), hasContent: false, error: &errorString)
        XCTAssertEqual(errorString, "payment gateway unavailable")
        XCTAssertNil(SupplierStatusCenter.shared.notice)
        SupplierStatusCenter.shared.clear()
    }

    func testClaimsViewClaimAmountLabelFormatsMinorUnits() {
        XCTAssertEqual(ClaimsView.claimAmountLabel(nil, currency: "UZS"), "—")
        let formatted = ClaimsView.claimAmountLabel(500_000, currency: "UZS")
        XCTAssertTrue(formatted.contains("5000") || formatted.contains("5 000"))
        XCTAssertTrue(formatted.contains("UZS"))
        let zero = ClaimsView.claimAmountLabel(0, currency: "UZS")
        XCTAssertTrue(zero.contains("0.00") || zero.contains("0"))
        XCTAssertTrue(zero.contains("UZS"))
    }

    func testClaimChargebacksViewFormatRowAmount() {
        XCTAssertEqual(ClaimChargebacksView.formatChargebackRowAmount(nil, currency: "UZS"), "—")
        let formatted = ClaimChargebacksView.formatChargebackRowAmount(150_000, currency: "UZS")
        XCTAssertTrue(formatted.contains("1500") || formatted.contains("1 500"))
        XCTAssertTrue(formatted.contains("UZS"))
        let zero = ClaimChargebacksView.formatChargebackRowAmount(0, currency: "UZS")
        XCTAssertTrue(zero.contains("0.00") || zero.contains("0"))
        XCTAssertTrue(zero.contains("UZS"))
    }

    func testTreasuryHubBatchPayoutFormatting() {
        XCTAssertEqual(PayoutsView.formatBatchPayout(status: "PENDING", netPayoutMinor: nil, currency: "UZS"), "PENDING · —")
        let formatted = PayoutsView.formatBatchPayout(status: "PAID", netPayoutMinor: 250_000, currency: "UZS")
        XCTAssertTrue(formatted.hasPrefix("PAID · "))
        XCTAssertTrue(formatted.contains("2500") || formatted.contains("2 500"))
        XCTAssertTrue(formatted.contains("UZS"))

        let emptyStatus = PayoutsView.formatBatchPayout(status: "", netPayoutMinor: 100_000, currency: "UZS")
        XCTAssertFalse(emptyStatus.contains("·"))
        XCTAssertTrue(emptyStatus.contains("1000") || emptyStatus.contains("1 000"))

        let zero = PayoutsView.formatBatchPayout(status: "SETTLED", netPayoutMinor: 0, currency: "USD")
        XCTAssertTrue(zero.hasPrefix("SETTLED · "))
        XCTAssertTrue(zero.contains("0.00") || zero.contains("0"))
        XCTAssertTrue(zero.contains("USD"))
    }

    func testCRMOrderAndLineAmountFormatting() {
        XCTAssertEqual(CRMView.formatCRMOrderAmount(nil, currency: "UZS"), "—")
        let orderFormatted = CRMView.formatCRMOrderAmount(1_200_000, currency: "UZS")
        XCTAssertTrue(orderFormatted.contains("12000") || orderFormatted.contains("12 000"))
        XCTAssertTrue(orderFormatted.contains("UZS"))

        XCTAssertEqual(CRMView.formatCRMOrderLineAmount(nil, currency: "UZS"), "—")
        let lineFormatted = CRMView.formatCRMOrderLineAmount(450_000, currency: "UZS")
        XCTAssertTrue(lineFormatted.contains("4500") || lineFormatted.contains("4 500"))
        XCTAssertTrue(lineFormatted.contains("UZS"))

        let lineZero = CRMView.formatCRMOrderLineAmount(0, currency: "USD")
        XCTAssertTrue(lineZero.contains("0.00") || lineZero.contains("0"))
        XCTAssertTrue(lineZero.contains("USD"))
    }

    func testCreditProfileSummaryFormatting() {
        let allNil = CreditProfilesView.formatCreditProfileSummary(limitMinor: nil, balanceMinor: nil, availMinor: nil)
        XCTAssertTrue(allNil.contains("—"))

        let formatted = CreditProfilesView.formatCreditProfileSummary(
            limitMinor: 10_000_000,
            balanceMinor: 2_500_000,
            availMinor: 7_500_000
        )
        XCTAssertTrue(formatted.contains("100000") || formatted.contains("100 000"))
        XCTAssertTrue(formatted.contains("25000") || formatted.contains("25 000"))
        XCTAssertTrue(formatted.contains("75000") || formatted.contains("75 000"))

        let zeroBalance = CreditProfilesView.formatCreditProfileSummary(
            limitMinor: 1_000_000,
            balanceMinor: 0,
            availMinor: 1_000_000
        )
        XCTAssertTrue(zeroBalance.contains("bal 0") || zeroBalance.contains("bal 0.00"))
    }

    func testParsePriceMinorExactDecimalParsing() {
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("29.99"), 2999)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("29,99"), 2999)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("100"), 10000)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("12.5"), 1250)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("0.05"), 5)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor(".50"), 50)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("10.006"), 1001)
        XCTAssertEqual(ProductPricingDetailView.parsePriceMinor("10.004"), 1000)
        XCTAssertNil(ProductPricingDetailView.parsePriceMinor("-15.00"))
        XCTAssertNil(ProductPricingDetailView.parsePriceMinor("abc"))
        XCTAssertNil(ProductPricingDetailView.parsePriceMinor("   "))
    }

    func testPromotionsViewFormatPromoSummary() {
        let promo = SupplierPromotion(
            promotionId: "p1",
            supplierId: "s1",
            name: "Summer Sale",
            description: nil,
            discountBps: 500,
            scopeType: "ALL_PRODUCTS",
            scopeProductId: nil,
            retailerScope: "ALL_RETAILERS",
            isActive: true,
            priority: 1
        )
        let summary = PromotionsView.formatPromoSummary(promo)
        XCTAssertEqual(summary, "5% · ALL_PRODUCTS · ALL_RETAILERS")

        let decimalPromo = SupplierPromotion(
            promotionId: "p2",
            supplierId: "s1",
            name: "Spring Sale",
            description: nil,
            discountBps: 150,
            scopeType: "CATEGORY",
            scopeProductId: nil,
            retailerScope: "VIP",
            isActive: true,
            priority: 2
        )
        let decimalSummary = PromotionsView.formatPromoSummary(decimalPromo)
        XCTAssertEqual(decimalSummary, "1.5% · CATEGORY · VIP")

        let inactivePromo = SupplierPromotion(
            promotionId: "p3",
            supplierId: "s1",
            name: "Winter Sale",
            description: nil,
            discountBps: 100,
            scopeType: "SKU",
            scopeProductId: "sku_1",
            retailerScope: "RETAILER",
            isActive: false,
            priority: 3
        )
        let inactiveSummary = PromotionsView.formatPromoSummary(inactivePromo)
        XCTAssertTrue(inactiveSummary.hasPrefix("1% · SKU · RETAILER"))
    }

    func testManifestDetailConnectionRestoredNotice() {
        let notice = ManifestDetailView.connectionRestoredNotice()
        XCTAssertEqual(notice, "Connection restored — verify manifest status before retrying.")
        XCTAssertNotEqual(notice, "mobile_supplier.chrome.connection_restored_manifest")
    }

    func testFleetOrdersViewFormatOrderTotal() {
        XCTAssertNil(FleetOrdersView.formatOrderTotal(totalMinor: nil, currency: "UZS"))

        let formatted = FleetOrdersView.formatOrderTotal(totalMinor: 450_000, currency: "UZS")
        XCTAssertNotNil(formatted)
        XCTAssertTrue(formatted?.contains("4500") == true || formatted?.contains("4 500") == true)
        XCTAssertTrue(formatted?.contains("UZS") == true)

        let zero = FleetOrdersView.formatOrderTotal(totalMinor: 0, currency: "USD")
        XCTAssertNotNil(zero)
        XCTAssertTrue(zero?.contains("0.00") == true || zero?.contains("0") == true)
        XCTAssertTrue(zero?.contains("USD") == true)

        let fallbackCurrency = FleetOrdersView.formatOrderTotal(totalMinor: 100_000, currency: nil)
        XCTAssertNotNil(fallbackCurrency)
        XCTAssertTrue(fallbackCurrency?.contains("UZS") == true)
    }
}
