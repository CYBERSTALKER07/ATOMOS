import SwiftUI

struct SupplyLanesView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var lanes: [SupplierSupplyLaneRow] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading supply lanes…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if lanes.isEmpty {
                SupplierEmptyView(title: "No lanes", message: "No active warehouse lanes. Configure nodes on topology.")
            } else {
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
        .background(SupplierTheme.background)
        .navigationTitle("Supply lanes")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            lanes = try await SupplierOperationsService.supplyLanes()
        } catch {
            self.error = error.localizedDescription
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
                Text("Lane utilization").foregroundStyle(.secondary)
                Spacer()
                Text(String(format: "%.0f%%", pct)).fontWeight(.medium).foregroundStyle(tint)
            }
            .font(.subheadline)
            ProgressView(value: min(100, max(0, pct)), total: 100)
                .tint(tint)
        }
    }
}
