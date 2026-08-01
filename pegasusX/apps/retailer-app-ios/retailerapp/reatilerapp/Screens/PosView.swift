import SwiftUI

struct PosView: View {
    @State private var registerId: String?
    @State private var sessionId: String?
    @State private var cart: [PosCartLine] = []
    @State private var sku = ""
    @State private var priceMajor = "0"
    @State private var qty = "1"
    @State private var banner: String?
    @State private var busy = false
    @State private var lastSaleId: String?

    private let api = APIClient.shared

    private var totalMinor: Int64 {
        cart.reduce(0) { $0 + $1.qty * $1.unitPriceMinor }
    }

    var body: some View {
        List {
            Section {
                Text("Open till → add SKU lines → complete cash sale (decrements FLOOR stock).")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Session") {
                if sessionId == nil {
                    Button(busy ? "…" : "Open session") { Task { await openSession() } }
                        .disabled(busy)
                } else {
                    Text("Open: \(sessionId!.prefix(12))…")
                    Button("Close session") { Task { await closeSession() } }
                }
            }
            if sessionId != nil {
                Section("Add line") {
                    TextField("SKU", text: $sku)
                    TextField("Qty", text: $qty)
                    TextField("Price (major)", text: $priceMajor)
                    Button("Add to cart") { addLine() }
                }
                Section("Cart") {
                    ForEach(cart) { line in
                        HStack {
                            Text("\(line.sku) × \(line.qty)")
                            Spacer()
                            Text(String(format: "%.2f", Double(line.qty * line.unitPriceMinor) / 100.0))
                        }
                    }
                    Text(String(format: "Total %.2f", Double(totalMinor) / 100.0))
                        .font(.headline)
                    Button(busy ? "…" : "Complete cash sale") { Task { await completeSale() } }
                        .disabled(busy || cart.isEmpty)
                    if lastSaleId != nil {
                        Button("Void last sale", role: .destructive) { Task { await voidLast() } }
                    }
                }
            }
        }
        .navigationTitle("POS")
        .navigationBarTitleDisplayMode(.inline)
        .task { await ensureRegister() }
    }

    private func ensureRegister() async {
        do {
            let regs = try await api.getRegisters()
            if let first = regs.items.first {
                registerId = first.registerId
            } else {
                let created = try await api.createRegister(label: "Register 1")
                registerId = created.registerId
                banner = "Register created"
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func openSession() async {
        guard let registerId else { return }
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
        do {
            let sale = try await api.createPosSale(
                sessionId: sessionId,
                lines: cart.map {
                    PosSaleLineWire(sku: $0.sku, name: $0.name, qty: $0.qty, unitPriceMinor: $0.unitPriceMinor)
                },
                totalMinor: totalMinor
            )
            lastSaleId = sale.saleId
            banner = "Sale \(sale.receiptNumber)"
            cart = []
        } catch {
            banner = error.localizedDescription
        }
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
