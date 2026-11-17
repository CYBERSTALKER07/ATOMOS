import Foundation
import Observation

struct HqSummaryWire: Decodable {
    let locationCount: Int
    let skuCount: Int
    let netMinor: Int64

    enum CodingKeys: String, CodingKey {
        case locationCount = "location_count"
        case skuCount = "sku_count"
        case netMinor = "net_minor"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        locationCount = try c.decodeIfPresent(Int.self, forKey: .locationCount) ?? 0
        skuCount = try c.decodeIfPresent(Int.self, forKey: .skuCount) ?? 0
        netMinor = try c.decodeIfPresent(Int64.self, forKey: .netMinor) ?? 0
    }
}

struct HqLocationWire: Decodable {
    let locationId: String
    let qtySold: Int
    let netMinor: Int64

    enum CodingKeys: String, CodingKey {
        case locationId = "location_id"
        case qtySold = "qty_sold"
        case netMinor = "net_minor"
    }

    init(locationId: String, qtySold: Int, netMinor: Int64) {
        self.locationId = locationId
        self.qtySold = qtySold
        self.netMinor = netMinor
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        locationId = try c.decodeIfPresent(String.self, forKey: .locationId) ?? "—"
        qtySold = try c.decodeIfPresent(Int.self, forKey: .qtySold) ?? 0
        netMinor = try c.decodeIfPresent(Int64.self, forKey: .netMinor) ?? 0
    }
}

struct HqLocationsResponse: Decodable {
    let items: [HqLocationWire]?
}

@Observable
@MainActor
final class HqViewModel {
    var summaryLine = "—"
    var locations: [HqLocationWire] = []
    var loadError: String?
    var summaryReady = false
    var isLoading = false

    private let api = APIClient.shared
    private static let dayFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withFullDate]
        return formatter
    }()

    var day: String {
        String(Self.dayFormatter.string(from: Date()).prefix(10))
    }

    func formatMinor(_ minor: Int64) -> String {
        let isNegative = minor < 0
        let whole = (minor / 100).magnitude
        let fraction = (minor % 100).magnitude
        return String(format: "%@%llu.%02llu", isNegative ? "-" : "", whole, fraction)
    }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        let currentDay = day
        do {
            let summary = try await api.getHqSummary(day: currentDay)
            summaryLine = "\(currentDay) · \(summary.locationCount) locations · \(summary.skuCount) SKUs · net \(formatMinor(summary.netMinor))"
            locations = try await api.getHqSalesByLocation(day: currentDay)
            summaryReady = true
            loadError = nil
        } catch {
            locations = []
            summaryReady = false
            loadError = error.localizedDescription
        }
    }
}
