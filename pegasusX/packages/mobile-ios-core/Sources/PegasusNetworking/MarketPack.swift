import Foundation
import SwiftUI

struct MarketPack: Sendable, Equatable {
    var code: String
    var name: String
    var timezone: String
    var currencyCode: String
    var fiscalAdapter: String
    var mapsAdapter: String
    var mapCenterLat: Double
    var mapCenterLng: Double
    var checkoutReadsThis: Bool

    var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
}

struct AuthSession: Sendable, Equatable {
    var marketCode: String
    var homeCell: String
    var apiUrl: String
    var pack: MarketPack?
    var checkoutReadsThis: Bool
}

func fiscalReceiptLabel(_ adapter: String?) -> String {
    switch (adapter ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased() {
    case "MY_SOLIQ": return "Soliq"
    case "COMMERCIAL", "PEGASUS", "FAKE": return "commercial"
    case "PEPPOL": return "PEPPOL"
    case "PLANNED", "": return "planned"
    default: return (adapter ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
    }
}

func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
    let code = pack?.currencyCode.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    return code.isEmpty ? fallback : code.uppercased()
}

/// Empty pack prints the number only. Never invents UZS.
func formatPackMoney(_ minor: Int64, pack: MarketPack?, decimalPlaces: Int = 2) -> String {
    let denom = pow(10.0, Double(max(decimalPlaces, 0)))
    let units = Double(minor) / denom
    let formatted = units.formatted(.number.precision(.fractionLength(0...max(decimalPlaces, 0))))
        .replacingOccurrences(of: ",", with: " ")
    let ccy = packCurrency(pack)
    return ccy.isEmpty ? formatted : "\(formatted) \(ccy)"
}

/// Stored/event currency, else session pack. Empty pack does not invent UZS.
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
    let fromEvent = (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    if !fromEvent.isEmpty { return fromEvent.uppercased() }
    return packCurrency(pack)
}

/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
    guard let pack else { return nil }
    if pack.mapCenterLat == 0 && pack.mapCenterLng == 0 { return nil }
    return (pack.mapCenterLat, pack.mapCenterLng)
}

func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
    packMapCenter(pack) ?? (0, 0)
}

func pinnedAPIBaseURL(bootstrap: URL) -> URL {
    let raw = CellApi.pinApiBaseUrl(
        bootstrap: bootstrap.absoluteString,
        homeCell: CellApi.homeCellFromJwt(CellTokenCache.token),
        sessionApiUrl: MarketPackStore.sessionApiUrl
    )
    let withSlash = raw.hasSuffix("/") ? raw : raw + "/"
    return URL(string: withSlash) ?? bootstrap
}

enum CellTokenCache {
    private static let lock = NSLock()
    private static var value = ""

    static var token: String {
        get { lock.lock(); defer { lock.unlock() }; return value }
        set { lock.lock(); value = newValue; lock.unlock() }
    }
}

enum MarketPackStore {
    private static let lock = NSLock()
    private static var session: AuthSession?

    static var current: AuthSession? {
        lock.lock(); defer { lock.unlock() }
        return session
    }

    static var pack: MarketPack? { current?.pack }

    static var sessionApiUrl: String? {
        let url = current?.apiUrl.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return url.isEmpty ? nil : url
    }

    static func set(_ next: AuthSession?) {
        lock.lock(); session = next; lock.unlock()
    }

    static func clear() {
        set(nil)
    }
}

enum MarketPackBinder {
    static func fetch(baseUrl: String, token: String) async -> AuthSession? {
        guard !token.isEmpty else { return nil }
        let pinned = CellApi.pinApiBaseUrl(
            bootstrap: baseUrl,
            homeCell: CellApi.homeCellFromJwt(token),
            sessionApiUrl: MarketPackStore.sessionApiUrl
        )
        guard let url = URL(string: "\(pinned)/v1/auth/session") else { return nil }
        var req = URLRequest(url: url)
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        guard let (data, resp) = try? await URLSession.shared.data(for: req),
              let http = resp as? HTTPURLResponse,
              (200...299).contains(http.statusCode),
              let parsed = parse(data)
        else { return nil }
        MarketPackStore.set(parsed)
        return parsed
    }

    static func parse(_ data: Data) -> AuthSession? {
        guard let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        var pack: MarketPack?
        if let obj = root["pack"] as? [String: Any] {
            pack = MarketPack(
                code: obj["code"] as? String ?? "",
                name: obj["name"] as? String ?? "",
                timezone: obj["timezone"] as? String ?? "",
                currencyCode: obj["currency_code"] as? String ?? "",
                fiscalAdapter: obj["fiscal_adapter"] as? String ?? "",
                mapsAdapter: obj["maps_adapter"] as? String ?? "",
                mapCenterLat: (obj["map_center_lat"] as? NSNumber)?.doubleValue ?? 0,
                mapCenterLng: (obj["map_center_lng"] as? NSNumber)?.doubleValue ?? 0,
                checkoutReadsThis: obj["checkout_reads_this"] as? Bool ?? false
            )
        }
        return AuthSession(
            marketCode: root["market_code"] as? String ?? "",
            homeCell: root["home_cell"] as? String ?? "",
            apiUrl: root["api_url"] as? String ?? "",
            pack: pack,
            checkoutReadsThis: root["checkout_reads_this"] as? Bool ?? false
        )
    }
}

/// GS-R splash: currency + receipts.
struct PackBanner: View {
    let pack: MarketPack?

    var body: some View {
        if let pack, !pack.currencyCode.isEmpty {
            Text("\(pack.currencyCode) · receipts: \(pack.receiptLabel)")
                .font(.caption)
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .overlay(Capsule().strokeBorder(.secondary, lineWidth: 1))
                .accessibilityIdentifier("gs-r-pack-chip")
        }
    }
}
