import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class PosViewModel {
    var registerId: String?
    var locationId: String?
    var sessionId: String?
    var cart: [PosCartLine] = []
    var sku = ""
    var priceMajor = "0"
    var qty = "1"
    var banner: String?
    var busy = false
    var lastSaleId: String?
    var pending: [PendingPosStore.Entry] = []
    var holdsEnabled = false
    var serverHolds: [PosHoldWire] = []
    var holdNote = ""

    private let api = APIClient.shared
    private let store = PendingPosStore.shared

    var totalMinor: Int64 {
        cart.reduce(0) { $0 + $1.qty * $1.unitPriceMinor }
    }

    func formatMinor(_ minor: Int64) -> String {
        let whole = minor / 100
        let fraction = abs(minor % 100)
        return String(format: "%lld.%02lld", whole, fraction)
    }

    func parseMajorToMinor(_ string: String) -> Int64 {
        let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
        let parts = trimmed.split(separator: ".")
        if parts.count == 1 {
            return (Int64(parts[0]) ?? 0) * 100
        } else if parts.count == 2 {
            let major = Int64(parts[0]) ?? 0
            let fractionStr = parts[1].prefix(2)
            let padded = fractionStr.padding(toLength: 2, withPad: "0", startingAt: 0)
            let minor = Int64(padded) ?? 0
            return major * 100 + minor
        }
        return 0
    }

    func refreshPending() {
        pending = store.active()
    }

    func ensureRegister() async {
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

    func loadServerHolds() async {
        guard NetworkLifecycleEngine.shared.isOnline else { return }
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

    func linesFromHold(_ hold: PosHoldWire) -> [PosCartLine] {
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

    func parkHold() async {
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

    func resumeHold(_ hold: PosHoldWire) async {
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

    func voidHold(_ hold: PosHoldWire) async {
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

    func openSession() async {
        guard let registerId else { return }
        guard NetworkLifecycleEngine.shared.isOnline else {
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

    func closeSession() async {
        guard let sessionId else { return }
        let active = store.countActive(sessionId: sessionId)
        if active > 0 {
            banner = "Sync \(active) offline sale(s) before close"
            return
        }
        guard NetworkLifecycleEngine.shared.isOnline else {
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

    func addLine() {
        let unit = parseMajorToMinor(priceMajor)
        let q = Int64(qty) ?? 1
        guard !sku.isEmpty, q > 0 else { return }
        if let i = cart.firstIndex(where: { $0.sku == sku }) {
            cart[i].qty += q
        } else {
            cart.append(PosCartLine(id: sku, sku: sku, name: sku, qty: q, unitPriceMinor: unit))
        }
        sku = ""; qty = "1"
    }

    func completeSale() async {
        guard let sessionId else { return }
        busy = true
        defer { busy = false }
        let lines = cart.map {
            PosSaleLineWire(sku: $0.sku, name: $0.name, qty: $0.qty, unitPriceMinor: $0.unitPriceMinor)
        }
        if !NetworkLifecycleEngine.shared.isOnline {
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

    func syncPending() async {
        guard NetworkLifecycleEngine.shared.isOnline else { return }
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

    func voidLast() async {
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
