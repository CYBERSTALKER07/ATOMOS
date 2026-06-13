import SwiftUI

struct GeoReportView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var lanes: [SupplierSupplyLaneRow] = []

    private var totalCells: Int { lanes.reduce(0) { $0 + $1.h3Cells } }

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading geo report…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if lanes.isEmpty {
                SupplierEmptyView(title: "No coverage", message: "No active lanes to report on.")
            } else {
                List {
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
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Geo report")
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
