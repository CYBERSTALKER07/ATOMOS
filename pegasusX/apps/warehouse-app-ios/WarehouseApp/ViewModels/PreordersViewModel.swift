import Foundation
import Observation

enum PreorderSheet: Identifiable {
    case propose(WarehousePreorderRow)
    case reject(WarehousePreorderRow)

    var id: String {
        switch self {
        case .propose(let row): return "propose-\(row.orderId)"
        case .reject(let row): return "reject-\(row.orderId)"
        }
    }
}

@Observable
@MainActor
final class PreordersViewModel {
    typealias FetchPreorders = @MainActor () async throws -> WarehousePreordersResponse
    typealias ProposeDelivery = @MainActor (String, String, String) async throws -> WarehouseOrderMutationResponse
    typealias RejectPreorder = @MainActor (String, String) async throws -> WarehouseOrderMutationResponse

    var rows: [WarehousePreorderRow] = []
    var loading = true
    var acting = false
    var activeSheet: PreorderSheet?
    var proposeDate = Date()
    var reasonInput = ""
    var statusMessage: String?

    private let fetchPreorders: FetchPreorders
    private let proposeDelivery: ProposeDelivery
    private let rejectPreorder: RejectPreorder

    init(
        fetchPreorders: FetchPreorders? = nil,
        proposeDelivery: ProposeDelivery? = nil,
        rejectPreorder: RejectPreorder? = nil
    ) {
        self.fetchPreorders = fetchPreorders ?? {
            try await WarehouseService.preorders()
        }
        self.proposeDelivery = proposeDelivery ?? { orderId, proposedDate, reason in
            try await WarehouseOperationsService.proposePreorderDelivery(
                orderId: orderId,
                proposedDeliveryDate: proposedDate,
                reason: reason
            )
        }
        self.rejectPreorder = rejectPreorder ?? { orderId, reason in
            try await WarehouseOperationsService.rejectPreorder(orderId: orderId, reason: reason)
        }
    }

    var trimmedReason: String {
        reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    var hasBlockingError: Bool {
        guard rows.isEmpty, let msg = statusMessage?.lowercased() else { return false }
        return msg.contains("network") || msg.contains("fail") || msg.contains("error")
    }

    func beginPropose(_ row: WarehousePreorderRow) {
        PegasusHaptics.light()
        reasonInput = ""
        proposeDate = Self.initialProposeDate(for: row)
        activeSheet = .propose(row)
    }

    func beginReject(_ row: WarehousePreorderRow) {
        PegasusHaptics.warning()
        reasonInput = ""
        activeSheet = .reject(row)
    }

    func load() async {
        loading = true
        defer { loading = false }
        do {
            let data = try await fetchPreorders()
            rows = data.preorders.isEmpty ? data.items : data.preorders
        } catch {
            rows = []
            statusMessage = error.localizedDescription
        }
    }

    func submitPropose(_ row: WarehousePreorderRow) async {
        let reason = trimmedReason
        guard !reason.isEmpty else { return }
        acting = true
        statusMessage = nil
        defer { acting = false }
        let iso = Self.isoDeliveryDate(from: proposeDate)
        do {
            let response = try await proposeDelivery(row.orderId, iso, reason)
            activeSheet = nil
            statusMessage = "Delivery date proposed · \(response.status)"
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func submitReject(_ row: WarehousePreorderRow) async {
        let reason = trimmedReason
        guard !reason.isEmpty else { return }
        acting = true
        statusMessage = nil
        defer { acting = false }
        do {
            let response = try await rejectPreorder(row.orderId, reason)
            activeSheet = nil
            statusMessage = "Pre-order rejected · \(response.status)"
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    static let tashkentDayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    static let tashkentIsoFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withColonSeparatorInTimeZone]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        return formatter
    }()

    static func initialProposeDate(for row: WarehousePreorderRow) -> Date {
        if let raw = row.requestedDeliveryDate?.prefix(10), !raw.isEmpty {
            if let parsed = tashkentDayFormatter.date(from: String(raw)) {
                return parsed
            }
        }
        return Date()
    }

    static func isoDeliveryDate(from date: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        let day = calendar.dateComponents([.year, .month, .day], from: date)
        var noon = DateComponents()
        noon.year = day.year
        noon.month = day.month
        noon.day = day.day
        noon.hour = 12
        noon.minute = 0
        noon.second = 0
        noon.timeZone = calendar.timeZone
        let normalized = calendar.date(from: noon) ?? date
        return tashkentIsoFormatter.string(from: normalized)
    }
}
