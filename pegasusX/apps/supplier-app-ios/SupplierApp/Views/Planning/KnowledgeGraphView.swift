import SwiftUI

struct KnowledgeGraphView: View {
    @State private var viewModel = KnowledgeGraphViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.graph == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.knowledge_graph")
            } else if let error = viewModel.error, viewModel.graph == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if let graph = viewModel.graph {
                ResponsiveGridContentWrapper {
                    Section(L10n.format("mobile_supplier.ui.nodes_count", "\(graph.nodes.count)")) {
                        ForEach(graph.nodes) { node in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text((node.name?.isEmpty == false ? node.name : nil) ?? node.id)
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.ui.type_id", "\(node.type)", "\(node.id)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\((node.name?.isEmpty == false ? node.name : nil) ?? node.id), type \(node.type)")
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
                            .accessibilityElement(children: .combine)
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("mobile_supplier.ui.knowledge_graph")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}
