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
}
