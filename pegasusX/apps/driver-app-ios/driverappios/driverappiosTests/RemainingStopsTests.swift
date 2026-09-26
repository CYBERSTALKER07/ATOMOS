import Testing
@testable import driverappios

struct RemainingStopsTests {
    @Test func shopClosedAndFiscalFailedAreFirstClassRemainingStops() {
        let stops = RemainingStops.remaining(from: [
            RemainingStop(id: "a", title: "Open", state: .IN_TRANSIT, sequenceIndex: 1),
            RemainingStop(id: "b", title: "Closed shop", state: .ARRIVED_SHOP_CLOSED, sequenceIndex: 2),
            RemainingStop(id: "c", title: "Fiscal", state: .FISCAL_FAILED, sequenceIndex: 3),
            RemainingStop(id: "d", title: "Done", state: .COMPLETED, sequenceIndex: 4),
            RemainingStop(id: "e", title: "Cancel", state: .CANCELLED, sequenceIndex: 5),
        ])
        #expect(stops.map(\.id) == ["a", "b", "c"])
        #expect(stops.contains { $0.state == .ARRIVED_SHOP_CLOSED && $0.firstClass })
        #expect(stops.contains { $0.state == .FISCAL_FAILED && $0.firstClass })
        #expect(!stops.contains { $0.state == .COMPLETED })
    }

    @Test func firstClassHelpers() {
        #expect(RemainingStops.isFirstClass(.ARRIVED_SHOP_CLOSED))
        #expect(RemainingStops.isFirstClass(.FISCAL_FAILED))
        #expect(!RemainingStops.isFirstClass(.ARRIVED))
        #expect(!RemainingStops.isFirstClass(.IN_TRANSIT))
    }

    @Test func moneyHealthDoesNotInventZeroAsEmptyMixup() {
        let empty = MoneyHealthCounts.from(orders: [])
        #expect(empty.pendingCash == 0)
        #expect(empty.openFiscal == 0)
        #expect(empty.creditLeave == 0)
    }

    @Test func pulseHonestyKeepsPreviousOnFailure() {
        let previous = ["old"]
        let failed = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: previous)
        #expect(failed.events == previous)
        #expect(failed.error == PulseHonesty.failed)
        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: previous)
        #expect(empty.events.isEmpty)
        #expect(empty.error == nil)
    }
}
