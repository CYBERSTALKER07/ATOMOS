import XCTest
@testable import SupplierAppIOS

final class DashboardTests: XCTestCase {
    func testMissingReceiptStaysQuiet() {
        XCTAssertTrue(ReceiptOpen.shouldStayQuiet(APIError.httpError(404)))
        XCTAssertTrue(ReceiptOpen.shouldStayQuiet(APIError.server("receipt_not_found")))
        XCTAssertFalse(ReceiptOpen.shouldStayQuiet(APIError.httpError(500)))
        XCTAssertFalse(ReceiptOpen.shouldStayQuiet(APIError.server("timeout")))
    }

    func testDashboardOmitsRevenueInsteadOfInventingZero() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertNil(dash.todayRevenueMinor)
        XCTAssertNil(dash.pendingOrders)
        XCTAssertNil(dash.inventorySKUs)
        XCTAssertNil(dash.deliveriesCompletedToday)
        XCTAssertNil(dash.deliveriesAttemptedToday)
    }

    func testMissingShownMoneyStaysNil() throws {
        let mismatch = try JSONDecoder().decode(ReconciliationMismatchRow.self, from: Data("""
        {"gateway":"payme","currency":"UZS"}
        """.utf8))
        XCTAssertNil(mismatch.netAmountMinor)
        XCTAssertNil(mismatch.creditAmountMinorTotal)
        XCTAssertNil(mismatch.debitAmountMinorTotal)

        let order = try JSONDecoder().decode(SupplierCRMOrder.self, from: Data("""
        {"order_id":"ord_1","state":"DELIVERED"}
        """.utf8))
        XCTAssertNil(order.amount)
    }

    func testEmptyPhotoIsNotAnImage() throws {
        XCTAssertThrowsError(try PhotoLoad.imageData(nil))
        XCTAssertThrowsError(try PhotoLoad.imageData(Data()))
        XCTAssertEqual(try PhotoLoad.imageData(Data([1])), Data([1]))
    }

    func testMalformedComplianceListDoesNotBecomeEmpty() {
        let json = Data("""
        {"summary":{"open_fiscal_count":1},"open_fiscal":[{}]}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(ComplianceDashboardResponse.self, from: json))
    }

    func testMissingComplianceListStaysEmpty() throws {
        let dash = try JSONDecoder().decode(ComplianceDashboardResponse.self, from: Data("""
        {"summary":{"open_fiscal_count":0}}
        """.utf8))
        XCTAssertEqual(dash.summary.openFiscalCount, 0)
        XCTAssertTrue(dash.openFiscal.isEmpty)
        XCTAssertTrue(dash.forceCompletes.isEmpty)
    }

    func testMalformedFlatComplianceSummaryDoesNotBecomeZero() {
        let malformedFlat = Data("""
        {"open_fiscal_count":"invalid"}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(ComplianceDashboardResponse.self, from: malformedFlat))
    }

    func testValidFlatComplianceSummaryParsesCorrectly() throws {
        let flat = try JSONDecoder().decode(ComplianceDashboardResponse.self, from: Data("""
        {"open_fiscal_count":4,"force_complete_count":2}
        """.utf8))
        XCTAssertEqual(flat.summary.openFiscalCount, 4)
        XCTAssertEqual(flat.summary.forceCompleteCount, 2)
        XCTAssertEqual(flat.summary.claimMismatchCount, 0)
        XCTAssertTrue(flat.openFiscal.isEmpty)
    }

    func testMalformedDispatchListDoesNotBecomeEmpty() {
        let json = Data("""
        {"undispatched_order_count":2,"undispatched_orders":[{}]}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(SupplierDispatchPreview.self, from: json))
    }

    func testDispatchCountWithoutListStaysCount() throws {
        let preview = try JSONDecoder().decode(SupplierDispatchPreview.self, from: Data("""
        {"undispatched_order_count":2}
        """.utf8))
        XCTAssertEqual(preview.undispatchedOrderCount, 2)
        XCTAssertTrue(preview.undispatchedOrders.isEmpty)
    }

    func testMalformedCreditProfilesDoNotBecomeEmpty() {
        let json = Data("""
        {"count":1,"profiles":[{}]}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(CreditProfilesResponse.self, from: json))
    }

    func testMalformedCreditProfileScoreDoesNotBecomeZero() {
        let json = Data("""
        {"profiles":[{"retailer_id":"ret_1","risk_score":"high","status":"ACTIVE","risk_tier":"LOW"}]}
        """.utf8)
        XCTAssertThrowsError(try JSONDecoder().decode(CreditProfilesResponse.self, from: json))

        let payloads = [
            #"{"retailer_id":"ret_1","status":1}"#,
            #"{"retailer_id":"ret_1","risk_tier":2}"#,
            #"{"retailer_id":"ret_1","needs_attention":"yes"}"#,
            #"{"retailer_id":"ret_1","delinquency_count":"no"}"#,
            #"{"retailer_id":"ret_1","utilization_bps":"wide"}"#,
            #"{"retailer_id":"ret_1","supplier_id":1}"#,
        ]
        for payload in payloads {
            XCTAssertThrowsError(try JSONDecoder().decode(CreditProfileRow.self, from: Data(payload.utf8)))
        }
    }

    func testMissingCreditProfileScoreStaysNilAndZeroStaysZero() throws {
        let missing = try JSONDecoder().decode(CreditProfileRow.self, from: Data("""
        {"retailer_id":"ret_1","status":"ACTIVE"}
        """.utf8))
        XCTAssertNil(missing.riskScore)
        XCTAssertEqual(missing.delinquencyCount, 0)
        XCTAssertEqual(missing.status, "ACTIVE")
        XCTAssertEqual(missing.riskTier, "")
        XCTAssertFalse(missing.needsAttention)
        XCTAssertNil(missing.utilizationBps)

        let zero = try JSONDecoder().decode(CreditProfileRow.self, from: Data("""
        {"retailer_id":"ret_1","risk_score":0,"delinquency_count":0,"needs_attention":false,"status":"FROZEN","risk_tier":"LOW","utilization_bps":0}
        """.utf8))
        XCTAssertEqual(zero.riskScore, 0)
        XCTAssertEqual(zero.delinquencyCount, 0)
        XCTAssertEqual(zero.status, "FROZEN")
        XCTAssertEqual(zero.riskTier, "LOW")
        XCTAssertFalse(zero.needsAttention)
        XCTAssertEqual(zero.utilizationBps, 0)
    }

    func testMalformedCreditNotesDoNotBecomeEmpty() {
        let malformedLists = [
            #"{"credit_notes":[{}]}"#,
            #"{"credit_notes":"none"}"#,
            #"{"credit_notes":[{"credit_note_id":"cn_1","order_id":"ord_1","status":123}]}"#,
        ]
        for payload in malformedLists {
            XCTAssertThrowsError(try JSONDecoder().decode(CreditNotesResponse.self, from: Data(payload.utf8)))
        }

        let malformedRows = [
            #"{"credit_note_id":"cn_1","order_id":"ord_1","status":123}"#,
            #"{"credit_note_id":"cn_1","order_id":"ord_1","status":["DRAFT"]}"#,
            #"{"credit_note_id":"cn_1","order_id":"ord_1","total_gross_minor":"invalid"}"#,
        ]
        for payload in malformedRows {
            XCTAssertThrowsError(try JSONDecoder().decode(CreditNoteRow.self, from: Data(payload.utf8)))
        }
    }

    func testMissingCreditNotesIsEmptyAndNullStaysEmpty() throws {
        let absent = try JSONDecoder().decode(CreditNotesResponse.self, from: Data("{}".utf8))
        XCTAssertTrue(absent.creditNotes.isEmpty)

        let nullList = try JSONDecoder().decode(CreditNotesResponse.self, from: Data(#"{"credit_notes":null}"#.utf8))
        XCTAssertTrue(nullList.creditNotes.isEmpty)

        let rowMissing = try JSONDecoder().decode(CreditNoteRow.self, from: Data(#"{"credit_note_id":"cn_1","order_id":"ord_1"}"#.utf8))
        XCTAssertEqual(rowMissing.creditNoteId, "cn_1")
        XCTAssertEqual(rowMissing.orderId, "ord_1")
        XCTAssertEqual(rowMissing.status, "")
        XCTAssertNil(rowMissing.totalGrossMinor)

        let rowZero = try JSONDecoder().decode(CreditNoteRow.self, from: Data(#"{"credit_note_id":"cn_1","order_id":"ord_1","status":"DRAFT","total_gross_minor":0}"#.utf8))
        XCTAssertEqual(rowZero.status, "DRAFT")
        XCTAssertEqual(rowZero.totalGrossMinor, 0)
    }

    func testDashboardParsesOrdersByStatus() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z",
          "orders_by_status": { "PENDING": 3, "COMPLETED": 1 }
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertEqual(dash.ordersByStatus["PENDING"], 3)
        XCTAssertEqual(dash.ordersByStatus["COMPLETED"], 1)
        XCTAssertEqual(orderStatusFunnel.count, 17)
    }

    func testDashboardMissingOrdersByStatusIsEmptyMap() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z"
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertTrue(dash.ordersByStatus.isEmpty)
        XCTAssertTrue(dash.manifestsByState.isEmpty)
    }

    func testMalformedDashboardStatusMapDoesNotBecomeEmpty() {
        let payloads = [
            #"{"supplier_id":"sup_123","is_configured":true,"orders_by_status":{"PENDING":"3"}}"#,
            #"{"supplier_id":"sup_123","is_configured":true,"orders_by_status":"live"}"#,
            #"{"supplier_id":"sup_123","is_configured":true,"manifests_by_state":[]}"#,
            #"{"supplier_id":"sup_123","is_configured":true,"manifests_by_state":{"DRAFT":"1"}}"#,
            #"{"supplier_id":"sup_123","is_configured":true,"updated_at":12345}"#,
            #"{"supplier_id":"sup_123","is_configured":true,"updated_at":["2026-06-30"]}"#,
        ]
        for payload in payloads {
            XCTAssertThrowsError(try JSONDecoder().decode(SupplierDashboard.self, from: Data(payload.utf8)))
        }
    }

    func testNullDashboardStatusMapsStayEmptyAndZeroStaysZero() throws {
        let absent = try JSONDecoder().decode(SupplierDashboard.self, from: Data("""
        {"supplier_id":"sup_123","is_configured":true,"orders_by_status":null,"manifests_by_state":null,"updated_at":null}
        """.utf8))
        XCTAssertTrue(absent.ordersByStatus.isEmpty)
        XCTAssertTrue(absent.manifestsByState.isEmpty)
        XCTAssertEqual(absent.updatedAt, "")

        let omitted = try JSONDecoder().decode(SupplierDashboard.self, from: Data("""
        {"supplier_id":"sup_123","is_configured":true}
        """.utf8))
        XCTAssertEqual(omitted.updatedAt, "")

        let zero = try JSONDecoder().decode(SupplierDashboard.self, from: Data("""
        {"supplier_id":"sup_123","is_configured":true,"orders_by_status":{"PENDING":0},"manifests_by_state":{"DRAFT":0}}
        """.utf8))
        XCTAssertEqual(zero.ordersByStatus["PENDING"], 0)
        XCTAssertEqual(zero.manifestsByState["DRAFT"], 0)
    }

    func testStatusStackModesStayDistinct() {
        let empty = statusStackModel(counts: nil)
        XCTAssertEqual(empty.mode, .empty)
        XCTAssertTrue(empty.rows.isEmpty)

        let zero = statusStackModel(counts: [:])
        XCTAssertEqual(zero.mode, .zero)
        XCTAssertEqual(zero.rows.count, 17)
        XCTAssertTrue(zero.rows.allSatisfy { $0.count == 0 })

        let unavailable = statusStackModel(counts: ["PENDING": 9], available: false)
        XCTAssertEqual(unavailable.mode, .unavailable)
        XCTAssertEqual(unavailable.rows.count, 17)
        XCTAssertTrue(unavailable.rows.allSatisfy { $0.count == nil })

        let live = statusStackModel(counts: ["PENDING": 2, "COMPLETED": 2])
        XCTAssertEqual(live.mode, .live)
        XCTAssertEqual(live.total, 4)
        XCTAssertEqual(live.rows.count, 17)
        XCTAssertEqual(live.rows.first { $0.key == "PENDING" }?.share, 0.5)
        XCTAssertEqual(live.rows.first { $0.key == "LOADED" }?.count, 0)
    }

    func testFiscalFailedIncrementsChip() {
        let next = incrementOrderStatusCount([:], status: "FISCAL_FAILED")
        XCTAssertEqual(next["FISCAL_FAILED"], 1)
        XCTAssertEqual(next.count, 17)
        XCTAssertEqual(incrementOrderStatusCount(next, status: "DISPATCHED")["LOADED"], 1)
    }

    func testDashboardParsesRevenueWithoutInventingCurrency() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z",
          "orders_by_status": { "FISCAL_FAILED": 1 },
          "today_revenue_minor": 1500
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertEqual(dash.todayRevenueMinor, Int64(1500))
        XCTAssertEqual(dash.ordersByStatus["FISCAL_FAILED"], 1)
        XCTAssertEqual(formatPackMoney(dash.todayRevenueMinor ?? -1, pack: nil), "15")
        XCTAssertFalse(formatPackMoney(dash.todayRevenueMinor ?? -1, pack: nil).contains("UZS"))
    }

    func testCommandChipUsesFunnelStatusNotCoarseTab() {
        let fiscal = resolveSupplierOrdersQuery(commandStatus: "FISCAL_FAILED", coarseFilter: "ACTIVE")
        XCTAssertEqual(fiscal.status, "FISCAL_FAILED")
        XCTAssertNil(fiscal.filter)
        let dispatched = resolveSupplierOrdersQuery(commandStatus: "DISPATCHED", coarseFilter: "ACTIVE")
        XCTAssertEqual(dispatched.status, "LOADED")
        let coarse = resolveSupplierOrdersQuery(commandStatus: nil, coarseFilter: "ACTIVE")
        XCTAssertEqual(coarse.filter, "ACTIVE")
        XCTAssertNil(coarse.status)
    }

    func testFailedMapOverlayKeepsTheLastCells() {
        let kept = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: ["cell-1"], failure: "timeout")
        XCTAssertEqual(kept.events, ["cell-1"])
        XCTAssertEqual(kept.error, "timeout")

        let cleared = PulseHonesty.apply(ok: true, incoming: [String](), previous: ["cell-1"], failure: "timeout")
        XCTAssertTrue(cleared.events.isEmpty)
        XCTAssertNil(cleared.error)
    }

    func testOptionalPanelKeepsTheLastValueWhenRefreshFails() {
        let kept = PulseHonesty.applyObject(ok: false, incoming: nil as String?, previous: "last", failure: "timeout")
        XCTAssertEqual(kept.value, "last")
        XCTAssertEqual(kept.error, "timeout")

        let first = PulseHonesty.applyObject(ok: false, incoming: nil as String?, previous: nil, failure: "timeout")
        XCTAssertNil(first.value)
        XCTAssertEqual(first.error, "timeout")

        let fresh = PulseHonesty.applyObject(ok: true, incoming: "next", previous: "last", failure: "timeout")
        XCTAssertEqual(fresh.value, "next")
        XCTAssertNil(fresh.error)
    }

    func testFailedCRMDetailKeepsTheLastRetailer() {
        struct Stub: Error {}
        let kept = CRMDetailLoad.selection(previous: "shop-1", loaded: .failure(Stub()))
        XCTAssertEqual(kept.value, "shop-1")
        XCTAssertNotNil(kept.failure)

        let opened = CRMDetailLoad.selection(previous: nil as String?, loaded: .failure(Stub()))
        XCTAssertNil(opened.value)
        XCTAssertNotNil(opened.failure)

        let fresh = CRMDetailLoad.selection(previous: "shop-1", loaded: Result<String, Stub>.success("shop-2"))
        XCTAssertEqual(fresh.value, "shop-2")
        XCTAssertNil(fresh.failure)
    }

    func testPulseHonestyKeepsPreviousOnFailure() {
        let previous = ["old"]
        let failed = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: previous)
        XCTAssertEqual(failed.events, previous)
        XCTAssertEqual(failed.error, PulseHonesty.failed)
        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: previous)
        XCTAssertEqual(empty.events, [])
        XCTAssertNil(empty.error)
    }
}
