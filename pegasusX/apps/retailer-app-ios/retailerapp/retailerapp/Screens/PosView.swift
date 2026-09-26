import SwiftUI
import Network

struct PosView: View {
    @State private var registerId: String?
    @State private var locationId: String?
    @State private var sessionId: String?
    @State private var cart: [PosCartLine] = []
    @State private var sku = ""
    @State private var priceMajor = "0"
    @State private var qty = "1"
    @State private var banner: String?
    @State private var busy = false
    @State private var lastSaleId: String?
    @State private var pending: [PendingPosStore.Entry] = []
    @State private var isOnline = true
    @State private var pathMonitor = NWPathMonitor()
    @State private var holdsEnabled = false
    @State private var serverHolds: [PosHoldWire] = []
    @State private var holdNote = ""

    private let api = APIClient.shared
    private let store = PendingPosStore.shared

    private var totalMinor: Int64 {
        cart.reduce(0) { $0 + $1.qty * $1.unitPriceMinor }
    }

    var body: some View {
        List {
            Section {
                Text("mobile_retailer.ui.open_till_online_cash_sales_work_offline_and_sync_on_reconnect_c")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if !isOnline {
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
                if sessionId == nil {
                    Button(busy ? "…" : (isOnline ? "Open session" : "Open needs network")) {
                        Task { await openSession() }
                    }
                    .disabled(busy || !isOnline)
                } else {
                    Text(L10n.format("mobile_retailer.ui.open_prefix", "\(sessionId!.prefix(12))"))
                    Button("mobile_retailer.ui.close_session") { Task { await closeSession() } }
                        .disabled(busy || !isOnline)
                    if !pending.isEmpty && isOnline {
                        Button(L10n.format("mobile_retailer.ui.sync_count_offline_sale_s", "\(pending.count)")) {
                            Task { await syncPending() }
                        }
                        .disabled(busy)
                    }
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
                    TextField("SKU", text: $sku)
                    TextField("retailer_desktop.pos.text.qty", text: $qty)
                    TextField("mobile_retailer.ui.price_major", text: $priceMajor)
                    Button("mobile_retailer.ui.add_to_cart") { addLine() }
                }
                Section("Cart") {
                    ForEach(cart) { line in
                        HStack {
                            Text(L10n.format("mobile_retailer.ui.sku_qty", "\(line.sku)", "\(line.qty)"))
                            Spacer()
                            Text(String(format: "%.2f", Double(line.qty * line.unitPriceMinor) / 100.0))
                        }
                    }
                    Text(String(format: "Total %.2f", Double(totalMinor) / 100.0))
                        .font(.headline)
                    if holdsEnabled && isOnline {
                        TextField("Hold note", text: $holdNote)
                        Button("Park hold") { Task { await parkHold() } }
                            .disabled(busy || cart.isEmpty || (locationId ?? "").isEmpty)
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
        .navigationTitle("POS")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await ensureRegister()
            refreshPending()
            startNetworkMonitor()
            if isOnline {
                await syncPending()
                await loadServerHolds()
            }
        }
        .onDisappear { pathMonitor.cancel() }
    }

    private func startNetworkMonitor() {
        pathMonitor.pathUpdateHandler = { path in
            DispatchQueue.main.async {
                isOnline = path.status == .satisfied
                if isOnline {
                    Task { await syncPending() }
                }
            }
        }
        pathMonitor.start(queue: DispatchQueue(label: "pos.network"))
    }

    private func refreshPending() {
        pending = store.active()
    }

    private func ensureRegister() async {
        do {
            let regs = try await api.getRegisters()
            if let first = regs.items.first {
                registerId = first.registerId
                locationId = first.locationId
            } else {
                let created = try await api.createRegister(label: "Register 1")
                registerId = created.registerId
                locationId = created.locationId
                banner = "Register created"
            }
            await loadServerHolds()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func loadServerHolds() async {
        guard isOnline else { return }
        do {
            let list = try await api.listPosHolds(locationId: locationId)
            holdsEnabled = true
            serverHolds = list.items
        } catch let APIError.serverError(statusCode, message) {
            if statusCode == 404 || message.contains("POS_HOLDS_DISABLED") {
                holdsEnabled = false
                serverHolds = []
            }
        } catch {
            let msg = error.localizedDescription
            if msg.contains("404") || msg.contains("POS_HOLDS_DISABLED") {
                holdsEnabled = false
                serverHolds = []
            }
        }
    }

    private func linesFromHold(_ hold: PosHoldWire) -> [PosCartLine] {
        (hold.cart?.lines ?? []).compactMap { line in
            guard !line.sku.isEmpty else { return nil }
            return PosCartLine(
                id: line.sku,
                sku: line.sku,
                name: line.name ?? line.sku,
                qty: line.qty ?? 1,
                unitPriceMinor: line.unitPriceMinor ?? 0
            )
        }
    }

    private func parkHold() async {
        guard let registerId, let locationId, !locationId.isEmpty else {
            banner = "Location required to park hold"
            return
        }
        busy = true
        defer { busy = false }
        let lines = cart.map {
            PosSaleLineWire(sku: $0.sku, name: $0.name, qty: $0.qty, unitPriceMinor: $0.unitPriceMinor)
        }
        do {
            _ = try await api.parkPosHold(
                locationId: locationId,
                registerId: registerId,
                lines: lines,
                note: holdNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : holdNote
            )
            cart = []
            holdNote = ""
            banner = "Cart parked on server (no stock held)"
            await loadServerHolds()
        } catch {
            let msg = error.localizedDescription
            if msg.contains("404") || msg.contains("POS_HOLDS_DISABLED") {
                holdsEnabled = false
                banner = "Server holds disabled"
            } else {
                banner = msg
            }
        }
    }

    private func resumeHold(_ hold: PosHoldWire) async {
        guard cart.isEmpty else {
            banner = "Clear or park current cart before resuming"
            return
        }
        busy = true
        defer { busy = false }
        do {
            let resumed = try await api.resumePosHold(
                holdId: hold.holdId,
                locationId: locationId ?? hold.locationId ?? ""
            )
            let lines = linesFromHold(resumed)
            cart = lines.isEmpty ? linesFromHold(hold) : lines
            banner = "Resumed hold \(hold.holdId.prefix(8))"
            await loadServerHolds()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func voidHold(_ hold: PosHoldWire) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.voidPosHold(holdId: hold.holdId)
            banner = "Hold voided"
            await loadServerHolds()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func openSession() async {
        guard let registerId else { return }
        guard isOnline else {
            banner = "Open session requires network"
            return
        }
        busy = true
        defer { busy = false }
        do {
            let sess = try await api.openPosSession(registerId: registerId)
            sessionId = sess.sessionId
            cart = []
            banner = "Session open"
        } catch {
            banner = error.localizedDescription
        }
    }

    private func closeSession() async {
        guard let sessionId else { return }
        let active = store.countActive(sessionId: sessionId)
        if active > 0 {
            banner = "Sync \(active) offline sale(s) before close"
            return
        }
        guard isOnline else {
            banner = "Close session requires network"
            return
        }
        busy = true
        defer { busy = false }
        do {
            _ = try await api.closePosSession(sessionId: sessionId, closingCashMinor: 0)
            self.sessionId = nil
            banner = "Session closed"
        } catch {
            banner = error.localizedDescription
        }
    }

    private func addLine() {
        let unit = Int64((Double(priceMajor) ?? 0) * 100)
        let q = Int64(qty) ?? 1
        guard !sku.isEmpty, q > 0 else { return }
        if let i = cart.firstIndex(where: { $0.sku == sku }) {
            cart[i].qty += q
        } else {
            cart.append(PosCartLine(id: sku, sku: sku, name: sku, qty: q, unitPriceMinor: unit))
        }
        sku = ""; qty = "1"
    }

    private func completeSale() async {
        guard let sessionId else { return }
        busy = true
        defer { busy = false }
        let lines = cart.map {
            PosSaleLineWire(sku: $0.sku, name: $0.name, qty: $0.qty, unitPriceMinor: $0.unitPriceMinor)
        }
        if !isOnline {
            do {
                let entry = try store.enqueue(sessionId: sessionId, lines: lines, totalMinor: totalMinor)
                lastSaleId = entry.clientSaleId
                banner = "Offline \(entry.clientReceipt) · will sync"
                cart = []
                refreshPending()
            } catch {
                banner = error.localizedDescription
            }
            return
        }
        do {
            let sale = try await api.createPosSale(
                sessionId: sessionId,
                lines: lines,
                totalMinor: totalMinor,
                origin: "online"
            )
            lastSaleId = sale.saleId
            banner = "Sale \(sale.receiptNumber)"
            cart = []
        } catch {
            // Network fail → queue offline cash
            do {
                let entry = try store.enqueue(sessionId: sessionId, lines: lines, totalMinor: totalMinor)
                lastSaleId = entry.clientSaleId
                banner = "Queued offline \(entry.clientReceipt)"
                cart = []
                refreshPending()
            } catch {
                banner = error.localizedDescription
            }
        }
    }

    private func syncPending() async {
        guard isOnline else { return }
        busy = true
        defer { busy = false }
        let result = await store.flush(using: api)
        if result.flushed > 0 {
            banner = "Synced \(result.flushed) offline sale(s)"
        } else if result.failed > 0 {
            banner = "\(result.failed) offline sale(s) failed"
        }
        refreshPending()
    }

    private func voidLast() async {
        guard let lastSaleId else { return }
        do {
            _ = try await api.voidPosSale(saleId: lastSaleId)
            banner = "Voided"
            self.lastSaleId = nil
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct PosCartLine: Identifiable {
    let id: String
    let sku: String
    let name: String
    var qty: Int64
    let unitPriceMinor: Int64
}
