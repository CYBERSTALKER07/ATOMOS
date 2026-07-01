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
                List {
                    Section("Nodes (\(graph.nodes.count))") {
                        ForEach(graph.nodes) { node in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(node.name?.isEmpty == false ? node.name! : node.id)
                                    .font(.headline)
                                Text("\(node.type) · \(node.id)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    Section("Edges (\(graph.edges.count))") {
                        ForEach(graph.edges) { edge in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(edge.relation)
                                    .font(.headline)
                                Text("\(edge.from) → \(edge.to)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Knowledge graph")
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
