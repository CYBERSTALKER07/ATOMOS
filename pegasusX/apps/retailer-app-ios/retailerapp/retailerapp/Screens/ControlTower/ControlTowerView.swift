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
                    Text("mobile_retailer.ui.live_counts_for_your_shop_never_demo_charts")
                        .font(.system(.footnote, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                    if !generatedAt.isEmpty {
                        Text(L10n.format("mobile_retailer.ui.updated_generatedat_packs_2", "\(generatedAt)", "\(packs)"))
                            .font(.caption2)
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    Button("portal.page.orders.action.refresh") { Task { await load() } }
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red) }
                }
                if loading && tiles.isEmpty {
                    Section { ProgressView() }
                } else if empty && error == nil {
                    Section {
                        Text("retailer_desktop.control_tower.text.no_live_ops_signals_yet")
                            .font(.headline)
                        Text("mobile_retailer.ui.place_orders_enable_stock_pos_open_a_shift_or_create_an_assist_t")
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
            .navigationTitle("warehouse_portal.network_pulse_panel.text.ops_pulse")
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
