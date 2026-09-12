import Foundation
import Testing
@testable import retailerapp

struct DashboardCommandTests {
    @Test func controlTowerPulseDecodesFiscalFailedSupplierFacet() throws {
        let raw = Data("""
        {
          "source": "spanner",
          "empty": false,
          "open_orders": 1,
          "active_fulfillments": 0,
          "dock_pending": 0,
          "pos_open_sessions": 0,
          "open_shifts": 0,
          "open_assist_tickets": 0,
          "low_stock_sku_bins": 0,
          "shift_variances_7d": 0,
          "sales_minor_7d": 0,
          "loyalty": { "enrolled": false },
          "orders_by_status": { "FISCAL_FAILED": 1, "COMPLETED": 1 },
          "orders_by_supplier": [
            { "supplier_id": "sup-a", "orders_by_status": { "FISCAL_FAILED": 1 } },
            { "supplier_id": "sup-b", "orders_by_status": { "COMPLETED": 1 } }
          ]
        }
        """.utf8)
        let pulse = try JSONDecoder().decode(ControlTowerPulseWire.self, from: raw)
        #expect(pulse.source == "spanner")
        #expect(pulse.empty == false)
        #expect(pulse.ordersByStatus["FISCAL_FAILED"] == 1)
        #expect(pulse.ordersBySupplier.first { $0.supplierId == "sup-a" }?.ordersByStatus["FISCAL_FAILED"] == 1)
        #expect(pulse.ordersBySupplier.first { $0.supplierId == "sup-b" }?.ordersByStatus["COMPLETED"] == 1)
        #expect(pulse.loyalty?.enrolled == false)
        #expect(orderStatusFunnel.contains("FISCAL_FAILED"))
        #expect(orderStatusFunnel.count == 17)
        #expect(statusStackModel(dictionary: orderStatusFunnel, counts: pulse.ordersByStatus).rows.count == 17)
    }

    @Test func commandChipMatchesCanonicalStatusAndSupplier() {
        #expect(retailerOrderMatchesCommand(statusRaw: "FISCAL_FAILED", supplierId: "sup-a", commandStatus: "FISCAL_FAILED", commandSupplierId: nil))
        #expect(retailerOrderMatchesCommand(statusRaw: "DISPATCHED", supplierId: "sup-a", commandStatus: "LOADED", commandSupplierId: "sup-a"))
        #expect(!retailerOrderMatchesCommand(statusRaw: "FISCAL_FAILED", supplierId: "sup-a", commandStatus: "FISCAL_FAILED", commandSupplierId: "sup-b"))
    }

    @Test func pulseHonestyKeepsPreviousOnFailure() {
        let previous = ["old"]
        let failed = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: previous)
        #expect(failed.events == previous)
        #expect(failed.error == PulseHonesty.failed)
        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: previous)
        #expect(empty.events.isEmpty)
        #expect(empty.error == nil)
    }

    @Test func commandPulseHonestyKeepsPreviousOnFailure() {
        let previous = "last"
        let failed = PulseHonesty.applyObject(ok: false, incoming: nil as String?, previous: previous)
        #expect(failed.value == previous)
        #expect(failed.error == PulseHonesty.commandFailed)
        let empty = PulseHonesty.applyObject(ok: true, incoming: "empty", previous: previous)
        #expect(empty.value == "empty")
        #expect(empty.error == nil)
    }
}
