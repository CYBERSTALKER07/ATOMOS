import SwiftUI

struct ControlTowerView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var empty = true
    @State private var generatedAt = ""
    @State private var packs = ""
    @State private var tiles: [(label: String, value: String, dest: PulseDest)] = []
    private let api = APIClient.shared

    private enum PulseDest {
        case orders, fulfillment, dock, pos, shifts, assist, stock, reports
    }

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
                            NavigationLink {
                                pulseDestination(tile.dest)
                            } label: {
                                HStack {
                                    Text(tile.label)
                                    Spacer()
                                    Text(tile.value).fontWeight(.semibold)
                                }
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
                ("Open orders", "\(p.openOrders)", .orders),
                ("Fulfillment", "\(p.activeFulfillments)", .fulfillment),
                ("Dock pending", "\(p.dockPending)", .dock),
                ("POS sessions", "\(p.posOpenSessions)", .pos),
                ("Open shifts", "\(p.openShifts)", .shifts),
                ("Assist", "\(p.openAssistTickets)", .assist),
                ("Low stock", "\(p.lowStockSkuBins)", .stock),
                ("Variances", "\(p.shiftVariances7d)", .shifts),
                ("Sales 7d", String(format: "%.2f", Double(p.salesMinor7d) / 100.0), .reports),
            ]
        } catch {
            self.error = error.localizedDescription
        }
    }

    @ViewBuilder
    private func pulseDestination(_ dest: PulseDest) -> some View {
        switch dest {
        case .orders: OrdersView()
        case .fulfillment: DeliveriesHubView()
        case .dock: DockView()
        case .pos: PosView()
        case .shifts: ShiftsView()
        case .assist: AssistView()
        case .stock: StoreStockView()
        case .reports: ReportsProView()
        }
    }
}

#Preview {
    ControlTowerView()
}
