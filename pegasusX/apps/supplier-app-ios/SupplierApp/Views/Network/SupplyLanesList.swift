import SwiftUI

struct SupplyLanesList: View {
    let lanes: [SupplierSupplyLaneRow]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(lanes) { lane in
                Section(lane.name.isEmpty ? lane.warehouseId : lane.name) {
                    LaneMetricRow(label: "H3 coverage estimate", value: "\(lane.h3Cells) cells")
                    LaneMetricRow(label: "Active drivers", value: "\(lane.drivers)")
                    LaneMetricRow(label: "Orders today", value: "\(lane.ordersToday)")
                    LaneMetricRow(label: "Capacity limit", value: "\(lane.capacity)")
                    LaneUtilizationRow(pct: lane.utilizationPct)
                }
            }
        }
    }
}

private struct LaneMetricRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(label).foregroundStyle(.secondary)
            Spacer()
            Text(value).fontWeight(.medium)
        }
        .font(.subheadline)
    }
}

private struct LaneUtilizationRow: View {
    let pct: Double

    private var tint: Color {
        if pct > 85 { return SupplierTheme.destructive }
        if pct > 75 { return SupplierTheme.warning }
        return .accentColor
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("mobile_supplier.ui.lane_utilization").foregroundStyle(.secondary)
                Spacer()
                Text(String(format: "%.0f%%", pct)).fontWeight(.medium).foregroundStyle(tint)
            }
            .font(.subheadline)
            ProgressView(value: min(100, max(0, pct)), total: 100)
                .tint(tint)
        }
    }
}
