import SwiftUI
import SwiftData

struct SyncQueueView: View {
    @Environment(\.modelContext) private var modelContext
    @State private var pending: [QueuedDriverAction] = []
    @State private var dead: [QueuedDriverAction] = []
    @State private var status: String?
    @State private var flushing = false

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack {
                        Text(L10n.format("mobile_driver.ui.count_pending_count_2_dead_letter", "\(pending.count)", "\(dead.count)"))
                            .font(.subheadline)
                        Spacer()
                        Button(flushing ? "Flushing…" : "Flush now") {
                            Task { await flush() }
                        }
                        .disabled(flushing || pending.isEmpty)
                    }
                    if let status {
                        Text(status).font(.caption).foregroundStyle(.secondary)
                    }
                }
                Section("Pending") {
                    if pending.isEmpty {
                        Text("mobile_driver.ui.no_pending_offline_actions").foregroundStyle(.secondary)
                    } else {
                        ForEach(pending, id: \.id) { row in
                            actionRow(row)
                        }
                    }
                }
                Section {
                    if dead.isEmpty {
                        Text("mobile_driver.ui.no_dead_lettered_actions").foregroundStyle(.secondary)
                    } else {
                        ForEach(dead, id: \.id) { row in
                            actionRow(row, dead: true)
                        }
                    }
                } header: {
                    HStack {
                        Text("supplier_portal.settings.integrations.text.dead_letter")
                        Spacer()
                        if !dead.isEmpty {
                            Button("mobile_driver.ui.dismiss_all") {
                                DriverOfflineQueue.shared.clearDead()
                                reload()
                            }
                            .font(.caption)
                        }
                    }
                }
            }
            .navigationTitle("mobile_driver.ui.sync_queue")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { reload() }
                }
            }
            .onAppear {
                DriverOfflineQueue.shared.attach(container: modelContext.container)
                reload()
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
    }

    private func reload() {
        pending = DriverOfflineQueue.shared.pending()
        dead = DriverOfflineQueue.shared.dead()
    }

    private func flush() async {
        flushing = true
        defer { flushing = false }
        let result = await DriverOfflineQueue.shared.flush(api: .shared)
        status = "Sent \(result.sent), remaining \(result.remaining)"
        reload()
    }
}
