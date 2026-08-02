import SwiftUI

struct ControlTowerView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var empty = true
    @State private var generatedAt = ""
    @State private var packs = ""
    @State private var tiles: [(label: String, value: String)] = []
    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Text("Live counts for your shop — never demo charts.")
                        .font(.system(.footnote, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                    if !generatedAt.isEmpty {
                        Text("Updated \(generatedAt) · \(packs)")
                            .font(.caption2)
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    Button("Refresh") { Task { await load() } }
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red) }
                }
                if loading && tiles.isEmpty {
                    Section { ProgressView() }
                } else if empty && error == nil {
                    Section {
                        Text("No live ops signals yet")
                            .font(.headline)
                        Text("Place orders, enable stock/POS, open a shift, or create an assist ticket. This stays empty until real activity exists.")
                            .font(.footnote)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                } else {
                    Section("Pulse") {
                        ForEach(tiles, id: \.label) { tile in
                            HStack {
                                Text(tile.label)
                                Spacer()
                                Text(tile.value).fontWeight(.semibold)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Ops pulse")
            .navigationBarTitleDisplayMode(.inline)
            .task { await load() }
            .refreshable { await load() }
        }
        .preferredColorScheme(.dark)
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let p = try await api.getControlTowerPulse()
            empty = p.empty
            generatedAt = String((p.generatedAt ?? "").prefix(19))
            packs = (p.capabilities ?? ["CORE"]).joined(separator: ", ")
            tiles = [
                ("Open orders", "\(p.openOrders)"),
                ("Fulfillment", "\(p.activeFulfillments)"),
                ("Dock pending", "\(p.dockPending)"),
                ("POS sessions", "\(p.posOpenSessions)"),
                ("Open shifts", "\(p.openShifts)"),
                ("Assist", "\(p.openAssistTickets)"),
                ("Low stock", "\(p.lowStockSkuBins)"),
                ("Variances", "\(p.shiftVariances7d)"),
                ("Sales 7d", String(format: "%.2f", Double(p.salesMinor7d) / 100.0)),
            ]
        } catch {
            self.error = error.localizedDescription
        }
    }
}

#Preview {
    ControlTowerView()
}
