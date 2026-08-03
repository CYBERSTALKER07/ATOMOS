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
                        Text("\(pending.count) pending · \(dead.count) dead-letter")
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
                        Text("No pending offline actions.").foregroundStyle(.secondary)
                    } else {
                        ForEach(pending, id: \.id) { row in
                            actionRow(row)
                        }
                    }
                }
                Section {
                    if dead.isEmpty {
                        Text("No dead-lettered actions.").foregroundStyle(.secondary)
                    } else {
                        ForEach(dead, id: \.id) { row in
                            actionRow(row, dead: true)
                        }
                    }
                } header: {
                    HStack {
                        Text("Dead letter")
                        Spacer()
                        if !dead.isEmpty {
                            Button("Dismiss all") {
                                DriverOfflineQueue.shared.clearDead()
                                reload()
                            }
                            .font(.caption)
                        }
                    }
                }
            }
            .navigationTitle("Sync Queue")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { reload() }
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
            Text("Order \(row.orderId.isEmpty ? "—" : row.orderId) · attempts \(row.attemptCount)")
                .font(.caption)
                .foregroundStyle(.secondary)
            if !row.clientTimestampIso.isEmpty {
                Text("client_ts \(row.clientTimestampIso)")
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
