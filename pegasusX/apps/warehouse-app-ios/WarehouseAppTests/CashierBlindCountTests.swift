import XCTest
@testable import WarehouseAppIOS

@MainActor
final class CashierBlindCountTests: XCTestCase {

    func testCleanEmptyDefaultsWithoutDemoPresets() {
        let vm = CashierViewModel(
            initialCellId: "",
            initialDepotId: "",
            initialSupplierId: ""
        )

        XCTAssertEqual(vm.depotId, "")
        XCTAssertEqual(vm.supplierId, "")
        XCTAssertEqual(vm.driverId, "")
        XCTAssertEqual(vm.routeId, "")
        XCTAssertEqual(vm.bagSealBarcode, "")
        XCTAssertEqual(vm.rrnOrderId, "")
        XCTAssertEqual(vm.rrnGateway, "")
        XCTAssertEqual(vm.rrnDisputedText, "")
        XCTAssertEqual(vm.citBadgeId, "")
        XCTAssertEqual(vm.citSealBarcode, "")
        XCTAssertEqual(vm.citHandoverText, "")
        XCTAssertEqual(vm.amlTin, "")
        XCTAssertEqual(vm.amlExposureText, "")
    }

    func testDenominationMinorArithmeticAndBlindMasking() {
        let vm = CashierViewModel(
            initialCellId: "cell-test",
            initialDepotId: "depot-01",
            initialSupplierId: "sup-01"
        )
        XCTAssertTrue(vm.isBlindCountMasked)
        XCTAssertEqual(vm.countedCashMinor, 0)
        XCTAssertEqual(vm.countedCashSom, 0)
        XCTAssertEqual(vm.amlCeilingMinor, 0)

        // 5 notes of 200,000 so'm + 3 notes of 100,000 so'm = 1,300,000 so'm (130,000,000 minor)
        vm.setDenominationCount(noteSom: 200000, count: 5)
        vm.setDenominationCount(noteSom: 100000, count: 3)
        XCTAssertEqual(vm.countedCashSom, 1_300_000)
        XCTAssertEqual(vm.countedCashMinor, 130_000_000)

        let row = DenominationRowItem(noteSom: 100000, count: 3)
        XCTAssertEqual(row.totalSom, 300_000)
        XCTAssertEqual(row.totalMinor, 30_000_000)

        // Negative counts clamp to zero
        vm.setDenominationCount(noteSom: 50000, count: -4)
        XCTAssertEqual(vm.denominations[50000], 0)
    }

    func testCashierSessionViewMasksExpectedCashUntilBalanced() throws {
        let json = """
        {
          "session_id": "sess-01",
          "supplier_id": "sup-01",
          "depot_id": "depot-01",
          "driver_id": "drv-01",
          "route_id": "route-01",
          "bag_seal_barcode": "BAG-01",
          "attempt_number": 1,
          "counted_cash_minor": 0,
          "status": "BLIND_COUNT_PENDING",
          "created_at": "2026-10-30T12:00:00Z"
        }
        """
        guard let data = json.data(using: .utf8) else {
            XCTFail("Failed to encode UTF-8 test payload")
            return
        }
        let session = try JSONDecoder().decode(CashierSessionView.self, from: data)
        XCTAssertNil(session.expectedCashMinor)
        XCTAssertNil(session.varianceMinor)
        XCTAssertEqual(session.status, "BLIND_COUNT_PENDING")
    }

    func testStrapBundlesAndTotalBanknoteCount() {
        let vm = CashierViewModel(initialCellId: "", initialDepotId: "", initialSupplierId: "")
        XCTAssertEqual(vm.totalBanknotesCount, 0)

        // 10-strap bundle of 200k + 50-strap bundle of 100k = 7,000,000 so'm (700,000,000 minor)
        vm.addStrap(noteSom: 200000, strapCount: 10)
        vm.addStrap(noteSom: 100000, strapCount: 50)

        XCTAssertEqual(vm.totalBanknotesCount, 60)
        XCTAssertEqual(vm.countedCashSom, 7_000_000)
        XCTAssertEqual(vm.countedCashMinor, 700_000_000)
    }

    func testWholeSomFormattingUsesSpaceGroupingAndNeverShowsTiyin() {
        XCTAssertEqual(CashierViewModel.formatSom(amount: 1425000), "1 425 000 so'm")
        XCTAssertEqual(CashierViewModel.formatWholeSom(minor: 142_500_099), "1 425 000 so'm")
        XCTAssertEqual(CashierViewModel.formatWholeSom(minor: 0), "0 so'm")
        XCTAssertEqual(CashierViewModel.formatWholeSom(minor: -50), "0 so'm")
        XCTAssertFalse(CashierViewModel.formatWholeSom(minor: 142_500_000).localizedCaseInsensitiveContains("tiyin"))
        XCTAssertFalse(CashierViewModel.formatWholeSom(minor: 142_500_000).localizedCaseInsensitiveContains("uzs"))
    }

    func testSelectorLoadingAutoLinkingAndManualFallback() async {
        let vm = CashierViewModel(
            initialCellId: "cell-uz",
            initialDepotId: "depot-01",
            initialSupplierId: "sup-01",
            loadDriversHandler: {
                [CashierDriverOption(driverId: "drv-live-1", label: "Sardor Karimov (drv-live-1)")]
            },
            loadManifestsHandler: {
                [CashierRouteOption(routeId: "mnf-77", label: "mnf-77 · Sardor Karimov (ISUZU 01A)", driverNameHint: "Sardor Karimov")]
            },
            loadFleetLiveMapHandler: { _ in
                (
                    [CashierDriverOption(driverId: "drv-live-2", label: "Dilshod Alimov (drv-live-2)")],
                    [CashierRouteOption(routeId: "route-live-2", label: "route-live-2 · Dilshod Alimov [DISPATCHED]", linkedDriverId: "drv-live-2")]
                )
            }
        )

        await vm.loadSelectors()
        XCTAssertTrue(vm.selectorsLoaded)
        XCTAssertFalse(vm.manualEntryMode)
        XCTAssertEqual(vm.driverOptions.map(\.driverId), ["drv-live-1", "drv-live-2"])
        XCTAssertEqual(vm.routeOptions.map(\.routeId), ["route-live-2", "mnf-77"])

        vm.selectRoute("mnf-77")
        XCTAssertEqual(vm.routeId, "mnf-77")
        XCTAssertEqual(vm.driverId, "drv-live-1")

        let offlineVm = CashierViewModel(
            initialCellId: "",
            initialDepotId: "depot-01",
            initialSupplierId: "sup-01",
            loadDriversHandler: { [] },
            loadManifestsHandler: { [] },
            loadFleetLiveMapHandler: { _ in ([], []) }
        )
        await offlineVm.loadSelectors()
        XCTAssertTrue(offlineVm.manualEntryMode)
    }

    func testValidationRejectsMissingSessionOrHandoverFieldsAndFormatsAMLInSom() async {
        var sessionRequests = 0
        let vm = CashierViewModel(
            initialCellId: "cell-01",
            initialDepotId: "",
            initialSupplierId: "",
            startSessionHandler: { _ in
                sessionRequests += 1
                throw APIError.httpError(500)
            },
            checkAMLHandler: { req in
                AMLCheckResponse(legalEntityTin: req.legalEntityTin, accumulatedCashMinor: 500_000_000, ceilingMinor: 1_000_000_000)
            }
        )

        await vm.startSession()
        XCTAssertEqual(sessionRequests, 0)
        XCTAssertNotNil(vm.errorMessage)

        vm.supplierId = "sup-01"
        let ok = await vm.checkAMLExposure(tin: "305678901", cashMinor: 500_000_000)
        XCTAssertTrue(ok)
        XCTAssertEqual(vm.statusMessage, "AML Exposure OK: 5 000 000 so'm / 10 000 000 so'm")
        XCTAssertFalse(vm.statusMessage?.contains("UZS") ?? true)
        XCTAssertFalse(vm.statusMessage?.localizedCaseInsensitiveContains("tiyin") ?? true)
    }
}
