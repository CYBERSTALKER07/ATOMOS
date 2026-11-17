import Foundation
import Observation

@Observable
@MainActor
final class LaborCapacityViewModel {
    var date: String
    var zones: [LaborZoneCapacity] = []
    var loading = true
    var error: String?
    var driverId = ""
    var score: LaborDriverScore?
    var availHours = "8"
    var availStatus = "AVAILABLE"
    var zoneH3 = ""
    var saving = false
    var statusMessage: String?

    private static let utcDateFormatter: DateFormatter = {
        let f = DateFormatter()
        f.calendar = Calendar(identifier: .gregorian)
        f.locale = Locale(identifier: "en_US_POSIX")
        f.timeZone = TimeZone(secondsFromGMT: 0)
        f.dateFormat = "yyyy-MM-dd"
        return f
    }()

    static func todayUTC() -> String {
        utcDateFormatter.string(from: Date())
    }

    init() {
        self.date = Self.utcDateFormatter.string(from: Date())
    }

    func loadZones() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await WarehouseService.laborZoneCapacity(date: date.trimmingCharacters(in: .whitespacesAndNewlines))
            zones = resp.zones
        } catch {
            zones = []
            self.error = error.localizedDescription
        }
    }

    func loadScore() async {
        let id = driverId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            error = "Driver ID required"
            return
        }
        do {
            score = try await WarehouseService.laborDriverScore(driverId: id)
            error = nil
        } catch {
            score = nil
            self.error = "Driver score not found"
        }
    }

    func saveAvailability() async {
        let id = driverId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            error = "Driver ID required"
            return
        }
        saving = true
        statusMessage = nil
        defer { saving = false }
        do {
            let body = LaborDriverAvailabilityRequest(
                driverId: id,
                date: date.trimmingCharacters(in: .whitespacesAndNewlines),
                availableHours: Double(availHours.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0,
                status: availStatus.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? "AVAILABLE"
                    : availStatus.trimmingCharacters(in: .whitespacesAndNewlines),
                zoneH3: zoneH3.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            _ = try await WarehouseService.setLaborDriverAvailability(body)
            statusMessage = "Availability saved"
            await loadZones()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
