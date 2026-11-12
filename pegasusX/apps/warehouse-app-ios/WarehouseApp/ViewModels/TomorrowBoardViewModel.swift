import Foundation
import Observation

@Observable
@MainActor
final class TomorrowBoardViewModel {
    typealias FetchBoard = @MainActor (String) async throws -> WarehouseOpsBoardResponse

    private static let utcDateFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter
    }()

    var date = Calendar.current.date(byAdding: .day, value: 1, to: Date()) ?? Date()
    var preorders: [WarehouseOpsBoardOrder] = []
    var deliverBefore: [WarehouseOpsBoardOrder] = []
    var loading = true
    var error: String?

    var dateString: String {
        Self.utcDateFormatter.string(from: date)
    }

    private let fetchBoard: FetchBoard

    init(fetchBoard: FetchBoard? = nil) {
        self.fetchBoard = fetchBoard ?? { date in
            try await WarehouseService.opsBoard(date: date)
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let body = try await fetchBoard(dateString)
            preorders = body.preorders
            deliverBefore = body.deliverBefore
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
