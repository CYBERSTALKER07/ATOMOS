import SwiftUI

struct KnowledgeGraphView: View {
    @State private var graph: SupplierKnowledgeGraph?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading knowledge graph…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let graph {
                ResponsiveGridContentWrapper {
                    Section(L10n.format("mobile_supplier.ui.nodes_count", "\(graph.nodes.count)")) {
                        ForEach(graph.nodes) { node in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(node.name?.isEmpty == false ? node.name! : node.id)
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.ui.type_id", "\(node.type)", "\(node.id)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    Section(L10n.format("mobile_supplier.ui.edges_count", "\(graph.edges.count)")) {
                        ForEach(graph.edges) { edge in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(edge.relation)
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.ui.from_to", "\(edge.from)", "\(edge.to)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("mobile_supplier.ui.knowledge_graph")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            graph = try await SupplierOperationsService.knowledgeGraph()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
