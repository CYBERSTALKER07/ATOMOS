import SwiftUI
import SwiftData

struct SyncQueueView: View {
    @Environment(\.modelContext) private var modelContext
    @State private var vm = SyncQueueViewModel()

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack {
                        Text(L10n.format("mobile_driver.ui.count_pending_count_2_dead_letter", "\(vm.pending.count)", "\(vm.dead.count)"))
                            .font(.subheadline)
                        Spacer()
                        Button(vm.flushing ? "Flushing…" : "Flush now") {
                            Task { await vm.flush() }
                        }
                        .frame(minHeight: 44)
                        .disabled(vm.flushing || vm.pending.isEmpty)
                    }
                    if let status = vm.status {
                        Text(status).font(.caption).foregroundStyle(.secondary)
                    }
                }
                Section("Pending") {
                    if vm.pending.isEmpty {
                        Text("mobile_driver.ui.no_pending_offline_actions").foregroundStyle(.secondary)
                    } else {
                        ForEach(vm.pending, id: \.id) { row in
                            actionRow(row)
                        }
                    }
                }
                Section {
                    if vm.dead.isEmpty {
                        Text("mobile_driver.ui.no_dead_lettered_actions").foregroundStyle(.secondary)
                    } else {
                        ForEach(vm.dead, id: \.id) { row in
                            actionRow(row, dead: true)
                        }
                    }
                } header: {
                    HStack {
                        Text("supplier_portal.settings.integrations.text.dead_letter")
                        Spacer()
                        if !vm.dead.isEmpty {
                            Button("mobile_driver.ui.dismiss_all") {
                                vm.dismissAllDead()
                            }
                            .font(.caption)
                            .frame(minHeight: 44)
                        }
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_driver.ui.sync_queue")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        vm.reload()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            }
            .onAppear {
                vm.attach(container: modelContext.container)
            }
        }
    }

    @ViewBuilder
    private func actionRow(_ row: QueuedDriverAction, dead: Bool = false) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(row.endpoint).font(.headline)
            Text(L10n.format("mobile_driver.ui.order_orderid_attempts_attemptcount", "\(row.orderId.isEmpty ? "—" : row.orderId)", "\(row.attemptCount)"))
                .font(.caption)
                .foregroundStyle(.secondary)
            if !row.clientTimestampIso.isEmpty {
                Text(L10n.format("mobile_driver.ui.client_ts_clienttimestampiso", "\(row.clientTimestampIso)"))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if !row.lastError.isEmpty {
                Text(row.lastError)
                    .font(.caption)
                    .foregroundStyle(dead ? .red : .secondary)
            }
        }
        .padding(.vertical, 2)
        .accessibilityElement(children: .combine)
    }
}
