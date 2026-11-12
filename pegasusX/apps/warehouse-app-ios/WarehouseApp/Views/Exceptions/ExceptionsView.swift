import SwiftUI

struct ExceptionsView: View {
    @State private var vm = ExceptionsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.rows.isEmpty {
                ContentUnavailableView("No open exceptions", systemImage: "exclamationmark.bubble")
            } else {
                List(vm.rows) { row in
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
                            .frame(minHeight: 44)
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
                    .frame(minHeight: 44)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(row.kind), \(row.reason.isEmpty ? (row.status.isEmpty ? "Needs review" : row.status) : row.reason)\(row.orderId.isEmpty ? "" : ", order \(row.orderId)")")
                }
                .listStyle(.plain)
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("warehouse_portal.exceptions.text.exception_triage")
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
                .accessibilityLabel("Refresh exceptions")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
