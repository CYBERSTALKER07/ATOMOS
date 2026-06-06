import SwiftUI

/// Portal parity hub for secondary warehouse admin routes (pegasus warehouse-portal nav).
struct MoreHubView: View {
    var body: some View {
        NavigationStack {
            List {
                Section("Fulfillment") {
                    NavigationLink { ManifestsView() } label: {
                        Label("Manifests", systemImage: "doc.text")
                    }
                    NavigationLink { DispatchView() } label: {
                        Label("Dispatch", systemImage: "paperplane")
                    }
                    NavigationLink { TransferActionsView() } label: {
                        Label("Transfer actions", systemImage: "arrow.left.arrow.right")
                    }
                }
                Section("Inventory") {
                    NavigationLink { ProductsView() } label: {
                        Label("Products", systemImage: "square.grid.2x2")
                    }
                    NavigationLink { SupplyRequestsHubView() } label: {
                        Label("Supply Requests", systemImage: "arrow.triangle.2.circlepath")
                    }
                    NavigationLink { DemandForecastView() } label: {
                        Label("Demand Forecast", systemImage: "chart.line.uptrend.xyaxis")
                    }
                }
                Section("Operations") {
                    NavigationLink { CRMView() } label: {
                        Label("Retailers", systemImage: "person.crop.rectangle")
                    }
                    NavigationLink { ReturnsView() } label: {
                        Label("Returns", systemImage: "arrow.uturn.backward")
                    }
                }
            }
            .navigationTitle("More")
        }
    }
}
