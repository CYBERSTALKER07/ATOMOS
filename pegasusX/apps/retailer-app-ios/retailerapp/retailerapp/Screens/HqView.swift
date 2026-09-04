import SwiftUI

struct HqView: View {
    @State private var summaryLine = "—"
    @State private var locations: [HqLocationWire] = []
    @State private var loadError: String?
    @State private var summaryReady = false
    private let api = APIClient.shared
    private var day: String {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withFullDate]
        return String(f.string(from: Date()).prefix(10))
    }

    var body: some View {
        List {
            if let loadError {
                Section { Text(loadError).font(.caption).foregroundStyle(AppTheme.destructive) }
            }
            if summaryReady && loadError == nil {
                Section("Summary") {
                    Text(summaryLine)
                }
                Section("Sales by location") {
                    if locations.isEmpty {
                        Text("No HQ rows for this day.")
                            .foregroundStyle(AppTheme.textSecondary)
                    } else {
                        ForEach(locations, id: \.locationId) { loc in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(loc.locationId).font(.headline)
                                Text(String(format: "Qty %d · Net %.2f", loc.qtySold, Double(loc.netMinor) / 100.0))
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("HQ multi-store")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        do {
            let summary = try await api.getHqSummary(day: day)
            summaryLine = String(
                format: "%@ · %d locations · %d SKUs · net %.2f",
                day,
                summary.locationCount,
                summary.skuCount,
                Double(summary.netMinor) / 100.0
            )
            locations = try await api.getHqSalesByLocation(day: day)
            summaryReady = true
            loadError = nil
        } catch {
            loadError = "hq_failed"
        }
    }
}

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
