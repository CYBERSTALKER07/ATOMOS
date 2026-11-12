import Foundation
import Observation

struct ShiftRow: Identifiable {
    let id: String
    let status: String
    let openingFloatMinor: Int64
    let varianceMinor: Int64?
}

@Observable
@MainActor
final class ShiftsViewModel {
    var clockedIn = false
    var shifts: [ShiftRow] = []
    var floatMinor = "0"
    var closingCash = "0"
    var registerId: String?
    var banner: String?
    var busy = false

    private let api = APIClient.shared

    func formatMinor(_ minor: Int64) -> String {
        let whole = minor / 100
        let fraction = abs(minor % 100)
        return String(format: "%lld.%02lld", whole, fraction)
    }

    func refresh() async {
        do {
            let time = try await api.getTimeEntries()
            clockedIn = time.clockedIn
            let list = try await api.getShifts()
            shifts = list.items.map {
                ShiftRow(
                    id: $0.shiftId,
                    status: $0.status,
                    openingFloatMinor: $0.openingFloatMinor,
                    varianceMinor: $0.varianceMinor
                )
            }
            if registerId == nil {
                let regs = try await api.getRegisters()
                registerId = regs.items.first?.registerId
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    func clockIn() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.clockIn()
            banner = "Clocked in"
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func clockOut() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.clockOut()
            banner = "Clocked out"
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func openShift() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.openShift(
                registerId: registerId,
                openingFloatMinor: Int64(floatMinor) ?? 0
            )
            banner = "Shift opened"
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func closeShift(id: String) async {
        busy = true
        defer { busy = false }
        do {
            let closed = try await api.closeShift(
                shiftId: id,
                closingCashMinor: Int64(closingCash) ?? 0
            )
            if let v = closed.varianceMinor {
                banner = "Closed · variance \(formatMinor(v))"
            } else {
                banner = "Shift closed"
            }
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
