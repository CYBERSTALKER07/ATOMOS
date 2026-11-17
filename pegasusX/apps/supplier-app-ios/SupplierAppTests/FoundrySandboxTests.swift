import XCTest
@testable import SupplierAppIOS

final class FoundrySandboxTests: XCTestCase {

    func testScenarioWatermarkConflictAndCommitResultSerialization() throws {
        let json = """
        {
          "scenario_id": "scen-surge-01",
          "committed_objects": 0,
          "conflicts": [
            {
              "cell_id": "cell-uz",
              "object_type": "SKU",
              "object_id": "sku-cola-1l",
              "base_version": 2,
              "live_version": 5
            }
          ]
        }
        """
        guard let data = json.data(using: .utf8) else {
            XCTFail("Failed to encode UTF-8 test payload")
            return
        }
        let result = try JSONDecoder().decode(ScenarioCommitResult.self, from: data)
        XCTAssertEqual(result.scenarioId, "scen-surge-01")
        XCTAssertEqual(result.committedObjects, 0)
        XCTAssertEqual(result.conflicts.count, 1)
        XCTAssertEqual(result.conflicts.first?.baseVersion, 2)
        XCTAssertEqual(result.conflicts.first?.liveVersion, 5)
    }

    func testRecallBlastRadiusReportDecoding() throws {
        let json = """
        {
          "cell_id": "cell-uz",
          "supplier_id": "sup-01",
          "contaminated_raw_material_lot": "LOT-SUGAR-X8",
          "affected_work_orders": ["WO-101"],
          "affected_finished_lots": ["FG-202"],
          "quarantined_sscc_pallets": ["SSCC-303"],
          "quarantined_warehouse_totes": ["TOTE-404"],
          "in_transit_driver_drop_blocks": ["DROP-505"],
          "impacted_soliq_facturas": ["FACT-606"]
        }
        """
        guard let data = json.data(using: .utf8) else {
            XCTFail("Failed to encode UTF-8 test payload")
            return
        }
        let report = try JSONDecoder().decode(RecallBlastRadiusReport.self, from: data)
        XCTAssertEqual(report.contaminatedRawMaterialLot, "LOT-SUGAR-X8")
        XCTAssertEqual(report.quarantinedSsccPallets, ["SSCC-303"])
        XCTAssertEqual(report.inTransitDriverDropBlocks, ["DROP-505"])
    }

    func testFoundryIdempotencyKeysStayStable() {
        let patch = FoundryIdempotency.scenarioPatch(
            scenarioId: "scen-1",
            supplierId: "sup-1",
            objectType: "SKU",
            objectId: "sku-1",
            key: "price_minor",
            value: "100"
        )
        XCTAssertEqual(
            patch,
            FoundryIdempotency.scenarioPatch(
                scenarioId: "scen-1",
                supplierId: "sup-1",
                objectType: "SKU",
                objectId: "sku-1",
                key: "price_minor",
                value: "100"
            )
        )
        XCTAssertNotEqual(
            patch,
            FoundryIdempotency.scenarioPatch(
                scenarioId: "scen-1",
                supplierId: "sup-1",
                objectType: "SKU",
                objectId: "sku-1",
                key: "price_minor",
                value: "200"
            )
        )
        XCTAssertEqual(FoundryIdempotency.scenarioCommit(scenarioId: "scen-1"), "foundry-scenario-commit:scen-1")
        XCTAssertEqual(FoundryIdempotency.lotRecall(supplierId: "sup-1", lot: "LOT-1"), "foundry-lot-recall:sup-1:LOT-1")
    }

    func testPayoutAndReplenishmentKeysStayStable() {
        XCTAssertEqual(SupplierIdempotencyKeys.payoutExport(batchId: "batch-1"), "supplier-payout-export:batch-1")
        XCTAssertEqual(SupplierIdempotencyKeys.payoutMarkPaid(batchId: "batch-1"), "supplier-payout-mark-paid:batch-1")
        XCTAssertEqual(
            SupplierIdempotencyKeys.payoutDispatch(batchId: "batch-1", live: true),
            "supplier-payout-dispatch:batch-1:live"
        )
        XCTAssertEqual(
            SupplierIdempotencyKeys.replenishmentTrigger(scopeId: "sup-1", day: "2026-11-10"),
            "supplier-replenishment-trigger:sup-1:2026-11-10"
        )
        var components = DateComponents()
        components.calendar = Calendar(identifier: .gregorian)
        components.timeZone = TimeZone(identifier: "UTC")
        components.year = 2026
        components.month = 11
        components.day = 10
        components.hour = 23
        let date = components.date!
        XCTAssertEqual(SupplierIdempotencyKeys.utcDay(date), "2026-11-10")
        XCTAssertEqual(
            SupplierIdempotencyKeys.claimApprove(claimId: "clm-1", settlementMode: "LEDGER_ONLY"),
            "supplier-claim-approve:clm-1:LEDGER_ONLY"
        )
        XCTAssertEqual(SupplierIdempotencyKeys.claimReject(claimId: "clm-1"), "supplier-claim-reject:clm-1")
        XCTAssertEqual(
            SupplierIdempotencyKeys.catalogCreate(scopeId: "sup-1", name: "Cola", priceMinor: 100, barcode: nil),
            SupplierIdempotencyKeys.catalogCreate(scopeId: "sup-1", name: "Cola", priceMinor: 100, barcode: nil)
        )
    }

    func testMissingPayoutModeIsNotTreatedAsHQ() throws {
        let json = Data(#"{"supplier_id":"sup-1","source":"DEFAULT"}"#.utf8)
        let policy = try JSONDecoder().decode(SupplierPayoutPolicy.self, from: json)
        XCTAssertEqual(policy.payoutMode, "")
        XCTAssertEqual(policy.source, "DEFAULT")

        let explicit = Data(#"{"supplier_id":"sup-1","payout_mode":"WAREHOUSE_LOCAL","source":"SUPPLIER_POLICY"}"#.utf8)
        let loaded = try JSONDecoder().decode(SupplierPayoutPolicy.self, from: explicit)
        XCTAssertEqual(loaded.payoutMode, "WAREHOUSE_LOCAL")

        let kept = PulseHonesty.applyObject(
            ok: false,
            incoming: nil as String?,
            previous: "WAREHOUSE_LOCAL",
            failure: "timeout"
        )
        XCTAssertEqual(kept.value, "WAREHOUSE_LOCAL")
        XCTAssertEqual(kept.error, "timeout")
    }

    func testSetupAndPayoutPolicyKeysFollowThePayload() {
        let policy = SupplierPayoutPolicyPatch(payoutMode: "HQ_SUPPLIER", feePolicyVersion: nil, reason: "audit")
        let fingerprint = SupplierIdempotencyKeys.encodedPayload(policy)
        XCTAssertFalse(fingerprint.isEmpty)
        XCTAssertEqual(fingerprint, SupplierIdempotencyKeys.encodedPayload(policy))
        let key = SupplierIdempotencyKeys.payoutPolicyPatch(scopeId: "sup-1", payloadFingerprint: fingerprint)
        XCTAssertEqual(key, SupplierIdempotencyKeys.payoutPolicyPatch(scopeId: "sup-1", payloadFingerprint: fingerprint))
        let changed = SupplierPayoutPolicyPatch(payoutMode: "HQ_SUPPLIER", feePolicyVersion: nil, reason: "other")
        XCTAssertNotEqual(
            key,
            SupplierIdempotencyKeys.payoutPolicyPatch(
                scopeId: "sup-1",
                payloadFingerprint: SupplierIdempotencyKeys.encodedPayload(changed)
            )
        )

        let setup = BusinessSetupRequest(
            taxId: "12345",
            registrationNumber: "R-1",
            headquartersAddress: "Amir Temur 1",
            city: "Tashkent",
            postalCode: "100000"
        )
        let setupFingerprint = SupplierIdempotencyKeys.encodedPayload(setup)
        let setupKey = SupplierIdempotencyKeys.businessSetup(scopeId: "sup-1", payloadFingerprint: setupFingerprint)
        XCTAssertEqual(
            setupKey,
            SupplierIdempotencyKeys.businessSetup(scopeId: "sup-1", payloadFingerprint: setupFingerprint)
        )
        XCTAssertNotEqual(setupKey, key)
    }

    func testExceptionReadAndDisableKeysNameTheAction() {
        let resolve = SupplierIdempotencyKeys.exceptionResolve(
            scopeId: "sup-1",
            kind: "credit_note_draft",
            id: "cn-1",
            creditNoteId: "cn-1"
        )
        XCTAssertEqual(
            resolve,
            SupplierIdempotencyKeys.exceptionResolve(
                scopeId: "sup-1",
                kind: "CREDIT_NOTE_DRAFT",
                id: "cn-1",
                creditNoteId: "cn-1"
            )
        )
        XCTAssertNotEqual(
            resolve,
            SupplierIdempotencyKeys.exceptionResolve(
                scopeId: "sup-1",
                kind: "CREDIT_NOTE_DRAFT",
                id: "cn-2",
                creditNoteId: "cn-2"
            )
        )

        let mark = SupplierIdempotencyKeys.notificationsMarkAll(scopeId: "sup-1", payloadFingerprint: "2|a,b")
        XCTAssertEqual(mark, SupplierIdempotencyKeys.notificationsMarkAll(scopeId: "sup-1", payloadFingerprint: "2|a,b"))
        XCTAssertNotEqual(mark, SupplierIdempotencyKeys.notificationsMarkAll(scopeId: "sup-1", payloadFingerprint: "3|a,b,c"))

        let disable = SupplierIdempotencyKeys.creditAdminDisable(
            mode: "relationship",
            supplierId: "sup-1",
            retailerId: "ret-1",
            ticketId: "T-9"
        )
        XCTAssertEqual(
            disable,
            SupplierIdempotencyKeys.creditAdminDisable(
                mode: "relationship",
                supplierId: "sup-1",
                retailerId: "ret-1",
                ticketId: "T-9"
            )
        )
        XCTAssertNotEqual(
            disable,
            SupplierIdempotencyKeys.creditAdminDisable(
                mode: "program",
                supplierId: "sup-1",
                retailerId: "",
                ticketId: "T-9"
            )
        )
    }
}
