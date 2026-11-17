import SwiftUI

struct SupplyLanesList: View {
    let lanes: [SupplierSupplyLaneRow]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(lanes) { lane in
                Section(lane.name.isEmpty ? lane.warehouseId : lane.name) {
                    LaneMetricRow(label: "mobile_supplier.chrome.h3_coverage_estimate", value: L10n.format("mobile_supplier.chrome.cells_value", "\(lane.h3Cells)"))
                    LaneMetricRow(label: "mobile_supplier.chrome.active_drivers", value: "\(lane.drivers)")
                    LaneMetricRow(label: "mobile_supplier.chrome.orders_today", value: "\(lane.ordersToday)")
                    LaneMetricRow(label: "mobile_supplier.chrome.capacity_limit", value: "\(lane.capacity)")
                    LaneUtilizationRow(pct: lane.utilizationPct)
                }
            }
        }
    }
}
