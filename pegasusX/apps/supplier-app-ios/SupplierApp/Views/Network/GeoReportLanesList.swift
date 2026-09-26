import SwiftUI

struct GeoReportLanesList: View {
    let lanes: [SupplierSupplyLaneRow]

    private var totalCells: Int { lanes.reduce(0) { $0 + $1.h3Cells } }

    var body: some View {
        ResponsiveGridContentWrapper {
            Section {
                VStack(alignment: .leading, spacing: 4) {
                    Text("mobile_supplier.ui.estimated_h3_cells_in_service").font(.caption).foregroundStyle(.secondary)
                    Text("\(totalCells)").font(.title2.bold())
                }
            }
            Section("Lane utilization") {
                ForEach(lanes) { lane in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(lane.name.isEmpty ? lane.warehouseId : lane.name).font(.body)
                        Text(String(format: "%d cells · %.0f%% utilization today", lane.h3Cells, lane.utilizationPct))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            Section {
                Text("supplier_portal.residual.text.h3_perimeter_coverage_and_lane_utilization_from_live_supplier_or")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
