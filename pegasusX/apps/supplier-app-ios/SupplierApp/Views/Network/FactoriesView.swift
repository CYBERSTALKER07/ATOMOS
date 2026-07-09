import SwiftUI

struct FactoriesView: View {
    @State private var factories: [SupplierTopologyFactory] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showAdd = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading factories…")
            } else if let error, factories.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if factories.isEmpty {
                TopologyCenteredEmptyState(
                    title: "No factories",
                    message: "Add a production node linked to your warehouse network.",
                    actionLabel: "Add first factory"
                ) {
                    showAdd = true
                }
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(factories) { factory in
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        Text(factory.name).font(.headline)
                        Text(factory.address.isEmpty ? "Coordinates on file" : factory.address)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                        SupplierStatusBadge(text: factory.isActive ? "ACTIVE" : "INACTIVE")
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Factories")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showAdd = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showAdd) {
            AddFactorySheet {
                Task { await load(silent: true) }
            }
        }
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
