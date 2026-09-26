import SwiftUI

struct ExceptionsView: View {
    @State private var rows: [WarehouseOpsException] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { Task { await load() } }
                    }
                } else if rows.isEmpty {
                    ContentUnavailableView("No open exceptions", systemImage: "exclamationmark.bubble")
                } else {
                    List(rows) { row in
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(row.kind.uppercased())
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            Text(row.reason.isEmpty ? (row.status.isEmpty ? "Needs review" : row.status) : row.reason)
                                .font(.headline)
                            if !row.orderId.isEmpty {
                                NavigationLink("Order \(row.orderId)") {
                                    OrderDetailView(orderId: row.orderId)
                                }
                            }
                            if !row.manifestId.isEmpty {
                                Text(L10n.format("mobile_warehouse.ui.manifest_manifestid", "\(row.manifestId)"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            if let label = row.deliveryExpectation?.targetLabel, !label.isEmpty {
                                Text(label)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("warehouse_portal.exceptions.text.exception_triage")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { Task { await load() } }
                }
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await WarehouseService.opsExceptions()
            rows = resp.exceptions
        } catch {
            self.error = error.localizedDescription
        }
    }
}
