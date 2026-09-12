import Foundation

/// Durable offline POS cash-sale queue (UserDefaults JSON). Survives app restart.
final class PendingPosStore {
    static let shared = PendingPosStore()
    private let key = "retailer_pending_pos_sales_v1"
    private let defaults = UserDefaults.standard
    private let lock = NSLock()

    struct Entry: Codable, Identifiable {
        var id: String { clientSaleId }
        let clientSaleId: String
        let clientReceipt: String
        let sessionId: String
        let payloadJson: String
        let createdAt: TimeInterval
        var retryCount: Int
        var status: String // PENDING | SYNCING | SYNCED | FAILED
        var lastError: String?
    }

    private init() {}

    func list() -> [Entry] {
        lock.lock()
        defer { lock.unlock() }
        guard let data = defaults.data(forKey: key),
              let rows = try? JSONDecoder().decode([Entry].self, from: data) else {
            return []
        }
        return rows.sorted { $0.createdAt < $1.createdAt }
    }

    func active() -> [Entry] {
        list().filter { ["PENDING", "FAILED", "SYNCING"].contains($0.status) }
    }

    func countActive(sessionId: String) -> Int {
        active().filter { $0.sessionId == sessionId }.count
    }

    func enqueue(sessionId: String, lines: [PosSaleLineWire], totalMinor: Int64) throws -> Entry {
        let clientSaleId = UUID().uuidString
        let short = String(Int(Date().timeIntervalSince1970), radix: 36).uppercased()
        let receipt = "OFF-\(short)-\(Int.random(in: 1...999))"
        let payload: [String: Any] = [
            "session_id": sessionId,
            "stock_bin": "FLOOR",
            "origin": "offline",
            "client_sale_id": clientSaleId,
            "client_created_at": ISO8601DateFormatter().string(from: Date()),
            "lines": lines.map { [
                "sku": $0.sku,
                "name": $0.name,
                "qty": $0.qty,
                "unit_price_minor": $0.unit_price_minor,
            ] },
            "tenders": [["method": "CASH", "amount_minor": totalMinor]],
        ]
        let data = try JSONSerialization.data(withJSONObject: payload)
        let entry = Entry(
            clientSaleId: clientSaleId,
            clientReceipt: receipt,
            sessionId: sessionId,
            payloadJson: String(data: data, encoding: .utf8) ?? "{}",
            createdAt: Date().timeIntervalSince1970,
            retryCount: 0,
            status: "PENDING",
            lastError: nil
        )
        var rows = list().filter { $0.clientSaleId != clientSaleId }
        rows.append(entry)
        save(rows)
        return entry
    }

    func update(_ entry: Entry) {
        var rows = list()
        if let i = rows.firstIndex(where: { $0.clientSaleId == entry.clientSaleId }) {
            rows[i] = entry
            save(rows)
        }
    }

    func remove(clientSaleId: String) {
        save(list().filter { $0.clientSaleId != clientSaleId })
    }

    private func save(_ rows: [Entry]) {
        lock.lock()
        defer { lock.unlock() }
        if let data = try? JSONEncoder().encode(rows) {
            defaults.set(data, forKey: key)
        }
    }

    /// Flush active queue to server (cash offline sales).
    func flush(using api: APIClient) async -> (flushed: Int, failed: Int) {
        var flushed = 0
        var failed = 0
        for var entry in active() {
            entry.status = "SYNCING"
            update(entry)
            do {
                guard let data = entry.payloadJson.data(using: .utf8),
                      let obj = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                      let sessionId = obj["session_id"] as? String,
                      let linesRaw = obj["lines"] as? [[String: Any]] else {
                    throw APIError.serverError(statusCode: 400, message: "bad_payload")
                }
                let lines: [PosSaleLineWire] = linesRaw.compactMap { row in
                    guard let sku = row["sku"] as? String else { return nil }
                    let name = row["name"] as? String ?? sku
                    let qty = (row["qty"] as? NSNumber)?.int64Value ?? 1
                    let unit = (row["unit_price_minor"] as? NSNumber)?.int64Value ?? 0
                    return PosSaleLineWire(sku: sku, name: name, qty: qty, unitPriceMinor: unit)
                }
                let total = lines.reduce(Int64(0)) { $0 + $1.qty * $1.unit_price_minor }
                _ = try await api.createPosSale(
                    sessionId: sessionId,
                    lines: lines,
                    totalMinor: total,
                    clientSaleId: entry.clientSaleId,
                    origin: "offline",
                    clientCreatedAt: obj["client_created_at"] as? String
                )
                remove(clientSaleId: entry.clientSaleId)
                flushed += 1
            } catch {
                entry.status = "FAILED"
                entry.retryCount += 1
                entry.lastError = error.localizedDescription
                update(entry)
                failed += 1
            }
        }
        return (flushed, failed)
    }
}
