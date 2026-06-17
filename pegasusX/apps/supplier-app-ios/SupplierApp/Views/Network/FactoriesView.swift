import SwiftUI

struct FactoriesView: View {
    @State private var factories: [SupplierTopologyFactory] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading factories…")
            } else if let error, factories.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if factories.isEmpty {
                SupplierEmptyView(title: "No factories", message: "Configure factories in topology.")
            } else {
                List(factories) { factory in
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        Text(factory.name).font(.headline)
                        Text(String(format: "%.4f, %.4f", factory.lat, factory.lng))
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                        SupplierStatusBadge(text: factory.isActive ? "ACTIVE" : "INACTIVE")
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Factories")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let topology = try await SupplierOperationsService.topology()
            factories = topology.factories
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
