import SwiftUI

struct WarehouseScoredExceptionsView: View {
    @State private var vm = WarehouseScoredExceptionsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.rows.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let errorText = vm.errorText, vm.rows.isEmpty {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(errorText)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.rows.isEmpty {
                Text("No open scored exceptions")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(vm.rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.type.isEmpty ? "—" : row.type)
                            .font(.headline)
                        Text("score \(row.score) · \(row.severity) · \(row.ageMinutes)m")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.orderId.isEmpty {
                            Text(row.orderId)
                                .font(.caption)
                                .foregroundStyle(.tint)
                        }
                        if !row.topPlaybookName.isEmpty {
                            Text(row.topPlaybookName).font(.caption)
                        }
                    }
                    .frame(minHeight: 44)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(row.type.isEmpty ? "Exception" : row.type), score \(row.score), \(row.severity), \(row.ageMinutes) minutes old\(row.orderId.isEmpty ? "" : ", order \(row.orderId)")")
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("Control tower")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh scored exceptions")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
