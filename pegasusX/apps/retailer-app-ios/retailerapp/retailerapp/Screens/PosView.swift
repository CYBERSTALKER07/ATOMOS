import SwiftUI
import Network

struct PosView: View {
    @State private var vm = PosViewModel()
    @State private var network = NetworkLifecycleEngine.shared
    private var isOnline: Bool { network.isOnline }

    private var sessionId: String? { vm.sessionId }
    private var cart: [PosCartLine] { vm.cart }
    private var banner: String? { vm.banner }
    private var busy: Bool { vm.busy }
    private var lastSaleId: String? { vm.lastSaleId }
    private var pending: [PendingPosStore.Entry] { vm.pending }
    private var holdsEnabled: Bool { vm.holdsEnabled }
    private var serverHolds: [PosHoldWire] { vm.serverHolds }
    private var totalMinor: Int64 { vm.totalMinor }

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                Text("mobile_retailer.ui.open_till_online_cash_sales_work_offline_and_sync_on_reconnect_c")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if !network.isOnline {
                Section {
                    Text("mobile_retailer.ui.offline_cash_queue" + (pending.isEmpty ? "" : " · \(pending.count) pending"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.orange)
                }
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Session") {
                if let sid = sessionId {
                    Text(L10n.format("mobile_retailer.ui.open_prefix", "\(sid.prefix(12))"))
                    Button("mobile_retailer.ui.close_session") { Task { await closeSession() } }
                        .disabled(busy || !isOnline)
                    if !pending.isEmpty && isOnline {
                        Button(L10n.format("mobile_retailer.ui.sync_count_offline_sale_s", "\(pending.count)")) {
                            Task { await syncPending() }
                        }
                        .disabled(busy)
                    }
                } else {
                    Button(busy ? "…" : (isOnline ? "Open session" : "Open needs network")) {
                        Task { await openSession() }
                    }
                    .disabled(busy || !isOnline)
                }
            }
            if !pending.isEmpty {
                Section("Offline queue") {
                    ForEach(pending) { p in
                        Text(L10n.format("mobile_retailer.ui.clientreceipt_status", "\(p.clientReceipt)", "\(p.status)") + (p.lastError.map { " · \($0)" } ?? ""))
                            .font(.caption2)
                            .foregroundStyle(p.status == "FAILED" ? .red : AppTheme.textTertiary)
                    }
                }
            }
            if sessionId != nil {
                Section("Add line") {
                    TextField("SKU", text: $vm.sku)
                    TextField("retailer_desktop.pos.text.qty", text: $vm.qty)
                    TextField("mobile_retailer.ui.price_major", text: $vm.priceMajor)
                    Button("mobile_retailer.ui.add_to_cart") { addLine() }
                }
                Section("Cart") {
                    ForEach(cart) { line in
                        HStack {
                            Text(L10n.format("mobile_retailer.ui.sku_qty", "\(line.sku)", "\(line.qty)"))
                            Spacer()
                            Text(vm.formatMinor(line.qty * line.unitPriceMinor))
                        }
                    }
                    Text("Total " + vm.formatMinor(totalMinor))
                        .font(.headline)
                    if holdsEnabled && isOnline {
                        TextField("Hold note", text: $vm.holdNote)
                        Button("Park hold") { Task { await parkHold() } }
                            .disabled(busy || cart.isEmpty || (vm.locationId ?? "").isEmpty)
                    }
                    Button(busy ? "…" : (isOnline ? "Complete cash sale" : "Complete cash sale offline")) {
                        Task { await completeSale() }
                    }
                    .disabled(busy || cart.isEmpty)
                    if lastSaleId != nil && isOnline {
                        Button("mobile_retailer.ui.void_last_sale", role: .destructive) { Task { await voidLast() } }
                    }
                }
            }
            if holdsEnabled && !serverHolds.isEmpty {
                Section("Parked holds") {
                    ForEach(serverHolds, id: \.holdId) { hold in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(
                                String(hold.holdId.prefix(8))
                                    + (hold.note.map { $0.isEmpty ? "" : " · \($0)" } ?? "")
                                    + " · \(hold.cart?.lines?.count ?? 0) lines"
                            )
                            .font(.caption)
                            HStack {
                                Button("Resume") { Task { await resumeHold(hold) } }
                                    .disabled(busy || !isOnline || sessionId == nil || !cart.isEmpty)
                                Button("Void", role: .destructive) { Task { await voidHold(hold) } }
                                    .disabled(busy || !isOnline)
                            }
                        }
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("POS")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await ensureRegister()
            refreshPending()
            if network.isOnline {
                await syncPending()
                await loadServerHolds()
            }
        }
        .onChange(of: network.syncEpoch) { _, _ in
            Task {
                await syncPending()
                await loadServerHolds()
            }
        }
    }

    private func refreshPending() {
        vm.refreshPending()
    }

    private func ensureRegister() async {
        await vm.ensureRegister()
    }

    private func loadServerHolds() async {
        await vm.loadServerHolds()
    }

    private func parkHold() async {
        await vm.parkHold()
    }

    private func resumeHold(_ hold: PosHoldWire) async {
        await vm.resumeHold(hold)
    }

    private func voidHold(_ hold: PosHoldWire) async {
        await vm.voidHold(hold)
    }

    private func openSession() async {
        await vm.openSession()
    }

    private func closeSession() async {
        await vm.closeSession()
    }

    private func addLine() {
        vm.addLine()
    }

    private func completeSale() async {
        await vm.completeSale()
    }

    private func syncPending() async {
        await vm.syncPending()
    }

    private func voidLast() async {
        await vm.voidLast()
    }
}

struct PosCartLine: Identifiable {
    let id: String
    let sku: String
    let name: String
    var qty: Int64
    let unitPriceMinor: Int64
}
