import SwiftUI

struct GeoReportLanesList: View {
    let lanes: [SupplierSupplyLaneRow]

    private var totalCells: Int { lanes.reduce(0) { $0 + $1.h3Cells } }

    var body: some View {
        ResponsiveGridContentWrapper {
            Section {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Estimated H3 cells in service").font(.caption).foregroundStyle(.secondary)
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
                Text("H3 perimeter coverage and lane utilization from live supplier orders.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
